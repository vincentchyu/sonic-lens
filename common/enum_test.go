package common

import "testing"

func TestParseAIModelPlatform(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected AIModelPlatform
	}{
		{name: "gemini", input: "gemini", expected: AIModelPlatformGemini},
		{name: "trim and lower", input: "  OLLAMA ", expected: AIModelPlatformOllama},
		{name: "omlx", input: "omlx", expected: AIModelPlatformOMLX},
		{name: "invalid", input: "unknown", expected: ""},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				if got := ParseAIModelPlatform(tt.input); got != tt.expected {
					t.Fatalf("ParseAIModelPlatform() = %q, want %q", got, tt.expected)
				}
			},
		)
	}
}
