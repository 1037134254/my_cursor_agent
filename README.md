# my_cursor

本项目是一个本地 AI 助手的后端原型，使用 Gin 提供 HTTP 接口，并通过本地 LLM（OpenAI 兼容接口）完成对话,后续期望对其CURSOR,整理自己的知识库+自己的模型优化接入codex能自己修改系统部分简单BUG。

## 功能概览

- 提供 `POST /api/chat` 接口，接收 `{"msg":"..."}`。
- 由 `internal/agent` 组织提示词并调用 `internal/llm`。
- 支持模型通过特殊指令访问工作目录：
  - `[READ]path` 读取 `workspace/` 下文件
  - `[WRITE]path||content` 写入 `workspace/` 下文件
- Web IDE：`GET /` 为 Vue3 + Monaco 构建产物（先执行 `cd web && npm install && npm run build`，生成 `web/dist`）

## 故障排查：401 令牌错误

- 启动时看控制台：应出现 `dotenv: 已加载 ...` 和 `LLM 密钥: 已加载（长度=…）`。若是「未检测到」，说明 `.env` 没被读到（工作目录不对时已改为向上查找项目根）。
- 修改 `.env` 后必须重启进程。
- Windows 下 `.env` 若为「UTF-8 带 BOM」，会在密钥前多出不可见字符导致 401；请保存为 **UTF-8 无 BOM**，或使用 `ZHIPU_API_KEY` / `BIGMODEL_API_KEY` 环境变量（代码会自动去 BOM）。
- 仍报 401：在开放平台核对 key 是否仍为「OpenAI 兼容 / Chat」类产品，必要时重新生成 key。
- **使用 One API / 自建网关时**：`LLM_API_URL` 必须填 **网关的** OpenAI 兼容地址（一般为 `https://<你的网关>/v1/chat/completions`），`LLM_API_KEY` 填 **网关「令牌管理」里发放的 sk**。**不要**再用智谱官方 `open.bigmodel.cn` 地址去验网关令牌，否则会一直 401。

## 环境要求

- Go `1.22+`
- 本地 LLM 服务，兼容 OpenAI Chat Completions 接口
- 默认配置（未设置环境变量时）：
  - Provider：`custom`
  - API 地址：`https://open.bigmodel.cn/api/paas/v4/chat/completions`
  - 模型：`glm-5.1`
- 可选环境变量：
  - `LLM_API_URL`：完整 `chat/completions` 地址（优先于下方网关根地址）
  - `LLM_MODEL`：模型名
  - `LLM_API_KEY`：服务端密钥（安全模式下页面不会提交密钥）
  - **One API / 网关（与部分工具链命名一致）**：`ANTHROPIC_BASE_URL`（仅填网关根，如 `http://your-host`，会自动拼 `/v1/chat/completions`）、`ANTHROPIC_API_KEY`、`ANTHROPIC_MODEL`。若已设置 `LLM_API_URL` 则不再使用 `ANTHROPIC_BASE_URL`。

## 快速开始

1. 安装依赖：

```bash
go mod tidy
```

2. 配置环境变量（推荐）：

```bash
cp .env.example .env
```

然后编辑 `.env` 填入真实密钥（推荐方案 B：`ANTHROPIC_API_KEY`，见 `.env.example`）。

3. 启动服务：

```bash
go run .
```

4.（可选）构建前端 IDE：

```bash
cd web && npm install && npm run build
```

5. 打开 IDE 或测试页：

浏览器访问 [http://127.0.0.1:8089](http://127.0.0.1:8089)（默认端口见 `main.go`）

6. 或直接请求接口：

```bash
curl -X POST http://127.0.0.1:8089/api/chat \
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

- `CHANGELOG.md`：本版功能与环境变量等行为变更摘要
- `main.go`：服务入口
- `api/`：路由与 HTTP 处理器
- `internal/agent/`：Agent 逻辑（工具指令解析）
- `internal/llm/`：LLM 请求与响应解析
- `internal/tool/`：文件读写工具（作用于 `workspace/`）
- `web/`：Vite + Vue3 + Monaco 源码；`web/dist` 为构建输出
- `workspace/`：运行时读写目录（自动创建）
