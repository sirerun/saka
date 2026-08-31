// Package htmd holds HTML-scraping helpers shared by the search providers:
// DOM walking/attribute lookups and the rotating desktop user-agent pool.
package htmd

import (
	"io"
	"math/rand"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

// UserAgents is a pool of realistic desktop browser user-agent strings
// rotated across provider requests to reduce fingerprinting.
var UserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64; rv:125.0) Gecko/20100101 Firefox/125.0",
}

// RandomUserAgent returns a random entry from UserAgents.
func RandomUserAgent() string {
	return UserAgents[rand.Intn(len(UserAgents))]
}

// SetUserAgent sets req's User-Agent header to a random entry from UserAgents.
func SetUserAgent(req *http.Request) {
	req.Header.Set("User-Agent", RandomUserAgent())
}

// Parse parses r as HTML and returns the document root.
func Parse(r io.Reader) (*html.Node, error) {
	return html.Parse(r)
}

// Walk calls fn on n and every descendant, in document order.
func Walk(n *html.Node, fn func(*html.Node)) {
	if n == nil {
		return
	}
	fn(n)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		Walk(c, fn)
	}
}

// Attr returns the value of n's key attribute, or "" if absent.
func Attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// Class returns n's class attribute verbatim.
func Class(n *html.Node) string {
	return Attr(n, "class")
}

// HasClass reports whether class is one of n's space-separated class tokens.
func HasClass(n *html.Node, class string) bool {
	for _, c := range strings.Fields(Class(n)) {
		if c == class {
			return true
		}
	}
	return false
}

// Text returns the concatenated, trimmed text content of n and its descendants.
func Text(n *html.Node) string {
	var sb strings.Builder
	Walk(n, func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
	})
	return strings.TrimSpace(sb.String())
}
