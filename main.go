package main

import (
	"log"
	"my_cursor/api"
	"my_cursor/internal/auth"
	"my_cursor/internal/auth/oauth"
	"my_cursor/internal/llm"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	loadDotEnv()
	oauth.InitProviders()
	_ = auth.InitRefreshStore()
	logLLMKeyHint()
	logAuthMode()

	// 初始化模型
	llm.Init()

	// 创建工作目录（含默认租户子目录）
	_ = os.MkdirAll("./workspace", 0755)
	_ = os.MkdirAll("./workspace/default", 0755)

	r := gin.Default()

	// 注册接口
	api.RegisterRoutes(r)

	println("项目启动成功 :8089")
	_ = r.Run(":8089")
}

// loadDotEnv 从当前目录逐级向上查找 .env（解决 IDE 启动时工作目录不在项目根导致读不到密钥）。
func loadDotEnv() {
	if p := findDotEnvPath(); p != "" {
		if err := godotenv.Load(p); err != nil {
			log.Printf("dotenv: 加载 %s 失败: %v", p, err)
			return
		}
		log.Printf("dotenv: 已加载 %s", p)
		return
	}
	log.Println("dotenv: 未找到 .env（将仅用系统环境变量；可把 .env 放在项目根或与 exe 同目录）")
}

func findDotEnvPath() string {
	var candidates []string
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates, filepath.Join(dir, ".env"))
	}
	wd, err := os.Getwd()
	if err != nil {
		for _, p := range candidates {
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p
			}
		}
		return ""
	}
	dir := wd
	for range 16 {
		candidates = append(candidates, filepath.Join(dir, ".env"))
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func logAuthMode() {
	if auth.Enabled() {
		anon := "关"
		if auth.AllowAnonymousDebug() {
			anon = "开"
		}
		log.Printf("认证: AUTH_ENABLED=true（JWT + RBAC）；匿名调试=%s", anon)
		return
	}
	log.Println("认证: 未启用 AUTH_ENABLED（开发模式：合成租户 default 管理员）")
}

func logLLMKeyHint() {
	k := llm.APIKeyFromEnv()
	if k == "" {
		log.Println("LLM 密钥: 未检测到（检查 .env 是否被加载、变量名是否正确）")
		return
	}
	log.Printf("LLM 密钥: 已加载（长度=%d，不回显内容）", len(k))
}
