# 变更记录

本文档概述与本仓库相关的**文档与行为变更**，便于评审与发布说明。

## 文档入口一览

| 文件 | 说明 |
|------|------|
| `README.md` | 项目说明、快速开始、401 排查、`ANTHROPIC_*`、**企业认证（JWT / `AUTH_USERS` / 微信扫码）**、目录结构 |
| `.env.example` | 环境变量模板（含 LLM 网关变量与**认证 / 微信 OAuth** 占位注释） |
| `http/glm-chat.http` | VS Code REST Client 示例请求 |
| `http/http-client.env.json.example` | REST Client 私有环境示例（勿提交真实密钥） |
| `postman/glm-my_cursor.postman_collection.json` | Postman 集合示例 |

## 本版主要改动（摘要）

### 配置与 LLM

- 支持 **`LLM_PROVIDER=custom`** 下使用 **`ANTHROPIC_BASE_URL`**（仅网关根地址），程序拼接为 `.../v1/chat/completions`。
- 已配置 **`ANTHROPIC_BASE_URL`** 时，密钥优先读 **`ANTHROPIC_API_KEY`**，再回退 `LLM_API_KEY` 等（避免与网关地址错位）。
- 若同时设置 **`LLM_API_URL`**，其优先级高于 **`ANTHROPIC_BASE_URL`**（完整 URL 显式指定）。
- 模型可通过 **`.env` 中 `ANTHROPIC_MODEL` / `LLM_MODEL`** 或 **`PUT /api/llm/config`**（页面「应用配置」）调整；名称须与网关可用模型 id 一致。
- 启动时从项目根（或向上查找）加载 **`.env`**（`godotenv`）；控制台会提示密钥是否加载（仅长度，不回显）。

### HTTP API（新增或强化）

- **`GET/PUT /api/llm/config`**：查看或热更新运行时 LLM 配置（**仅本机 loopback**）；请求体**不允许**携带 `api_key`（须用环境变量）。
- **`PUT`** 若 `api_url` 为网关根地址（不含 `chat/completions`），服务端会做与 `ANTHROPIC_BASE_URL` 相同的 URL 规范化。
- **`GET/PUT /api/workspace/file`**、**`GET /api/workspace/files`**：工作区文件读写与列表（防 `..` 穿越）。

### 认证与多租户（可选）

- **`AUTH_ENABLED`**：未开启时与旧版兼容（开发模式合成管理员，租户 **`default`**）；开启后需 **`JWT_SECRET`（≥32）** 与 **`AUTH_USERS`**（`用户名:密码:租户ID:角色`，多条英文 **`;`** 分隔）。
- **`POST /api/auth/login`**、刷新令牌、**`Authorization: Bearer`**；WebSocket 使用 **`/api/chat/ws?access_token=...`**（前端已拼接）。
- **RBAC**：`anon` / `viewer` / `user` / `admin`（详见 `README.md`「企业认证」）。
- **多租户**：工作区 **`workspace/<租户ID>/`**；RAG 集合按租户区分。
- **微信网站扫码**：`internal/auth/oauth`，环境变量见 `.env.example`；扩展其它 IdP 可实现同一 **`oauth.Provider`** 接口。

### 前端

- **`web/`**：Vite + Vue3 + Monaco；构建产物输出到 **`web/dist`**，由 Gin 托管 **`/`**。
- 右侧 AI：WebSocket 流式对话；模型/API URL 可在页面配置（密钥仍在服务端环境变量）。
- 可选 **账号密码登录**（`web/src/lib/auth.ts`）：令牌存 **`localStorage`**，请求自动带 Bearer。

### 仓库与忽略规则

- **`.gitignore`**：忽略 `.env`、`web/dist`、`*.exe`、`.idea/`、`http/http-client.private.env.json` 等。
- 已移除旧版 **`web/legacy-chat.html`**（独立 HTML 测试页），统一使用 Vue 构建入口。

### 测试

- **`internal/llm_test/config_test.go`**：custom + Anthropic 方案相关加载逻辑。
- **`internal/llm_test/live_glm_test.go`**：可选真实网络测试（需环境变量开关，默认不参与 CI）。

---

更新日期：以仓库当前分支为准；细节以代码与 `README.md` 为准。
