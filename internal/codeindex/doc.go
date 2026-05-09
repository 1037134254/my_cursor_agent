// Package codeindex 将仓库源码按行切片并向量化写入 Qdrant，供对话前检索。
//
// 最近更新（2026-05-09 15:00:00）：
// - IndexCodeRoot：遍历目录、跳过常见噪声目录、按扩展名过滤。
// - 使用 internal/codechunk 切片与 rag.IngestPreparedChunks 批量入库。
// - 环境变量：CODE_INDEX_EXTS、CODE_INDEX_MAX_FILE_MB。
package codeindex
