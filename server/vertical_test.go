package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	saka "github.com/sirerun/saka"
)

// fakeVerticalProvider is a test-only saka.Provider returning a fixed,
// distinguishable result set, standing in for a real vertical provider
// (e.g. gdelt) without reaching the network. Registered under names
// distinct from any real provider ("fake-general"/"fake-news") to avoid
// colliding with saka.go's production registrations.
type fakeVerticalProvider struct {
	name    string
	results []saka.Result
}

func (p *fakeVerticalProvider) Name() string { return p.name }

func (p *fakeVerticalProvider) Search(_ context.Context, _ saka.Query) ([]saka.Result, error) {
	return p.results, nil
}

func init() {
	if err := saka.Register("fake-general", func(_ saka.ProviderConfig) (saka.Provider, error) {
		return &fakeVerticalProvider{
			name: "fake-general",
			results: []saka.Result{
				{Title: "General T", URL: "https://general.example/a", Snippet: "general s", Source: "fake-general", Position: 1},
			},
		}, nil
	}); err != nil {
		panic(err)
	}
	if err := saka.Register("fake-news", func(_ saka.ProviderConfig) (saka.Provider, error) {
		return &fakeVerticalProvider{
			name: "fake-news",
			results: []saka.Result{
				{Title: "News T", URL: "https://news.example/a", Snippet: "news s", Source: "fake-news", Position: 1},
			},
		}, nil
	}); err != nil {
		panic(err)
	}
}

// TestHandleSearchVerticalDistinctFromGeneral is T7.4's acceptance test: it
// builds a real saka.Engine (saka.New, not the fakeEngine mock used
// elsewhere in this package) from a saka.Config with one provider on the
// general (empty) vertical and one on vertical "news" -- mirroring how a
// real saka.json would configure the gdelt provider under
// "vertical": "news" (docs/adr/003-search-verticals.md) -- and asserts
// that GET /v1/search?...&vertical=news returns different results than
// the same query with no vertical param.
func TestHandleSearchVerticalDistinctFromGeneral(t *testing.T) {
	cfg := saka.Config{
		Providers: []saka.ProviderConfig{
			{Name: "fake-general", RPS: 1},
			{Name: "fake-news", RPS: 1, Vertical: "news"},
		},
		Fetch: saka.FetchConfig{RPS: 1, RespectRobots: false},
	}
	engine, err := saka.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	s := New(engine)

	generalRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(generalRR, httptest.NewRequest(http.MethodGet, "/v1/search?q=climate", nil))
	if generalRR.Code != http.StatusOK {
		t.Fatalf("general search: status %d: %s", generalRR.Code, generalRR.Body.String())
	}
	var general saka.Results
	if err := json.Unmarshal(generalRR.Body.Bytes(), &general); err != nil {
		t.Fatal(err)
	}

	newsRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(newsRR, httptest.NewRequest(http.MethodGet, "/v1/search?q=climate&vertical=news", nil))
	if newsRR.Code != http.StatusOK {
		t.Fatalf("vertical=news search: status %d: %s", newsRR.Code, newsRR.Body.String())
	}
	var news saka.Results
	if err := json.Unmarshal(newsRR.Body.Bytes(), &news); err != nil {
		t.Fatal(err)
	}

	if general.Provider != "fake-general" {
		t.Fatalf("general search: want provider fake-general, got %q", general.Provider)
	}
	if news.Provider != "fake-news" {
		t.Fatalf("vertical=news search: want provider fake-news, got %q", news.Provider)
	}
	if news.Provider == general.Provider {
		t.Fatalf("vertical=news search returned the same provider as general search: %q", news.Provider)
	}
	if reflect.DeepEqual(news.Results, general.Results) {
		t.Fatalf("vertical=news search returned results identical to the general search: %+v", news.Results)
	}
}
