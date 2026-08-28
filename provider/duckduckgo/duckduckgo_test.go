package duckduckgo

import (
	"strings"
	"testing"
)

const fixture = `<!DOCTYPE html>
<html><body>
<div class="result">
  <a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fone">First Result</a>
  <a class="result__snippet">Snippet one</a>
</div>
<div class="result">
  <a class="result__a" href="https://example.com/two">Second Result</a>
  <a class="result__snippet">Snippet two</a>
</div>
<div class="result">
  <a class="result__a" href="https://duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fthree&amp;rut=abc">Third</a>
  <a class="result__snippet">Snippet three</a>
</div>
</body></html>`

func TestParseResults(t *testing.T) {
	results, err := parseResults(strings.NewReader(fixture), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("want 3 results, got %d", len(results))
	}
	first := results[0]
	if first.URL != "https://example.com/one" {
		t.Errorf("unwrapDDG failed: got %q", first.URL)
	}
	if first.Title != "First Result" {
		t.Errorf("title wrong: %q", first.Title)
	}
	if results[1].URL != "https://example.com/two" {
		t.Errorf("plain URL mangled: %q", results[1].URL)
	}
	if results[2].Position != 3 {
		t.Errorf("position not set: %+v", results[2])
	}
}

func TestParseResultsMaxRespect(t *testing.T) {
	results, _ := parseResults(strings.NewReader(fixture), 2)
	if len(results) != 2 {
		t.Errorf("want 2, got %d", len(results))
	}
}

func TestUnwrapDDG(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://example.com/x", "https://example.com/x"},
		{"//duckduckgo.com/l/?uddg=https%3A%2F%2Fa.io", "https://a.io"},
		{"", ""},
		{"javascript:void(0)", ""},
	}
	for _, c := range cases {
		if got := unwrapDDG(c.in); got != c.want {
			t.Errorf("unwrapDDG(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
