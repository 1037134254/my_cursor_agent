package llm

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// Gateway AI 网关：限流、超时、重试；底层 Provider 可替换，集群调度通过 EndpointScheduler 扩展。
type Gateway struct {
	prov       Provider
	timeout    time.Duration
	maxRetries int
	backoff    time.Duration
	limiter    *rpmLimiter
}

// NewGateway 由 LoadConfig 结果构造。
func NewGateway(cfg Config) *Gateway {
	return &Gateway{
		prov:       newOpenAICompatProvider(cfg),
		timeout:    time.Duration(cfg.TimeoutSeconds) * time.Second,
		maxRetries: cfg.MaxRetries,
		backoff:    time.Duration(cfg.RetryBackoffMs) * time.Millisecond,
		limiter:    newRPMLimiter(cfg.RPM),
	}
}

// Chat 非流式调用（带限流 + 重试）。
func (g *Gateway) Chat(ctx context.Context, prompt string) (string, error) {
	if err := g.limiter.Wait(ctx); err != nil {
		return "", err
	}
	var lastErr error
	for attempt := 0; attempt <= g.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(g.backoff * time.Duration(attempt)):
			}
		}
		ctx2, cancel := context.WithTimeout(ctx, g.timeout)
		out, err := g.prov.ChatCompletion(ctx2, prompt)
		cancel()
		if err == nil {
			return out, nil
		}
		lastErr = err
		if !isRetryable(err) {
			return "", err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("unknown llm failure")
	}
	return "", lastErr
}

// ChatStream 流式调用（限流；可重试仅在连接阶段失败时有意义，此处做有限次重试）。
func (g *Gateway) ChatStream(ctx context.Context, prompt string, onDelta func(string) error) error {
	if err := g.limiter.Wait(ctx); err != nil {
		return err
	}
	var lastErr error
	for attempt := 0; attempt <= g.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(g.backoff * time.Duration(attempt)):
			}
		}
		err := g.prov.ChatCompletionStream(ctx, prompt, onDelta)
		if err == nil {
			return nil
		}
		lastErr = err
		if !isRetryable(err) {
			return err
		}
	}
	return lastErr
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	s := err.Error()
	if strings.Contains(s, "429") || strings.Contains(s, "502") || strings.Contains(s, "503") || strings.Contains(s, "504") {
		return true
	}
	if strings.Contains(s, "connection refused") || strings.Contains(s, "EOF") || strings.Contains(s, "timeout") {
		return true
	}
	return false
}
