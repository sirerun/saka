package htmd

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// ddgFixture mirrors provider/duckduckgo/duckduckgo_test.go's fixture shape.
const ddgFixture = `<!DOCTYPE html>
<html><body>
<div class="result">
  <a class="result__a" href="https://example.com/one">First Result</a>
  <a class="result__snippet">Snippet one</a>
</div>
<div class="result">
  <a class="result__a" href="https://example.com/two">Second Result</a>
  <a class="result__snippet">Snippet two</a>
</div>
</body></html>`

type link struct{ title, href string }

func extractDDG(doc *html.Node) (links []link, snippets []string) {
	Walk(doc, func(n *html.Node) {
		if n.Type != html.ElementNode || n.Data != "a" {
			return
		}
		switch {
		case HasClass(n, "result__a"):
			links = append(links, link{title: Text(n), href: Attr(n, "href")})
		case HasClass(n, "result__snippet"):
			snippets = append(snippets, Text(n))
		}
	})
	return links, snippets
}

func TestDuckDuckGoFixtureParsing(t *testing.T) {
	doc, err := Parse(strings.NewReader(ddgFixture))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	links, snippets := extractDDG(doc)
	if len(links) != 2 {
		t.Fatalf("want 2 result links, got %d", len(links))
	}
	if links[0].href != "https://example.com/one" || links[0].title != "First Result" {
		t.Errorf("bad first result: %+v", links[0])
	}
	if links[1].href != "https://example.com/two" || links[1].title != "Second Result" {
		t.Errorf("bad second result: %+v", links[1])
	}
	if len(snippets) != 2 || snippets[0] != "Snippet one" || snippets[1] != "Snippet two" {
		t.Errorf("bad snippets: %+v", snippets)
	}
}

// spFixture mirrors provider/startpage/startpage_test.go's fixture shape.
const spFixture = `<!DOCTYPE html>
<html><body>
<div class="w-gl">
  <a class="w-gl__result-title" href="https://example.com/one"><h2>First</h2></a>
  <p class="w-gl__description">Snippet one</p>
  <a class="w-gl__result-title" href="https://example.com/two"><h2>Second</h2></a>
  <p class="w-gl__description">Snippet two</p>
</div>
</body></html>`

// spChallengeFixture mirrors the CAPTCHA/challenge page Startpage serves
// instead of results when it wants to block a scrape.
const spChallengeFixture = `<html><body><div class="captcha-container">Please verify</div></body></html>`

func extractStartpage(doc *html.Node) (links []link, snippets []string) {
	Walk(doc, func(n *html.Node) {
		switch {
		case n.Type == html.ElementNode && n.Data == "a" && HasClass(n, "w-gl__result-title"):
			links = append(links, link{title: Text(n), href: Attr(n, "href")})
		case n.Type == html.ElementNode && n.Data == "p" && HasClass(n, "w-gl__description"):
			snippets = append(snippets, Text(n))
		}
	})
	return links, snippets
}

func hasCaptchaClass(doc *html.Node) bool {
	found := false
	Walk(doc, func(n *html.Node) {
		if strings.Contains(Class(n), "captcha") {
			found = true
		}
	})
	return found
}

func TestStartpageFixtureParsing(t *testing.T) {
	doc, err := Parse(strings.NewReader(spFixture))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	links, snippets := extractStartpage(doc)
	if len(links) != 2 {
		t.Fatalf("want 2 result links, got %d", len(links))
	}
	if links[0].href != "https://example.com/one" || links[0].title != "First" {
		t.Errorf("bad first result: %+v", links[0])
	}
	if len(snippets) != 2 || snippets[0] != "Snippet one" || snippets[1] != "Snippet two" {
		t.Errorf("bad snippets: %+v", snippets)
	}
}

func TestStartpageCaptchaChallengeDetected(t *testing.T) {
	doc, err := Parse(strings.NewReader(spChallengeFixture))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	links, _ := extractStartpage(doc)
	if len(links) != 0 {
		t.Fatalf("challenge page should yield 0 results, got %d", len(links))
	}
	if !hasCaptchaClass(doc) {
		t.Fatal("expected to detect the captcha class on the challenge page")
	}
}
