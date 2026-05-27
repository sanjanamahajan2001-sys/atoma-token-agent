package optimizer

import (
	"testing"
)

func TestTextOptimizer_Run(t *testing.T) {
	opt := NewTextOptimizer()
	
	tests := []struct {
		name     string
		input    string
		want     string
		minSugg  int
	}{
		{
			name:  "Conversational fillers",
			input: "I would like you to basically explain how quantum computers work. Actually, it is important to note that I am writing this to ask you to be very detailed. Please kindly provide a step-by-step guide. In my opinion, this should be simple enough for a child to understand at the end of the day.",
			want:  "Explain how quantum computers work. Note: be very detailed. Provide a step-by-step guide. This should be simple enough for a child to understand finally.",
			minSugg: 5,
		},
		{
			name:  "Case-insensitivity",
			input: "please KINDLY basically explain.",
			want:  "Explain.",
			minSugg: 3,
		},
		{
			name:  "Leading punctuation cleanup",
			input: ", please explain.",
			want:  "Explain.",
			minSugg: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, suggs := opt.Run(tt.input)
			if got != tt.want {
				t.Errorf("Run() got = %q, want %q", got, tt.want)
			}
			if len(suggs) < tt.minSugg {
				t.Errorf("Run() suggestions count = %d, want at least %d", len(suggs), tt.minSugg)
			}
		})
	}
}
