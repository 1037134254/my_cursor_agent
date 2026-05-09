package llm

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// readChatStreamSSE 解析 OpenAI 兼容 SSE（每行 data: ...）。
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
