package common

import "testing"

func TestParseAIModelPlatform(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected AIModelPlatform
	}{
		{name: "openai", input: "openai", expected: AIModelPlatformOpenAI},
		{name: "trim and lower", input: "  OLLAMA ", expected: AIModelPlatformOllama},
		{name: "custom", input: "custom", expected: AIModelPlatformCustom},
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
