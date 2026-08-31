package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	saka "github.com/sirerun/saka"
)

// TestHandleSearchVerticalImagesEndToEnd is T8.3's acceptance test: it builds
// a real saka.Engine (saka.New) from a saka.Config configuring the
// "searxng-images" provider (T8.2) on vertical "images" alongside a general
// "searxng" provider, both pointing at one httptest-mocked SearXNG instance,
// and asserts that GET /v1/search?...&vertical=images returns 200 with
// results carrying non-empty ThumbnailURL, distinct from the same query with
// no vertical param. It proves T7.4's generic &vertical= REST plumbing needs
// no images-specific code.
func TestHandleSearchVerticalImagesEndToEnd(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("categories") == "images" {
			_, _ = w.Write([]byte(`{"results":[
				{"title":"A cat","img_src":"https://x/full.jpg","thumbnail_src":"https://x/thumb.jpg","engine":"bing images","resolution":"1920x1080"}
			]}`))
			return
		}
		_, _ = w.Write([]byte(`{"results":[
			{"title":"Cats - Wikipedia","url":"https://en.wikipedia.org/wiki/Cat","content":"The cat is a domestic species.","engine":"wikipedia"}
		]}`))
	}))
	defer mock.Close()

	cfg := saka.Config{
		Providers: []saka.ProviderConfig{
			{Name: "searxng", URL: mock.URL, RPS: 5},
			{Name: "searxng-images", URL: mock.URL, RPS: 5, Vertical: "images"},
		},
		Fetch: saka.FetchConfig{RPS: 1, RespectRobots: false},
	}
	engine, err := saka.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	s := New(engine)

	generalRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(generalRR, httptest.NewRequest(http.MethodGet, "/v1/search?q=cats", nil))
	if generalRR.Code != http.StatusOK {
		t.Fatalf("general search: status %d: %s", generalRR.Code, generalRR.Body.String())
	}
	var general saka.Results
	if err := json.Unmarshal(generalRR.Body.Bytes(), &general); err != nil {
		t.Fatal(err)
	}
	if len(general.Results) == 0 {
		t.Fatalf("general search: expected results, got none")
	}
	if general.Results[0].ThumbnailURL != "" {
		t.Fatalf("general search: expected no ThumbnailURL, got %q", general.Results[0].ThumbnailURL)
	}

	imagesRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(imagesRR, httptest.NewRequest(http.MethodGet, "/v1/search?q=cats&vertical=images", nil))
	if imagesRR.Code != http.StatusOK {
		t.Fatalf("vertical=images search: status %d: %s", imagesRR.Code, imagesRR.Body.String())
	}
	var images saka.Results
	if err := json.Unmarshal(imagesRR.Body.Bytes(), &images); err != nil {
		t.Fatal(err)
	}
	if images.Provider != "searxng-images" {
		t.Fatalf("vertical=images search: want provider searxng-images, got %q", images.Provider)
	}
	if len(images.Results) == 0 || images.Results[0].ThumbnailURL == "" {
		t.Fatalf("vertical=images search: expected non-empty ThumbnailURL, got %+v", images.Results)
	}
	if images.Results[0].URL != "https://x/full.jpg" {
		t.Fatalf("vertical=images search: URL = %q, want img_src value", images.Results[0].URL)
	}

	if images.Provider == general.Provider {
		t.Fatalf("vertical=images search returned the same provider as general search: %q", images.Provider)
	}
}
