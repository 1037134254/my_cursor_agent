package tool

import (
	"os"
	"path/filepath"
)

const ROOT = "./workspace"

func ReadFile(path string) (string, error) {
	full := filepath.Join(ROOT, path)
	b, err := os.ReadFile(full)
	return string(b), err
}

func WriteFile(path, content string) error {
	full := filepath.Join(ROOT, path)
	_ = os.MkdirAll(filepath.Dir(full), 0755)
	return os.WriteFile(full, []byte(content), 0644)
}
