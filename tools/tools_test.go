package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	saka "github.com/sirerun/saka"
	"github.com/sirerun/saka/types"
)

type fakeSearcher struct{}

func (fakeSearcher) Search(_ context.Context, q saka.Query) (*saka.Results, error) {
	return &saka.Results{Query: q.Text, Results: []saka.Result{
		{Title: "T", URL: "https://x", Snippet: "snip", Position: 1},
	}, Provider: "fake"}, nil
}
func (fakeSearcher) Fetch(_ context.Context, u string) (*saka.Page, error) {
	return &saka.Page{URL: u, Title: "T", Text: "body text"}, nil
}
func (fakeSearcher) FetchStream(ctx context.Context, u string) (<-chan saka.Chunk, <-chan *saka.Page, <-chan error) {
	return nil, nil, nil
}
func (fakeSearcher) SearchStream(ctx context.Context, q saka.Query) (<-chan saka.Result, <-chan *saka.Results, <-chan error) {
	return nil, nil, nil
}

func TestExecuteToolSearch(t *testing.T) {
	out, err := ExecuteTool(context.Background(), fakeSearcher{}, "web_search",
		json.RawMessage(`{"query":"ai news","max_results":3}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "https://x") {
		t.Errorf("output missing URL: %s", out)
	}
}

func TestExecuteToolFetch(t *testing.T) {
	out, err := ExecuteTool(context.Background(), fakeSearcher{}, "fetch_page",
		json.RawMessage(`{"url":"https://x/a"}`))
	if err != nil || !strings.Contains(out, "body text") {
		t.Fatalf("fetch_page failed: %v %s", err, out)
	}
}

func TestSchemasAreValidJSON(t *testing.T) {
	for _, s := range []string{SearchSchema(), FetchSchema()} {
		var v any
		if err := json.Unmarshal([]byte(s), &v); err != nil {
			t.Errorf("invalid schema: %v", err)
		}
	}
}

func TestSearchSchemaHasOptionalVertical(t *testing.T) {
	var schema struct {
		Properties map[string]any `json:"properties"`
		Required   []string       `json:"required"`
	}
	if err := json.Unmarshal([]byte(SearchSchema()), &schema); err != nil {
		t.Fatal(err)
	}
	if _, ok := schema.Properties["vertical"]; !ok {
		t.Fatal(`schema missing "vertical" property`)
	}
	for _, r := range schema.Required {
		if r == "vertical" {
			t.Fatal("vertical must be optional, not required")
		}
	}
}

func TestSearchArgsUnmarshalsVertical(t *testing.T) {
	var a searchArgs
	if err := json.Unmarshal([]byte(`{"query":"q","vertical":"news"}`), &a); err != nil {
		t.Fatal(err)
	}
	if a.Vertical != "news" {
		t.Errorf("Vertical = %q, want %q", a.Vertical, "news")
	}
}

type capturingSearcher struct {
	got *saka.Query
}

func (c capturingSearcher) Search(_ context.Context, q saka.Query) (*saka.Results, error) {
	*c.got = q
	return &saka.Results{Query: q.Text, Provider: "fake"}, nil
}
func (c capturingSearcher) Fetch(_ context.Context, u string) (*saka.Page, error) {
	return &saka.Page{URL: u}, nil
}
func (c capturingSearcher) FetchStream(_ context.Context, _ string) (<-chan saka.Chunk, <-chan *saka.Page, <-chan error) {
	return nil, nil, nil
}
func (c capturingSearcher) SearchStream(_ context.Context, _ saka.Query) (<-chan saka.Result, <-chan *saka.Results, <-chan error) {
	return nil, nil, nil
}

func TestExecuteToolPassesVerticalThroughToQuery(t *testing.T) {
	var got saka.Query
	if _, err := ExecuteTool(context.Background(), capturingSearcher{got: &got}, "web_search",
		json.RawMessage(`{"query":"ai news","vertical":"news"}`)); err != nil {
		t.Fatal(err)
	}
	if got.Vertical != "news" {
		t.Errorf("Query.Vertical = %q, want %q", got.Vertical, "news")
	}
}

// fakeVerticalProvider is a types.Provider stand-in registered under a
// test-only name so TestExecuteToolRoutesNewsVerticalToGdeltProvider can
// exercise the full ExecuteTool -> Engine -> chain -> provider path without
// hitting GDELT's live API. It cannot be registered under the real "gdelt"
// name: saka.go blank-imports provider/gdelt (T7.4), which self-registers
// "gdelt" for real in this same test binary, so a second Register("gdelt",
// ...) here would panic with "already registered".
type fakeVerticalProvider struct {
	name  string
	title string
}

func (f fakeVerticalProvider) Name() string { return f.name }
func (f fakeVerticalProvider) Search(_ context.Context, _ types.Query) ([]types.Result, error) {
	return []types.Result{{Title: f.title, URL: "https://x", Position: 1}}, nil
}

func init() {
	if err := types.Register("test-gdelt-fake", func(types.ProviderConfig) (types.Provider, error) {
		return fakeVerticalProvider{name: "test-gdelt-fake", title: "GDELT-NEWS-MARKER"}, nil
	}); err != nil {
		panic(err)
	}
	if err := types.Register("test-general-web", func(types.ProviderConfig) (types.Provider, error) {
		return fakeVerticalProvider{name: "test-general-web", title: "GENERAL-WEB-MARKER"}, nil
	}); err != nil {
		panic(err)
	}
}

func TestExecuteToolRoutesNewsVerticalToGdeltProvider(t *testing.T) {
	engine, err := saka.New(saka.Config{
		Providers: []saka.ProviderConfig{
			{Name: "test-general-web", RPS: 1},
			{Name: "test-gdelt-fake", RPS: 1, Vertical: "news"},
		},
		Fetch: saka.FetchConfig{RPS: 1},
	})
	if err != nil {
		t.Fatal(err)
	}

	out, err := ExecuteTool(context.Background(), engine, "web_search",
		json.RawMessage(`{"query":"ai regulation","vertical":"news"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "GDELT-NEWS-MARKER") {
		t.Errorf("vertical=news should route through gdelt, got: %s", out)
	}

	out, err = ExecuteTool(context.Background(), engine, "web_search",
		json.RawMessage(`{"query":"ai regulation"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "GENERAL-WEB-MARKER") {
		t.Errorf("default search should stay on the general web chain, got: %s", out)
	}
}
