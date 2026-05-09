package codechunk

import (
	"strings"
	"testing"
)

func TestSplitLinesOverlap(t *testing.T) {
	var lines []string
	for i := 0; i < 10; i++ {
		lines = append(lines, "x")
	}
	text := strings.Join(lines, "\n")
	ch := SplitLines(text, 4, 1)
	if len(ch) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(ch))
	}
}
