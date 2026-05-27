package tokenizer

import (
	"testing"
)

func TestEstimationTokenizer_CountTokens(t *testing.T) {
	tk := NewAnthropicTokenizer("claude-3-5-sonnet")
	
	// Claude Ratio: 1.35
	// "Hello world" = 2 words * 1.35 = 2.7 = 2
	got, _ := tk.CountTokens("Hello world")
	if got < 2 || got > 3 {
		t.Errorf("Claude CountTokens() = %v; want 2 or 3", got)
	}
}

func TestEstimationTokenizer_CountMessages(t *testing.T) {
	tk := NewAnthropicTokenizer("claude-3-5")
	messages := []Message{
		{Role: "user", Content: "Hello world"},
	}

	// 2 (content) + 5 (delta) + 3 (primer) = 10
	got, _ := tk.CountMessages(messages)
	if got < 9 || got > 11 {
		t.Errorf("Claude CountMessages() = %v; want ~10", got)
	}
}

func TestEstimationTokenizer_Gemini(t *testing.T) {
	tk := NewGeminiTokenizer("gemini-1.5-pro")
	messages := []Message{
		{Role: "user", Content: "Explain AI"},
	}

	// 2 words * 1.2 = 2.4 = 2 (content)
	// 2 (content) + 3 (delta) + 3 (primer) = 8
	got, _ := tk.CountMessages(messages)
	if got < 7 || got > 9 {
		t.Errorf("Gemini CountMessages() = %v; want ~8", got)
	}
}
