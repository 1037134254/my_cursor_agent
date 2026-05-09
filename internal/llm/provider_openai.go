package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Provider OpenAI /chat/completions 兼容实现（Ollama、DeepSeek 等均适用）。
type Provider interface {
	ChatCompletion(ctx context.Context, prompt string) (string, error)
	ChatCompletionStream(ctx context.Context, prompt string, onDelta func(string) error) error
}

type openAICompatProvider struct {
	sched  EndpointScheduler
	model  string
	apiKey string
}

func newOpenAICompatProvider(cfg Config) Provider {
	return &openAICompatProvider{
		sched:  NewRoundRobinScheduler(cfg.ChatEndpoints),
		model:  cfg.Model,
		apiKey: cfg.APIKey,
	}
}

func (p *openAICompatProvider) ChatCompletion(ctx context.Context, prompt string) (string, error) {
	url := p.sched.Next()
	if url == "" {
		return "", fmt.Errorf("未配置聊天 endpoint（检查 LLM_API_URL / LLM_CLUSTER_ENDPOINTS）")
	}
	body, err := json.Marshal(map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("llm error: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return parseChatContent(raw)
}

func (p *openAICompatProvider) ChatCompletionStream(ctx context.Context, prompt string, onDelta func(string) error) error {
	url := p.sched.Next()
	if url == "" {
		return fmt.Errorf("未配置聊天 endpoint")
	}
	body, err := json.Marshal(map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"stream": true,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("call llm failed (url=%s): %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("llm error: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return readChatStreamSSE(ctx, resp.Body, onDelta)
}
