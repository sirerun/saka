package htmd

import (
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestRandomUserAgent(t *testing.T) {
	if len(UserAgents) == 0 {
		t.Fatal("UserAgents is empty")
	}
	for i := 0; i < 50; i++ {
		ua := RandomUserAgent()
		found := false
		for _, want := range UserAgents {
			if ua == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("RandomUserAgent() returned %q, not in UserAgents", ua)
		}
	}
}

func TestSetUserAgent(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	SetUserAgent(req)
	if req.Header.Get("User-Agent") == "" {
		t.Fatal("SetUserAgent did not set the User-Agent header")
	}
}

const doc = `<html><body><div class="result foo"><a class="result__a" href="/x">Hello <b>World</b></a><p class="result__snippet">snip text</p></div></body></html>`

func TestParseWalkAttrClass(t *testing.T) {
	node, err := Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	var found *html.Node
	Walk(node, func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" && HasClass(n, "result__a") {
			found = n
		}
	})
	if found == nil {
		t.Fatal("Walk/HasClass did not find a.result__a")
	}
	if got := Attr(found, "href"); got != "/x" {
		t.Fatalf("Attr href = %q, want /x", got)
	}
	if got := Class(found); !strings.Contains(got, "result__a") {
		t.Fatalf("Class = %q", got)
	}
	if HasClass(found, "resul") {
		t.Fatal("HasClass matched a non-token substring; must be exact token match")
	}
	if got := Text(found); got != "Hello World" {
		t.Fatalf("Text = %q, want %q", got, "Hello World")
	}
}
