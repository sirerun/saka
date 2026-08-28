package fetch

import (
	"fmt"
	"strings"
	"time"

	"github.com/you/saka"
	"golang.org/x/net/html"
)

// skipTags are non-content elements removed before scoring.
var skipTags = map[string]bool{
	"script": true, "style": true, "nav": true, "header": true,
	"footer": true, "aside": true, "form": true, "noscript": true,
	"svg": true, "iframe": true, "button": true,
}

// Extract parses HTML and pulls out the dominant readable text (readability-style).
func Extract(rawURL string, r interface {
	Read([]byte) (int, error)
}) (*saka.Page, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("extract: parse: %w", err)
	}

	title := findMeta(doc, "og:title")
	if title == "" {
		if t := findTag(doc, "title"); t != "" {
			title = t
		}
	}
	published := parseTime(findMeta(doc, "article:published_time"))

	// Score candidate containers: prefer <article>, then highest text-density div.
	best := findBestContainer(doc)
	var sb strings.Builder
	collectText(best, &sb)
	text := normalize(sb.String())

	return &saka.Page{
		URL:         rawURL,
		Title:       title,
		Text:        text,
		PublishedAt: published,
	}, nil
}

func findBestContainer(doc *html.Node) *html.Node {
	var best *html.Node
	bestScore := 0.0
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "article":
				if s := density(n); s > bestScore {
					best, bestScore = n, s
				}
			case "div", "main", "section":
				if s := density(n) * 0.9; s > bestScore { // slight penalty vs <article>
					best, bestScore = n, s
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if best == nil {
		return doc // fallback: whole document
	}
	return best
}

// density = text length weighted by paragraph share.
func density(n *html.Node) float64 {
	textLen, pCount, aLen := 0, 0, 0
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if skipTags[n.Data] {
				return // don't descend
			}
			if n.Data == "p" {
				pCount++
			}
			if n.Data == "a" {
				aLen += textContentLen(n)
				return
			}
		}
		if n.Type == html.TextNode {
			textLen += len(strings.TrimSpace(n.Data))
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	if textLen == 0 {
		return 0
	}
	score := float64(textLen) + float64(pCount)*100
	if aLen > 0 {
		score *= float64(textLen) / float64(textLen+aLen) // link-density penalty
	}
	return score
}

func textContentLen(n *html.Node) int {
	n2 := 0
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			n2 += len(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return n2
}

func collectText(n *html.Node, sb *strings.Builder) {
	if n.Type == html.ElementNode {
		if skipTags[n.Data] {
			return
		}
		if n.Data == "p" || n.Data == "h1" || n.Data == "h2" || n.Data == "h3" ||
			n.Data == "li" || n.Data == "blockquote" {
			sb.WriteString("\n\n")
		}
	}
	if n.Type == html.TextNode {
		sb.WriteString(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectText(c, sb)
	}
}

func normalize(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n\n")
}

func findMeta(doc *html.Node, prop string) string {
	var found string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var key, content string
			for _, a := range n.Attr {
				if a.Key == "property" || a.Key == "name" {
					key = a.Val
				}
				if a.Key == "content" {
					content = a.Val
				}
			}
			if key == prop {
				found = content
			}
		}
		for c := n.FirstChild; c != nil && found == ""; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return found
}

func findTag(doc *html.Node, tag string) string {
	var found string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == tag {
			var sb strings.Builder
			collectText(n, &sb)
			found = strings.TrimSpace(sb.String())
		}
		for c := n.FirstChild; c != nil && found == ""; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return found
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
