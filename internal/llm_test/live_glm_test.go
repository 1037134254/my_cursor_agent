package llm_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joho/godotenv"

	llm "my_cursor/internal/llm"
)

// loadDotEnvFromAncestors 便于在任意工作目录下跑测试时仍能读到项目根 .env。
func loadDotEnvFromAncestors() {
	wd, err := os.Getwd()
	if err != nil {
		return
	}
	dir := wd
	for range 24 {
		p := filepath.Join(dir, ".env")
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			_ = godotenv.Load(p)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
}

// TestLiveGLM51Ping 最小连通性测试：真实请求 GLM 5.1（需 LLM_API_KEY）。
// 设置 LLM_LIVE_TEST=1 且配置密钥后才会发起网络请求；否则跳过，避免 CI / 误伤。
func TestLiveGLM51Ping(t *testing.T) {
	if strings.TrimSpace(os.Getenv("LLM_LIVE_TEST")) != "1" {
		t.Skip("未开启 LLM_LIVE_TEST=1：跳过真实 GLM 请求（确认密钥后再跑）")
	}
	loadDotEnvFromAncestors()

	key := llm.APIKeyFromEnv()
	if key == "" {
		t.Skip("未设置密钥：请在环境变量或项目根 .env 中配置 LLM_API_KEY / ZHIPU_API_KEY / BIGMODEL_API_KEY 之一")
	}
	t.Logf("使用密钥长度=%d（不回显内容）", len(key))

	cfg := llm.Config{
		Kind:           llm.ProviderCustom,
		ChatEndpoints:  []string{"https://open.bigmodel.cn/api/paas/v4/chat/completions"},
		Model:          "glm-5.1",
		APIKey:         key,
		TimeoutSeconds: 90,
		MaxRetries:     1,
		RetryBackoffMs: 400,
		RPM:            0,
	}
	if err := llm.SetConfig(cfg); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { llm.Init() })

	reply, err := llm.Chat("只回复两个汉字：你好")
	if err != nil {
		if strings.Contains(err.Error(), "status=401") {
			t.Skipf("GLM 返回 401（密钥无效或未生效）：%v", err)
		}
		t.Fatalf("GLM 调用失败: %v", err)
	}
	if strings.TrimSpace(reply) == "" {
		t.Fatal("回复为空")
	}
	t.Logf("收到回复（截断）: %s", truncateRunes(reply, 160))
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "..."
}
