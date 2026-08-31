package saka

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSearchStreamRoutesNewsVerticalToRealGdeltProvider is T9.4's proof
// that SearchStream's vertical routing (Search's chains map, keyed by
// ProviderConfig.Vertical -- see Engine.Search) reaches the actual gdelt
// provider, not a stand-in. It points the real "gdelt" provider at an
// httptest.Server via ProviderConfig.URL (see provider/gdelt's
// newFromConfig) instead of mocking Engine or Provider.
func TestSearchStreamRoutesNewsVerticalToRealGdeltProvider(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"articles":[{"url":"https://example.com/climate","title":"Climate News","domain":"example.com"}]}`))
	}))
	defer srv.Close()

	cfg := Config{
		Providers: []ProviderConfig{
			{Name: "duckduckgo", RPS: 1},
			{Name: "gdelt", URL: srv.URL, RPS: 1, Vertical: "news"},
		},
		Fetch: FetchConfig{RPS: 1},
	}
	e, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	itemCh, doneCh, errCh := e.SearchStream(context.Background(), Query{Text: "climate", Vertical: "news"})

	var got []Result
	for r := range itemCh {
		got = append(got, r)
	}

	select {
	case err := <-errCh:
		t.Fatalf("unexpected error on error channel: %v", err)
	default:
	}

	done := <-doneCh
	if done == nil {
		t.Fatal("expected non-nil *Results on done channel")
	}
	if done.Provider != "gdelt" {
		t.Fatalf("done.Provider = %q, want gdelt", done.Provider)
	}
	if len(got) != 1 {
		t.Fatalf("streamed %d results, want 1: %+v", len(got), got)
	}
	if got[0].URL != "https://example.com/climate" || got[0].Title != "Climate News" || got[0].Source != "example.com" {
		t.Errorf("unexpected result: %+v", got[0])
	}
}
