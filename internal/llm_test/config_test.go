package llm_test

// 对应 internal/llm/config.go：从环境变量加载 LLM 配置。

import (
	"testing"

	llm "my_cursor/internal/llm"
)

// TestLoadConfigDeepSeekPreset 验证 DEEPSEEK 预设：Kind、Key、默认 URL 与模型。
func TestLoadConfigDeepSeekPreset(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "deepseek")
	t.Setenv("DEEPSEEK_API_KEY", "sk-test")
	t.Setenv("DEEPSEEK_API_URL", "")
	t.Setenv("LLM_API_URL", "")
	t.Setenv("LLM_CLUSTER_ENDPOINTS", "")
	t.Setenv("LLM_MODEL", "")
	t.Setenv("DEEPSEEK_MODEL", "")

	cfg := llm.LoadConfig()
	if cfg.Kind != llm.ProviderDeepSeek {
		t.Fatalf("kind %s", cfg.Kind)
	}
	if cfg.APIKey != "sk-test" {
		t.Fatalf("api key")
	}
	const wantURL = "https://api.deepseek.com/v1/chat/completions"
	if len(cfg.ChatEndpoints) != 1 || cfg.ChatEndpoints[0] != wantURL {
		t.Fatalf("endpoints %+v", cfg.ChatEndpoints)
	}
	if cfg.Model != "deepseek-chat" {
		t.Fatalf("model %s", cfg.Model)
	}
}

// TestLoadConfigClusterEndpoints 验证 LLM_CLUSTER_ENDPOINTS 逗号分隔解析为多 endpoint。
func TestLoadConfigClusterEndpoints(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "qwen")
	t.Setenv("LLM_CLUSTER_ENDPOINTS", "http://a/v1/chat/completions, http://b/v1/chat/completions")

	cfg := llm.LoadConfig()
	if len(cfg.ChatEndpoints) != 2 {
		t.Fatalf("got %+v", cfg.ChatEndpoints)
	}
	if cfg.ChatEndpoints[0] != "http://a/v1/chat/completions" {
		t.Fatal(cfg.ChatEndpoints[0])
	}
}

func TestLoadConfigCustomAnthropic(t *testing.T) {
	cases := []struct {
		name       string
		wantKey    string
		wantURL    string
		wantModel  string
		anthropicK string
		llmKey     string
	}{
		{
			name:       "anthropic_key_wins",
			wantKey:    "sk-from-gateway",
			wantURL:    "http://gw.example.com/v1/chat/completions",
			wantModel:  "glm-5.1",
			anthropicK: "sk-from-gateway",
			llmKey:     "sk-should-not-win",
		},
		{
			name:       "fallback_llm_key",
			wantKey:    "sk-fallback",
			wantURL:    "http://gw.example.com/v1/chat/completions",
			wantModel:  "glm-5.1",
			anthropicK: "",
			llmKey:     "sk-fallback",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LLM_PROVIDER", "custom")
			t.Setenv("LLM_CLUSTER_ENDPOINTS", "")
			t.Setenv("LLM_API_URL", "")
			t.Setenv("ANTHROPIC_BASE_URL", "http://gw.example.com")
			t.Setenv("ANTHROPIC_API_KEY", tc.anthropicK)
			t.Setenv("LLM_API_KEY", tc.llmKey)
			t.Setenv("ANTHROPIC_MODEL", "glm-5.1")
			t.Setenv("LLM_MODEL", "")

			cfg := llm.LoadConfig()
			if cfg.APIKey != tc.wantKey {
				t.Fatalf("api key got %q want %q", cfg.APIKey, tc.wantKey)
			}
			if len(cfg.ChatEndpoints) != 1 || cfg.ChatEndpoints[0] != tc.wantURL {
				t.Fatalf("endpoints %+v", cfg.ChatEndpoints)
			}
			if cfg.Model != tc.wantModel {
				t.Fatalf("model %s", cfg.Model)
			}
		})
	}
}
