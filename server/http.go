package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	saka "github.com/you/saka"
)

// NOTE: the source chat never wrote handleSearch/handleFetch, or the
// original single-arg `New` constructor — only handleStream (v1.0),
// then Options/NewWithOptions/Handler (v1.1, for optional API-key auth),
// then a `usage *UsageStats` field implied by usage.go's wiring (v1.2).
// This struct is assembled from all three passes; handleSearch and
// handleFetch are still exactly what the chat left them: described in
// prose (GET /v1/search, GET /v1/fetch) but never coded. See NOTES.md.

// Options configures optional server behavior.
type Options struct {
	Keys     KeySource   // nil = open access (free/self-hosted mode)
	AdminKey string      // referenced by usage.go's /v1/usage wiring, added here so it compiles
	Usage    *UsageStats // nil = usage tracking disabled
}

type Server struct {
	engine saka.Searcher
	opts   Options
}

// New is a plain constructor with no auth/usage — equivalent to
// NewWithOptions(engine, Options{}).
func New(engine saka.Searcher) *Server {
	return &Server{engine: engine}
}

func NewWithOptions(engine saka.Searcher, opts Options) *Server {
	return &Server{engine: engine, opts: opts}
}

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
	if s.opts.Keys != nil {
		return AuthMiddleware(s.opts.Keys, mux)
	}
	return mux // keyless self-hosted mode
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	// TODO: not written out in the source chat — wire up saka.Query from
	// r.URL.Query() (q, n, format) and call s.engine.Search.
	http.Error(w, "not implemented in source chat", http.StatusNotImplemented)
}

func (s *Server) handleFetch(w http.ResponseWriter, r *http.Request) {
	// TODO: not written out in the source chat — call s.engine.Fetch and
	// render text/markdown/json per the ?format= param.
	http.Error(w, "not implemented in source chat", http.StatusNotImplemented)
}

// GET /v1/stream?url=... — Server-Sent Events of extracted text chunks.
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		http.Error(w, "missing url param", http.StatusBadRequest)
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
			fmt.Fprintf(w, "event: error\ndata: %q\n\n", err.Error())
			fl.Flush()
			return
		case <-r.Context().Done():
			return
		}
	}
}
