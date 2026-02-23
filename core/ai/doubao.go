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

	// 构建请求消息
	dReq := model.CreateChatCompletionRequest{
		Model: p.model,
		Messages: []*model.ChatCompletionMessage{
			{
				Role: model.ChatMessageRoleSystem,
				Content: &model.ChatCompletionMessageContent{
					StringValue: volcengine.String(buildSystemPrompt(req.FeedbackContext)),
				},
			},
			{
				Role: model.ChatMessageRoleUser,
				Content: &model.ChatCompletionMessageContent{
					StringValue: volcengine.String(buildUserPrompt(req)),
				},
			},
		},
		Thinking: &model.Thinking{
			Type: model.ThinkingTypeEnabled, // 禁用深度思考以加快响应
		},
	}

	// 打印请求体
	reqJSON, _ := json.Marshal(dReq)
	log.Debug(ctx, "豆包请求体", zap.String("body", string(reqJSON)))

	// 调用豆包 API
	resp, err := p.client.CreateChatCompletion(ctx, dReq)

	// 异步保存调用流水
	var respJSON string
	if err == nil {
		rb, _ := json.Marshal(resp)
		respJSON = string(rb)
	}
	p.SaveCallLog(ctx, req, respJSON, err, startTime, "sync")

	if err != nil {
		log.Error(ctx, "调用豆包 API 失败", zap.Error(err))
		return nil, err
	}

	// 打印响应体
	log.Debug(ctx, "豆包响应体", zap.String("body", respJSON))

	// 检查响应内容
	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == nil || resp.Choices[0].Message.Content.StringValue == nil {
		log.Error(ctx, "豆包 API 返回内容为空")
		return nil, errors.New("豆包 API 返回内容为空")
	}

	// 获取响应内容
	raw := TrimCodeFence(*resp.Choices[0].Message.Content.StringValue)

	// 解析 JSON 响应
	var result TrackAnalysisResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		// 如果解析失败，尝试从文本中提取 JSON 块
		if extracted := extractJSON(raw); extracted != "" {
			if err := json.Unmarshal([]byte(extracted), &result); err == nil {
				goto SUCCESS
			}
		}
		log.Error(ctx, "解析豆包响应失败", zap.Error(err), zap.String("raw", raw))
		return nil, err
	}

