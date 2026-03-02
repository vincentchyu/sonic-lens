package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ollama/ollama/api"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/log"
)

// --- Ollama Provider ---

// OllamaProvider 使用本地 Ollama 服务实现 LLMProvider
type OllamaProvider struct {
	BaseProvider
	host   string
	model  string
	client *api.Client
}

// newOllamaProvider 从 config 创建 Ollama Provider
// host 默认 http://localhost:11434，model 默认 gpt-oss:latest
func newOllamaProvider(cfg config.OllamaConfig) (LLMProvider, error) {
	host := cfg.Host
	if host == "" {
		host = "http://localhost:11434"
	}
	model := cfg.Model
	if model == "" {
		model = "gpt-oss:latest"
	}

	u, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	// 初始化 Ollama SDK Client
	client := api.NewClient(
		u, &http.Client{
			Timeout: 30 * time.Minute,
		},
	)

	return &OllamaProvider{
		BaseProvider: BaseProvider{
			ProviderName: "ollama",
			ModelName:    model,
		},
		host:   host,
		model:  model,
		client: client,
	}, nil
}

// 移除旧的手动结构体定义，改用 api.* 结构体

// AnalyzeTrack 调用本地 Ollama 接口，对歌词进行翻译和深度解析
func (p *OllamaProvider) AnalyzeTrack(
	ctx context.Context, req TrackAnalysisRequest,
) (*TrackAnalysisResult, error) {
	startTime := time.Now()
	prompt := buildTrackInsightUserPrompt(req)

	ollamaReq := &api.GenerateRequest{
		Model:  p.model,
		Prompt: prompt,
		Stream: new(bool), // set streaming to false
		Think:  &api.ThinkValue{Value: "medium"},
	}

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
		p.SaveCallLog(ctx, req, fullResponse.String(), err, startTime, "sync")
		log.Warn(ctx, "调用 Ollama 接口失败", zap.Error(err))
		return nil, err
	}

	raw := TrimCodeFence(fullContent.String())
	var result TrackAnalysisResult
	if err = json.Unmarshal([]byte(raw), &result); err != nil {
		// 如果解析失败，尝试从文本中提取 JSON 块
		if extracted := extractJSON(raw); extracted != "" {
			if err = json.Unmarshal([]byte(extracted), &result); err == nil {
				goto SUCCESS
			}
		}
		log.Error(ctx, "解析ollama响应失败", zap.Error(err), zap.String("raw", raw))
		p.SaveCallLog(ctx, req, fullResponse.String(), err, startTime, "sync")
		return nil, err
	}
SUCCESS:
	p.SaveCallLog(ctx, req, fullResponse.String(), nil, startTime, "sync")
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	// 修复：将字面量 \n 转换为实际换行符
	result.LyricsTranslation = strings.ReplaceAll(result.LyricsTranslation, "\\n", "\n")

	result.LLMProvider = "ollama:" + p.model
	return &result, nil
}

// AnalyzeTrackStream 实现流式输出
func (p *OllamaProvider) AnalyzeTrackStream(ctx context.Context, req TrackAnalysisRequest) (<-chan string, error) {
	startTime := time.Now()
	prompt := buildTrackInsightUserPrompt(req)
	ollamaReq := &api.GenerateRequest{
		Model:  p.model,
		Prompt: prompt,
		// Stream: &stream, // 默认为 false，SDK 会根据调用方式处理
		Think: &api.ThinkValue{Value: true},
	}

	log.Info(ctx, "Ollama 流式请求详情", zap.String("host", p.host), zap.String("model", p.model))

	out := make(chan string, 100)
	go func() {
		defer close(out)

		var fullResponse strings.Builder
		var finalErr error

		err := p.client.Generate(
			ctx, ollamaReq, func(resp api.GenerateResponse) error {
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
			log.Error(ctx, "Ollama 流式请求失败", zap.Error(err))
			finalErr = err
		}

		// 流结束，保存日志
		p.SaveCallLog(ctx, req, fullResponse.String(), finalErr, startTime, "stream")
	}()

	return out, nil
}
