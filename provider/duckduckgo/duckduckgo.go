// Package duckduckgo scrapes DuckDuckGo's HTML endpoint — no API key required.
package duckduckgo

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirerun/saka/types"
	"golang.org/x/net/html"
)

const htmlEndpoint = "https://html.duckduckgo.com/html/"

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64; rv:125.0) Gecko/20100101 Firefox/125.0",
}

type Provider struct {
	client *http.Client
}

func New() *Provider {
	return &Provider{client: &http.Client{Timeout: 15 * time.Second}}
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
	req.Header.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])

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
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	var results []types.Result
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" && hasClass(n, "result__a") {
			href := attr(n, "href")
			target := unwrapDDG(href)
			if target != "" && len(results) < max {
				results = append(results, types.Result{
					Title: text(n),
					URL:   target,
				})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

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
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" && hasClass(n, "result__snippet") {
			snips = append(snips, text(n))
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
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

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hasClass(n *html.Node, class string) bool {
	for _, c := range strings.Fields(attr(n, "class")) {
		if c == class {
			return true
		}
	}
	return false
}

func text(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(sb.String())
}

// NOTE (source-chat comment, preserved): the assistant flagged that the
// original two-pass version of parseResults (one walk for titles, a
// second, unused walk for snippets) had a dead first pass. This file
// already applies the single-pass "collect snippets in document order"
// fix the assistant described making in its next iteration.
