package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	saka "github.com/sirerun/saka"
)

// capturingRealEngine wraps a real *saka.Engine and records the *saka.Results
// it returns from Search, so a test can assert on the exact value ExecuteTool
// received from the real Engine -> chain -> provider dispatch path, not a
// stand-in. It embeds the engine so Fetch/FetchStream/SearchStream pass
// straight through.
type capturingRealEngine struct {
	*saka.Engine
	got *saka.Results
}

func (c capturingRealEngine) Search(ctx context.Context, q saka.Query) (*saka.Results, error) {
	res, err := c.Engine.Search(ctx, q)
	if err != nil {
		return nil, err
	}
	*c.got = *res
	return res, nil
}

// TestExecuteToolRoutesImagesVerticalToSearxngImagesProvider proves T8.4: a
// web_search tool call with "vertical":"images" travels through
// ExecuteTool -> a real *saka.Engine -> the images-vertical chain -> the
// real, registered "searxng-images" provider (provider/searxng/images.go,
// T8.2) hitting a mocked SearXNG categories=images endpoint. It is registered
// for real via saka.go's blank import of provider/searxng, so this test
// configures it by its real name against an httptest server rather than
// faking a provider under that name (see tools_test.go's fakeVerticalProvider
// doc comment for why a literal-name fake would panic on double-registration).
func TestExecuteToolRoutesImagesVerticalToSearxngImagesProvider(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("categories"); got != "images" {
			t.Errorf("categories=%q, want images", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[
			{"title":"A cat","img_src":"https://x/full.jpg","thumbnail_src":"https://x/thumb.jpg","engine":"bing images","resolution":"1920x1080"}
		]}`))
	}))
	defer srv.Close()

	engine, err := saka.New(saka.Config{
		Providers: []saka.ProviderConfig{
			{Name: "searxng-images", URL: srv.URL, Vertical: "images", RPS: 1},
		},
		Fetch: saka.FetchConfig{RPS: 1},
	})
	if err != nil {
		t.Fatal(err)
	}

	var got saka.Results
	capturing := capturingRealEngine{Engine: engine, got: &got}

	out, err := ExecuteTool(context.Background(), capturing, "web_search",
		json.RawMessage(`{"query":"cats","vertical":"images"}`))
	if err != nil {
		t.Fatal(err)
	}
	if out == "" {
		t.Fatal("ExecuteTool returned empty output")
	}

	if got.Provider != "searxng-images" {
		t.Errorf("Results.Provider = %q, want %q", got.Provider, "searxng-images")
	}
	if len(got.Results) == 0 {
		t.Fatal("Results.Results is empty")
	}
	if got.Results[0].ThumbnailURL == "" {
		t.Error("Results.Results[0].ThumbnailURL is empty, want non-empty")
	}
}
