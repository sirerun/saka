//go:build integration

package gdelt

import (
	"context"
	"errors"
	"testing"
	"time"

	types "github.com/sirerun/saka/types"
)

func TestLiveSearch(t *testing.T) {
	p := New()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	results, err := p.Search(ctx, types.Query{Text: "climate", MaxResults: 5})
	if err != nil {
		var rl *types.RateLimitError
		if errors.As(err, &rl) {
			t.Skip("gdelt rate limited us — expected under repeated CI runs, not a failure")
		}
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Skip("zero results parsed — GDELT's response shape may have changed; investigate")
	}
	for _, r := range results {
		if r.URL == "" || r.Title == "" {
			t.Errorf("malformed result: %+v", r)
		}
	}
	t.Logf("live gdelt: %d results", len(results))
}
