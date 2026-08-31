package searxng

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirerun/saka/types"
)

func TestImagesSearchOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("categories"); got != "images" {
			t.Errorf("categories=%q, want images", got)
		}
		if r.URL.Query().Get("format") != "json" {
			t.Errorf("format=%q", r.URL.Query().Get("format"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[
			{"title":"A cat","img_src":"https://x/full.jpg","thumbnail_src":"https://x/thumb.jpg","engine":"bing images","resolution":"1920x1080"},
			{"title":"Malformed resolution","img_src":"https://x/full2.jpg","thumbnail_src":"https://x/thumb2.jpg","engine":"bing images","resolution":"not-a-resolution"},
			{"title":"Missing resolution","img_src":"https://x/full3.jpg","thumbnail_src":"https://x/thumb3.jpg","engine":"bing images"}
		]}`))
	}))
	defer srv.Close()

	p := NewImages(srv.URL)
	if p.Name() != "searxng-images" {
		t.Fatalf("Name() = %q, want searxng-images", p.Name())
	}

	res, err := p.Search(context.Background(), types.Query{Text: "cats", MaxResults: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 3 {
		t.Fatalf("got %d results, want 3: %+v", len(res), res)
	}

	first := res[0]
	if first.URL != "https://x/full.jpg" {
		t.Errorf("URL = %q, want img_src value", first.URL)
	}
	if first.ThumbnailURL != "https://x/thumb.jpg" {
		t.Errorf("ThumbnailURL = %q, want thumbnail_src value", first.ThumbnailURL)
	}
	if first.Width != 1920 || first.Height != 1080 {
		t.Errorf("Width/Height = %d/%d, want 1920/1080 parsed from resolution", first.Width, first.Height)
	}

	malformed := res[1]
	if malformed.Width != 0 || malformed.Height != 0 {
		t.Errorf("malformed resolution: Width/Height = %d/%d, want 0/0", malformed.Width, malformed.Height)
	}
	if malformed.URL == "" || malformed.ThumbnailURL == "" {
		t.Errorf("malformed resolution result should not be dropped: %+v", malformed)
	}

	missing := res[2]
	if missing.Width != 0 || missing.Height != 0 {
		t.Errorf("missing resolution: Width/Height = %d/%d, want 0/0", missing.Width, missing.Height)
	}
}

func TestImagesSearchRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	_, err := NewImages(srv.URL).Search(context.Background(), types.Query{Text: "q", MaxResults: 1})
	if _, ok := err.(*types.RateLimitError); !ok {
		t.Fatalf("want RateLimitError, got %v", err)
	}
}

func TestParseResolution(t *testing.T) {
	cases := []struct {
		name         string
		in           string
		wantW, wantH int
	}{
		{"well-formed", "1920x1080", 1920, 1080},
		{"well-formed small", "640x480", 640, 480},
		{"empty", "", 0, 0},
		{"garbage", "garbage", 0, 0},
		{"missing height", "1920x", 0, 0},
		{"missing width", "x1080", 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w, h := parseResolution(c.in)
			if w != c.wantW || h != c.wantH {
				t.Errorf("parseResolution(%q) = %d,%d, want %d,%d", c.in, w, h, c.wantW, c.wantH)
			}
		})
	}
}
