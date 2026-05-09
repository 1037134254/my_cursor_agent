package codechunk_test

// 对应 internal/codechunk：按行切分与重叠。

import (
	"strings"
	"testing"

	"my_cursor/internal/codechunk"
)

// TestSplitLinesOverlap 验证多行文本在 maxLines 与 overlap 下产生多个 chunk。
func TestSplitLinesOverlap(t *testing.T) {
	var lines []string
	for i := 0; i < 10; i++ {
		lines = append(lines, "x")
	}
	text := strings.Join(lines, "\n")
	ch := codechunk.SplitLines(text, 4, 1)
	if len(ch) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(ch))
	}
}
