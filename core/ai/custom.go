package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/log"
)

// CustomProvider 使用自定义的 OpenAI 兼容接口实现 LLMProvider
type CustomProvider struct {
	BaseProvider
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

func newCustomProviderFromConfigOrEnv(customConfig config.CustomAIConfig) (LLMProvider, error) {
	apiKey := customConfig.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("CUSTOM_API_KEY")
	}
	if apiKey == "" {
		return nil, errors.New("未配置 Custom AI API Key (config.ai.custom.apiKey 变量 CUSTOM_API_KEY)")
	}

	baseURL := customConfig.BaseURL
	if baseURL == "" {
		baseURL = os.Getenv("CUSTOM_BASE_URL")
	}
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8000"
	}

	model := customConfig.Model
	if model == "" {
		model = os.Getenv("CUSTOM_MODEL")
	}
	if model == "" {
		model = "default-model"
	}

	return &CustomProvider{
		BaseProvider: BaseProvider{
			ProviderName: "custom",
			ModelName:    model,
		},
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		httpClient: &http.Client{
			Timeout: 60 * time.Minute,
		},
	}, nil
}

type customChatRequest struct {
	Model    string              `json:"model"`
	Messages []openAIChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
}

type customChatResponse struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			ReasoningContent string `json:"reasoning_content,omitempty"`
			Content          string `json:"content,omitempty"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type customCompletionsRequest struct {
	Model       string `json:"model"`
	Prompt      string `json:"prompt"`
	MaxTokens   int    `json:"max_tokens"`
	Temperature int    `json:"temperature"`
	Stream      bool   `json:"stream"`
}
type customCompletionsResponse struct {
	Id                string `json:"id"`
	Object            string `json:"object"`
	Created           int    `json:"created"`
	Model             string `json:"model"`
	SystemFingerprint string `json:"system_fingerprint"`
	Choices           []struct {
		Text         string      `json:"text"`
		Index        int         `json:"index"`
		Logprobs     interface{} `json:"logprobs"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type customChatChunk struct {
	Id      string `json:"id"`
	Choices []struct {
		Delta struct {
			ReasoningContent string `json:"reasoning_content,omitempty"`
			Content          string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
}

func (p *CustomProvider) buildRequest(ctx context.Context, req TrackAnalysisRequest, stream bool) (
	*http.Request, error,
) {
	payload := customChatRequest{
		Model: p.model,
		Messages: []openAIChatMessage{
			{Role: "system", Content: buildTrackInsightMergedPrompt(req)},
			{Role: "user", Content: "开始的表演"},
		},
		Stream: stream,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	return httpReq, nil
}

func (p *CustomProvider) buildCompletionsRequest(ctx context.Context, req TrackAnalysisRequest) (
	*http.Request, error,
) {
	payload := customCompletionsRequest{
		Model:  p.model,
		Prompt: buildTrackInsightMergedPrompt(req),
		Stream: false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	return httpReq, nil
}

// AnalyzeTrack 调用 Custom API（非流式）
func (p *CustomProvider) AnalyzeTrack(
	ctx context.Context, req TrackAnalysisRequest,
) (*TrackAnalysisResult, error) {
	startTime := time.Now()

	httpReq, err := p.buildRequest(ctx, req, false)
	if err != nil {
		p.SaveCallLog(ctx, req, "", err, startTime, "sync")
		return nil, err
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		p.SaveCallLog(ctx, req, "", err, startTime, "sync")
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = errors.New("调用 Custom API 失败，状态码: " + resp.Status)
		p.SaveCallLog(ctx, req, "", err, startTime, "sync")
		return nil, err
	}

	var chatResp customChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		p.SaveCallLog(ctx, req, "", err, startTime, "sync")
		return nil, err
	}
	if len(chatResp.Choices) == 0 {
		err = errors.New("Custom AI 返回结果为空")
		p.SaveCallLog(ctx, req, "", err, startTime, "sync")
		return nil, err
	}

	chatRespBytes, _ := json.Marshal(chatResp)
	p.SaveCallLog(ctx, req, string(chatRespBytes), nil, startTime, "sync")

	rawContent := chatResp.Choices[0].Message.Content
	rawReasoning := chatResp.Choices[0].Message.ReasoningContent

	raw := TrimCodeFence(rawContent)

	var result TrackAnalysisResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		if extracted := extractJSON(raw); extracted != "" {
			if err = json.Unmarshal([]byte(extracted), &result); err == nil {
				goto SUCCESS
			}
		}
		log.Error(
			ctx, "解析 Custom AI 响应失败", zap.Error(err), zap.String("raw", raw),
			zap.String("reasoning", rawReasoning),
		)
		p.SaveCallLog(ctx, req, rawContent, err, startTime, "sync")
		return nil, err
	}

SUCCESS:
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	result.LyricsTranslation = strings.ReplaceAll(result.LyricsTranslation, "\\n", "\n")

	// 将深度推理内容放置在 Metadata 内
	if rawReasoning != "" {
		result.Metadata["reasoning_content"] = rawReasoning
	}

	result.LLMProvider = "oMLX:" + p.model
	return &result, nil
}

// AnalyzeTrackStream 返回流式分析结果
func (p *CustomProvider) AnalyzeTrackStream(ctx context.Context, req TrackAnalysisRequest) (<-chan string, error) {
	httpReq, err := p.buildRequest(ctx, req, true)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, errors.New("调用 Custom API 失败，状态码: " + resp.Status)
	}

	ch := make(chan string)
	go func() {
		defer resp.Body.Close()
		defer close(ch)

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					log.Error(ctx, "Custom AI 流式解析异常", zap.Error(err))
				}
				break
			}
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if line == "data: [DONE]" {
				break
			}
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			dataJson := strings.TrimPrefix(line, "data: ")
			var chunk customChatChunk
			if err := json.Unmarshal([]byte(dataJson), &chunk); err != nil {
				log.Error(ctx, "Stream json parse error", zap.Error(err), zap.String("line", line))
				continue
			}

			if len(chunk.Choices) > 0 {
				if rc := chunk.Choices[0].Delta.ReasoningContent; rc != "" {
					ch <- rc
				}
				if mc := chunk.Choices[0].Delta.Content; mc != "" {
					ch <- mc
				}
			}
		}
	}()

	return ch, nil
}
