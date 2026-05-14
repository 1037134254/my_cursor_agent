# 变更记录

本文档概述与本仓库相关的**文档与行为变更**，便于评审与发布说明。

## 文档入口一览

| 文件 | 说明 |
|------|------|
| `README.md` | 项目说明、快速开始、401 排查、`ANTHROPIC_*`、**企业认证（JWT / `AUTH_USERS` / 微信扫码）**、目录结构 |
| `docs/llm-concepts.md` | LLM、Tokenizer、上下文与窗口、RAG、Prompt、Tool、MCP、Agent、元数据/指令层等术语梳理 |
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

### 多租户模型注册表（企业级）

- 新增 **`model_profile`** 与 **`tenant_model_binding`** 两张 MySQL 表（启动时 `CREATE IF NOT EXISTS`）；推理时按 `(tenant_id, purpose)` 路由到对应 profile，租户 A/B 可分别走 GLM / DeepSeek / Ollama / 自建 vLLM。
- **零信任密钥**：`model_profile.api_key_ref` 仅存 env 变量名（如 `GLM_KEY_A`），DB 永远没有明文 key；旋转 key 改 env 即可。
- **租户级 RPM 覆盖**：`tenant_model_binding.rpm_override` 让每个租户在同一 profile 上拥有独立限流额度（0 表示沿用 profile 默认）。
- **新 API**（均归在 `PermLLMAdmin` / `PermLLMRead`）：
  - `GET/POST/PUT/DELETE /api/llm/profiles[/:id]`、`POST /api/llm/profiles/:id/probe`（一次性 ping 真实下游，30s 短超时）
  - `GET/PUT /api/tenants/:tid/models`（admin 改任意租户，user 仅看本租户）
  - `GET /api/llm/current?purpose=chat`（当前租户该用途的默认 profile，脱敏）
- **运行期降级**：MySQL 不可达或 `LLM_REGISTRY=env` 时降级到只读 env 模式，仅 default 租户可用，便于本地开发零依赖。
- **兼容期**：旧 `PUT /api/llm/config` 在 env 模式下保留语义，MySQL 模式下返回 `410 Gone` 并提示走新 API；下个版本删除。
- 推理入口签名变更：`llm.Chat(ctx, prompt)` / `llm.ChatStream(ctx, prompt, onDelta)`，从 `tenant.FromContext(ctx)` 取租户走 registry；`agent.Chat / ChatStream` 已同步。

### 认证与多租户（可选）

- **`AUTH_ENABLED`**：未开启时与旧版兼容（开发模式合成管理员，租户 **`default`**）；开启后需 **`JWT_SECRET`（≥32）** 与 **`AUTH_USERS`**（`用户名:密码:租户ID:角色`，多条英文 **`;`** 分隔）。
- **`POST /api/auth/login`**、刷新令牌、**`Authorization: Bearer`**；WebSocket 使用 **`/api/chat/ws?access_token=...`**（前端已拼接）。
- **刷新令牌持久化**：默认探测 MySQL（`AUTH_REFRESH_MYSQL_DSN`，库不存在自动建），不可达则降级内存。Token 仅以 **SHA-256 哈希** 入库，一次性消费 + 自动轮换。
- **RBAC**：`anon` / `viewer` / `user` / `admin`（详见 `README.md`「企业认证」）。
- **多租户**：工作区 **`workspace/<租户ID>/`**；RAG 集合按租户区分。
- **第三方登录**：`internal/auth/oauth` 统一 `Provider` 抽象，已内置：
  - **GitHub OAuth**（个人 5 分钟可申请，开发期使用）
  - **微信开放平台 · 网站应用扫码**（需企业主体 + 备案域名，代码已就绪，等 `WECHAT_*` 配齐自动启用）
- **前端登录页**：根据 `GET /api/auth/oauth/providers` 自动渲染对应按钮（未配置则不显示），登录态用 `isLoggedIn`/`currentUser()` 驱动。

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

## 路线图（TODO）

按企业化优先级排序，每条都不影响现有功能，可独立推进：

- [ ] **结构化审计落盘**：把 `Audit()` 改为 JSON 行追加到 `data/audit/YYYY-MM-DD.jsonl`，并可选输出到 stdout（便于对接 ELK / Loki / SIEM）。
- [ ] **LLM 调用计量**：新增 `llm_call_log` 表（tenant/user/profile/tokens/latency），异步批量写；管理后台出成本曲线。
- [ ] **function calling 升级**：把 `[TOOL:read_file]xxx` 字符串协议升级到 OpenAI function calling JSON，提升 Agent 工具调用稳定性。
- [ ] **Embedding profile 化**：RAG 当前硬编码 Ollama embedding；接入 `Purpose=embed` 的注册表后可按租户切 BGE/Jina/OpenAI embedding。
- [ ] **通用 OIDC SSO**：基于 `github.com/coreos/go-oidc/v3` 增加 `oauth.OIDCProvider`，支持 Keycloak / Authentik / Azure AD / Okta，环境变量描述 `OIDC_ISSUER / CLIENT_ID / CLIENT_SECRET / REDIRECT_URI`，复用现有 binding 与 RBAC 模型。
- [ ] **微信网站应用接入**：等申请到企业主体 + 备案域名后，填 `WECHAT_OPEN_APP_ID / SECRET / REDIRECT_URI` 即可启用；当前 `internal/auth/oauth/wechat_web.go` 已实现完整 qrconnect 流程。
- [ ] **更多第三方**：复用 `oauth.Provider` 接口可低成本加 Gitee / 钉钉 / 企业微信 / 飞书；统一回调地址使用 `OAUTH_AFTER_LOGIN_REDIRECT`。
- [ ] **组织 → 项目 二级隔离**：在 `Principal` 与 RAG payload 中加 `project_id`，工作区路径升级为 `workspace/<tenant>/<project>/`。

---

更新日期：以仓库当前分支为准；细节以代码与 `README.md` 为准。
