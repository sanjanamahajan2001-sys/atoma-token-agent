package optimizer

import (
	"atoma/pkg/tokenizer"
	"encoding/json"
	"regexp"
	"strings"
	"unicode"
)

type OptimizationRule struct {
	Phrase      string
	Replacement string
	Reason      string
}

// DefaultRules contains conversational fillers and redundant prompt instructions
var DefaultRules = []OptimizationRule{
	{Phrase: "I would like you to", Replacement: "", Reason: "Indirect instruction"},
	{Phrase: "Please", Replacement: "", Reason: "Conversational filler"},
	{Phrase: "Kindly", Replacement: "", Reason: "Conversational filler"},
	{Phrase: "basically", Replacement: "", Reason: "Stop word"},
	{Phrase: "actually", Replacement: "", Reason: "Stop word"},
	{Phrase: "In my opinion", Replacement: "", Reason: "Redundant"},
	{Phrase: "It is important to note that", Replacement: "Note:", Reason: "Wordy"},
	{Phrase: "I am writing this to ask you to", Replacement: "", Reason: "Introductory filler"},
	{Phrase: "Could you please", Replacement: "", Reason: "Polite filler"},
	{Phrase: "At the end of the day", Replacement: "finally", Reason: "Cliché/Wordy"},
	{Phrase: "Due to the fact that", Replacement: "because", Reason: "Wordy"},
	{Phrase: "In order to", Replacement: "to", Reason: "Wordy"},
}

// TextOptimizer handles role-aware and structured optimization
type TextOptimizer struct {
	Rules []OptimizationRule
}

func NewTextOptimizer() *TextOptimizer {
	return &TextOptimizer{Rules: DefaultRules}
}

func (o *TextOptimizer) Run(text string) (string, []Suggestion) {
	// 1. JSON Detection
	if tokenizer.IsJSON([]byte(text)) {
		msgs, err := tokenizer.ParseChat([]byte(text))
		if err == nil {
			return o.OptimizeChat(msgs)
		}
	}

	// 2. JSONL Detection
	if strings.HasPrefix(text, "{") && strings.Contains(text, "\n") {
		return o.OptimizeJSONL(text)
	}

	// 3. Fallback to Standard Text
	return o.cleanText(text)
}

func (o *TextOptimizer) OptimizeChat(messages []tokenizer.Message) (string, []Suggestion) {
	allSuggestions := []Suggestion{}
	for i, msg := range messages {
		contentStr, ok := msg.Content.(string)
		if ok {
			clean, suggestions := o.cleanText(contentStr)
			messages[i].Content = clean
			allSuggestions = append(allSuggestions, suggestions...)
		}
	}

	// Minify output for maximum token savings
	data, _ := json.Marshal(messages)
	return string(data), allSuggestions
}

func (o *TextOptimizer) OptimizeJSONL(text string) (string, []Suggestion) {
	allSuggestions := []Suggestion{}
	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		var messages []tokenizer.Message
		if err := json.Unmarshal([]byte(line), &messages); err == nil && len(messages) > 0 {
			_, suggestions := o.OptimizeChat(messages)
			allSuggestions = append(allSuggestions, suggestions...)
			data, _ := json.Marshal(messages)
			result = append(result, string(data))
		} else {
			// Wrapped format
			var wrapped struct {
				Messages []tokenizer.Message `json:"messages"`
			}
			if err := json.Unmarshal([]byte(line), &wrapped); err == nil && len(wrapped.Messages) > 0 {
				_, suggestions := o.OptimizeChat(wrapped.Messages)
				allSuggestions = append(allSuggestions, suggestions...)
				data, _ := json.Marshal(wrapped)
				result = append(result, string(data))
			}
		}
	}
	return strings.Join(result, "\n"), allSuggestions
}

func (o *TextOptimizer) cleanText(text string) (string, []Suggestion) {
	optimized := text
	suggestions := []Suggestion{}

	for _, rule := range o.Rules {
		re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(rule.Phrase))
		if re.MatchString(optimized) {
			optimized = re.ReplaceAllString(optimized, rule.Replacement)
			suggestions = append(suggestions, Suggestion{
				Original:    rule.Phrase,
				Replacement: rule.Replacement,
				Reason:      rule.Reason,
			})
		}
	}

	optimized = o.cleanup(optimized)
	return optimized, suggestions
}

func (o *TextOptimizer) cleanup(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	rePunc := regexp.MustCompile(`[.,;]\s*[.,;]`)
	text = rePunc.ReplaceAllStringFunc(text, func(match string) string {
		return string(match[0])
	})
	text = strings.TrimLeft(text, " ,.;")
	reSpacePunc := regexp.MustCompile(`\s+([.,!?;])`)
	text = reSpacePunc.ReplaceAllString(text, "$1")

	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	capitalizeNext := true
	for i, word := range words {
		if capitalizeNext && len(word) > 0 {
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
			capitalizeNext = false
		}
		if strings.HasSuffix(word, ".") || strings.HasSuffix(word, "!") || strings.HasSuffix(word, "?") {
			capitalizeNext = true
		}
	}
	return strings.Join(words, " ")
}
