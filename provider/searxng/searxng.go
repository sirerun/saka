// Package searxng queries a self-hosted SearXNG instance via its JSON API.
package searxng

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	types "github.com/sirerun/saka/types"
)

type Provider struct {
	baseURL string // e.g. "http://localhost:8888"
	client  *http.Client
}

func New(baseURL string) *Provider {
	return &Provider{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *Provider) Name() string { return "searxng" }

func init() {
	if err := types.Register("searxng", newFromConfig); err != nil {
		panic(err)
	}
}

func newFromConfig(cfg types.ProviderConfig) (types.Provider, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("saka: searxng provider requires url")
	}
	return New(cfg.URL), nil
}

// searxResponse models the subset of SearXNG's JSON we need.
type searxResponse struct {
	Results []struct {
		Title         string `json:"title"`
		URL           string `json:"url"`
		Content       string `json:"content"`                 // snippet
		PublishedDate string `json:"publishedDate,omitempty"` // RFC3339 or "2024-05-01T12:00:00Z"
		Engine        string `json:"engine,omitempty"`
	} `json:"results"`
}

func (p *Provider) Search(ctx context.Context, q types.Query) ([]types.Result, error) {
	params := url.Values{
		"q":        {q.Text},
		"format":   {"json"},
		"language": {langOrDefault(q.Region)},
	}
	if q.MaxResults > 0 {
		params.Set("pageno", "1")
	}
	if q.Site != "" {
		params.Set("q", q.Text+" site:"+q.Site)
	}
	if q.SafeSearch {
		params.Set("safesearch", "1")
	}

	endpoint := p.baseURL + "/search?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	// SearXNG requires a browser-like UA or it may return 403.
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; saka/0.1)")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("searxng: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusTooManyRequests, http.StatusForbidden:
		return nil, &types.RateLimitError{Provider: "searxng", RetryAfter: 30 * time.Second}
	case http.StatusOK:
		// continue
	default:
		return nil, fmt.Errorf("searxng: status %d", resp.StatusCode)
	}

	var sr searxResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, fmt.Errorf("searxng: decode: %w", err)
	}

	out := make([]types.Result, 0, len(sr.Results))
	for i, r := range sr.Results {
		if i >= q.MaxResults {
			break
		}
		out = append(out, types.Result{
			Title:    r.Title,
			URL:      r.URL,
			Snippet:  r.Content,
			Source:   "searxng",
			Position: i + 1,
		})
	}
	return out, nil
}

func langOrDefault(region string) string {
	if region == "" {
		return "en-US"
	}
	return region
}

// NOTE: SearXNG requires `search: formats: [html, json]` in your instance's
// settings.yml to enable the JSON API — document that in the README.
