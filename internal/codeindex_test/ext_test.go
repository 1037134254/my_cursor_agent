package codeindex_test

// 对应 internal/codeindex：扩展名列表解析。

import (
	"testing"

	"my_cursor/internal/codeindex"
)

// TestParseExtList 验证空串返回 nil；逗号分隔、去空格、扩展名归一为小写键。
func TestParseExtList(t *testing.T) {
	if codeindex.ParseExtList("") != nil {
		t.Fatal()
	}
	m := codeindex.ParseExtList(" go , .MD , , tsx ")
	if !m["go"] || !m["md"] || !m["tsx"] || len(m) != 3 {
		t.Fatalf("%+v", m)
	}
}
