package rag_test

// 对应 internal/rag：chunk 点 ID 生成。

import (
	"testing"

	"my_cursor/internal/rag"
)

// TestChunkPointIDStable 验证相同 source+正文 ID 一致，正文不同则 ID 不同。
func TestChunkPointIDStable(t *testing.T) {
	a := rag.ChunkPointIDForTest("a.go", "body1")
	b := rag.ChunkPointIDForTest("a.go", "body1")
	c := rag.ChunkPointIDForTest("a.go", "body2")
	if a != b {
		t.Fatal("same source+text should match")
	}
	if a == c {
		t.Fatal("different text should differ")
	}
}
