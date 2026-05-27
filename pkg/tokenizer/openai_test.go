package tokenizer

import (
	"testing"
)

func TestOpenAITokenizer_CountTokens(t *testing.T) {
	tk, err := NewOpenAITokenizer("gpt-4o")
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}

	tests := []struct {
		text     string
		expected int
	}{
		{"Hello, world!", 4},
		{"Quantum computing is awesome.", 5},
		{"", 0},
	}

	for _, tt := range tests {
		got, err := tk.CountTokens(tt.text)
		if err != nil {
			t.Errorf("CountTokens(%q) error = %v", tt.text, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("CountTokens(%q) = %v; want %v", tt.text, got, tt.expected)
		}
	}
}

func TestOpenAITokenizer_CountMessages(t *testing.T) {
	tk, _ := NewOpenAITokenizer("gpt-4o")

	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Hello!"},
	}

	// OpenAI gpt-4o: 
	// System: 3 (overhead) + 6 (content) + 1 (role) = 10
	// User:   3 (overhead) + 2 (content) + 1 (role) = 6
	// Primer: 3
	// Total:  10 + 6 + 3 = 19
	expected := 19
	got, _ := tk.CountMessages(messages)

	if got != expected {
		t.Errorf("CountMessages() = %v; want %v", got, expected)
	}
}
