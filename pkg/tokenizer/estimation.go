package tokenizer

import (
	"encoding/json"
	"fmt"
	"strings"
)

type EstimationTokenizer struct {
	Model string
	Info  ModelInfo
	Ratio float64 // Words to tokens ratio (e.g. 1.3)
}

func NewAnthropicTokenizer(model string) *EstimationTokenizer {
	info := ModelInfo{
		Name:          model,
		Provider:      Anthropic,
		IsEstimated:   true,
	}

	return &EstimationTokenizer{
		Model: model,
		Info:  info,
		Ratio: 1.35,
	}
}

func NewGeminiTokenizer(model string) *EstimationTokenizer {
	info := ModelInfo{
		Name:          model,
		Provider:      Gemini,
		IsEstimated:   true,
	}

	return &EstimationTokenizer{
		Model: model,
		Info:  info,
		Ratio: 1.2,
	}
}

func (t *EstimationTokenizer) CountTokens(text string) (int, error) {
	words := len(strings.Fields(text))
	if words == 0 {
		return 0, nil
	}
	return int(float64(words) * t.Ratio), nil
}

func (t *EstimationTokenizer) CountMessages(messages []Message) (int, error) {
	var totalTokens int
	var turnOverhead int

	switch t.Info.Provider {
	case Anthropic:
		turnOverhead = 5
	case Gemini:
		turnOverhead = 3
	default:
		turnOverhead = 4
	}

	for _, msg := range messages {
		contentT, _ := t.CountContent(msg.Content)
		totalTokens += contentT
		totalTokens += turnOverhead
		
		if len(msg.ToolCalls) > 0 {
			totalTokens += 15
			for _, call := range msg.ToolCalls {
				tc, _ := t.CountTokens(call.Function.Arguments)
				totalTokens += tc
			}
		}
	}
	
	totalTokens += 3 
	return totalTokens, nil
}

func (t *EstimationTokenizer) CountContent(content interface{}) (int, error) {
	switch v := content.(type) {
	case string:
		return t.CountTokens(v)
	case []interface{}:
		var count int
		for _, part := range v {
			if m, ok := part.(map[string]interface{}); ok {
				if m["type"] == "text" {
					count, _ = t.CountTokens(fmt.Sprint(m["text"]))
				} else if m["type"] == "image_url" {
					// Vision Heuristic:
					// Gemini: 258 per image
					// Anthropic: roughly 1600 tokens per full-res image
					if t.Info.Provider == Gemini {
						count += 258
					} else {
						count += 1600
					}
				}
			}
		}
		return count, nil
	}
	return t.CountTokens(fmt.Sprint(content))
}

func (t *EstimationTokenizer) CountTools(tools []Tool) (int, error) {
	if len(tools) == 0 {
		return 0, nil
	}
	var totalTokens int
	totalTokens += 10
	for _, tool := range tools {
		nameT, _ := t.CountTokens(tool.Function.Name)
		totalTokens += nameT
		descT, _ := t.CountTokens(tool.Function.Description)
		totalTokens += descT
		paramsJSON, _ := json.Marshal(tool.Function.Parameters)
		paramsT, _ := t.CountTokens(string(paramsJSON))
		totalTokens += paramsT
		totalTokens += 4
	}
	return totalTokens, nil
}

func (t *EstimationTokenizer) GetInfo() ModelInfo {
	return t.Info
}

func (t *EstimationTokenizer) UpdateInfo(info ModelInfo) {
	t.Info = info
	// Restore IsEstimated flag which might be lost in YAML load
	t.Info.IsEstimated = true
}
