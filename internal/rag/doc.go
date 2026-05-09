// Package rag 演示 RAG 最小链路：文本切片 -> 嵌入（Ollama）-> Qdrant 向量库写入与检索。
//
// 最近更新（2026-05）：
//   - Ingest：切片 + EmbedOne + Upsert，payload 含 text、chunk_index、source。
//   - Search：查询句嵌入 + Query 近邻检索。
//   - 环境变量：QDRANT_HOST、QDRANT_GRPC_PORT、RAG_COLLECTION、RAG_VECTOR_SIZE；
//     嵌入侧见 internal/embed。
package rag
