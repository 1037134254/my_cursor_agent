# my_cursor

本项目是一个本地 AI 助手的后端原型，使用 Gin 提供 HTTP 接口，并通过本地 LLM（OpenAI 兼容接口）完成对话,后续期望对其CURSOR,整理自己的知识库+自己的模型优化接入codex能自己修改系统部分简单BUG。

## 功能概览

- 提供 `POST /api/chat` 接口，接收 `{"msg":"..."}`。
- 由 `internal/agent` 组织提示词并调用 `internal/llm`。
- 支持模型通过特殊指令访问工作目录：
  - `[READ]path` 读取 `workspace/` 下文件
  - `[WRITE]path||content` 写入 `workspace/` 下文件
- 内置最小测试页面：`GET /`（页面文件在 `web/index.html`）

## 环境要求

- Go `1.22+`
- 本地 LLM 服务，兼容 OpenAI Chat Completions 接口
- 默认配置：
  - API 地址：`http://127.0.0.1:11434/v1/chat/completions`
  - 模型：`qwen2.5-coder:7b-instruct-q4_K_M`
- 可选环境变量：
  - `LLM_API_URL`：覆盖默认接口地址
  - `LLM_MODEL`：覆盖默认模型

## 快速开始

1. 安装依赖：

```bash
go mod tidy
```

2. 启动服务：

```bash
go run .
```

3. 打开测试页面：

浏览器访问 [http://localhost:8080](http://localhost:8080)

4. 或直接请求接口：

```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d "{\"msg\":\"帮我创建 hello.txt，内容是 hello world\"}"
```

## 返回格式

成功示例：

```json
{
  "code": 0,
  "message": "success",
  "data": "..."
}
```

失败示例：

```json
{
  "error": "..."
}
```

## 目录说明

- `main.go`：服务入口
- `api/`：路由与 HTTP 处理器
- `internal/agent/`：Agent 逻辑（工具指令解析）
- `internal/llm/`：LLM 请求与响应解析
- `internal/tool/`：文件读写工具（作用于 `workspace/`）
- `web/`：最小前端测试页面
- `workspace/`：运行时读写目录（自动创建）
