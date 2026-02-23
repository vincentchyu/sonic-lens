package ai

import (
	"testing"
)

func TestCleanLyrics(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Remove timestamps",
			input:    "[00:12.34]Line 1\n[01:02.03]Line 2",
			expected: "Line 1\nLine 2",
		},
		{
			name:     "Remove metadata tags",
			input:    "[ar:Artist]\n[ti:Title]\n[al:Album]\nLine 1\nLine 2",
			expected: "Line 1\nLine 2",
		},
		{
			name:     "Remove section labels",
			input:    "[Verse]\nLine 1\n[Chorus]\nLine 2",
			expected: "Line 1\nLine 2",
		},
		{
			name:     "Preserve empty lines",
			input:    "Line 1\n\nLine 2",
			expected: "Line 1\n\nLine 2",
		},
		{
			name:     "Mixed input",
			input:    "[ar:Radiohead]\n[00:27.000]Karma police arrest this man\n[Verse]\n[00:37.000]He buzzes like a fridge",
			expected: "Karma police arrest this man\nHe buzzes like a fridge",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got := CleanLyrics(tt.input)
				if got != tt.expected {
					t.Errorf("CleanLyrics() = %q, want %q", got, tt.expected)
				}
			},
		)
	}
}
