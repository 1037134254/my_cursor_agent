package llm_test

// 对应 internal/llm/gateway.go：重试策略、Gateway.Chat 与占位 Provider。

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	llm "my_cursor/internal/llm"
)

// TestIsRetryable 验证各类错误是否应触发重试（含 net.Error 超时接口）。
func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"503 body", fmt.Errorf("llm error: status=503 body=x"), true},
		{"429", fmt.Errorf("429 too many"), true},
		{"502", errors.New("upstream 502"), true},
		{"504", errors.New("504"), true},
		{"timeout substring", errors.New("context deadline exceeded timeout"), true},
		{"EOF", errors.New("read EOF"), true},
		{"connection refused", errors.New("connection refused"), true},
		{"400 bad", fmt.Errorf("llm error: status=400"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := llm.IsRetryableForTest(tt.err); got != tt.want {
				t.Fatalf("IsRetryableForTest(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}

	var te net.Error = &timeoutErrStub{}
	if !llm.IsRetryableForTest(te) {
		t.Fatal("expected timeout net.Error to be retryable")
	}
}

type timeoutErrStub struct{}

func (*timeoutErrStub) Error() string   { return "timeout" }
func (*timeoutErrStub) Timeout() bool   { return true }
func (*timeoutErrStub) Temporary() bool { return false }

type stubProvider struct {
	chatCalls int
	chatFn    func(ctx context.Context, prompt string) (string, error)
	streamFn  func(ctx context.Context, prompt string, onDelta func(string) error) error
}

func (s *stubProvider) ChatCompletion(ctx context.Context, prompt string) (string, error) {
	s.chatCalls++
	if s.chatFn != nil {
		return s.chatFn(ctx, prompt)
	}
	return "", nil
}

func (s *stubProvider) ChatCompletionStream(ctx context.Context, prompt string, onDelta func(string) error) error {
	if s.streamFn != nil {
		return s.streamFn(ctx, prompt, onDelta)
	}
	return nil
}

// TestGatewayChatRetriesRetryable 验证可重试错误（如 503）会再次调用 Provider 直至成功。
func TestGatewayChatRetriesRetryable(t *testing.T) {
	var calls int
	st := &stubProvider{
		chatFn: func(ctx context.Context, prompt string) (string, error) {
			calls++
			if calls < 2 {
				return "", fmt.Errorf("llm error: status=503 body=x")
			}
			return "ok", nil
		},
	}
	g := llm.NewGatewayForTest(st, 5*time.Second, 3, time.Millisecond, nil)
	out, err := g.Chat(context.Background(), "hi")
	if err != nil {
		t.Fatal(err)
	}
	if out != "ok" {
		t.Fatalf("got %q", out)
	}
	if calls != 2 {
		t.Fatalf("expected 2 provider calls, got %d", calls)
	}
}

// TestGatewayChatNoRetryOn400 验证客户端错误（400）不重试，仅调用 Provider 一次。
func TestGatewayChatNoRetryOn400(t *testing.T) {
	var calls int
	st := &stubProvider{
		chatFn: func(ctx context.Context, prompt string) (string, error) {
			calls++
			return "", fmt.Errorf("llm error: status=400 body=bad")
		},
	}
	g := llm.NewGatewayForTest(st, time.Second, 3, time.Millisecond, nil)
	_, err := g.Chat(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}
