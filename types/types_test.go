package types

import (
	"strings"
	"testing"
	"time"
)

func TestSplitChunks(t *testing.T) {
	text := strings.Repeat("a", 2000)
	parts := splitChunks(text, 900)
	if len(parts) != 3 {
		t.Fatalf("want 3 chunks, got %d", len(parts))
	}
	if len(parts[0]) != 900 || len(parts[2]) != 200 {
		t.Errorf("chunk sizes wrong: %d %d %d", len(parts[0]), len(parts[1]), len(parts[2]))
	}
}

func TestQueryWithDefaults(t *testing.T) {
	q := Query{Text: "x"}.WithDefaults()
	if q.MaxResults != 10 {
		t.Errorf("want default MaxResults 10, got %d", q.MaxResults)
	}

	q = Query{Text: "x", MaxResults: 5}.WithDefaults()
	if q.MaxResults != 5 {
		t.Errorf("want MaxResults preserved at 5, got %d", q.MaxResults)
	}
}

func TestPageChunksNoText(t *testing.T) {
	p := &Page{URL: "https://x"}
	if _, err := p.Chunks(); err == nil {
		t.Error("want error for page with no text")
	}
}

func TestPageChunks(t *testing.T) {
	p := &Page{Text: strings.Repeat("a", 2000)}
	ch, err := p.Chunks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var chunks []Chunk
	for c := range ch {
		chunks = append(chunks, c)
	}

	if len(chunks) != 4 {
		t.Fatalf("want 3 text chunks + 1 done chunk, got %d", len(chunks))
	}
	for i, c := range chunks[:3] {
		if c.Seq != i {
			t.Errorf("chunk %d: want Seq %d, got %d", i, i, c.Seq)
		}
	}
	if !chunks[3].Done {
		t.Error("want final chunk marked Done")
	}
}

func TestRateLimitErrorError(t *testing.T) {
	err := &RateLimitError{Provider: "duckduckgo", RetryAfter: time.Second}
	want := "saka: duckduckgo rate limited"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
