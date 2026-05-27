package tokenizer

import (
	"testing"
)

func TestParseChat_Detection(t *testing.T) {
	// 1. Valid Chat
	chatData := `[{"role": "user", "content": "Hello"}]`
	msgs, err := ParseChat([]byte(chatData))
	if err != nil {
		t.Errorf("Valid chat failed: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Role != "user" {
		t.Errorf("Message normalization failed: %+v", msgs)
	}

	// 2. Metadata (Should fail)
	metaData := `[{"id": "usr_01", "name": "Sanjana"}]`
	_, err = ParseChat([]byte(metaData))
	if err == nil {
		t.Error("Non-chat metadata should have triggered fallback error")
	}

	// 3. Normalization
	altData := `[{"role": "human", "content": "Howdy"}]`
	msgsAlt, _ := ParseChat([]byte(altData))
	if msgsAlt[0].Role != "user" {
		t.Errorf("Role normalization (human -> user) failed: %s", msgsAlt[0].Role)
	}
}

func TestIsJSON(t *testing.T) {
	if !IsJSON([]byte(`{"test":1}`)) { t.Error("Object failed") }
	if !IsJSON([]byte(`[1,2,3]`)) { t.Error("Array failed") }
	if IsJSON([]byte(`not json`)) { t.Error("Non-json passed") }
}

func TestParseChat_Wrapped(t *testing.T) {
	wrappedData := `{"messages": [{"role": "user", "content": "Wrapped message"}]}`
	msgs, err := ParseChat([]byte(wrappedData))
	if err != nil {
		t.Errorf("Wrapped chat failed: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Content != "Wrapped message" {
		t.Error("Wrapped content extraction failed")
	}
}
