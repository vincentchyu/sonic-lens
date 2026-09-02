package ai

import (
	"cmp"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/ollama/ollama/api"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/core/telemetry"
)

// --- Ollama Provider ---

// OllamaProvider 使用本地 Ollama 服务实现 LLMProvider
type OllamaProvider struct {
	BaseProvider
	host   string
	model  string
	client *api.Client
}

type ollamaProviderFactory struct {
	cfg          config.OllamaConfig
	host         string
	defaultModel string
	client       *api.Client
	initErr      error
}

// newOllamaProvider 从 config 创建 Ollama Provider
// host 默认 http://localhost:11434，model 默认 gpt-oss:latest
func newOllamaProvider(cfg config.OllamaConfig) (LLMProvider, error) {
	return newOllamaProviderFactory(cfg).Create("")
}

func newOllamaProviderFactory(cfg config.OllamaConfig) providerFactory {
	factory := &ollamaProviderFactory{cfg: cfg}
	host := resolveOllamaHost(cfg)
	factory.host = host
	factory.defaultModel = resolveOllamaModel(cfg)

	u, err := url.Parse(host)
	if err != nil {
		factory.initErr = err
		return factory
	}

	factory.client = api.NewClient(
		u, telemetry.WrapHTTPClient(
			&http.Client{
				Timeout: 30 * time.Minute,
			},
		),
	)
	return factory
}

func (f *ollamaProviderFactory) Platform() common.AIModelPlatform {
	return common.AIModelPlatformOllama
}

func (f *ollamaProviderFactory) DisplayName() string {
	return "Ollama"
}

func (f *ollamaProviderFactory) Configured() bool {
	return strings.TrimSpace(f.host) != "" && f.initErr == nil
}

func (f *ollamaProviderFactory) DefaultModel() string {
	return f.defaultModel
}

func (f *ollamaProviderFactory) Create(model string) (LLMProvider, error) {
	if f.initErr != nil {
		return nil, f.initErr
	}

	resolvedModel := strings.TrimSpace(model)
	if resolvedModel == "" {
		resolvedModel = f.defaultModel
	}

	return &OllamaProvider{
		BaseProvider: BaseProvider{
			ProviderName: "ollama",
			ModelName:    resolvedModel,
		},
		host:   f.host,
		model:  resolvedModel,
		client: f.client,
	}, nil
}

func (f *ollamaProviderFactory) ListModels(ctx context.Context) ([]ModelOption, error) {
	if f.initErr != nil {
		return nil, f.initErr
	}

	resp, err := f.client.List(ctx)
	if err != nil {
		return nil, err
	}

	defaultModel := f.DefaultModel()
	models := make([]ModelOption, 0, len(resp.Models))
	seen := make(map[string]struct{}, len(resp.Models))
	for _, item := range resp.Models {
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
				DisplayName: id,
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

func (f *ollamaProviderFactory) CacheFingerprint() string {
	return strings.Join([]string{string(f.Platform()), resolveOllamaHost(f.cfg), f.DefaultModel()}, "|")
}

func (f *ollamaProviderFactory) InitErr() error {
	return f.initErr
}

func resolveOllamaHost(cfg config.OllamaConfig) string {
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		host = "http://localhost:11434"
	}
	return host
}

func resolveOllamaModel(cfg config.OllamaConfig) string {
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = "gpt-oss:latest"
	}
	return model
}

// 移除旧的手动结构体定义，改用 api.* 结构体

// AnalyzeTrack 调用本地 Ollama 接口，对歌词进行翻译和深度解析
func (p *OllamaProvider) AnalyzeTrack(
	ctx context.Context, req TrackAnalysisRequest,
) (*TrackAnalysisResult, error) {
	startTime := time.Now()
	prompt := buildTrackInsightMergedPrompt(req)

	ollamaReq := &api.GenerateRequest{
		Model:  p.model,
		System: prompt,
		Format: json.RawMessage("json"),
		Stream: new(bool), // set streaming to false
		Options: map[string]any{
			"temperature": float32(DefaultInsightTemperature),
		},
		Think:  &api.ThinkValue{Value: "medium"},
		Prompt: "由于本地模型的能力有限，不做analysis_by_section.appreciate_analysis字段的填充和分析",
	}
	reqJSONBytes, _ := json.Marshal(ollamaReq)
	requestJSON := string(reqJSONBytes)

	var fullResponse strings.Builder
	var fullContent strings.Builder

	log.Info(
		ctx, "Ollama 请求详情", zap.String("host", p.host), zap.String("model", p.model),
		zap.Any("Ollama request", ollamaReq),
	)

	err := p.client.Generate(
		ctx, ollamaReq, func(resp api.GenerateResponse) error {
			// 记录全量响应数据，用于日志审计
			cb, _ := json.Marshal(resp)
			fullResponse.WriteString(string(cb))
			fullResponse.WriteString("\n")

			if resp.Response != "" {
				fullContent.WriteString(resp.Response)
			}
			return nil
		},
	)

	if err != nil {
		p.SaveCallLog(ctx, req, requestJSON, fullResponse.String(), err, startTime, "sync")
		log.Warn(ctx, "调用 Ollama 接口失败", zap.Error(err))
		return nil, err
	}

	result, raw, err := ParseTrackResult(fullContent.String())
	if err != nil {
		log.Error(ctx, "解析ollama响应失败", zap.Error(err), zap.String("raw", raw))
		p.SaveCallLog(ctx, req, requestJSON, fullResponse.String(), err, startTime, "sync")
		return nil, err
	}

	p.SaveCallLog(ctx, req, requestJSON, fullResponse.String(), nil, startTime, "sync")
	result.LLMProvider = "ollama:" + p.model
	return result, nil
}

