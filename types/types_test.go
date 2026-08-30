package types

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestSplitChunks(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		size       int
		wantChunks []int // want rune count per chunk
	}{
		{
			name:       "empty text",
			text:       "",
			size:       900,
			wantChunks: nil,
		},
		{
			name:       "exact size boundary, no trailing empty chunk",
			text:       strings.Repeat("a", 1800),
			size:       900,
			wantChunks: []int{900, 900},
		},
		{
			name:       "trailing partial chunk",
			text:       strings.Repeat("a", 2000),
			size:       900,
			wantChunks: []int{900, 900, 200},
		},
		{
			name:       "multi-byte runes split on rune boundaries, not byte boundaries",
			text:       strings.Repeat("€", 950), // 3 bytes/rune; a byte-based split would corrupt a rune here
			size:       900,
			wantChunks: []int{900, 50},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := splitChunks(tt.text, tt.size)
			if len(parts) != len(tt.wantChunks) {
				t.Fatalf("want %d chunks, got %d: %v", len(tt.wantChunks), len(parts), parts)
			}
			var reassembled strings.Builder
			for i, part := range parts {
				if !utf8.ValidString(part) {
					t.Errorf("chunk %d is not valid UTF-8 (rune split mid-character): %q", i, part)
				}
				if n := utf8.RuneCountInString(part); n != tt.wantChunks[i] {
					t.Errorf("chunk %d: want %d runes, got %d", i, tt.wantChunks[i], n)
				}
				reassembled.WriteString(part)
			}
			if got := reassembled.String(); got != tt.text {
				t.Errorf("reassembled chunks do not equal original text: got %q, want %q", got, tt.text)
			}
		})
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

func TestQueryWithDefaultsLeavesVerticalUntouched(t *testing.T) {
	q := Query{Text: "x"}.WithDefaults()
	if q.Vertical != "" {
		t.Errorf("want empty Vertical left untouched, got %q", q.Vertical)
	}

	q = Query{Text: "x", Vertical: "news"}.WithDefaults()
	if q.Vertical != "news" {
		t.Errorf("want Vertical preserved as %q, got %q", "news", q.Vertical)
	}
}

func TestQueryVerticalField(t *testing.T) {
	q := Query{Vertical: "news"}
	if q.Vertical != "news" {
		t.Errorf("want Vertical %q, got %q", "news", q.Vertical)
	}
}

func TestProviderConfigVerticalField(t *testing.T) {
	pc := ProviderConfig{Name: "gdelt", Vertical: "news"}
	if pc.Vertical != "news" {
		t.Errorf("want Vertical %q, got %q", "news", pc.Vertical)
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
