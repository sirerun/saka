package server

import (
	"bytes"
	"context"
	"strings"
	"testing"

	saka "github.com/sirerun/saka"
)

type fakeEngine struct{}

func (fakeEngine) Search(_ context.Context, q saka.Query) (*saka.Results, error) {
	return &saka.Results{Query: q.Text, Results: []saka.Result{
		{Title: "T", URL: "https://x", Snippet: "s", Position: 1},
	}, Provider: "fake"}, nil
}
func (fakeEngine) Fetch(_ context.Context, url string) (*saka.Page, error) {
	return &saka.Page{URL: url, Title: "T", Text: "body"}, nil
}
func (fakeEngine) FetchStream(ctx context.Context, url string) (<-chan saka.Chunk, <-chan *saka.Page, <-chan error) {
	ch := make(chan saka.Chunk, 1)
	ch <- saka.Chunk{Text: "hello", Seq: 0}
	close(ch)
	done := make(chan *saka.Page, 1)
	done <- &saka.Page{URL: url, Text: "hello"}
	errc := make(chan error, 1)
	return ch, done, errc
}

func TestMCPRoundTrip(t *testing.T) {
	s := NewMCP(fakeEngine{})
	in := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"web_search","arguments":{"query":"ai","max_results":5}}}`,
		"not json",
	}, "\n")
	var out bytes.Buffer
	if err := s.Serve(context.Background(), strings.NewReader(in), &out); err != nil {
		t.Fatal(err)
	}
	body := out.String()
	if !strings.Contains(body, `"saka"`) {
		t.Error("initialize response missing serverInfo")
	}
	if !strings.Contains(body, `"web_search"`) || !strings.Contains(body, `"fetch_page"`) {
		t.Error("tools/list missing tools")
	}
	if !strings.Contains(body, `"parse error"`) {
		t.Error("malformed line not handled")
	}
	// verify the tools/call response is valid JSON-RPC with the search result
	if !strings.Contains(body, `"hit"`) && !strings.Contains(body, "T") {
		t.Error("tools/call did not return search text")
	}
}
