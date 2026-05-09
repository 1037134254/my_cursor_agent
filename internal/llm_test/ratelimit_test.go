package llm_test

// 对应 internal/llm/ratelimit.go：RPM 限流与 nil 安全行为。

import (
	"context"
	"testing"

	llm "my_cursor/internal/llm"
)

// TestNewRPMLimiterNil 验证 rpm≤0 不创建限流器；nil 上 Wait 不报错。
func TestNewRPMLimiterNil(t *testing.T) {
	if llm.NewRPMLimiterForTest(0) != nil {
		t.Fatal("expected nil for 0 rpm")
	}
	if llm.NewRPMLimiterForTest(-1) != nil {
		t.Fatal("expected nil for negative")
	}
	var l *llm.RPMLimiter
	if err := l.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// TestRPMLimiterCancelWhileWaiting 验证排队等待过程中取消上下文返回 Canceled。
func TestRPMLimiterCancelWhileWaiting(t *testing.T) {
	l := llm.NewRPMLimiterForTest(1)
	if err := l.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- l.Wait(ctx)
	}()
	cancel()
	err := <-done
	if err != context.Canceled {
		t.Fatalf("want canceled, got %v", err)
	}
}
