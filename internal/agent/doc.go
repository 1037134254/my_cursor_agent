// Package agent 为自研智能体核心逻辑。
//
// 最近更新（2026-05-09 12:00:00）：
// - 重构为可插拔工具注册模式（RegisterTool）。
// - 增加 Agent 核心循环：模型输出工具指令 -> 自动执行 -> 回注上下文 -> 继续推理。
// - 每次请求为单一任务（不按分号/换行拆子任务）。
// - 支持会话上下文（session_id）与多轮连续对话。
// - 兼容新工具协议 [TOOL:name]args，同时兼容旧 [READ]/[WRITE]。
// - 对话前可选向量检索（internal/rag）：将相关代码片段注入提示词（RAG_CHAT_ENABLED、RAG_TOP_K）。
package agent