// AnalyzeAlbum 调用本地 Ollama 接口，对专辑聚合上下文进行深度解析。
func (p *OllamaProvider) AnalyzeAlbum(
	ctx context.Context, req AlbumAnalysisRequest,
) (*AlbumAnalysisResult, error) {
	startTime := time.Now()
	prompt := buildAlbumInsightMergedPrompt(req)

	ollamaReq := &api.GenerateRequest{
		Model:  p.model,
		System: prompt,
		Format: json.RawMessage("json"),
		Stream: new(bool),
		Options: map[string]any{
			"temperature": float32(DefaultInsightTemperature),
		},
		Think:  &api.ThinkValue{Value: "medium"},
		Prompt: "请仅输出符合 schema 的专辑分析 JSON。",
	}
	reqJSONBytes, _ := json.Marshal(ollamaReq)
	requestJSON := string(reqJSONBytes)

	var fullResponse strings.Builder
	var fullContent strings.Builder

	log.Info(
		ctx, "Ollama 专辑分析请求详情", zap.String("host", p.host), zap.String("model", p.model),
		zap.Any("Ollama request", ollamaReq),
	)

	err := p.client.Generate(
		ctx, ollamaReq, func(resp api.GenerateResponse) error {
			cb, _ := json.Marshal(resp)
			fullResponse.WriteString(string(cb))
			fullResponse.WriteString("\n")

			if resp.Response != "" {
				fullContent.WriteString(resp.Response)
			}
			return nil
		},
	)

	if err != nil {
		p.SaveAlbumCallLog(ctx, req, requestJSON, fullResponse.String(), err, startTime, "sync")
		log.Warn(ctx, "调用 Ollama 专辑分析失败", zap.Error(err))
		return nil, err
	}

	result, raw, err := ParseAlbumResult(fullContent.String())
	if err != nil {
		log.Error(ctx, "解析ollama专辑分析响应失败", zap.Error(err), zap.String("raw", raw))
		p.SaveAlbumCallLog(ctx, req, requestJSON, fullResponse.String(), err, startTime, "sync")
		return nil, err
	}

	p.SaveAlbumCallLog(ctx, req, requestJSON, fullResponse.String(), nil, startTime, "sync")
	result.LLMProvider = "ollama:" + p.model
	return result, nil
}

// AnalyzeTrackStream 实现流式输出
// Deprecated: 流式接口已废弃
func (p *OllamaProvider) AnalyzeTrackStream(ctx context.Context, req TrackAnalysisRequest) (<-chan string, error) {
	startTime := time.Now()
	prompt := buildTrackInsightMergedPrompt(req)
	ollamaReq := &api.GenerateRequest{
		Model:  p.model,
		Prompt: prompt,
		// Stream: &stream, // 默认为 false，SDK 会根据调用方式处理
		Think: &api.ThinkValue{Value: true},
	}
	reqJSONBytes, _ := json.Marshal(ollamaReq)
	requestJSON := string(reqJSONBytes)

	log.Info(ctx, "Ollama 流式请求详情", zap.String("host", p.host), zap.String("model", p.model))

	out := make(chan string, 100)
	telemetry.GoSafe(
		ctx, "ai.ollama.stream_track_analysis", func(asyncCtx context.Context) {
			defer close(out)

			var fullResponse strings.Builder
			var finalErr error

			err := p.client.Generate(
				asyncCtx, ollamaReq, func(resp api.GenerateResponse) error {
					// 记录全量内容，用于 SaveCallLog
					rb, _ := json.Marshal(resp)
					fullResponse.WriteString(string(rb))
					fullResponse.WriteString("\n")

					// 依次处理思考内容和回复内容
					// 注意：SDK 的 GenerateResponse 结构中可能有 Response 字段。
					// SDK 目前主要通过 Response 包含所有输出片段。
					if !resp.Done {
						if resp.Thinking != "" {
							out <- resp.Thinking
						}
						if resp.Response != "" {
							out <- resp.Response
						}
					}
					return nil
				},
			)

			if err != nil {
				log.Error(asyncCtx, "Ollama 流式请求失败", zap.Error(err))
				finalErr = err
			}

			// 流结束，保存日志
			p.SaveCallLog(asyncCtx, req, requestJSON, fullResponse.String(), finalErr, startTime, "stream")
		},
	)

	return out, nil
}
