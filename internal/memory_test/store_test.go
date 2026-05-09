package memory_test

// 对应 internal/memory：会话 ID 与消息环形缓冲。

import (
	"fmt"
	"testing"
	"time"

	"my_cursor/internal/memory"
)

const testTenant = "acme"

// TestEnsureSessionID 验证非空 ID 原样保留；空串时生成新会话 ID。
func TestEnsureSessionID(t *testing.T) {
	s := memory.EnsureSessionID(testTenant, "keep-me")
	if s != "keep-me" {
		t.Fatal(s)
	}
	gen := memory.EnsureSessionID(testTenant, "")
	if gen == "" || gen == "keep-me" {
		t.Fatal(gen)
	}
}

// TestAppendGetRecent 验证追加消息后 GetRecent 条数与顺序（含 limit=0 取全部）。
func TestAppendGetRecent(t *testing.T) {
	sid := fmt.Sprintf("test-session-%d", time.Now().UnixNano())
	memory.Append(testTenant, sid, memory.Message{Role: "user", Content: "a"})
	memory.Append(testTenant, sid, memory.Message{Role: "assistant", Content: "b"})
	memory.Append(testTenant, sid, memory.Message{Role: "user", Content: "c"})

	all := memory.GetRecent(testTenant, sid, 0)
	if len(all) != 3 {
		t.Fatalf("len=%d", len(all))
	}
	last2 := memory.GetRecent(testTenant, sid, 2)
	if len(last2) != 2 || last2[0].Content != "b" || last2[1].Content != "c" {
		t.Fatalf("%+v", last2)
	}
}

func TestTenantIsolation(t *testing.T) {
	sid := "shared-sid"
	memory.Append("t1", sid, memory.Message{Role: "user", Content: "x"})
	memory.Append("t2", sid, memory.Message{Role: "user", Content: "y"})
	a := memory.GetRecent("t1", sid, 0)
	b := memory.GetRecent("t2", sid, 0)
	if len(a) != 1 || a[0].Content != "x" || len(b) != 1 || b[0].Content != "y" {
		t.Fatal(a, b)
	}
}
