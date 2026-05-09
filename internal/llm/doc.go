// Package llm 封装本地或远端模型调用（AI 网关形态）。
//
// 最近更新（2026-05-09 12:00:00）：
// - Gateway：超时、重试（429/5xx/网络错误）、RPM 限流（LLM_RPM，0 表示不限流）。
// - 一键切换预设：LLM_PROVIDER=qwen|ollama|deepseek|custom（见 LoadConfig）。
// - DeepSeek：DEEPSEEK_API_KEY、DEEPSEEK_API_URL、DEEPSEEK_MODEL。
// - 集群：LLM_CLUSTER_ENDPOINTS 逗号分隔多个 chat/completions 完整 URL，轮询调度（EndpointScheduler）。
// - EndpointScheduler（scheduler.go）预留多副本路由扩展。
// - 环境变量：LLM_API_URL、LLM_MODEL、LLM_API_KEY、LLM_TIMEOUT_SECONDS、LLM_MAX_RETRIES、LLM_RETRY_BACKOFF_MS。
package llm
