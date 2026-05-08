package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAPIURL = "http://127.0.0.1:11434/v1/chat/completions"
	defaultModel  = "qwen2.5-coder:7b-instruct-q4_K_M"
)

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Error   any          `json:"error"`
}

func Init() {}

func Chat(prompt string) (string, error) {
	apiURL := envOrDefault("LLM_API_URL", defaultAPIURL)
	model := envOrDefault("LLM_MODEL", defaultModel)
	timeout := envIntOrDefault("LLM_TIMEOUT_SECONDS", 180)
	httpClient := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	reqBody := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request failed: %w", err)
	}

	resp, err := httpClient.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("call llm failed (url=%s): %w", apiURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read llm response failed: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("llm error: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	content, err := parseChatContent(body)
	if err != nil {
		return "", err
	}

	return content, nil
}

// ChatStream 以流式方式调用兼容 OpenAI Chat Completions 的接口（SSE：每行 data: ...）。
// onDelta 收到每个增量文本片段；ctx 取消时会中止读取。
func ChatStream(ctx context.Context, prompt string, onDelta func(string) error) error {
	apiURL := envOrDefault("LLM_API_URL", defaultAPIURL)
	model := envOrDefault("LLM_MODEL", defaultModel)

	reqBody := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"stream": true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("build request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 流式响应可能持续较久，不在 Client 层设总超时，由 ctx 控制。
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("call llm failed (url=%s): %w", apiURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("llm error: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return readChatStreamSSE(ctx, resp.Body, onDelta)
}

func readChatStreamSSE(ctx context.Context, body io.Reader, onDelta func(string) error) error {
	scanner := bufio.NewScanner(body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("read stream failed: %w", err)
			}
			return nil
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			return nil
		}

		delta, streamErr, err := parseStreamChunk(payload)
		if err != nil {
			return err
		}
		if streamErr != nil {
			return streamErr
		}
		if delta != "" {
			if err := onDelta(delta); err != nil {
				return err
			}
		}
	}
}

type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Error json.RawMessage `json:"error"`
}

func parseStreamChunk(payload string) (delta string, streamErr error, err error) {
	var chunk streamChunk
	if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
		return "", nil, fmt.Errorf("decode stream chunk failed: %w", err)
	}
	if len(chunk.Error) > 0 && string(chunk.Error) != "null" {
		return "", fmt.Errorf("llm stream error: %s", string(chunk.Error)), nil
	}
	if len(chunk.Choices) == 0 {
		return "", nil, nil
	}
	return chunk.Choices[0].Delta.Content, nil, nil
}

func envOrDefault(key, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	return value
}

func envIntOrDefault(key string, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return defaultValue
	}
	return parsed
}

func parseChatContent(body []byte) (string, error) {
	var result chatResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decode llm response failed: %w", err)
	}

	if len(result.Choices) == 0 {
		if result.Error != nil {
			return "", fmt.Errorf("llm returned error payload: %v", result.Error)
		}
		return "", errors.New("llm response has no choices")
	}

	content := strings.TrimSpace(result.Choices[0].Message.Content)
	if content == "" {
		return "", errors.New("llm message content is empty")
	}

	return content, nil
}
