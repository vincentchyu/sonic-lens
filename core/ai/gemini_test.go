package ai

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vincentchyu/sonic-lens/config"
)

func TestGeminiProvider_AnalyzeTrack(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("跳过测试：未配置 GEMINI_API_KEY 环境变量")
	}

	cfg := config.GeminiConfig{
		APIKey: apiKey,
		Model:  "gemini-2.0-flash", // 使用最新稳定的 Flash 模型
	}

	provider, err := newGeminiProvider(cfg)
	assert.NoError(t, err)

	req := TrackAnalysisRequest{
		Title:  "Hotel California",
		Artist: "Eagles",
		Lyrics: "On a dark desert highway, cool wind in my hair...",
	}

	ctx := context.Background()
	result, err := provider.AnalyzeTrack(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.LyricsTranslation)
	assert.NotEmpty(t, result.AnalysisSummary)

	t.Logf("分析结果: %+v", result)
}

func TestGeminiProvider_AnalyzeTrackStream(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("跳过测试：未配置 GEMINI_API_KEY 环境变量")
	}

	cfg := config.GeminiConfig{
		APIKey: apiKey,
		Model:  "gemini-2.0-flash",
	}

	provider, err := newGeminiProvider(cfg)
	assert.NoError(t, err)

	req := TrackAnalysisRequest{
		Title:  "Hotel California",
		Artist: "Eagles",
		Lyrics: "On a dark desert highway, cool wind in my hair...",
	}

	ctx := context.Background()
	ch, err := provider.AnalyzeTrackStream(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, ch)

	var fullContent string
	for chunk := range ch {
		fullContent += chunk
	}

	assert.NotEmpty(t, fullContent)
	t.Logf("流式输出内容: %s", fullContent)
}

func TestTrimCodeFence(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"```json\n{\"a\": 1}\n```", "{\"a\": 1}"},
		{"```\n{\"a\": 1}\n```", "{\"a\": 1}"},
		{"{\"a\": 1}", "{\"a\": 1}"},
		{"  ```json\n{\"a\": 1}\n```  ", "{\"a\": 1}"},
	}

	for _, c := range cases {
		assert.Equal(t, c.expected, TrimCodeFence(c.input))
	}
}
