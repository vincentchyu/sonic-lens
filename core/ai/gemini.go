package ai

import (
	"context"
	"encoding/json/v2"
	"io"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/genai"

	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/log"
)

// --- Gemini Provider ---

// GeminiProvider 使用本地 Gemini 服务实现 LLMProvider
type GeminiProvider struct {
	BaseProvider
	host   string
	model  string
	client *genai.Client
}

// newGeminiProvider 从 config 创建 Gemini Provider
func newGeminiProvider(cfg config.GeminiConfig) (LLMProvider, error) {
	// 修正：genai.NewClient 会自动从 ClientConfig 中读取 APIKey。
	// 如果 SDK 版本较新，可能不需要显式指定 Backend。
	client, err := genai.NewClient(
		context.TODO(), &genai.ClientConfig{
			APIKey: cfg.APIKey,
		},
	)
	if err != nil {
		return nil, err
	}

	return &GeminiProvider{
		BaseProvider: BaseProvider{
			ProviderName: "gemini",
			ModelName:    cfg.Model,
		},
		host:   cfg.BaseURL,
		model:  cfg.Model,
		client: client,
	}, nil
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
				ThinkingLevel:   genai.ThinkingLevelHigh,
			},
			SystemInstruction: genai.NewContentFromText(insightSystemPrompt, genai.RoleUser),
		},
	)
	if err != nil {
		log.Error(ctx, "调用Gemini API 失败", zap.Error(err))
		p.SaveCallLog(ctx, req, "", err, startTime, "sync")
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
		p.SaveCallLog(ctx, req, respJSON, err, startTime, "sync")
		return nil, err
	}

SUCCESS:
	p.SaveCallLog(ctx, req, respJSON, nil, startTime, "sync")
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	// 修复：将字面量 \n 转换为实际换行符
	result.LyricsTranslation = strings.ReplaceAll(result.LyricsTranslation, "\\n", "\n")

	result.LLMProvider = "gemini:" + p.model
	return &result, nil
}

// AnalyzeTrackStream 实现流式输出
func (p *GeminiProvider) AnalyzeTrackStream(ctx context.Context, req TrackAnalysisRequest) (<-chan string, error) {
	prompt := buildTrackInsightUserPrompt(req)
	iter := p.client.Models.GenerateContentStream(ctx, p.model, genai.Text(prompt), nil)

	ch := make(chan string)
	go func() {
		defer close(ch)
		for resp, err := range iter {
			if err != nil {
				if err != io.EOF {
					log.Error(ctx, "Gemini流式输出异常", zap.Error(err))
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
	}()

	return ch, nil
}
