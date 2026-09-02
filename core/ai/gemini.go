package ai

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/genai"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/core/telemetry"
)

// --- Gemini Provider ---

// GeminiProvider 使用本地 Gemini 服务实现 LLMProvider
type GeminiProvider struct {
	BaseProvider
	host   string
	model  string
	client *genai.Client
}

type geminiProviderFactory struct {
	cfg          config.GeminiConfig
	client       *genai.Client
	defaultModel string
	initErr      error
}

// newGeminiProvider 从 config 创建 Gemini Provider
func newGeminiProvider(cfg config.GeminiConfig) (LLMProvider, error) {
	return newGeminiProviderFactory(cfg).Create("")
}

func newGeminiProviderFactory(cfg config.GeminiConfig) providerFactory {
	factory := &geminiProviderFactory{cfg: cfg}
	if !factory.hasConfig() {
		return factory
	}
	client, err := genai.NewClient(
		context.Background(), &genai.ClientConfig{
			APIKey: resolveGeminiAPIKey(cfg),
			HTTPClient: telemetry.WrapHTTPClient(
				&http.Client{
					Transport: &http.Transport{
						Proxy: http.ProxyFromEnvironment,
					},
				},
			),
			HTTPOptions: genai.HTTPOptions{
				BaseURL: strings.TrimSpace(cfg.BaseURL),
				Timeout: new(time.Second * 60 * 5),
			},
		},
	)
	if err != nil {
		factory.initErr = err
		return factory
	}

	factory.client = client
	factory.defaultModel = resolveGeminiModel(cfg)
	return factory
}

func (f *geminiProviderFactory) Platform() common.AIModelPlatform {
	return common.AIModelPlatformGemini
}

func (f *geminiProviderFactory) DisplayName() string {
	return "Gemini"
}

func (f *geminiProviderFactory) Configured() bool {
	return f.hasConfig() && f.initErr == nil
}

func (f *geminiProviderFactory) DefaultModel() string {
	return f.defaultModel
}

func (f *geminiProviderFactory) Create(model string) (LLMProvider, error) {
	if f.initErr != nil {
		return nil, f.initErr
	}

	resolvedModel := strings.TrimSpace(model)
	if resolvedModel == "" {
		resolvedModel = f.defaultModel
	}

	return &GeminiProvider{
		BaseProvider: BaseProvider{
			ProviderName: "gemini",
			ModelName:    resolvedModel,
		},
		host:   f.cfg.BaseURL,
		model:  resolvedModel,
		client: f.client,
	}, nil
}

func (f *geminiProviderFactory) ListModels(ctx context.Context) ([]ModelOption, error) {
	if f.initErr != nil {
		return nil, f.initErr
	}
	defaultModel := f.DefaultModel()
	models := make([]ModelOption, 0)
	seen := make(map[string]struct{})
	for item, iterErr := range f.client.Models.All(ctx) {
		if iterErr != nil {
			return nil, iterErr
		}
		if item == nil {
			continue
		}
		id := strings.TrimSpace(item.Name)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		models = append(
			models, ModelOption{
				ID:          id,
				DisplayName: item.DisplayName,
				IsDefault:   id == defaultModel,
			},
		)
	}
	if defaultModel != "" {
		if _, exists := seen[defaultModel]; !exists {
			models = append(
				models, ModelOption{
					ID:          defaultModel,
					DisplayName: defaultModel,
					IsDefault:   true,
				},
			)
		}
	}
	slices.SortFunc(
		models, func(a, b ModelOption) int {
			if a.IsDefault != b.IsDefault {
				if a.IsDefault {
					return -1
				}
				return 1
			}
			return cmp.Compare(a.ID, b.ID)
		},
	)
	return models, nil
}

func (f *geminiProviderFactory) CacheFingerprint() string {
	return strings.Join(
		[]string{
			string(f.Platform()), strings.TrimSpace(f.cfg.BaseURL), f.DefaultModel(), resolveGeminiAPIKey(f.cfg),
		}, "|",
	)
}

func (f *geminiProviderFactory) InitErr() error {
	return f.initErr
}

func (f *geminiProviderFactory) hasConfig() bool {
	return resolveGeminiAPIKey(f.cfg) != ""
}

func resolveGeminiAPIKey(cfg config.GeminiConfig) string {
	return strings.TrimSpace(cfg.APIKey)
}

func resolveGeminiModel(cfg config.GeminiConfig) string {
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = "gemini-2.5-flash"
	}
	return model
}

