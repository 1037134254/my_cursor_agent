package llm

import "testing"

func TestAnthropicBaseToChatURL(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"http://one-api.example.com", "http://one-api.example.com/v1/chat/completions"},
		{"http://one-api.example.com/v1", "http://one-api.example.com/v1/chat/completions"},
		{"http://one-api.example.com/v1/chat/completions", "http://one-api.example.com/v1/chat/completions"},
	}
	for _, tc := range cases {
		if got := AnthropicBaseToChatURL(tc.in); got != tc.want {
			t.Fatalf("AnthropicBaseToChatURL(%q) = %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestCleanSecret_BOMAndQuotes(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"\ufeffsk-abc", "sk-abc"},
		{`"sk-abc"`, "sk-abc"},
		{" 'sk-abc' ", "sk-abc"},
		{"\x60sk-abc\x60", "sk-abc"}, // `sk-abc`
	}
	for _, tc := range cases {
		if got := cleanSecret(tc.in); got != tc.want {
			t.Fatalf("cleanSecret(%q) = %q want %q", tc.in, got, tc.want)
		}
	}
}
