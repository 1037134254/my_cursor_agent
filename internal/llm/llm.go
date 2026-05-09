package llm

import (
	"context"
)

var defaultGW *Gateway

// Init 读取环境变量并初始化默认网关（应在 main 启动时调用一次）。
func Init() {
	cfg := LoadConfig()
	defaultGW = NewGateway(cfg)
}

func ensureGW() {
	if defaultGW == nil {
		Init()
	}
}

// Chat 非流式对话（默认 Provider + 超时/重试/限流）。
func Chat(prompt string) (string, error) {
	ensureGW()
	return defaultGW.Chat(context.Background(), prompt)
}

// ChatStream 流式对话（ctx 可由上层控制总时长）。
func ChatStream(ctx context.Context, prompt string, onDelta func(string) error) error {
	ensureGW()
	return defaultGW.ChatStream(ctx, prompt, onDelta)
}
