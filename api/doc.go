// Package api 为 HTTP/RPC 等对外接口层。
//
// 最近更新（2026-05）：
// - 新增 WebSocket 流式接口：GET /api/chat/ws。
// - Chat 接口请求增加 session_id，响应返回 session_id。
// - Stream 接口支持 session_id 透传，并返回 start/meta/delta/done/error 事件。
// - 增加流式超时控制（LLM_STREAM_TIMEOUT_SECONDS）与部分输出超时优雅结束。
// - RAG：POST /api/rag/ingest、POST /api/rag/search（Qdrant + Ollama 嵌入）。
package api
