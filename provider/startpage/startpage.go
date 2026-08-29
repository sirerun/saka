// Package startpage scrapes Startpage's HTML endpoint (Google-backed results).
package startpage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sirerun/saka/internal/htmd"
	types "github.com/sirerun/saka/types"
	"golang.org/x/net/html"
)

const endpoint = "https://www.startpage.com/sp/search"

type Provider struct {
	client *http.Client
}

func New() *Provider {
	jar, _ := cookiejar.New(nil)
	return &Provider{
		client: &http.Client{Timeout: 15 * time.Second, Jar: jar},
	}
}

func (p *Provider) Name() string { return "startpage" }

func init() {
	if err := types.Register("startpage", newFromConfig); err != nil {
		panic(err)
	}
}

func newFromConfig(_ types.ProviderConfig) (types.Provider, error) {
	return New(), nil
}

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
	htmd.SetUserAgent(req)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("startpage: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

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
	doc, err := htmd.Parse(r)
	if err != nil {
		return nil, err
	}
	var results []types.Result

	htmd.Walk(doc, func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			cls := htmd.Class(n)
			if strings.Contains(cls, "w-gl__result-title") && len(results) < max {
				href := htmd.Attr(n, "href")
				if strings.HasPrefix(href, "http") {
					results = append(results, types.Result{
						Title:  htmd.Text(n),
						URL:    href,
						Source: "startpage",
					})
				}
			}
		}
		if n.Type == html.ElementNode && n.Data == "p" &&
			strings.Contains(htmd.Class(n), "w-gl__description") &&
			len(results) > 0 {
			last := &results[len(results)-1]
			if last.Snippet == "" {
				last.Snippet = htmd.Text(n)
			}
		}
	})

	for i := range results {
		results[i].Position = i + 1
	}
	return results, nil
}
