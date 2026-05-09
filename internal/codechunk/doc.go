// Package codechunk 提供源代码按行切片与格式化，用于代码知识库 RAG。
//
// 最近更新（2026-05-09 15:00:00）：
// - SplitLines：控制最大行数与重叠窗口。
// - FormatChunk：为每个切片打上 file 路径前缀，便于模型关联路径。
package codechunk
