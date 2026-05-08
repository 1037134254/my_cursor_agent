// Package agent 为自研智能体核心逻辑。
//
// 最近更新（2026-05）：
// - 重构为可插拔工具注册模式（RegisterTool）。
// - 增加 Agent 核心循环：模型输出工具指令 -> 自动执行 -> 回注上下文 -> 继续推理。
// - 支持任务自动拆解（按分号/换行拆分子任务）。
// - 支持会话上下文（session_id）与多轮连续对话。
// - 兼容新工具协议 [TOOL:name]args，同时兼容旧 [READ]/[WRITE]。
package agent
