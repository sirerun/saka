package types

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
