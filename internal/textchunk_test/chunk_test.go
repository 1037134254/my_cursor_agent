package textchunk_test

// 对应 internal/textchunk：通用文本 Split。

import (
	"testing"

	"my_cursor/internal/textchunk"
)

// TestSplitOverlap 验证定长窗口加重叠时得到多块。
func TestSplitOverlap(t *testing.T) {
	chunks := textchunk.Split("abcdefghij", 4, 1)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %v", chunks)
	}
}

// TestSplitEmpty 验证仅空白输入返回 nil。
func TestSplitEmpty(t *testing.T) {
	if textchunk.Split("   ", 100, 0) != nil {
		t.Fatal("expected nil")
	}
}