// SendChatRequest 实现 RawChatClient 接口
func (p *GeminiProvider) SendChatRequest(
	ctx context.Context, req TrackAnalysisRequest, systemPrompt, userPrompt string, schema map[string]any, step string,
) (string, error) {
	startTime := time.Now()

	requestPayload := map[string]any{
		"model":                p.model,
		"contents":             []string{userPrompt},
		"system_instruction":   systemPrompt,
		"response_mime_type":   "application/json",
		"response_json_schema": schema,
		"thinking_config": map[string]any{
			"include_thoughts": true,
			"thinking_level":   "medium",
		},
	}
	requestBytes, _ := json.Marshal(requestPayload)
	requestJSON := string(requestBytes)

	gResult, err := p.client.Models.GenerateContent(
		ctx,
		p.model,
		genai.Text(userPrompt),
		&genai.GenerateContentConfig{
			Temperature: genai.Ptr(float32(DefaultInsightTemperature)),
			ThinkingConfig: &genai.ThinkingConfig{
				IncludeThoughts: true,
				ThinkingLevel:   genai.ThinkingLevelMedium,
			},
			SystemInstruction:  genai.NewContentFromText(systemPrompt, genai.RoleUser),
			ResponseMIMEType:   "application/json",
			ResponseJsonSchema: schema,
		},
	)
	if err != nil {
		p.SaveCallLog(ctx, req, requestJSON, "", err, startTime, step)
		return "", err
	}

	respText := gResult.Text()
	respBytes, _ := json.Marshal(gResult)
	respJSON := string(respBytes)

	p.SaveCallLog(ctx, req, requestJSON, respJSON, nil, startTime, step)
	return respText, nil
}

// AnalyzeTrack 调Gemini API，对歌词进行翻译和深度解析
func (p *GeminiProvider) AnalyzeTrack(
	ctx context.Context, req TrackAnalysisRequest,
) (*TrackAnalysisResult, error) {
	if config.ConfigObj.AI.MultiStep {
		return multiStepAnalyzeTrack(ctx, p, req)
	}
	return p.analyzeTrackSingleStep(ctx, req)
}

func (p *GeminiProvider) analyzeTrackSingleStep(
	ctx context.Context, req TrackAnalysisRequest,
) (*TrackAnalysisResult, error) {
	respStr, err := p.SendChatRequest(
		ctx, req, buildTrackInsightSystemPromptWithoutSchema(req), buildTrackInsightUserPrompt(req),
		GetTrackInsightSchema(), "sync",
	)
	if err != nil {
		return nil, err
	}

	result, raw, err := ParseTrackResult(respStr)
	if err != nil {
		log.Error(ctx, "解析Gemini响应失败", zap.Error(err), zap.String("raw", raw))
		return nil, err
	}

	result.LLMProvider = "gemini:" + p.model
	return result, nil
}

// AnalyzeAlbum 调 Gemini API，对专辑聚合上下文进行深度解析。
func (p *GeminiProvider) AnalyzeAlbum(
	ctx context.Context, req AlbumAnalysisRequest,
) (*AlbumAnalysisResult, error) {
	startTime := time.Now()

	userPrompt := buildAlbumInsightUserPrompt(req)
	systemPrompt := buildAlbumInsightSystemPromptWithoutSchema(req)
	requestPayload := map[string]any{
		"model":                p.model,
		"contents":             []string{userPrompt},
		"system_instruction":   systemPrompt,
		"response_mime_type":   "application/json",
		"response_json_schema": GetAlbumInsightSchema(),
		"thinking_config": map[string]any{
			"include_thoughts": true,
			"thinking_level":   "medium",
		},
	}
	requestBytes, _ := json.Marshal(requestPayload)
	requestJSON := string(requestBytes)
	log.Info(
		ctx, "Gemini专辑分析请求体", zap.String("model", p.model), zap.String("userPrompt", userPrompt),
		zap.String("systemPrompt", systemPrompt),
	)

	gResult, err := p.client.Models.GenerateContent(
		ctx,
		p.model,
		genai.Text(userPrompt),
		&genai.GenerateContentConfig{
			Temperature: genai.Ptr(float32(DefaultInsightTemperature)),
			ThinkingConfig: &genai.ThinkingConfig{
				IncludeThoughts: true,
				ThinkingLevel:   genai.ThinkingLevelMedium,
			},
			SystemInstruction:  genai.NewContentFromText(systemPrompt, genai.RoleUser),
			ResponseMIMEType:   "application/json",
			ResponseJsonSchema: GetAlbumInsightSchema(),
		},
	)
	if err != nil {
		log.Error(ctx, "调用Gemini专辑分析失败", zap.Error(err))
		p.SaveAlbumCallLog(ctx, req, requestJSON, "", err, startTime, "sync")
		return nil, err
	}

	respText := gResult.Text()
	log.Debug(ctx, "Gemini专辑分析响应内容", zap.String("content", respText))

	respBytes, _ := json.Marshal(gResult)
	respJSON := string(respBytes)

	result, raw, err := ParseAlbumResult(respText)
	if err != nil {
		log.Error(ctx, "解析Gemini专辑分析响应失败", zap.Error(err), zap.String("raw", raw))
		p.SaveAlbumCallLog(ctx, req, requestJSON, respJSON, err, startTime, "sync")
		return nil, err
	}

	p.SaveAlbumCallLog(ctx, req, requestJSON, respJSON, nil, startTime, "sync")
	result.LLMProvider = "gemini:" + p.model
	return result, nil
}

// AnalyzeTrackStream 实现流式输出
func (p *GeminiProvider) AnalyzeTrackStream(ctx context.Context, req TrackAnalysisRequest) (<-chan string, error) {
	return nil, errors.New("流式接口已废弃")
}
