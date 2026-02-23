package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/log"
)

// --- Ollama Provider ---

// OllamaProvider 使用本地 Ollama 服务实现 LLMProvider
type OllamaProvider struct {
	BaseProvider
	host       string
	model      string
	httpClient *http.Client
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
	return &OllamaProvider{
		BaseProvider: BaseProvider{
			ProviderName: "ollama",
			ModelName:    model,
		},
		host:  host,
		model: model,
		httpClient: &http.Client{
			Timeout: 30 * time.Minute, // 本地模型可能较慢，给足超时
		},
	}, nil
}

// ollamaGenerateRequest 与 Ollama /api/generate 请求体一致
type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	// / Format  string                 `json:"format,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"` // 可选参数，如 temperature
	// Context []int                  `json:"context,omitempty"` // 用于保持上下文的 token 列表
}

// ollamaGenerateResponse 与 Ollama /api/generate 响应体一致
type ollamaGenerateResponse struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
	Thinking           string `json:"thinking"` // 模型思考过程
	Done               bool   `json:"done"`
	DoneReason         string `json:"done_reason"`
	Context            []int  `json:"context,omitempty"`
	TotalDuration      int64  `json:"total_duration"`
	LoadDuration       int64  `json:"load_duration"`
	PromptEvalCount    int    `json:"prompt_eval_count"`
	PromptEvalDuration int64  `json:"prompt_eval_duration"`
	EvalCount          int    `json:"eval_count"`
	EvalDuration       int64  `json:"eval_duration"`
}

// AnalyzeTrack 调用本地 Ollama 接口，对歌词进行翻译和深度解析
func (p *OllamaProvider) AnalyzeTrack(
	ctx context.Context, req TrackAnalysisRequest,
) (*TrackAnalysisResult, error) {
	startTime := time.Now()

	payload := ollamaGenerateRequest{
		Model:  p.model,
		Prompt: p.buildPrompt(req),
		Stream: false, //
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	apiURL := p.host
	if !strings.HasSuffix(apiURL, "/api/generate") && !strings.HasSuffix(apiURL, "/api/chat") {
		apiURL = strings.TrimSuffix(apiURL, "/") + "/api/generate"
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		p.SaveCallLog(ctx, req, "", err, startTime, "sync")
		log.Warn(
			ctx, "创建 Ollama 请求失败",
			zap.Error(err),
		)
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	log.Info(ctx, "Ollama 请求详情", zap.String("url", apiURL), zap.String("payload", string(body)))
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		p.SaveCallLog(ctx, req, "", err, startTime, "sync")
		log.Warn(
			ctx, "调用 Ollama 接口失败",
			zap.Error(err),
		)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = errors.New("Ollama 接口返回错误: " + resp.Status)
		p.SaveCallLog(ctx, req, "", err, startTime, "sync")
		log.Warn(
			ctx, "调用大模型进行歌词解析失败",
			zap.Error(err),
		)
		return nil, err
	}

	var fullResponse strings.Builder
	var fullContent strings.Builder
	// 使用 TeeReader 来捕获原始响应流
	decoder := json.NewDecoder(resp.Body)
	for {
		var chunk ollamaGenerateResponse
		// 这里无法简单用 TeeReader 记录全量原始 JSON，因为 Decoder 会消费。
		// 但由于是内部结构，我们可以重新 Marshal chunk 记录。
		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			p.SaveCallLog(ctx, req, fullResponse.String(), err, startTime, "sync")
			log.Warn(ctx, "Ollama 解析响应片段失败", zap.Error(err))
			return nil, err
		}

		cb, _ := json.Marshal(chunk)
		fullResponse.WriteString(string(cb))
		fullResponse.WriteString("\n")

		if chunk.Response != "" {
			fullContent.WriteString(chunk.Response)
		}
		if chunk.Done {
			break
		}
	}

	// 记录成功日志
	p.SaveCallLog(ctx, req, fullResponse.String(), nil, startTime, "sync")

	raw := TrimCodeFence(fullContent.String())
	var result TrackAnalysisResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		// 如果解析失败，尝试从文本中提取 JSON 块
		if extracted := extractJSON(raw); extracted != "" {
			if err := json.Unmarshal([]byte(extracted), &result); err == nil {
				goto SUCCESS
			}
		}
		return nil, err
	}

