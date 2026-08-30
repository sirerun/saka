// Package types holds shared leaf contracts for saka subpackages.
// Subpackages (chain, fetch, providers) import types only — never the
// root saka package — to avoid import cycles.
package types

import (
	"context"
	"errors"
	"time"
)

// ErrNoResults is returned when all providers fail or return nothing.
var ErrNoResults = errors.New("saka: no results from any provider")

// Query describes a web search.
type Query struct {
	Text       string
	MaxResults int    // default 10
	Region     string // e.g. "us-en"
	SafeSearch bool
	Site       string // restrict to host
	// Vertical selects a non-general search vertical (e.g. "news",
	// "images"). Empty means the general web chain. See
	// docs/adr/003-search-verticals.md.
	Vertical string
}

// WithDefaults returns a copy with MaxResults set when unset.
func (q Query) WithDefaults() Query {
	if q.MaxResults <= 0 {
		q.MaxResults = 10
	}
	return q
}

// Result is a single search hit.
type Result struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	Snippet  string `json:"snippet"`
	Source   string `json:"source"`
	Position int    `json:"position"`
}

// Results is a search response.
type Results struct {
	Query    string   `json:"query"`
	Results  []Result `json:"results"`
	Provider string   `json:"provider"`
	TookMs   int64    `json:"took_ms"`
}

// Chunk is an incremental piece of extracted page text.
type Chunk struct {
	Text string `json:"text"`
	Seq  int    `json:"seq"`
	Done bool   `json:"done,omitempty"`
	Err  string `json:"error,omitempty"`
}

const chunkSize = 900

// Page is fetched, extracted page content.
type Page struct {
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Text        string    `json:"text"`
	PublishedAt time.Time `json:"published_at,omitempty"`
}

// Chunks returns the page text in ordered chunks.
func (p *Page) Chunks() (<-chan Chunk, error) {
	src := p.Text
	if src == "" {
		return nil, errors.New("saka: page has no text")
	}
	out := make(chan Chunk, 8)
	go func() {
		defer close(out)
		for i, part := range splitChunks(src, chunkSize) {
			out <- Chunk{Text: part, Seq: i}
		}
		out <- Chunk{Done: true}
	}()
	return out, nil
}

func splitChunks(s string, size int) []string {
	var out []string
	runes := []rune(s)
	for i := 0; i < len(runes); i += size {
		end := i + size
		if end > len(runes) {
			end = len(runes)
		}
		out = append(out, string(runes[i:end]))
	}
	return out
}

// Searcher is the primary library interface.
type Searcher interface {
	Search(ctx context.Context, q Query) (*Results, error)
	Fetch(ctx context.Context, url string) (*Page, error)
	FetchStream(ctx context.Context, url string) (<-chan Chunk, <-chan *Page, <-chan error)
}

// Provider is implemented by each search backend.
type Provider interface {
	Name() string
	Search(ctx context.Context, q Query) ([]Result, error)
}

// ProviderConfig describes one provider entry in config.
type ProviderConfig struct {
	Name    string  `json:"name"`
	URL     string  `json:"url,omitempty"`
	RPS     float64 `json:"rps,omitempty"`
	Retries int     `json:"retries,omitempty"`
	// Vertical assigns this provider to a non-general search vertical
	// (e.g. "news", "images"). Empty means the general web chain. See
	// docs/adr/003-search-verticals.md.
	Vertical string `json:"vertical,omitempty"`
}

// RateLimitError signals a provider is throttled; triggers chain fallback.
type RateLimitError struct {
	Provider   string
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return "saka: " + e.Provider + " rate limited"
}
