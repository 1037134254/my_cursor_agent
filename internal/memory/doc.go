// Package memory 管理对话与上下文记忆。
//
// 最近更新（2026-05）：
// - 新增内存会话存储（session_id -> 消息列表）。
// - 提供自动生成 session_id 能力。
// - 提供最近消息窗口读取能力，用于多轮上下文注入。
package memory
