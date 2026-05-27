package tokenizer

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ChatPayload represents a generic wrapped message format
type ChatPayload struct {
	Messages []Message `json:"messages"`
}

// ParseChat attempts to detect and parse a conversation from JSON
func ParseChat(data []byte) ([]Message, error) {
	// Try parsing as a raw array first (OpenAI standard)
	var messages []Message
	if err := json.Unmarshal(data, &messages); err == nil && len(messages) > 0 {
		// VALIDATION: If the first message has no role and no content, it's likely not a chat
		if messages[0].Role == "" && (messages[0].Content == "" || messages[0].Content == nil) && len(messages[0].ToolCalls) == 0 {
			return nil, fmt.Errorf("not a chat schema")
		}
		return normalizeMessages(messages), nil
	}

	// Try parsing as a wrapped "messages" object
	var payload ChatPayload
	if err := json.Unmarshal(data, &payload); err == nil && len(payload.Messages) > 0 {
		return normalizeMessages(payload.Messages), nil
	}

	return nil, fmt.Errorf("could not detect valid JSON conversation format")
}

// normalizeMessages ensures roles are consistent (e.g. "human" -> "user")
func normalizeMessages(msgs []Message) []Message {
	for i, m := range msgs {
		role := strings.ToLower(m.Role)
		switch role {
		case "human", "speaker", "customer":
			msgs[i].Role = "user"
		case "bot", "ai", "model":
			msgs[i].Role = "assistant"
		default:
			msgs[i].Role = role
		}
	}
	return msgs
}

// IsJSON checks if the data is a valid JSON object or array
func IsJSON(data []byte) bool {
	var js json.RawMessage
	return json.Unmarshal(data, &js) == nil
}
