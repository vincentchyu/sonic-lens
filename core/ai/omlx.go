package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/core/telemetry"
)

// OMLXProvider 使用自定义的 OpenAI 兼容接口实现 LLMProvider
type OMLXProvider struct {
	BaseProvider
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

type omlxProviderFactory struct {
	cfg               config.OMLXConfig
	apiKey            string
	baseURL           string
	defaultModel      string
	runtimeHTTPClient *http.Client
	catalogHTTPClient *http.Client
	initErr           error
}

func newOMLXProviderFromConfigOrEnv(omlxConfig config.OMLXConfig) (LLMProvider, error) {
	return newOmlxProviderFactory(omlxConfig).Create("")
}

func newOmlxProviderFactory(cfg config.OMLXConfig) providerFactory {
	factory := &omlxProviderFactory{cfg: cfg}
	if !factory.hasConfig() {
		return factory
	}

	apiKey, baseURL, defaultModel, err := resolveOMLXConfig(cfg, "")
	if err != nil {
		factory.initErr = err
		return factory
	}

	factory.apiKey = apiKey
	factory.baseURL = strings.TrimRight(baseURL, "/")
	factory.defaultModel = defaultModel
	factory.runtimeHTTPClient = telemetry.WrapHTTPClient(&http.Client{Timeout: 60 * time.Minute})
	factory.catalogHTTPClient = telemetry.WrapHTTPClient(&http.Client{Timeout: 30 * time.Second})
	return factory
}

func (f *omlxProviderFactory) Platform() common.AIModelPlatform {
	return common.AIModelPlatformOMLX
}

func (f *omlxProviderFactory) DisplayName() string {
	return "OMLX"
}

func (f *omlxProviderFactory) Configured() bool {
	return f.hasConfig() && f.initErr == nil
}

func (f *omlxProviderFactory) DefaultModel() string {
	return f.defaultModel
}

func (f *omlxProviderFactory) Create(model string) (LLMProvider, error) {
	if f.initErr != nil {
		return nil, f.initErr
	}

	resolvedModel := strings.TrimSpace(model)
	if resolvedModel == "" {
		resolvedModel = f.defaultModel
	}

	return &OMLXProvider{
		BaseProvider: BaseProvider{
			ProviderName: "omlx",
			ModelName:    resolvedModel,
		},
		apiKey:     f.apiKey,
		baseURL:    f.baseURL,
		model:      resolvedModel,
		httpClient: f.runtimeHTTPClient,
	}, nil
}

func (f *omlxProviderFactory) ListModels(ctx context.Context) ([]ModelOption, error) {
	if f.initErr != nil {
		return nil, f.initErr
	}
	return listOpenAICompatibleModelsWithClient(ctx, f.catalogHTTPClient, f.baseURL, f.apiKey, f.defaultModel)
}

func (f *omlxProviderFactory) CacheFingerprint() string {
	apiKey := resolveOMLXAPIKey(f.cfg)
	baseURL := resolveOMLXBaseURL(f.cfg)
	model := f.DefaultModel()
	return strings.Join([]string{string(f.Platform()), baseURL, model, apiKey}, "|")
}

func (f *omlxProviderFactory) InitErr() error {
	return f.initErr
}

func (f *omlxProviderFactory) hasConfig() bool {
	return resolveOMLXAPIKey(f.cfg) != ""
}

func resolveOMLXConfig(omlxConfig config.OMLXConfig, modelOverride string) (string, string, string, error) {
	apiKey := resolveOMLXAPIKey(omlxConfig)
	if apiKey == "" {
		return "", "", "", errors.New("未配置 OMLX AI API Key (config.ai.omlx.apiKey 变量 OMLX_API_KEY)")
	}

	baseURL := resolveOMLXBaseURL(omlxConfig)
	model := strings.TrimSpace(modelOverride)
	if model == "" {
		model = resolveOMLXModel(omlxConfig)
	}
	return apiKey, baseURL, model, nil
}

func resolveOMLXAPIKey(omlxConfig config.OMLXConfig) string {
	apiKey := strings.TrimSpace(omlxConfig.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("OMLX_API_KEY"))
	}
	return apiKey
}

