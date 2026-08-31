package gdelt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirerun/saka/types"
)

func TestSearchOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("mode") != "artlist" {
			t.Errorf("mode=%q", r.URL.Query().Get("mode"))
		}
		if r.URL.Query().Get("format") != "json" {
			t.Errorf("format=%q", r.URL.Query().Get("format"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"articles":[{"url":"https://example.com/a","title":"T","domain":"example.com"}]}`))
	}))
	defer srv.Close()

	p := newWithEndpoint(srv.URL)
	res, err := p.Search(context.Background(), types.Query{Text: "climate", MaxResults: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Fatalf("%+v", res)
	}
	if res[0].URL != "https://example.com/a" || res[0].Title != "T" || res[0].Source != "example.com" {
		t.Errorf("unexpected result: %+v", res[0])
	}
	if res[0].Position != 1 {
		t.Errorf("position = %d, want 1", res[0].Position)
	}
	if p.Name() != "gdelt" {
		t.Fatal(p.Name())
	}
}

func TestSearchRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	_, err := newWithEndpoint(srv.URL).Search(context.Background(), types.Query{Text: "q", MaxResults: 1})
	if _, ok := err.(*types.RateLimitError); !ok {
		t.Fatalf("want RateLimitError, got %v", err)
	}
}

// TestSearchMissingSnippetGraceful asserts that GDELT's artlist response --
// which never includes an excerpt/snippet field -- is handled gracefully:
// Snippet stays empty rather than the decode failing.
func TestSearchMissingSnippetGraceful(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"articles":[{"url":"https://example.com/a","title":"T","domain":"example.com"}]}`))
	}))
	defer srv.Close()

	res, err := newWithEndpoint(srv.URL).Search(context.Background(), types.Query{Text: "q", MaxResults: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Fatalf("%+v", res)
	}
	if res[0].Snippet != "" {
		t.Errorf("snippet = %q, want empty", res[0].Snippet)
	}
}

// TestSearchMalformedJSON asserts a malformed response body surfaces as an
// error instead of panicking.
func TestSearchMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	_, err := newWithEndpoint(srv.URL).Search(context.Background(), types.Query{Text: "q", MaxResults: 5})
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestSearchSiteFilter(t *testing.T) {
	var gotQ string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQ = r.URL.Query().Get("query")
		_, _ = w.Write([]byte(`{"articles":[]}`))
	}))
	defer srv.Close()

	_, err := newWithEndpoint(srv.URL).Search(context.Background(), types.Query{Text: "ai", Site: "arxiv.org", MaxResults: 3})
	if err != nil {
		t.Fatal(err)
	}
	if gotQ != "ai domainis:arxiv.org" {
		t.Fatalf("query=%q", gotQ)
	}
}

// TestNewFromConfigHonorsURL asserts that a ProviderConfig.URL (e.g. so
// tests or a self-hosted mirror can point gdelt at a non-default endpoint)
// overrides the built-in docEndpoint, and that an empty URL falls back to
// the real GDELT API.
func TestNewFromConfigHonorsURL(t *testing.T) {
	var gotHit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHit = true
		_, _ = w.Write([]byte(`{"articles":[]}`))
	}))
	defer srv.Close()

	p, err := newFromConfig(types.ProviderConfig{Name: "gdelt", URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Search(context.Background(), types.Query{Text: "q", MaxResults: 1}); err != nil {
		t.Fatal(err)
	}
	if !gotHit {
		t.Fatal("expected newFromConfig to route Search through the configured URL")
	}

	defaultProvider, err := newFromConfig(types.ProviderConfig{Name: "gdelt"})
	if err != nil {
		t.Fatal(err)
	}
	if got := defaultProvider.(*Provider).endpoint; got != docEndpoint {
		t.Errorf("endpoint = %q, want default %q", got, docEndpoint)
	}
}

func TestSearchMaxResultsCap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"articles":[
			{"url":"https://a","title":"A","domain":"a.com"},
			{"url":"https://b","title":"B","domain":"b.com"},
			{"url":"https://c","title":"C","domain":"c.com"}
		]}`))
	}))
	defer srv.Close()

	res, err := newWithEndpoint(srv.URL).Search(context.Background(), types.Query{Text: "q", MaxResults: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 {
		t.Fatalf("got %d results, want 2: %+v", len(res), res)
	}
}
