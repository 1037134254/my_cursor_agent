package llm_test

// 对应 internal/llm/provider_openai.go：OpenAI 兼容 HTTP 客户端（本地 httptest 模拟）。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	llm "my_cursor/internal/llm"
)

// TestOpenAICompatProviderChatCompletion 验证请求路径与 JSON 响应解析为助手正文。
func TestOpenAICompatProviderChatCompletion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "hello"}},
			},
		})
	}))
	defer ts.Close()

	cfg := llm.Config{
		ChatEndpoints: []string{ts.URL + "/v1/chat/completions"},
		Model:         "m",
		APIKey:        "",
	}
	p := llm.NewOpenAICompatProviderForTest(cfg)
	out, err := p.ChatCompletion(context.Background(), "ping")
	if err != nil {
		t.Fatal(err)
	}
	if out != "hello" {
		t.Fatalf("got %q", out)
	}
}

// TestOpenAICompatProviderBearer 验证配置了 APIKey 时携带 Authorization: Bearer。
func TestOpenAICompatProviderBearer(t *testing.T) {
	var auth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "x"}},
			},
		})
	}))
	defer ts.Close()

	cfg := llm.Config{
		ChatEndpoints: []string{ts.URL + "/v1/chat/completions"},
		Model:         "m",
		APIKey:        "secret",
	}
	p := llm.NewOpenAICompatProviderForTest(cfg)
	_, err := p.ChatCompletion(context.Background(), "p")
	if err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer secret" {
		t.Fatalf("auth %q", auth)
	}
}
