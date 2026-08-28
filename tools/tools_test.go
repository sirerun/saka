package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	saka "github.com/you/saka"
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
