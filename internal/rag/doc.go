// Package rag 演示 RAG 最小链路：文本切片 -> 嵌入（Ollama）-> Qdrant 向量库写入与检索。
//
// 最近更新（2026-05-09 12:00:00）：
//   - Ingest：切片 + EmbedOne + Upsert，payload 含 text、chunk_index、source。
//   - Search：查询句嵌入 + Query 近邻检索。
//   - 环境变量：QDRANT_HOST、QDRANT_GRPC_PORT、RAG_COLLECTION、RAG_VECTOR_SIZE；
//     嵌入侧见 internal/embed。
//
// - IngestPreparedChunks：代码块等预切片文本批量入库，payload 含 source/kind。
// - Search 返回 Hit.Source / Hit.Kind 供 Agent 引用路径。
package rag
