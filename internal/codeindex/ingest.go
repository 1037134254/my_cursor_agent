package codeindex

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"my_cursor/internal/codechunk"
	"my_cursor/internal/rag"
)

var skipDirNames = map[string]bool{
	".git":         true,
	"vendor":       true,
	"node_modules": true,
	".idea":        true,
	"qdrant_data":  true,
	"__pycache__":  true,
}

const ingestBatchSize = 24

// IndexCodeRoot 遍历 root 下源码文件，按行切片后嵌入写入 Qdrant。
// exts 如 {"go":true,"md":true}；为空时使用默认 go/mod/md/yaml/html。
func IndexCodeRoot(ctx context.Context, root string, maxLines, overlap int, exts map[string]bool) (files int, chunks int, err error) {
	root, err = filepath.Abs(root)
	if err != nil {
		return 0, 0, err
	}
	svc, err := rag.Get()
	if err != nil {
		return 0, 0, err
	}
	if len(exts) == 0 {
		exts = defaultExts()
	}
	maxBytes := int64(1024 * 1024)
	if s := strings.TrimSpace(os.Getenv("CODE_INDEX_MAX_FILE_MB")); s != "" {
		if mb, e := strconv.ParseFloat(s, 64); e == nil && mb > 0 {
			maxBytes = int64(mb * 1024 * 1024)
		}
	}

	var prepared []rag.PreparedChunk
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := d.Name()
			if skipDirNames[name] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
		if ext == "" || !exts[ext] {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		if fi.Size() > maxBytes {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if len(data) == 0 {
			return nil
		}
		text := string(data)
		if strings.Contains(text, "\x00") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		parts := codechunk.SplitLines(text, maxLines, overlap)
		for _, body := range parts {
			prepared = append(prepared, rag.PreparedChunk{
				Text:   codechunk.FormatChunk(rel, body),
				Source: rel,
				Kind:   "code",
			})
		}
		files++
		return nil
	})
	if err != nil {
		return 0, 0, err
	}

	total := 0
	for start := 0; start < len(prepared); start += ingestBatchSize {
		end := start + ingestBatchSize
		if end > len(prepared) {
			end = len(prepared)
		}
		n, err := svc.IngestPreparedChunks(ctx, prepared[start:end])
		if err != nil {
			return files, total, fmt.Errorf("ingest batch %d-%d: %w", start, end, err)
		}
		total += n
	}
	return files, total, nil
}

func defaultExts() map[string]bool {
	return map[string]bool{
		"go":   true,
		"mod":  true,
		"md":   true,
		"yml":  true,
		"yaml": true,
		"json": true,
		"html": true,
		"css":  true,
		"js":   true,
		"ts":   true,
		"tsx":  true,
		"vue":  true,
	}
}

// ParseExtList 解析环境变量 CODE_INDEX_EXTS（逗号分隔，如 go,md,mod）。
func ParseExtList(s string) map[string]bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	out := make(map[string]bool)
	for _, p := range strings.Split(s, ",") {
		e := strings.TrimSpace(strings.ToLower(p))
		e = strings.TrimPrefix(e, ".")
		if e != "" {
			out[e] = true
		}
	}
	return out
}
