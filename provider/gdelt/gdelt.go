// Package gdelt queries the GDELT Project's DOC 2.0 API for news articles.
// It requires no API key.
//
// This provider is scoped to the "news" search vertical, not general web
// search (see docs/adr/003-search-verticals.md). The provider itself has
// no way to enforce this -- it's ProviderConfig.Vertical that routes it
// into the news chain instead of the general web chain -- so a saka.json
// entry for "gdelt" should always set "vertical": "news". Do not add it
// to the general chain: mixing GDELT news results into general web
// results would make general search nondeterministic depending on which
// provider happened to answer first.
//
// GDELT is a shared public resource with no auth to throttle by, so
// configure it conservatively: RPS <= 1.
package gdelt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/sirerun/saka/types"
)

// docEndpoint is GDELT's DOC 2.0 API, article-list mode.
const docEndpoint = "https://api.gdeltproject.org/api/v2/doc/doc"

// defaultMaxRecords is used to bound the outbound request when the
// caller didn't set Query.MaxResults (e.g. calling the provider
// directly rather than through saka.Engine, which applies
// Query.WithDefaults first).
const defaultMaxRecords = 10

// maxRecordsCeiling is GDELT's own upper bound on maxrecords.
const maxRecordsCeiling = 250

// Provider queries the GDELT DOC 2.0 API.
type Provider struct {
	endpoint string
	client   *http.Client
}

var _ types.Provider = (*Provider)(nil)

// New returns a Provider ready to query GDELT.
func New() *Provider {
	return newWithEndpoint(docEndpoint)
}

// newWithEndpoint builds a Provider against a custom endpoint, letting
// tests point at an httptest.Server instead of the real GDELT API.
func newWithEndpoint(endpoint string) *Provider {
	return &Provider{endpoint: endpoint, client: &http.Client{Timeout: 15 * time.Second}}
}

func (p *Provider) Name() string { return "gdelt" }

func init() {
	if err := types.Register("gdelt", newFromConfig); err != nil {
		panic(err)
	}
}

func newFromConfig(_ types.ProviderConfig) (types.Provider, error) {
	return New(), nil
}

// gdeltResponse models the subset of GDELT's DOC 2.0 JSON we need.
type gdeltResponse struct {
	Articles []struct {
		URL    string `json:"url"`
		Title  string `json:"title"`
		Domain string `json:"domain"`
	} `json:"articles"`
}

func (p *Provider) Search(ctx context.Context, q types.Query) ([]types.Result, error) {
	query := q.Text
	if q.Site != "" {
		query += " domainis:" + q.Site
	}

	maxRecords := q.MaxResults
	if maxRecords <= 0 {
		maxRecords = defaultMaxRecords
	}
	if maxRecords > maxRecordsCeiling {
		maxRecords = maxRecordsCeiling
	}

	params := url.Values{
		"query":      {query},
		"mode":       {"artlist"},
		"format":     {"json"},
		"maxrecords": {fmt.Sprintf("%d", maxRecords)},
	}

	endpoint := p.endpoint + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gdelt: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusTooManyRequests, http.StatusForbidden:
		return nil, &types.RateLimitError{Provider: "gdelt", RetryAfter: 30 * time.Second}
	case http.StatusOK:
		// continue
	default:
		return nil, fmt.Errorf("gdelt: status %d", resp.StatusCode)
	}

	var gr gdeltResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return nil, fmt.Errorf("gdelt: decode: %w", err)
	}

	out := make([]types.Result, 0, len(gr.Articles))
	for i, a := range gr.Articles {
		if q.MaxResults > 0 && i >= q.MaxResults {
			break
		}
		out = append(out, types.Result{
			Title:    a.Title,
			URL:      a.URL,
			Snippet:  "", // GDELT's artlist mode doesn't return an excerpt
			Source:   a.Domain,
			Position: i + 1,
		})
	}
	return out, nil
}
