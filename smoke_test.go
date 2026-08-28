//go:build integration

// Live smoke tests against real endpoints. Run manually:
//
//	go test -tags=integration -run TestSmoke -v -timeout 120s
package saka_test

import (
	"context"
	"strings"
	"testing"
	"time"

	saka "github.com/sirerun/saka"
)

func mustEngine(t *testing.T) *saka.Engine {
	t.Helper()
	e, err := saka.New(saka.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestSmokeSearchDDG(t *testing.T) {
	e := mustEngine(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := e.Search(ctx, saka.Query{Text: "open source large language models", MaxResults: 5})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(res.Results) == 0 {
		t.Fatal("no results")
	}
	for _, r := range res.Results {
		if r.URL == "" || r.Title == "" {
			t.Errorf("malformed result: %+v", r)
		}
		if !strings.HasPrefix(r.URL, "http") {
			t.Errorf("unwrapped URL: %s", r.URL)
		}
	}
	t.Logf("got %d results in %dms via %s", len(res.Results), res.TookMs, res.Provider)
}

func TestSmokeFetchAndExtract(t *testing.T) {
	e := mustEngine(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// example.com is stable and robots-friendly — ideal smoke target.
	page, err := e.Fetch(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if page.Title == "" {
		t.Error("empty title")
	}
	if len(page.Text) < 20 {
		t.Errorf("extracted text too short: %q", page.Text)
	}
}

func TestSmokeStream(t *testing.T) {
	e := mustEngine(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	chunks, _, errCh := e.FetchStream(ctx, "https://example.com")
	got := ""
	for c := range chunks {
		if c.Done {
			break
		}
		got += c.Text
	}
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("stream error: %v", err)
		}
	default:
	}
	if len(got) < 20 {
		t.Errorf("streamed text too short: %q", got)
	}
}

func TestSmokeCacheHit(t *testing.T) {
	e := mustEngine(t)
	ctx := context.Background()
	start := time.Now()
	if _, err := e.Fetch(ctx, "https://example.com"); err != nil {
		t.Fatal(err)
	}
	first := time.Since(start)

	start = time.Now()
	if _, err := e.Fetch(ctx, "https://example.com"); err != nil {
		t.Fatal(err)
	}
	second := time.Since(start)

	if second > first/2 && first > 100*time.Millisecond {
		t.Errorf("cache miss suspected: first=%v second=%v", first, second)
	}
}
