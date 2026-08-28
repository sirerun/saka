// Package startpage scrapes Startpage's HTML endpoint (Google-backed results).
package startpage

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	types "github.com/sirerun/saka/types"
	"golang.org/x/net/html"
)

const endpoint = "https://www.startpage.com/sp/search"

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64; rv:125.0) Gecko/20100101 Firefox/125.0",
}

type Provider struct {
	client *http.Client
	jar    http.CookieJar // hold consent cookies across requests
}

func New() *Provider {
	jar, _ := cookiejar.New(nil)
	return &Provider{
		client: &http.Client{Timeout: 15 * time.Second, Jar: jar},
	}
}

func (p *Provider) Name() string { return "startpage" }

func (p *Provider) Search(ctx context.Context, q types.Query) ([]types.Result, error) {
	params := url.Values{
		"query":    {q.Text},
		"cat":      {"web"},
		"language": {langOrDefault(q.Region)},
	}
	if q.MaxResults > 0 {
		params.Set("num", strconv.Itoa(q.MaxResults))
	}
	if q.Site != "" {
		params.Set("query", q.Text+" site:"+q.Site)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("startpage: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusTooManyRequests, http.StatusForbidden:
		// Startpage sends CAPTCHA/challenge pages as 200 too — checked in parse.
		return nil, &types.RateLimitError{Provider: "startpage", RetryAfter: time.Minute}
	case http.StatusOK:
	default:
		return nil, fmt.Errorf("startpage: status %d", resp.StatusCode)
	}

	results, err := parseResults(resp.Body, q.MaxResults)
	if err != nil {
		return nil, err
	}
	// Challenge pages parse to zero results — treat as rate limit so
	// the chain falls back and the breaker opens.
	if len(results) == 0 {
		return nil, &types.RateLimitError{Provider: "startpage", RetryAfter: time.Minute}
	}
	return results, nil
}

func langOrDefault(r string) string {
	if r == "" {
		return "english"
	}
	return r
}

// parseResults: results live in section.w-gl > a.w-gl__result-title (href is
// a direct URL) with snippet in p.w-gl__description. One walk collects both.
func parseResults(r io.Reader, max int) ([]types.Result, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	var results []types.Result

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			cls := classOf(n)
			if strings.Contains(cls, "w-gl__result-title") && len(results) < max {
				href := attrOf(n, "href")
				if strings.HasPrefix(href, "http") {
					results = append(results, types.Result{
						Title:  textOf(n),
						URL:    href,
						Source: "startpage",
					})
				}
			}
		}
		if n.Type == html.ElementNode && n.Data == "p" &&
			strings.Contains(classOf(n), "w-gl__description") &&
			len(results) > 0 {
			last := &results[len(results)-1]
			if last.Snippet == "" {
				last.Snippet = textOf(n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	for i := range results {
		results[i].Position = i + 1
	}
	return results, nil
}

// attrOf/classOf/textOf are shared helpers — extract to an internal
// package (saka/internal/htmd) so duckduckgo and startpage both use them.
func attrOf(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func classOf(n *html.Node) string { return attrOf(n, "class") }

func textOf(n *html.Node) string {
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
