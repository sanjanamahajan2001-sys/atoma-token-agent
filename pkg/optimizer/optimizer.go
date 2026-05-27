package optimizer

// Suggestion represents a single optimization found in the text
type Suggestion struct {
	Original    string
	Replacement string
	Reason      string
}

// Result contains the full findings of an optimization pass
type Result struct {
	OriginalText   string
	OptimizedText  string
	Suggestions    []Suggestion
	TokensOriginal int
	TokensSaved    int
	PercentSaved   float64
}

// Optimizer is the interface for prompt simplification
type Optimizer interface {
	Optimize(text string) (Result, error)
}
