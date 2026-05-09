package llm_test

// 对应 internal/llm/parse.go、stream.go：聊天 JSON 与流式 chunk 解析。

import (
	"testing"

	llm "my_cursor/internal/llm"
)

// TestParseChatContentSuccess 验证标准 choices[0].message.content 解析。
func TestParseChatContentSuccess(t *testing.T) {
	body := []byte(`{"choices":[{"message":{"role":"assistant","content":"hello"}}]}`)
	got, err := llm.ParseChatContentForTest(body)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "hello" {
		t.Fatalf("expected hello, got %q", got)
	}
}

// TestParseChatContentNoChoices 验证 choices 为空时返回错误。
func TestParseChatContentNoChoices(t *testing.T) {
	body := []byte(`{"choices":[]}`)
	_, err := llm.ParseChatContentForTest(body)
	if err == nil {
		t.Fatal("expected error when choices are empty")
	}
}

// TestParseChatContentInvalidJSON 验证非法 JSON 返回错误。
func TestParseChatContentInvalidJSON(t *testing.T) {
	body := []byte(`{"choices":[}`)
	_, err := llm.ParseChatContentForTest(body)
	if err == nil {
		t.Fatal("expected error on invalid json")
	}
}

// TestParseChatContentEmptyMessage 验证仅空白 content 视为无效。
func TestParseChatContentEmptyMessage(t *testing.T) {
	body := []byte(`{"choices":[{"message":{"role":"assistant","content":"   "}}]}`)
	_, err := llm.ParseChatContentForTest(body)
	if err == nil {
		t.Fatal("expected error when message content is empty")
	}
}

// TestParseStreamChunkDelta 验证流式 delta.content 片段解析。
func TestParseStreamChunkDelta(t *testing.T) {
	delta, streamErr, err := llm.ParseStreamChunkForTest(`{"choices":[{"delta":{"content":"hi"}}]}`)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if streamErr != nil {
		t.Fatalf("unexpected stream err: %v", streamErr)
	}
	if delta != "hi" {
		t.Fatalf("expected hi, got %q", delta)
	}
}

// TestParseStreamChunkErrorField 验证响应体中带 error 字段时解析为流错误。
func TestParseStreamChunkErrorField(t *testing.T) {
	_, streamErr, err := llm.ParseStreamChunkForTest(`{"error":{"message":"bad"}}`)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if streamErr == nil {
		t.Fatal("expected stream error")
	}
}
