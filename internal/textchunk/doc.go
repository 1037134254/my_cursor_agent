// Package textchunk 提供 RAG 用的文本切片（chunk）工具。
//
// 最近更新（2026-05-09 12:00:00）：
// - 按 rune 切分，支持最大长度与重叠窗口，便于向量检索保留上下文。
// - 清理未使用辅助函数，减少 IDE 告警噪声。
package textchunk
