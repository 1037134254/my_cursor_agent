package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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
