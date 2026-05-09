package embed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Ollama 原生接口：POST /api/embeddings，body: {"model","prompt"}。
const defaultEmbedURL = "http://127.0.0.1:11434/api/embeddings"

func envOrDefault(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

// EmbedOne 对单段文本生成向量（float32），用于写入 Qdrant。
func EmbedOne(text string) ([]float32, error) {
	url := envOrDefault("EMBED_API_URL", defaultEmbedURL)
	model := envOrDefault("EMBED_MODEL", "nomic-embed-text")
	timeout := 60 * time.Second
	if s := strings.TrimSpace(os.Getenv("EMBED_TIMEOUT_SECONDS")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			timeout = time.Duration(n) * time.Second
		}
	}

	body, err := json.Marshal(map[string]string{
		"model":  model,
		"prompt": text,
	})
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("embed request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("embed error: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var out struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("embed decode failed: %w", err)
	}
	if len(out.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding")
	}
	vec := make([]float32, len(out.Embedding))
	for i, v := range out.Embedding {
		vec[i] = float32(v)
	}
	return vec, nil
}
