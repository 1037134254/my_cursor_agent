package textchunk

import (
	"strings"
)

// Split 按 rune 切片长文本，支持重叠（overlap < maxRunes）。
func Split(text string, maxRunes, overlap int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if maxRunes <= 0 {
		maxRunes = 500
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= maxRunes {
		overlap = maxRunes / 4
	}

	runes := []rune(text)
	if len(runes) <= maxRunes {
		return []string{text}
	}

	var out []string
	step := maxRunes - overlap
	if step <= 0 {
		step = maxRunes
	}
	for start := 0; start < len(runes); start += step {
		end := start + maxRunes
		if end > len(runes) {
			end = len(runes)
		}
		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			out = append(out, chunk)
		}
		if end >= len(runes) {
			break
		}
	}
	return out
}
