package tokenizer

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Provider represents an LLM provider (OpenAI, Anthropic, Gemini)
type Provider string

const (
	OpenAI    Provider = "openai"
	Anthropic Provider = "anthropic"
	Gemini    Provider = "gemini"
)

// ModelInfo contains metadata about a model's context window and pricing
type ModelInfo struct {
	Name          string   `yaml:"name"`
	Provider      Provider `yaml:"provider"`
	ContextWindow int      `yaml:"context_window"`
	InputPrice    float64  `yaml:"input_price_per_1m"`
	IsEstimated   bool     `yaml:"-"`
}

// Config represents the prices.yaml structure
type Config struct {
	Models []ModelInfo `yaml:"models"`
}

// Tokenizer defines the interface for counting tokens and messages
type Tokenizer interface {
	CountTokens(text string) (int, error)
	CountMessages(messages []Message) (int, error)
	CountTools(tools []Tool) (int, error)
	GetInfo() ModelInfo
	UpdateInfo(info ModelInfo)
}

// Message represents a single chat message
type Message struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content"` // Support for string or []ContentPart
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

// ContentPart represents a multimodal part (text or image)
type ContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL *Image `json:"image_url,omitempty"`
}

// Image represents the image_url schema
type Image struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // "low" or "high"
}

// ToolCall represents a call to a tool from the assistant
type ToolCall struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// Tool represents a tool definition
type Tool struct {
	Type     string           `json:"type"`
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition represents the schema for a tool's function
type FunctionDefinition struct {
	Name        string      `name:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

// Function represents a called function
type Function struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Registry manages model-specific tokenizers
type Registry struct {
	tokenizers map[string]Tokenizer
}

func NewRegistry() *Registry {
	return &Registry{
		tokenizers: make(map[string]Tokenizer),
	}
}

func (r *Registry) Register(name string, t Tokenizer) {
	r.tokenizers[name] = t
}

func (r *Registry) Get(name string) (Tokenizer, error) {
	if t, ok := r.tokenizers[name]; ok {
		return t, nil
	}
	return nil, fmt.Errorf("tokenizer for model %s not found", name)
}

// LoadConfig loads pricing and context data from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
