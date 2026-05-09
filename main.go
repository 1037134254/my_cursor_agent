package main

import (
	"my_cursor/api"
	"my_cursor/internal/llm"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化模型
	llm.Init()

	// 创建工作目录
	_ = os.MkdirAll("./workspace", 0755)

	r := gin.Default()

	// 注册接口
	api.RegisterRoutes(r)

	println("项目启动成功 :8089")
	_ = r.Run(":8089")
}
