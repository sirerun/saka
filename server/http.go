// Package server implements the saka HTTP and MCP servers.
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	saka "github.com/sirerun/saka"
)

// Options configures optional server behavior.
type Options struct {
	Keys     KeySource   // nil = open access (free/self-hosted mode)
	AdminKey string      // Bearer token that can dump full /v1/usage
	Usage    *UsageStats // nil = usage tracking disabled
}

type Server struct {
	engine saka.Searcher
	opts   Options
}

// New is a plain constructor with no auth/usage.
func New(engine saka.Searcher) *Server {
	return &Server{engine: engine}
}

// NewWithOptions constructs a Server with optional auth and usage.
func NewWithOptions(engine saka.Searcher, opts Options) *Server {
	return &Server{engine: engine, opts: opts}
}

// Handler returns the HTTP mux, optionally wrapped with API-key auth.
//
// /health is always open. When Keys is set, /v1/usage still accepts the
// configured AdminKey Bearer token without going through SignedKeys
// verification (billing jobs are not API consumers).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/search", s.handleSearch)
	mux.HandleFunc("/v1/fetch", s.handleFetch)
	mux.HandleFunc("/v1/stream", s.handleStream)
	if s.opts.Usage != nil {
		mux.Handle("/v1/usage", s.opts.Usage.Handler(s.opts.AdminKey))
	}
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})
	if s.opts.Keys == nil {
		return mux
	}
	keys := s.opts.Keys
	if s.opts.Usage != nil {
		keys = recordingKeySource{inner: keys, stats: s.opts.Usage}
	}
	protected := AuthMiddleware(keys, mux)
	adminKey := s.opts.AdminKey
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			mux.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/v1/usage" && adminKey != "" {
			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if ok && token == adminKey {
				mux.ServeHTTP(w, r)
				return
			}
		}
		protected.ServeHTTP(w, r)
	})
}

func (s *Server) record(r *http.Request, field func(*KeyUsage)) {
	if s.opts.Usage == nil {
		return
	}
	if key := APIKeyFromContext(r.Context()); key != "" {
		s.opts.Usage.Record(key, field)
	}
}

// GET /v1/search?q=&n=&format=json|markdown
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		http.Error(w, `{"error":"missing q param"}`, http.StatusBadRequest)
		s.record(r, func(ku *KeyUsage) { ku.Errors4xx++ })
		return
	}
	n := 10
	if ns := r.URL.Query().Get("n"); ns != "" {
		v, err := strconv.Atoi(ns)
		if err != nil || v <= 0 {
			http.Error(w, `{"error":"invalid n param"}`, http.StatusBadRequest)
			s.record(r, func(ku *KeyUsage) { ku.Errors4xx++ })
			return
		}
		n = v
	}
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	res, err := s.engine.Search(r.Context(), saka.Query{Text: q, MaxResults: n})
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadGateway)
		s.record(r, func(ku *KeyUsage) { ku.Errors5xx++ })
		return
	}
	s.record(r, func(ku *KeyUsage) { ku.Searches++ })

	switch format {
	case "markdown":
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		var b strings.Builder
		fmt.Fprintf(&b, "# Results for %q _(via %s)_\n\n", res.Query, res.Provider)
		for _, hit := range res.Results {
			fmt.Fprintf(&b, "%d. **[%s](%s)**\n   > %s\n\n", hit.Position, hit.Title, hit.URL, hit.Snippet)
		}
		w.Write([]byte(b.String()))
	case "json":
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	default:
		http.Error(w, `{"error":"format must be json or markdown"}`, http.StatusBadRequest)
		s.record(r, func(ku *KeyUsage) { ku.Errors4xx++ })
	}
}

// GET /v1/fetch?url=&format=text|json|markdown
func (s *Server) handleFetch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rawURL := strings.TrimSpace(r.URL.Query().Get("url"))
	if rawURL == "" {
		http.Error(w, `{"error":"missing url param"}`, http.StatusBadRequest)
		s.record(r, func(ku *KeyUsage) { ku.Errors4xx++ })
		return
	}
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "text"
	}

	page, err := s.engine.Fetch(r.Context(), rawURL)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadGateway)
		s.record(r, func(ku *KeyUsage) { ku.Errors5xx++ })
		return
	}
	s.record(r, func(ku *KeyUsage) { ku.Fetches++ })

	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(page)
	case "markdown":
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		fmt.Fprintf(w, "# %s\n\n_Source: %s_\n\n%s\n", page.Title, page.URL, page.Text)
	case "text":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(page.Text))
	default:
		http.Error(w, `{"error":"format must be text, json, or markdown"}`, http.StatusBadRequest)
		s.record(r, func(ku *KeyUsage) { ku.Errors4xx++ })
	}
}

// GET /v1/stream?url=... — Server-Sent Events of extracted text chunks.
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		http.Error(w, "missing url param", http.StatusBadRequest)
		s.record(r, func(ku *KeyUsage) { ku.Errors4xx++ })
		return
	}
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	s.record(r, func(ku *KeyUsage) { ku.Streams++ })

	chunks, _, errCh := s.engine.FetchStream(r.Context(), rawURL)
	for {
		select {
		case c, ok := <-chunks:
			if !ok {
				fmt.Fprint(w, "event: done\ndata: {}\n\n")
				fl.Flush()
				return
			}
			b, _ := json.Marshal(c)
			fmt.Fprintf(w, "event: chunk\ndata: %s\n\n", b)
			fl.Flush()
		case err := <-errCh:
			if err != nil {
				fmt.Fprintf(w, "event: error\ndata: %q\n\n", err.Error())
				fl.Flush()
				s.record(r, func(ku *KeyUsage) { ku.Errors5xx++ })
			}
			return
		case <-r.Context().Done():
			return
		}
	}
}
