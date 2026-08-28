// Package saka provides free web search with no API keys.
package saka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/you/saka/chain"
	"github.com/you/saka/fetch"
	"github.com/you/saka/provider/duckduckgo"
	"github.com/you/saka/provider/searxng"
	"github.com/you/saka/provider/startpage"
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
}

func (q Query) withDefaults() Query {
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
	Text string `json:"text"`            // text content
	Seq  int    `json:"seq"`             // ordering
	Done bool   `json:"done,omitempty"`  // set on final chunk
	Err  string `json:"error,omitempty"` // set if stream aborted
}

const chunkSize = 900 // ~900 chars per chunk: good granularity for LLM context

// Page is fetched, extracted page content.
//
// NOTE (source-chat bug, preserved verbatim): the chat's later revision of
// this struct adds a lowercase `text` field alongside the exported `Text`
// field, and has fetch.ExtractStream / the server & tools test fakes set
// `text` directly from other packages. Unexported fields can't be set
// from outside package saka, so as designed this does not compile from
// fetch, server, or tools. See NOTES.md.
type Page struct {
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Text        string    `json:"text"` // extracted readable text
	PublishedAt time.Time `json:"published_at,omitempty"`

	text string // internal: set by the extractor
}

// Chunks returns the page text in ordered chunks (from cache if already fetched).
func (p *Page) Chunks() (<-chan Chunk, error) {
	src := p.text
	if src == "" {
		src = p.Text
	}
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

// ---- Config ----

type ProviderConfig struct {
	Name    string  `json:"name"`
	URL     string  `json:"url,omitempty"` // for searxng
	RPS     float64 `json:"rps,omitempty"`
	Retries int     `json:"retries,omitempty"`
}

// DiskCacheConfig enables an optional on-disk L2 cache for fetched pages.
type DiskCacheConfig struct {
	Dir        string `json:"dir"`
	TTLSeconds int    `json:"ttl_seconds"`
}

type FetchConfig struct {
	RPS             float64          `json:"rps"`
	CacheTTLSeconds int              `json:"cache_ttl_seconds"`
	RespectRobots   bool             `json:"respect_robots"`
	DiskCache       *DiskCacheConfig `json:"disk_cache,omitempty"`
}

type Config struct {
	Providers []ProviderConfig `json:"providers"`
	Fetch     FetchConfig      `json:"fetch"`
}

// DefaultConfig tries SearXNG first (fastest/cleanest when you have an
// instance running), falls back to DuckDuckGo, then Startpage last (the
// chat called it "slowest, most fragile").
//
// NOTE (source-chat, preserved verbatim): the chat flagged its own change
// here with "auto-skipped if not running? No: see note" and never
// resolved what "see note" refers to — there's no actual skip-if-down
// behavior implemented anywhere (an unreachable SearXNG at rps 5 will
// just fail and fall through the chain like any other provider error).
func DefaultConfig() Config {
	return Config{
		Providers: []ProviderConfig{
			{Name: "searxng", URL: "http://localhost:8888", RPS: 5, Retries: 2}, // auto-skipped if not running? No: see note
			{Name: "duckduckgo", RPS: 1, Retries: 2},
			{Name: "startpage", RPS: 0.2, Retries: 1}, // slowest, most fragile — last
		},
		Fetch: FetchConfig{RPS: 2, CacheTTLSeconds: 3600, RespectRobots: true},
	}
}

// Validate checks a Config for correctness before engine construction.
func (c Config) Validate() error {
	if len(c.Providers) == 0 {
		return fmt.Errorf("saka: config has no providers")
	}
	seen := make(map[string]bool)
	for i, p := range c.Providers {
		if p.Name == "" {
			return fmt.Errorf("saka: providers[%d]: name is required", i)
		}
		if seen[p.Name] {
			return fmt.Errorf("saka: providers[%d]: duplicate provider %q", i, p.Name)
		}
		seen[p.Name] = true
		switch p.Name {
		case "duckduckgo", "startpage":
			// no required fields
		case "searxng":
			if p.URL == "" {
				return fmt.Errorf("saka: providers[%d]: searxng requires \"url\"", i)
			}
			if !strings.HasPrefix(p.URL, "http") {
				return fmt.Errorf("saka: providers[%d]: searxng url must start with http(s)", i)
			}
		default:
			return fmt.Errorf("saka: providers[%d]: unknown provider %q (known: duckduckgo, searxng, startpage)", i, p.Name)
		}
		if p.RPS < 0 || p.RPS > 10 {
			return fmt.Errorf("saka: providers[%d]: rps must be 0-10 (got %v)", i, p.RPS)
		}
		if p.Retries < 0 || p.Retries > 5 {
			return fmt.Errorf("saka: providers[%d]: retries must be 0-5 (got %d)", i, p.Retries)
		}
	}
	if c.Fetch.RPS < 0 || c.Fetch.RPS > 10 {
		return fmt.Errorf("saka: fetch.rps must be 0-10 (got %v)", c.Fetch.RPS)
	}
	if c.Fetch.CacheTTLSeconds < 0 {
		return fmt.Errorf("saka: fetch.cache_ttl_seconds must be >= 0")
	}
	return nil
}

func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("saka: read config: %w", err)
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("saka: parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// homeDir returns the user's home directory, or "" if it can't be
// determined. Added because the chat's disk-cache wiring calls a
// homeDir() helper (to expand "~" in the configured cache dir) that was
// never actually defined anywhere in the conversation.
func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

// ---- Engine ----

// Engine ties providers + fetcher together and implements Searcher.
//
// NOTE (source-chat bug, preserved verbatim): package saka imports
// chain and fetch here, while chain and fetch both import package saka
// for the shared types (Query, Result, Page, ...). That's a circular
// import and Go will refuse to build this as laid out. See NOTES.md.
type Engine struct {
	chain   *chain.Chain
	fetcher *fetch.Fetcher
}

func New(cfg Config) (*Engine, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	var ps []Provider
	for _, pc := range cfg.Providers {
		switch pc.Name {
		case "duckduckgo":
			ps = append(ps, duckduckgo.New())
		case "searxng":
			if pc.URL == "" {
				return nil, fmt.Errorf("saka: searxng provider requires url")
			}
			ps = append(ps, searxng.New(pc.URL))
		case "startpage":
			ps = append(ps, startpage.New())
		default:
			return nil, fmt.Errorf("saka: unknown provider %q", pc.Name)
		}
	}
	ttl := time.Duration(cfg.Fetch.CacheTTLSeconds) * time.Second
	fetcher := fetch.New(cfg.Fetch.RPS, ttl, cfg.Fetch.RespectRobots)

	if cfg.Fetch.DiskCache != nil && cfg.Fetch.DiskCache.Dir != "" {
		dir := strings.ReplaceAll(cfg.Fetch.DiskCache.Dir, "~", homeDir())
		dcTTL := time.Duration(cfg.Fetch.DiskCache.TTLSeconds) * time.Second
		if dcTTL == 0 {
			dcTTL = 24 * time.Hour
		}
		dc, err := fetch.NewDiskCache(dir, dcTTL)
		if err != nil {
			return nil, err
		}
		go func() { dc.GC() }() // opportunistic startup GC; server can schedule more
		fetcher.SetDiskCache(dc)
	}

	return &Engine{
		chain:   chain.New(cfg.Providers, ps),
		fetcher: fetcher,
	}, nil
}

func (e *Engine) Search(ctx context.Context, q Query) (*Results, error) {
	return e.chain.Search(ctx, q.withDefaults())
}

func (e *Engine) Fetch(ctx context.Context, url string) (*Page, error) {
	return e.fetcher.Fetch(ctx, url)
}

func (e *Engine) FetchStream(ctx context.Context, url string) (<-chan Chunk, <-chan *Page, <-chan error) {
	return e.fetcher.FetchStream(ctx, url)
}
