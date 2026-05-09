// Package embed 封装文本向量化（嵌入），当前对接 Ollama /api/embeddings。
//
// 最近更新（2026-05）：
// - EmbedOne：单段文本 -> float32 向量。
// - 环境变量：EMBED_API_URL、EMBED_MODEL、EMBED_TIMEOUT_SECONDS。
package embed
