package ai

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestCustomProviderBuildRequestUsesSystemAndUserMessages(t *testing.T) {
	provider := &CustomProvider{
		apiKey:  "test-key",
		baseURL: "http://localhost:8000",
		model:   "test-model",
	}

	req := TrackAnalysisRequest{
		Title:           "山雀",
		Artist:          "万能青年旅店",
		Album:           "冀西南林路行",
		Lyrics:          "自然赠予你",
		LangSource:      "auto",
		LangTarget:      "zh-CN",
		FeedbackContext: "用户反馈：请避免重复",
	}

	httpReq, body, err := provider.buildRequest(context.Background(), req, false)
	if err != nil {
		t.Fatalf("buildRequest() error = %v", err)
	}

	if got := httpReq.URL.String(); got != "http://localhost:8000/v1/chat/completions" {
		t.Fatalf("request url = %q, want %q", got, "http://localhost:8000/v1/chat/completions")
	}

	var payload customChatRequest
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("unmarshal request body failed: %v", err)
	}

	if payload.Model != "test-model" {
		t.Fatalf("payload.Model = %q, want %q", payload.Model, "test-model")
	}
	if len(payload.Messages) != 2 {
		t.Fatalf("payload.Messages len = %d, want 2", len(payload.Messages))
	}
	if payload.Messages[0].Role != "system" {
		t.Fatalf("system message role = %q, want %q", payload.Messages[0].Role, "system")
	}
	if got := payload.Messages[0].Content; !strings.Contains(got, "【JSON Schema】") || !strings.Contains(got, "analysis_summary") || !strings.Contains(got, "analysis_by_section") {
		t.Fatalf("system message content missing required prompt sections: %q", got)
	}
	if payload.Messages[1].Role != "user" {
		t.Fatalf("user message role = %q, want %q", payload.Messages[1].Role, "user")
	}
	if got := payload.Messages[1].Content; !strings.Contains(got, "输入数据（JSON）") || !strings.Contains(got, "\"title\":\"山雀\"") || !strings.Contains(got, "\"artist\":\"万能青年旅店\"") || !strings.Contains(got, "用户反馈：请避免重复") {
		t.Fatalf("user message content missing required fields: %q", got)
	}
}

func TestCustomProviderBuildAlbumRequestUsesSystemAndUserMessages(t *testing.T) {
	provider := &CustomProvider{
		apiKey:  "test-key",
		baseURL: "http://localhost:8000",
		model:   "test-model",
	}

	req := AlbumAnalysisRequest{
		AlbumID:         12,
		Artist:          "万能青年旅店",
		Album:           "冀西南林路行",
		ReleaseDate:     "2020-11-11",
		Genre:           "摇滚",
		TrackCount:      9,
		AnalyzedTracks:  7,
		FeedbackContext: "用户反馈：请补全专辑背景",
	}

	httpReq, body, err := provider.buildAlbumRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("buildAlbumRequest() error = %v", err)
	}

	if got := httpReq.URL.String(); got != "http://localhost:8000/v1/chat/completions" {
		t.Fatalf("request url = %q, want %q", got, "http://localhost:8000/v1/chat/completions")
	}

	var payload customChatRequest
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("unmarshal request body failed: %v", err)
	}

	if payload.Model != "test-model" {
		t.Fatalf("payload.Model = %q, want %q", payload.Model, "test-model")
	}
	if len(payload.Messages) != 2 {
		t.Fatalf("payload.Messages len = %d, want 2", len(payload.Messages))
	}
	if payload.Messages[0].Role != "system" {
		t.Fatalf("system message role = %q, want %q", payload.Messages[0].Role, "system")
	}
	if got := payload.Messages[0].Content; !strings.Contains(got, "【JSON Schema】") || !strings.Contains(got, "analysis_summary") || !strings.Contains(got, "analysis_by_section") {
		t.Fatalf("system message content missing required prompt sections: %q", got)
	}
	if payload.Messages[1].Role != "user" {
		t.Fatalf("user message role = %q, want %q", payload.Messages[1].Role, "user")
	}
	if got := payload.Messages[1].Content; !strings.Contains(got, "输入数据（JSON）") || !strings.Contains(got, "\"album_id\":12") || !strings.Contains(got, "\"artist\":\"万能青年旅店\"") || !strings.Contains(got, "用户反馈：请补全专辑背景") {
		t.Fatalf("user message content missing required fields: %q", got)
	}
}

func TestCustomProviderBuildCompletionsRequestUsesCompletionsEndpoint(t *testing.T) {
	provider := &CustomProvider{
		apiKey:  "test-key",
		baseURL: "http://localhost:8000",
		model:   "test-model",
	}

	req := TrackAnalysisRequest{
		Title:      "山雀",
		Artist:     "万能青年旅店",
		Album:      "冀西南林路行",
		Lyrics:     "自然赠予你",
		LangSource: "auto",
		LangTarget: "zh-CN",
	}

	httpReq, body, err := provider.buildCompletionsRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("buildCompletionsRequest() error = %v", err)
	}

	if got := httpReq.URL.String(); got != "http://localhost:8000/v1/completions" {
		t.Fatalf("request url = %q, want %q", got, "http://localhost:8000/v1/completions")
	}

	var payload customCompletionsRequest
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("unmarshal request body failed: %v", err)
	}

	if payload.Model != "test-model" {
		t.Fatalf("payload.Model = %q, want %q", payload.Model, "test-model")
	}
	if got := payload.Prompt; !strings.Contains(got, "系统提示：") || !strings.Contains(got, "输入数据（JSON）") || !strings.Contains(got, "\"title\":\"山雀\"") || !strings.Contains(got, "\"artist\":\"万能青年旅店\"") {
		t.Fatalf("prompt missing required merged content: %q", got)
	}
}
