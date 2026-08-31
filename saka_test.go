package saka

import (
	"strings"
	"testing"
)

func TestPageChunks(t *testing.T) {
	p := &Page{URL: "http://x", Text: strings.Repeat("word ", 400)}
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

func TestPageChunksEmpty(t *testing.T) {
	p := &Page{URL: "http://x"}
	_, err := p.Chunks()
	if err == nil {
		t.Fatal("expected error for empty page")
	}
}
