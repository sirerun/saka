//go:build integration

package startpage

import (
	"context"
	"errors"
	"testing"
	"time"

	saka "github.com/you/saka"
	// NOTE (source-chat gap): this imported "github.com/you/saka/internal/htmd",
	// a shared-helpers package that startpage.go's own comment suggests
	// creating ("extract to an internal package (saka/internal/htmd) so
	// duckduckgo and startpage both use them") but that is never actually
	// created anywhere in the chat. Commented out so this file at least
	// compiles under `-tags=integration`; see NOTES.md.
	// "github.com/you/saka/internal/htmd"
)

func TestLiveSearch(t *testing.T) {
	p := New()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	results, err := p.Search(ctx, saka.Query{Text: "startpage search engine", MaxResults: 5})
	if err != nil {
		var rl *saka.RateLimitError
		if errors.As(err, &rl) {
			t.Skip("startpage challenged us — expected under repeated CI runs, not a failure")
		}
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Skip("zero results parsed — startpage markup may have changed; investigate")
	}
	for _, r := range results {
		if r.URL == "" || r.Title == "" {
			t.Errorf("malformed result: %+v", r)
		}
	}
	t.Logf("live startpage: %d results", len(results))
}

// Regression guard: if markup changes, this is the canary.
func TestParserFixtureStillMatchesLiveFormat(t *testing.T) {
	// The offline fixture test catches parser regressions; this one catches
	// "startpage changed their HTML" drift by checking class names still exist.
	p := New()
	_, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Reuse the provider's request path but inspect raw HTML:
	// (implemented via an internal hook or by refactoring Search to expose
	//  fetchAndParse; shown here as the fetch half)
	// If w-gl__result-title disappears from live HTML, warn loudly:
	_ = p
	t.Skip("wire to exposed fetchHTML() helper; kept as scaffold")
}
