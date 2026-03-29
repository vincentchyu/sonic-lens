package ai

import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"sort"
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
			APIKey:     resolveGeminiAPIKey(cfg),
			HTTPClient: telemetry.WrapHTTPClient(&http.Client{}),
			HTTPOptions: genai.HTTPOptions{
				BaseURL: strings.TrimSpace(cfg.BaseURL),
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
	sort.Slice(
		models, func(i, j int) bool {
			if models[i].IsDefault != models[j].IsDefault {
				return models[i].IsDefault
			}
			return models[i].ID < models[j].ID
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

// AnalyzeTrack 调Gemini API，对歌词进行翻译和深度解析
func (p *GeminiProvider) AnalyzeTrack(
	ctx context.Context, req TrackAnalysisRequest,
) (*TrackAnalysisResult, error) {
	// 记录开始时间
	startTime := time.Now()

	// 构建请求消息
	userPrompt := buildTrackInsightUserPrompt(req)
	insightSystemPrompt := buildTrackInsightSystemPrompt()
	requestPayload := map[string]any{
		"model":                p.model,
		"contents":             []string{userPrompt},
		"system_instruction":   insightSystemPrompt,
		"response_mime_type":   "application/json",
		"response_json_schema": GetTrackInsightSchema(),
		"thinking_config": map[string]any{
			"include_thoughts": true,
			"thinking_level":   "medium",
		},
	}
	requestBytes, _ := json.Marshal(requestPayload)
	requestJSON := string(requestBytes)
	// 打印请求体
	log.Info(
		ctx, "Gemini请求体", zap.String("model", p.model), zap.String("user userPrompt", userPrompt),
		zap.String("insightSystemPrompt", insightSystemPrompt),
	)

	gResult, err := p.client.Models.GenerateContent(
		ctx,
		p.model,
		genai.Text(userPrompt),
		&genai.GenerateContentConfig{
			ThinkingConfig: &genai.ThinkingConfig{
				IncludeThoughts: true,
				ThinkingBudget:  nil,
				ThinkingLevel:   genai.ThinkingLevelMedium,
			},
			SystemInstruction:  genai.NewContentFromText(insightSystemPrompt, genai.RoleUser),
			ResponseMIMEType:   "application/json",
			ResponseJsonSchema: GetTrackInsightSchema(),
		},
	)
	if err != nil {
		log.Error(ctx, "调用Gemini API 失败", zap.Error(err))
		p.SaveCallLog(ctx, req, requestJSON, "", err, startTime, "sync")
		return nil, err
	}

	// 获取响应文本
	respText := gResult.Text()
	log.Debug(ctx, "Gemini响应内容", zap.String("content", respText))

	// 记录调用流水
	respBytes, _ := json.Marshal(gResult)
	respJSON := string(respBytes)

	// 提取 JSON 内容
	raw := TrimCodeFence(respText)

	// 解析 JSON 响应
	var result TrackAnalysisResult
	if err = json.Unmarshal([]byte(raw), &result); err != nil {
		// 如果解析失败，尝试从文本中提取 JSON 块
		if extracted := extractJSON(raw); extracted != "" {
			if err = json.Unmarshal([]byte(extracted), &result); err == nil {
				goto SUCCESS
			}
		}
		log.Error(ctx, "解析Gemini响应失败", zap.Error(err), zap.String("raw", raw))
		p.SaveCallLog(ctx, req, requestJSON, respJSON, err, startTime, "sync")
		return nil, err
	}

SUCCESS:
	p.SaveCallLog(ctx, req, requestJSON, respJSON, nil, startTime, "sync")
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	// 修复：将字面量 \n 转换为实际换行符
	result.LyricsTranslation = strings.ReplaceAll(result.LyricsTranslation, "\\n", "\n")

	result.LLMProvider = "gemini:" + p.model
	return &result, nil
}

// AnalyzeAlbum 调 Gemini API，对专辑聚合上下文进行深度解析。
func (p *GeminiProvider) AnalyzeAlbum(
	ctx context.Context, req AlbumAnalysisRequest,
) (*AlbumAnalysisResult, error) {
	startTime := time.Now()

	userPrompt := buildAlbumInsightUserPrompt(req)
	systemPrompt := buildAlbumInsightSystemPrompt()
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
			ThinkingConfig: &genai.ThinkingConfig{
				IncludeThoughts: true,
				ThinkingBudget:  nil,
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

	raw := TrimCodeFence(respText)

	var result AlbumAnalysisResult
	if err = json.Unmarshal([]byte(raw), &result); err != nil {
		if extracted := extractJSON(raw); extracted != "" {
			if err = json.Unmarshal([]byte(extracted), &result); err == nil {
				goto SUCCESS
			}
		}
		log.Error(ctx, "解析Gemini专辑分析响应失败", zap.Error(err), zap.String("raw", raw))
		p.SaveAlbumCallLog(ctx, req, requestJSON, respJSON, err, startTime, "sync")
		return nil, err
	}

SUCCESS:
	p.SaveAlbumCallLog(ctx, req, requestJSON, respJSON, nil, startTime, "sync")
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	result.LLMProvider = "gemini:" + p.model
	return &result, nil
}

// AnalyzeTrackStream 实现流式输出
func (p *GeminiProvider) AnalyzeTrackStream(ctx context.Context, req TrackAnalysisRequest) (<-chan string, error) {
	prompt := buildTrackInsightUserPrompt(req)
	iter := p.client.Models.GenerateContentStream(ctx, p.model, genai.Text(prompt), nil)

	ch := make(chan string)
	telemetry.GoSafe(
		ctx, "ai.gemini.stream_track_analysis", func(asyncCtx context.Context) {
			defer close(ch)
			for resp, err := range iter {
				if err != nil {
					if err != io.EOF {
						log.Error(asyncCtx, "Gemini流式输出异常", zap.Error(err))
					}
					return
				}
				if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
					// 修正：尝试获取 Text 内容
					part := resp.Candidates[0].Content.Parts[0]
					if part.Text != "" {
						ch <- part.Text
					}
				}
			}
		},
	)

	return ch, nil
}
