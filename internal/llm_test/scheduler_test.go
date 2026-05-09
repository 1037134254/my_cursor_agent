package llm_test

// 对应 internal/llm/scheduler.go：RoundRobin 端点轮询与空列表行为。

import (
	"testing"

	llm "my_cursor/internal/llm"
)

// TestRoundRobinScheduler 验证三个 endpoint 按 a→b→c→a 顺序轮询。
func TestRoundRobinScheduler(t *testing.T) {
	s := llm.NewRoundRobinScheduler([]string{"a", "b", "c"})
	if s.Len() != 3 {
		t.Fatal(s.Len())
	}
	order := []string{s.Next(), s.Next(), s.Next(), s.Next()}
	want := []string{"a", "b", "c", "a"}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("i=%d got %q want %q", i, order[i], want[i])
		}
	}
}

// TestRoundRobinSchedulerEmpty 验证无 endpoint 时 Next 返回空串且 Len 为 0。
func TestRoundRobinSchedulerEmpty(t *testing.T) {
	s := llm.NewRoundRobinScheduler(nil)
	if s.Next() != "" {
		t.Fatal("expected empty")
	}
	if s.Len() != 0 {
		t.Fatal(s.Len())
	}
}