SUCCESS:
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
					StringValue: volcengine.String(buildSystemPrompt(req.FeedbackContext)),
				},
			},
			{
				Role: model.ChatMessageRoleUser,
				Content: &model.ChatCompletionMessageContent{
					StringValue: volcengine.String(buildUserPrompt(req)),
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

// buildSystemPrompt 提供与 Ollama 一致的系统提示词
func buildSystemPrompt(feedbackContext string) string {
	feedbackSection := ""
	if feedbackContext != "" {
		feedbackSection = `
═══════════════════════════════════════════
【重要：历史用户反馈】
═══════════════════════════════════════════
` + feedbackContext + `

⚠️ 请在分析时特别注意避免重复上述问题，确保本次分析质量更高。

`
	}

	return `你是一位多维度音乐分析专家，精通文学翻译、乐评分析、文化史研究。请按以下四个角色顺序，深度分析这首歌曲：
` + feedbackSection + `

═══════════════════════════════════════════
【角色一：文学翻译家(信雅达)】
═══════════════════════════════════════════

任务 1.1 - 双语翻译：
• 原文是【非中文】：严格执行"原文+翻译"逐行对照格式
  示例：
  Hello darkness, my old friend
  你好黑暗，我的老友
• 原文是【纯中文】：无需翻译，直接输出原文，不添加任何解释

任务 1.2 - 文学分析：
• 解析歌词中的核心意象、隐喻和修辞手法
• 分析叙事结构和情感递进
• 说明歌词的语言风格（诗意/叙事/口语等）
• 尤其对中文歌词，和外语原文进行详细的解读，不少于300字，重点关注歌词含义、隐喻、立意等，帮助用户解读深刻含义
• 针对每一段进行赏析和解读，文本解析格式：每一段原文（解读）
• ⚠️ 重要：分段解读(appreciate_analysis)必须包含完整的歌词内容，不要遗漏任何歌词行
• 格式要求：
  - 每一句/每一段歌词原文必须出现在解读中
  - 原文后紧跟（解读），解读前后括号要换行
	示例：
	就在一瞬间
    就在一瞬间 握紧我矛盾密布的手
	（解读：此处分析...")
	是谁来自山川湖海
	却囿于昼夜厨房与爱
	（解读：此处分析...")

═══════════════════════════════════════════
【角色二：乐评人】
═══════════════════════════════════════════

任务 2.1 - 音乐风格：
• 判断歌曲的音乐流派和风格特征
• 分析编曲的层次感和乐器运用特点
• 评价歌曲的情感基调和氛围营造

任务 2.2 - 演唱表现：
• 分析歌手的演唱技巧和情感表达
• 评价声音特质与歌曲主题的契合度
• 说明歌曲的记忆点（hook）设计

═══════════════════════════════════════════
【角色三：文化史学家】
═══════════════════════════════════════════

任务 3.1 - 创作背景：
• 说明这首歌或其所在专辑的大致创作背景
• 分析歌手/乐队的创作动机和当时状态
• 提及专辑在艺术家生涯中的位置

任务 3.2 - 时代语境：
• 说明歌曲所处时代的大致文化/社会语境
• 分析歌曲是否反映了当时的社会议题或思潮
• 如果信息不足，明确说明"背景信息有限"

═══════════════════════════════════════════
【角色四：综合分析师】
═══════════════════════════════════════════

任务 4 - 整体评价：
• 总结这首歌的核心价值和艺术成就
• 提炼 2-3 个最突出的亮点
• 给出欣赏这首歌的建议视角

═══════════════════════════════════════════
【输出格式 - 严格遵守】
═══════════════════════════════════════════

必须输出以下 JSON 结构，不要包含任何 Markdown 或自然语言：

{
  "lyrics_translation": "逐行双语对照结果（非中文歌曲）或原文（中文歌曲）",
  "analysis_summary": "综合分析师的整体评价（200-300字）",
  "analysis_by_section": {
    "literary_analysis": "文学翻译家的深度解读（意象、修辞、叙事）",
    "appreciate_analysis": "分句进行赏析和解读",
    "musical_analysis": "乐评人的专业评价（风格、编曲、演唱）",
    "cultural_context": "文化史学家的背景与时代分析",
    "translation_notes": "翻译难点说明或语言特色分析"
  },
  "background_info": "创作背景信息",
  "era_context": "时代文化语境",
  "metadata": {
    "analysis_depth": "深度分析",
    "model_size": "Doubao-Pro"
  }
}

═══════════════════════════════════════════
【重要约束】
═══════════════════════════════════════════

1. 只能输出 JSON，不要 Markdown 代码块标记
2. 所有字符串使用 UTF-8 编码
3. 如信息不足，在相关字段填入"背景信息有限"
4. 优先保证 lyrics_translation 和 analysis_summary 的完整性
5. 使用 \n 表示换行，不要在 JSON 中使用实际换行符
6. 不要输出任何思考过程，只输出最终 JSON

请根据以下歌曲信息进行深度分析：`
}

// buildUserPrompt 格式化用户输入数据
func buildUserPrompt(req TrackAnalysisRequest) string {
	userPromptData := map[string]interface{}{
		"title":       req.Title,
		"artist":      req.Artist,
		"album":       req.Album,
		"lyrics":      req.Lyrics,
		"lang_source": req.LangSource,
		"lang_target": req.LangTarget,
	}
	userPromptBytes, _ := json.Marshal(userPromptData)

	jsonSchema := `{
  "type": "object",
  "properties": {
    "lyrics_translation": { "type": "string" },
    "analysis_summary": { "type": "string" },
    "analysis_by_section": {
      "type": "object",
      "additionalProperties": { "type": "string" }
    },
    "background_info": { "type": "string" },
    "era_context": { "type": "string" },
    "metadata": { "type": "object" }
  },
  "required": ["lyrics_translation", "analysis_summary"],
  "additionalProperties": false
}`

	return "系统提示：\n" + buildSystemPrompt("") + "\n\n输入数据（JSON）：\n" +
		string(userPromptBytes) +
		"\n\n请严格按照如下 JSON Schema 输出解析结果：" +
		jsonSchema +
		"\n\n注意：只能输出 JSON，不要 Markdown 或自然语言解释。"
}
