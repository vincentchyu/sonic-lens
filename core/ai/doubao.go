package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/log"
)

// --- Doubao Provider ---

// DoubaoProvider 使用本地 Doubao 服务实现 LLMProvider
type DoubaoProvider struct {
	BaseProvider
	host   string
	model  string
	client *arkruntime.Client
}

// newDoubaoProvider 从 config 创建 Doubao Provider
func newDoubaoProvider(cfg config.DoubaoConfig) (LLMProvider, error) {
	c := arkruntime.NewClientWithApiKey(
		cfg.APIKey,
		// The base URL for model invocation
		arkruntime.WithBaseUrl(cfg.BaseURL),
	)

	return &DoubaoProvider{
		BaseProvider: BaseProvider{
			ProviderName: "doubao",
			ModelName:    cfg.Model,
		},
		host:   cfg.BaseURL,
		model:  cfg.Model,
		client: c,
	}, nil
}

// AnalyzeTrack 调用豆包 API，对歌词进行翻译和深度解析
func (p *DoubaoProvider) AnalyzeTrack(
	ctx context.Context, req TrackAnalysisRequest,
) (*TrackAnalysisResult, error) {
	// 记录开始时间
	startTime := time.Now()
	var err error
	// 构建请求消息
	dReq := model.CreateChatCompletionRequest{
		Model: p.model,
		Messages: []*model.ChatCompletionMessage{
			{
				Role: model.ChatMessageRoleSystem,
				Content: &model.ChatCompletionMessageContent{
					StringValue: volcengine.String(buildTrackInsightSystemPrompt(req.FeedbackContext)),
				},
			},
			{
				Role: model.ChatMessageRoleUser,
				Content: &model.ChatCompletionMessageContent{
					StringValue: volcengine.String(buildTrackInsightUserPrompt(req)),
				},
			},
		},
		Thinking: &model.Thinking{
			Type: model.ThinkingTypeEnabled, // 禁用深度思考以加快响应
		},
	}

	// 打印请求体
	reqJSON, _ := json.Marshal(dReq)
	log.Info(ctx, "豆包请求体", zap.String("body", string(reqJSON)))

	// 调用豆包 API
	resp, err := p.client.CreateChatCompletion(ctx, dReq)

	// 异步保存调用流水
	var respJSON string
	if err == nil {
		rb, _ := json.Marshal(resp)
		respJSON = string(rb)
	}

	if err != nil {
		log.Error(ctx, "调用豆包 API 失败", zap.Error(err))
		return nil, err
	}

	// 打印响应体
	log.Debug(ctx, "豆包响应体", zap.String("body", respJSON))

	// 检查响应内容
	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == nil || resp.Choices[0].Message.Content.StringValue == nil {
		log.Warn(ctx, "豆包 API 返回内容为空")
		err = errors.New("豆包 API 返回内容为空")
		return nil, err
	}

	// 获取响应内容
	raw := TrimCodeFence(*resp.Choices[0].Message.Content.StringValue)

	// 解析 JSON 响应
	var result TrackAnalysisResult
	if err = json.Unmarshal([]byte(raw), &result); err != nil {
		// 如果解析失败，尝试从文本中提取 JSON 块
		if extracted := extractJSON(raw); extracted != "" {
			if err = json.Unmarshal([]byte(extracted), &result); err == nil {
				goto SUCCESS
			}
		}
		log.Error(ctx, "解析豆包响应失败", zap.Error(err), zap.String("raw", raw))
		p.SaveCallLog(ctx, req, respJSON, err, startTime, "sync")
		return nil, err
	}

SUCCESS:
	p.SaveCallLog(ctx, req, respJSON, err, startTime, "sync")
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	// 修复：将字面量 \n 转换为实际换行符
	result.LyricsTranslation = strings.ReplaceAll(result.LyricsTranslation, "\\n", "\n")

	result.LLMProvider = "doubao:" + p.model
	return &result, nil
}

// AnalyzeTrackStream 实现流式输出
func (p *DoubaoProvider) AnalyzeTrackStream(ctx context.Context, req TrackAnalysisRequest) (<-chan string, error) {
	startTime := time.Now()

	// 构建请求消息
	dReq := model.CreateChatCompletionRequest{
		Model: p.model,
		Messages: []*model.ChatCompletionMessage{
			{
				Role: model.ChatMessageRoleSystem,
				Content: &model.ChatCompletionMessageContent{
					StringValue: volcengine.String(buildTrackInsightSystemPrompt(req.FeedbackContext)),
				},
			},
			{
				Role: model.ChatMessageRoleUser,
				Content: &model.ChatCompletionMessageContent{
					StringValue: volcengine.String(buildTrackInsightUserPrompt(req)),
				},
			},
		},
		Thinking: &model.Thinking{
			Type: model.ThinkingTypeDisabled,
		},
		StreamOptions: &model.StreamOptions{
			IncludeUsage: true,
		},
	}

	// 打印请求体
	reqJSON, _ := json.Marshal(dReq)
	log.Debug(ctx, "豆包流式请求体", zap.String("body", string(reqJSON)))

	// 调用豆包流式 API
	stream, err := p.client.CreateChatCompletionStream(ctx, dReq)
	if err != nil {
		p.SaveCallLog(ctx, req, "", err, startTime, "stream")
		log.Error(ctx, "调用豆包流式 API 失败", zap.Error(err))
		return nil, err
	}

	out := make(chan string, 100)
	go func() {
		defer close(out)
		defer stream.Close()

		var fullResponse strings.Builder
		var finalErr error

		for {
			recv, err := stream.Recv()
			if err != nil {
				if err != io.EOF {
					log.Error(ctx, "接收豆包流式响应失败", zap.Error(err))
					finalErr = err
				}
				break
			}

			// 记录全量内容，用于 SaveCallLog
			rb, _ := json.Marshal(recv)
			fullResponse.WriteString(string(rb))
			fullResponse.WriteString("\n")

			// 发送流式内容
			if len(recv.Choices) > 0 {
				content := recv.Choices[0].Delta.Content
				if content != "" {
					out <- content
				}
			}
		}

		// 流结束，保存日志
		p.SaveCallLog(ctx, req, fullResponse.String(), finalErr, startTime, "stream")
	}()

	return out, nil
}
