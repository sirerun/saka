// Package saka provides free web search with no API keys.
package saka

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirerun/saka/chain"
	"github.com/sirerun/saka/fetch"
	_ "github.com/sirerun/saka/provider/duckduckgo"
	_ "github.com/sirerun/saka/provider/searxng"
	_ "github.com/sirerun/saka/provider/startpage"
	"github.com/sirerun/saka/types"
)

// Re-export leaf types so library callers keep using saka.Query, etc.
type (
	Query           = types.Query
	Result          = types.Result
	Results         = types.Results
	Chunk           = types.Chunk
	Page            = types.Page
	Searcher        = types.Searcher
	Provider        = types.Provider
	ProviderConfig  = types.ProviderConfig
	RateLimitError  = types.RateLimitError
	ProviderFactory = types.ProviderFactory
)

// ErrNoResults is returned when all providers fail or return nothing.
var ErrNoResults = types.ErrNoResults

// Register is a convenience wrapper over types.Register for third-party
// provider packages, so they depend on saka's public API, not types.
func Register(name string, factory ProviderFactory) error {
	return types.Register(name, factory)
}

// DiskCacheConfig enables an optional on-disk L2 cache for fetched pages.
type DiskCacheConfig struct {
	Dir        string `json:"dir"`
	TTLSeconds int    `json:"ttl_seconds"`
}

// FetchConfig configures page fetching.
type FetchConfig struct {
	RPS             float64          `json:"rps"`
	CacheTTLSeconds int              `json:"cache_ttl_seconds"`
	RespectRobots   bool             `json:"respect_robots"`
	DiskCache       *DiskCacheConfig `json:"disk_cache,omitempty"`
}

// Config is the top-level engine configuration.
type Config struct {
	Providers []ProviderConfig `json:"providers"`
	Fetch     FetchConfig      `json:"fetch"`
}

// DefaultConfig tries SearXNG first (fastest/cleanest when you have an
// instance running), falls back to DuckDuckGo, then Startpage last.
func DefaultConfig() Config {
	return Config{
		Providers: []ProviderConfig{
			{Name: "searxng", URL: "http://localhost:8888", RPS: 5, Retries: 2},
			{Name: "duckduckgo", RPS: 1, Retries: 2},
			{Name: "startpage", RPS: 0.2, Retries: 1},
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
		if _, ok := types.Lookup(p.Name); !ok {
			return fmt.Errorf("saka: providers[%d]: provider %q is not registered (known: %s)", i, p.Name, strings.Join(types.Registered(), ", "))
		}
		if p.Name == "searxng" {
			if p.URL == "" {
				return fmt.Errorf("saka: providers[%d]: searxng requires \"url\"", i)
			}
			if !strings.HasPrefix(p.URL, "http") {
				return fmt.Errorf("saka: providers[%d]: searxng url must start with http(s)", i)
			}
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

// LoadConfig reads and validates a saka.json config file.
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

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

// Engine ties providers + fetcher together and implements Searcher.
type Engine struct {
	chain   *chain.Chain
	fetcher *fetch.Fetcher
}

// New constructs an Engine from cfg.
func New(cfg Config) (*Engine, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	var ps []Provider
	for _, pc := range cfg.Providers {
		factory, ok := types.Lookup(pc.Name)
		if !ok {
			return nil, fmt.Errorf("saka: provider %q is not registered", pc.Name)
		}
		p, err := factory(pc)
		if err != nil {
			return nil, fmt.Errorf("saka: provider %q: %w", pc.Name, err)
		}
		ps = append(ps, p)
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
		go func() { dc.GC() }()
		fetcher.SetDiskCache(dc)
	}

	return &Engine{
		chain:   chain.New(cfg.Providers, ps),
		fetcher: fetcher,
	}, nil
}

// Search runs the provider chain.
func (e *Engine) Search(ctx context.Context, q Query) (*Results, error) {
	return e.chain.Search(ctx, q.WithDefaults())
}

// Fetch retrieves and extracts a page.
func (e *Engine) Fetch(ctx context.Context, url string) (*Page, error) {
	return e.fetcher.Fetch(ctx, url)
}

// FetchStream streams extracted text chunks for a URL.
func (e *Engine) FetchStream(ctx context.Context, url string) (<-chan Chunk, <-chan *Page, <-chan error) {
	return e.fetcher.FetchStream(ctx, url)
}
