package agent_test

// 对应 internal/agent：parseToolCall 解析。

import (
	"testing"

	"my_cursor/internal/agent"
)

// TestParseToolCall 表驱动验证 [READ]/[WRITE]/[TOOL:name] 及无效输入。
func TestParseToolCall(t *testing.T) {
	tests := []struct {
		in       string
		wantName string
		wantArgs string
		wantOK   bool
	}{
		{"[READ]foo/bar.go", "read_file", "foo/bar.go", true},
		{"[WRITE]x.go||hello", "write_file", "x.go||hello", true},
		{"[TOOL:read_file]path", "read_file", "path", true},
		{"[TOOL:write_file]a||b", "write_file", "a||b", true},
		{"plain text", "", "", false},
		{"[TOOL:]", "", "", false},
	}
	for _, tt := range tests {
		n, a, ok := agent.ParseToolCallForTest(tt.in)
		if ok != tt.wantOK || n != tt.wantName || a != tt.wantArgs {
			t.Fatalf("ParseToolCallForTest(%q) = (%q,%q,%v) want (%q,%q,%v)", tt.in, n, a, ok, tt.wantName, tt.wantArgs, tt.wantOK)
		}
	}
}
