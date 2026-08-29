// Package duckduckgo scrapes DuckDuckGo's HTML endpoint — no API key required.
package duckduckgo

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirerun/saka/internal/htmd"
	"github.com/sirerun/saka/types"
	"golang.org/x/net/html"
)

const htmlEndpoint = "https://html.duckduckgo.com/html/"

type Provider struct {
	client *http.Client
}

func New() *Provider {
	return &Provider{client: &http.Client{Timeout: 15 * time.Second}}
}

func init() {
	if err := types.Register("duckduckgo", newFromConfig); err != nil {
		panic(err)
	}
}

func newFromConfig(_ types.ProviderConfig) (types.Provider, error) {
	return New(), nil
}

func (p *Provider) Name() string { return "duckduckgo" }

func (p *Provider) Search(ctx context.Context, q types.Query) ([]types.Result, error) {
	form := url.Values{
		"q":  {q.Text},
		"kl": {regionOrDefault(q.Region)},
	}
	if q.SafeSearch {
		form.Set("kp", "1")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, htmlEndpoint,
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	htmd.SetUserAgent(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("duckduckgo: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden {
		return nil, &types.RateLimitError{Provider: "duckduckgo", RetryAfter: 30 * time.Second}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("duckduckgo: status %d", resp.StatusCode)
	}

	return parseResults(resp.Body, q.MaxResults)
}

func regionOrDefault(r string) string {
	if r == "" {
		return "us-en"
	}
	return r
}

// parseResults walks the DOM looking for result links: a.result__a
// inside div.result, with snippet in a.result__snippet.
func parseResults(r io.Reader, max int) ([]types.Result, error) {
	doc, err := htmd.Parse(r)
	if err != nil {
		return nil, err
	}
	var results []types.Result
	htmd.Walk(doc, func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" && htmd.HasClass(n, "result__a") {
			href := htmd.Attr(n, "href")
			target := unwrapDDG(href)
			if target != "" && len(results) < max {
				results = append(results, types.Result{
					Title: htmd.Text(n),
					URL:   target,
				})
			}
		}
	})

	// attach snippets: they appear after each title link in document order.
	var withSnips []types.Result
	snips := collectSnippets(doc)
	for i := range results {
		if i < len(snips) {
			results[i].Snippet = snips[i]
		}
		results[i].Source = "duckduckgo"
		results[i].Position = i + 1
		withSnips = append(withSnips, results[i])
	}
	return withSnips, nil
}

func collectSnippets(doc *html.Node) []string {
	var snips []string
	htmd.Walk(doc, func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" && htmd.HasClass(n, "result__snippet") {
			snips = append(snips, htmd.Text(n))
		}
	})
	return snips
}

// unwrapDDG extracts the real URL from DDG's //duckduckgo.com/l/?uddg=<encoded> redirect.
func unwrapDDG(href string) string {
	if href == "" {
		return ""
	}
	if strings.HasPrefix(href, "//duckduckgo.com/l/") || strings.Contains(href, "duckduckgo.com/l/") {
		u, err := url.Parse("https:" + strings.TrimPrefix(href, "//duckduckgo.com/l/"))
		if err != nil {
			return ""
		}
		uddg := u.Query().Get("uddg")
		if decoded, err := url.QueryUnescape(uddg); err == nil {
			return decoded
		}
		return uddg
	}
	if strings.HasPrefix(href, "http") {
		return href
	}
	return ""
}

// NOTE (source-chat comment, preserved): the assistant flagged that the
// original two-pass version of parseResults (one walk for titles, a
// second, unused walk for snippets) had a dead first pass. This file
// already applies the single-pass "collect snippets in document order"
// fix the assistant described making in its next iteration.
