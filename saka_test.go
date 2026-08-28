package saka

import (
	"strings"
	"testing"
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

func TestPageChunks(t *testing.T) {
	p := &Page{URL: "http://x", text: strings.Repeat("word ", 400)}
	ch, err := p.Chunks()
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for c := range ch {
		if c.Done {
			break
		}
		if c.Text == "" {
			t.Error("empty chunk")
		}
		if c.Seq != n {
			t.Errorf("out of order: seq %d, want %d", c.Seq, n)
		}
		n++
	}
	if n == 0 {
		t.Error("no chunks emitted")
	}
}