func resolveOMLXBaseURL(omlxConfig config.OMLXConfig) string {
	baseURL := strings.TrimSpace(omlxConfig.BaseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(os.Getenv("OMLX_BASE_URL"))
	}
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8000"
	}
	return strings.TrimRight(baseURL, "/")
}

func resolveOMLXModel(omlxConfig config.OMLXConfig) string {
	model := strings.TrimSpace(omlxConfig.Model)
	if model == "" {
		model = strings.TrimSpace(os.Getenv("OMLX_MODEL"))
	}
	if model == "" {
		model = "default-model"
	}
	return model
}

type omlxChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type omlxChatRequest struct {
	Model       string            `json:"model"`
	Messages    []omlxChatMessage `json:"messages"`
	Stream      bool              `json:"stream"`
	Temperature float32           `json:"temperature"`
}

type omlxChatResponse struct {
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

type omlxCompletionsRequest struct {
	Model       string `json:"model"`
	Prompt      string `json:"prompt"`
	MaxTokens   int    `json:"max_tokens"`
	Temperature int    `json:"temperature"`
	Stream      bool   `json:"stream"`
}
type omlxCompletionsResponse struct {
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

type omlxChatChunk struct {
	Id      string `json:"id"`
	Choices []struct {
		Delta struct {
			ReasoningContent string `json:"reasoning_content,omitempty"`
			Content          string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
}

func (p *OMLXProvider) buildAlbumRequest(ctx context.Context, req AlbumAnalysisRequest) (
	*http.Request, string, error,
) {
	payload := omlxChatRequest{
		Model: p.model,
		Messages: []omlxChatMessage{
			{Role: "system", Content: buildAlbumInsightSystemPromptWithSchema(req)},
			{Role: "user", Content: buildAlbumInsightUserPrompt(req)},
		},
		Stream:      false,
		Temperature: DefaultInsightTemperature,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(body),
	)
	if err != nil {
		return nil, "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	return httpReq, string(body), nil
}

// SendChatRequest 实现 RawChatClient 接口
func (p *OMLXProvider) SendChatRequest(
	ctx context.Context, req TrackAnalysisRequest, systemPrompt, userPrompt string, schema map[string]any, step string,
) (string, error) {
	startTime := time.Now()
	payload := omlxChatRequest{
		Model: p.model,
		Messages: []omlxChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Stream:      false,
		Temperature: DefaultInsightTemperature,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(body),
	)
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		p.SaveCallLog(ctx, req, string(body), "", err, startTime, step)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = errors.New("调用 OMLX API 失败，状态码: " + resp.Status)
		p.SaveCallLog(ctx, req, string(body), "", err, startTime, step)
		return "", err
	}

	var chatResp omlxChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		p.SaveCallLog(ctx, req, string(body), "", err, startTime, step)
		return "", err
	}
	if len(chatResp.Choices) == 0 {
		err = errors.New("OMLX AI 返回结果为空")
		p.SaveCallLog(ctx, req, string(body), "", err, startTime, step)
		return "", err
	}

	chatRespBytes, _ := json.Marshal(chatResp)
	p.SaveCallLog(ctx, req, string(body), string(chatRespBytes), nil, startTime, step)

	return chatResp.Choices[0].Message.Content, nil
}

func (p *OMLXProvider) buildCompletionsRequest(ctx context.Context, req TrackAnalysisRequest) (
	*http.Request, string, error,
) {
	payload := omlxCompletionsRequest{
		Model:  p.model,
		Prompt: buildTrackInsightMergedPrompt(req),
		Stream: false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx, http.MethodPost, p.baseURL+"/v1/completions", bytes.NewReader(body),
	)
	if err != nil {
		return nil, "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	return httpReq, string(body), nil
}

// AnalyzeTrack 调用 OMLX API（分支支持单步/多步）
func (p *OMLXProvider) AnalyzeTrack(
	ctx context.Context, req TrackAnalysisRequest,
) (*TrackAnalysisResult, error) {
	if config.ConfigObj.AI.MultiStep {
		return multiStepAnalyzeTrack(ctx, p, req)
	}
	return p.analyzeTrackSingleStep(ctx, req)
}

// analyzeTrackSingleStep 原有的单步分析逻辑
func (p *OMLXProvider) analyzeTrackSingleStep(
	ctx context.Context, req TrackAnalysisRequest,
) (*TrackAnalysisResult, error) {
	respStr, err := p.SendChatRequest(
		ctx, req, buildTrackInsightSystemPromptWithSchema(req), buildTrackInsightUserPrompt(req), nil, "sync",
	)
	if err != nil {
		return nil, err
	}

	result, raw, err := ParseTrackResult(respStr)
	if err != nil {
		log.Error(
			ctx, "解析 OMLX AI 响应失败", zap.Error(err), zap.String("raw", raw),
		)
		return nil, err
	}

	result.LLMProvider = "oMLX:" + p.model
	return result, nil
}

// AnalyzeAlbum 调用 OMLX API（非流式专辑分析）。
func (p *OMLXProvider) AnalyzeAlbum(
	ctx context.Context, req AlbumAnalysisRequest,
) (*AlbumAnalysisResult, error) {
	startTime := time.Now()

	httpReq, requestJSON, err := p.buildAlbumRequest(ctx, req)
	if err != nil {
		p.SaveAlbumCallLog(ctx, req, requestJSON, "", err, startTime, "sync")
		return nil, err
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		p.SaveAlbumCallLog(ctx, req, requestJSON, "", err, startTime, "sync")
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = errors.New("调用 OMLX API 失败，状态码: " + resp.Status)
		p.SaveAlbumCallLog(ctx, req, requestJSON, "", err, startTime, "sync")
		return nil, err
	}

	var chatResp omlxChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		p.SaveAlbumCallLog(ctx, req, requestJSON, "", err, startTime, "sync")
		return nil, err
	}
	if len(chatResp.Choices) == 0 {
		err = errors.New("OMLX AI 返回结果为空")
		p.SaveAlbumCallLog(ctx, req, requestJSON, "", err, startTime, "sync")
		return nil, err
	}

	chatRespBytes, _ := json.Marshal(chatResp)
	p.SaveAlbumCallLog(ctx, req, requestJSON, string(chatRespBytes), nil, startTime, "sync")

	rawContent := chatResp.Choices[0].Message.Content
	rawReasoning := chatResp.Choices[0].Message.ReasoningContent

	result, raw, err := ParseAlbumResult(rawContent)
	if err != nil {
		log.Error(
			ctx, "解析 OMLX AI 专辑分析响应失败", zap.Error(err), zap.String("raw", raw),
			zap.String("reasoning", rawReasoning),
		)
		p.SaveAlbumCallLog(ctx, req, requestJSON, rawContent, err, startTime, "sync")
		return nil, err
	}

	if rawReasoning != "" {
		if result.Metadata == nil {
			result.Metadata = make(map[string]interface{})
		}
		result.Metadata["reasoning_content"] = rawReasoning
	}
	result.LLMProvider = "oMLX:" + p.model
	return result, nil
}

// AnalyzeTrackStream 返回流式分析结果
// Deprecated: 流式接口已废弃
// AnalyzeTrackStream 返回流式分析结果
// Deprecated: 流式接口已废弃
func (p *OMLXProvider) AnalyzeTrackStream(ctx context.Context, req TrackAnalysisRequest) (<-chan string, error) {
	return nil, errors.New("流式接口已废弃")
}
