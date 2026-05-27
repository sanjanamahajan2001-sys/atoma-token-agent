package tokenizer

import (
	"encoding/json"
	"fmt"
	"github.com/pkoukk/tiktoken-go"
)

type OpenAITokenizer struct {
	Model    string
	Tke      *tiktoken.Tiktoken
	Info     ModelInfo
}

func NewOpenAITokenizer(model string) (*OpenAITokenizer, error) {
	tke, err := tiktoken.EncodingForModel(model)
	if err != nil {
		tke, err = tiktoken.GetEncoding("cl100k_base")
		if err != nil {
			return nil, err
		}
	}

	info := ModelInfo{
		Name:          model,
		Provider:      OpenAI,
		IsEstimated:   false,
	}

	return &OpenAITokenizer{
		Model: model,
		Tke:   tke,
		Info:  info,
	}, nil
}

func (t *OpenAITokenizer) CountTokens(text string) (int, error) {
	token := t.Tke.Encode(text, nil, nil)
	return len(token), nil
}

func (t *OpenAITokenizer) CountMessages(messages []Message) (int, error) {
	var totalTokens int
	var tokensPerMessage int

	if t.Model == "gpt-3.5-turbo-0301" {
		tokensPerMessage = 4
	} else {
		tokensPerMessage = 3
	}

	for _, msg := range messages {
		totalTokens += tokensPerMessage
		
		contentT, _ := t.CountContent(msg.Content)
		totalTokens += contentT

		roleTokens, _ := t.CountTokens(msg.Role)
		totalTokens += roleTokens

		if len(msg.ToolCalls) > 0 {
			for _, call := range msg.ToolCalls {
				totalTokens += 10
				argTokens, _ := t.CountTokens(call.Function.Arguments)
				totalTokens += argTokens
				nameTokens, _ := t.CountTokens(call.Function.Name)
				totalTokens += nameTokens
			}
		}

		if msg.ToolCallID != "" {
			idTokens, _ := t.CountTokens(msg.ToolCallID)
			totalTokens += idTokens
		}
	}

	totalTokens += 3 // primer
	return totalTokens, nil
}

func (t *OpenAITokenizer) CountContent(content interface{}) (int, error) {
	switch v := content.(type) {
	case string:
		return t.CountTokens(v)
	case []interface{}: // JSON unmarshals to []interface{}
		var count int
		for _, part := range v {
			if m, ok := part.(map[string]interface{}); ok {
				if m["type"] == "text" {
					count += 4 // overhead for text part
					count, _ = t.CountTokens(fmt.Sprint(m["text"]))
				} else if m["type"] == "image_url" {
					// Vision: https://openai.com/api/pricing/
					// gpt-4o: low detail = 85 tokens, high detail = 170 + 680 per 512px tile
					detail := "high"
					if img, ok := m["image_url"].(map[string]interface{}); ok {
						if d, ok := img["detail"].(string); ok {
							detail = d
						}
					}
					if detail == "low" {
						count += 85
					} else {
						count += 170 + 680 // assuming 1 tile for default high detail
					}
				}
			}
		}
		return count, nil
	}
	return 0, nil
}

func (t *OpenAITokenizer) CountTools(tools []Tool) (int, error) {
	if len(tools) == 0 {
		return 0, nil
	}

	var totalTokens int
	totalTokens += 10

	for _, tool := range tools {
		totalTokens += 2
		nameT, _ := t.CountTokens(tool.Function.Name)
		totalTokens += nameT
		descT, _ := t.CountTokens(tool.Function.Description)
		totalTokens += descT
		paramsJSON, _ := json.Marshal(tool.Function.Parameters)
		paramsT, _ := t.CountTokens(string(paramsJSON))
		totalTokens += paramsT
	}

	return totalTokens, nil
}

func (t *OpenAITokenizer) GetInfo() ModelInfo {
	return t.Info
}

func (t *OpenAITokenizer) UpdateInfo(info ModelInfo) {
	t.Info = info
}