SUCCESS:
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	// 修复：将字面量 \n 转换为实际换行符
	result.LyricsTranslation = strings.ReplaceAll(result.LyricsTranslation, "\\n", "\n")

	result.LLMProvider = "ollama:" + p.model
	return &result, nil
}

// buildPrompt 将请求封装为 Prompt（四角色分层分析，适配 20B 本地模型）
func (p *OllamaProvider) buildPrompt(req TrackAnalysisRequest) string {
	feedbackSection := ""
	if req.FeedbackContext != "" {
		feedbackSection = `
═══════════════════════════════════════════
【重要：历史用户反馈】
═══════════════════════════════════════════
` + req.FeedbackContext + `

⚠️ 请在分析时特别注意避免重复上述问题，确保本次分析质量更高。

`
	}

	systemPrompt := `你是一位多维度音乐分析专家，精通文学翻译、乐评分析、文化史研究。请按以下四个角色顺序，深度分析这首歌曲：
` + feedbackSection + `
═══════════════════════════════════════════
【角色一：文学翻译家】
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
    "model_size": "20B"
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

	return "系统提示：\n" + systemPrompt + "\n\n输入数据（JSON）：\n" +
		string(userPromptBytes) +
		"\n\n请严格按照如下 JSON Schema 输出解析结果：" +
		jsonSchema +
		"\n\n注意：只能输出 JSON，不要 Markdown 或自然语言解释。"
}

// AnalyzeTrackStream 实现流式输出
func (p *OllamaProvider) AnalyzeTrackStream(ctx context.Context, req TrackAnalysisRequest) (<-chan string, error) {
	startTime := time.Now()

	payload := ollamaGenerateRequest{
		Model:  p.model,
		Prompt: p.buildPrompt(req),
		Stream: true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// 智能处理 URL
	apiURL := p.host
	if !strings.HasSuffix(apiURL, "/api/generate") && !strings.HasSuffix(apiURL, "/api/chat") {
		apiURL = strings.TrimSuffix(apiURL, "/") + "/api/generate"
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		p.SaveCallLog(ctx, req, "", err, startTime, "stream")
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	log.Info(ctx, "Ollama 流式请求详情", zap.String("url", apiURL), zap.String("payload", string(body)))

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		p.SaveCallLog(ctx, req, "", err, startTime, "stream")
		return nil, err
	}

	out := make(chan string, 100) // 缓冲区加大，应对大量数据推送
	go func() {
		defer resp.Body.Close()
		defer close(out)

		var fullResponse strings.Builder
		var finalErr error

		if resp.StatusCode != http.StatusOK {
			finalErr = errors.New("Ollama 接口返回错误: " + resp.Status)
			p.SaveCallLog(ctx, req, "", finalErr, startTime, "stream")
			log.Error(ctx, "Ollama 请求失败", zap.Int("status", resp.StatusCode))
			return
		}

		// 使用 json.Decoder 的流式读取能力直接解析 ndjson
		decoder := json.NewDecoder(resp.Body)
		for {
			var chunk ollamaGenerateResponse
			if err := decoder.Decode(&chunk); err != nil {
				if err != io.EOF {
					log.Error(ctx, "解析 Ollama ndjson 响应失败", zap.Error(err))
					finalErr = err
				}
				break
			}

			// 记录全量内容，用于 SaveCallLog
			rb, _ := json.Marshal(chunk)
			fullResponse.WriteString(string(rb))
			fullResponse.WriteString("\n")

			// 依次处理思考内容和回复内容
			if chunk.Thinking != "" {
				out <- chunk.Thinking
			}
			if chunk.Response != "" {
				out <- chunk.Response
			}

			if chunk.Done {
				break
			}
		}

		// 流结束，保存日志
		p.SaveCallLog(ctx, req, fullResponse.String(), finalErr, startTime, "stream")
	}()

	return out, nil
}
