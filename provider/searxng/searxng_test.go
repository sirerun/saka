package searxng

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirerun/saka/types"
)

func TestSearchOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("format") != "json" {
			t.Errorf("format=%q", r.URL.Query().Get("format"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"title":"T","url":"https://x","content":"snip"}]}`))
	}))
	defer srv.Close()

	p := New(srv.URL)
	res, err := p.Search(context.Background(), types.Query{Text: "q", MaxResults: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0].URL != "https://x" {
		t.Fatalf("%+v", res)
	}
	if p.Name() != "searxng" {
		t.Fatal(p.Name())
	}
}

func TestSearchRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	_, err := New(srv.URL).Search(context.Background(), types.Query{Text: "q", MaxResults: 1})
	if _, ok := err.(*types.RateLimitError); !ok {
		t.Fatalf("want RateLimitError, got %v", err)
	}
}

func TestSearchSiteFilter(t *testing.T) {
	var gotQ string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQ = r.URL.Query().Get("q")
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer srv.Close()

	_, err := New(srv.URL).Search(context.Background(), types.Query{Text: "ai", Site: "arxiv.org", MaxResults: 3})
	if err != nil {
		t.Fatal(err)
	}
	if gotQ != "ai site:arxiv.org" {
		t.Fatalf("q=%q", gotQ)
	}
}
