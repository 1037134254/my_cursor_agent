package codechunk

import (
	"strings"
)

// SplitLines 按行切片源代码，适合保留函数/结构大致边界感。
// maxLines/overlap 为行数；overlap 表示相邻块重叠行数。
func SplitLines(content string, maxLines, overlap int) []string {
	content = strings.TrimRight(content, "\n\t ")
	if content == "" {
		return nil
	}
	if maxLines <= 0 {
		maxLines = 120
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= maxLines {
		overlap = maxLines / 4
	}

	lines := strings.Split(content, "\n")
	if len(lines) <= maxLines {
		return []string{strings.Join(lines, "\n")}
	}

	var out []string
	step := maxLines - overlap
	if step <= 0 {
		step = maxLines
	}
	for start := 0; start < len(lines); start += step {
		end := start + maxLines
		if end > len(lines) {
			end = len(lines)
		}
		chunk := strings.TrimSpace(strings.Join(lines[start:end], "\n"))
		if chunk != "" {
			out = append(out, chunk)
		}
		if end >= len(lines) {
			break
		}
	}
	return out
}

// FormatChunk 为入库文本加上文件路径头，便于检索结果可读、全局理解。
func FormatChunk(relPath, body string) string {
	return "// file: " + relPath + "\n\n" + body
}
