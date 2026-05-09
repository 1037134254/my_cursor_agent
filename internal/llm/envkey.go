package llm

import (
	"os"
	"strings"
)

// cleanSecret 去掉首尾空白、UTF-8 BOM、常见引号包裹（.env 里手写 "sk-..." 时可能残留引号）。
func cleanSecret(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "\ufeff")
	s = strings.Trim(s, `"'`+"`")
	return strings.TrimSpace(s)
}

func firstNonEmptyEnv(keys ...string) string {
	for _, k := range keys {
		if v := cleanSecret(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

// APIKeyFromEnv 读取密钥；已配置 ANTHROPIC_BASE_URL 时优先 ANTHROPIC_API_KEY。
func APIKeyFromEnv() string {
	if strings.TrimSpace(os.Getenv("ANTHROPIC_BASE_URL")) != "" {
		if k := firstNonEmptyEnv("ANTHROPIC_API_KEY"); k != "" {
			return k
		}
	}
	return firstNonEmptyEnv("LLM_API_KEY", "ZHIPU_API_KEY", "BIGMODEL_API_KEY", "ANTHROPIC_API_KEY")
}

// AnthropicBaseToChatURL 将网关根地址转为 OpenAI 兼容 chat/completions URL（兼容 One API：仅主机名或已含 /v1）。
func AnthropicBaseToChatURL(base string) string {
	base = strings.TrimSpace(base)
	base = strings.TrimRight(base, "/")
	if base == "" {
		return ""
	}
	if strings.Contains(base, "/chat/completions") {
		return base
	}
	if strings.HasSuffix(base, "/v1") {
		return base + "/chat/completions"
	}
	return base + "/v1/chat/completions"
}
