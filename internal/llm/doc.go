// Package llm 封装本地或远端模型调用。
//
// 最近更新（2026-05-09 12:00:00）：
// - 非流式 Chat 增强：状态码检查、JSON 解析兜底、可配置超时。
// - 新增流式 ChatStream：适配 Ollama/OpenAI 兼容 SSE（data: ...）分片协议。
// - 支持 [DONE] 终止、流分片解析与错误分片处理。
// - 增加可配置项：LLM_API_URL、LLM_MODEL、LLM_TIMEOUT_SECONDS。
package llm
