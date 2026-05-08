package llm

import "testing"

func TestParseChatContentSuccess(t *testing.T) {
	body := []byte(`{"choices":[{"message":{"role":"assistant","content":"hello"}}]}`)
	got, err := parseChatContent(body)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "hello" {
		t.Fatalf("expected hello, got %q", got)
	}
}

func TestParseChatContentNoChoices(t *testing.T) {
	body := []byte(`{"choices":[]}`)
	_, err := parseChatContent(body)
	if err == nil {
		t.Fatal("expected error when choices are empty")
	}
}

func TestParseChatContentInvalidJSON(t *testing.T) {
	body := []byte(`{"choices":[}`)
	_, err := parseChatContent(body)
	if err == nil {
		t.Fatal("expected error on invalid json")
	}
}

func TestParseChatContentEmptyMessage(t *testing.T) {
	body := []byte(`{"choices":[{"message":{"role":"assistant","content":"   "}}]}`)
	_, err := parseChatContent(body)
	if err == nil {
		t.Fatal("expected error when message content is empty")
	}
}

func TestParseStreamChunkDelta(t *testing.T) {
	delta, streamErr, err := parseStreamChunk(`{"choices":[{"delta":{"content":"hi"}}]}`)
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

func TestParseStreamChunkErrorField(t *testing.T) {
	_, streamErr, err := parseStreamChunk(`{"error":{"message":"bad"}}`)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if streamErr == nil {
		t.Fatal("expected stream error")
	}
}
