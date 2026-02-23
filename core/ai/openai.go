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

	"github.com/vincentchyu/sonic-lens/config"
)

// OpenAIProvider 使用 OpenAI Chat Completions 接口实现 LLMProvider
type OpenAIProvider struct {
	BaseProvider
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// newOpenAIProviderFromConfigOrEnv 优先使用 config.yaml 中的 openai 配置，必要字段缺失时回退到环境变量。
// 支持的配置来源优先级：
// 1. config.ai.openai.apiKey / baseUrl / model
// 2. 环境变量：OPENAI_API_KEY / OPENAI_BASE_URL / OPENAI_MODEL
func newOpenAIProviderFromConfigOrEnv(openAIConfig config.OpenAIConfig) (LLMProvider, error) {
	apiKey := openAIConfig.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		return nil, errors.New("未配置 OpenAI API Key（config.ai.openai.apiKey 或环境变量 OPENAI_API_KEY）")
	}

	baseURL := openAIConfig.BaseURL
	if baseURL == "" {
		baseURL = os.Getenv("OPENAI_BASE_URL")
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}

	model := openAIConfig.Model
	if model == "" {
		model = os.Getenv("OPENAI_MODEL")
	}
	if model == "" {
		model = "gpt-4.1-mini"
	}

	return &OpenAIProvider{
		BaseProvider: BaseProvider{
			ProviderName: "openai",
			ModelName:    model,
		},
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 30 * time.Minute,
		},
	}, nil
}

type openAIChatRequest struct {
	Model    string              `json:"model"`
	Messages []openAIChatMessage `json:"messages"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// AnalyzeTrack 调用 OpenAI 接口，对歌词进行翻译和深度解析
func (p *OpenAIProvider) AnalyzeTrack(
	ctx context.Context, req TrackAnalysisRequest,
) (*TrackAnalysisResult, error) {
	startTime := time.Now()
	systemPrompt := `你是一名专业的乐评人和文学翻译，擅长理解歌词意象、叙事与时代语境。
请根据给出的歌曲信息和原文歌词，完成以下任务：
1. 语言识别与处理：
   - 如果原文是【非中文】（如英文、日文等）：必须严格执行“原文+翻译”的双语对照格式。每一句原文下面必须紧跟一句中文翻译。
     格式示例：
     Hello darkness, my old friend
     你好黑暗，我的老友
     I've come to talk with you again
     我又来找你聊天了
   - 如果原文是【纯中文】：无需翻译，直接输出原文。不要输出任何重复翻译或解释。
2. 给出一段整体性的歌词解读，说明这首歌在表达什么情绪、主题或故事。
3. 按段落或重要意象，对歌词做更细致的解析（可以按“段1/段2/副歌/桥段”等维度分块）。
4. 结合你已有的知识，说明这首歌或其所在专辑的大致创作背景（如果你知道的话），以及它所处时代的大致文化/社会语境；如果信息不足，请明确说明“背景信息有限”即可。
5. 输出严格的 JSON，不要包含多余文本。`

	userPrompt := map[string]interface{}{
		"title":       req.Title,
		"artist":      req.Artist,
		"album":       req.Album,
		"lyrics":      req.Lyrics,
		"lang_source": req.LangSource,
		"lang_target": req.LangTarget,
	}
	userPromptBytes, _ := json.Marshal(userPrompt)

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

	content := "下面是本次分析的输入数据（JSON）：\n" +
		string(userPromptBytes) +
		"\n\n请根据系统提示完成分析，并仅输出一个 JSON，对应如下 JSON Schema：" +
		jsonSchema +
		"\n\n注意：\n- 只能输出 JSON，不要输出 Markdown 或自然语言解释。\n- 所有字符串请使用 UTF-8 编码。"

	payload := openAIChatRequest{
		Model: p.model,
		Messages: []openAIChatMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: content,
			},
		},
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

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		p.SaveCallLog(ctx, req, "", err, startTime, "sync")
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = errors.New("调用 OpenAI 接口失败，状态码: " + resp.Status)
		p.SaveCallLog(ctx, req, "", err, startTime, "sync")
		return nil, err
	}

	var chatResp openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		p.SaveCallLog(ctx, req, "", err, startTime, "sync")
		return nil, err
	}
	if len(chatResp.Choices) == 0 {
		err = errors.New("OpenAI 返回结果为空")
		p.SaveCallLog(ctx, req, "", err, startTime, "sync")
		return nil, err
	}

	// 记录原始响应
	chatRespBytes, _ := json.Marshal(chatResp)
	p.SaveCallLog(ctx, req, string(chatRespBytes), nil, startTime, "sync")

	raw := chatResp.Choices[0].Message.Content
	// 有些模型会在内容外面包裹 ```json ```，这里做一下简单清理
	raw = TrimCodeFence(raw)

	var result TrackAnalysisResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, err
	}
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	// 修复：将字面量 \n 转换为实际换行符
	result.LyricsTranslation = strings.ReplaceAll(result.LyricsTranslation, "\\n", "\n")

	result.LLMProvider = "openai:" + p.model
	return &result, nil
}

func (p *OpenAIProvider) AnalyzeTrackStream(ctx context.Context, req TrackAnalysisRequest) (<-chan string, error) {
	// 暂不实现 OpenAI 的流式输出，可抛出错误或是直接聚合后返回
	err := errors.New("OpenAI Provider 暂未实现流式接口")
	p.SaveCallLog(ctx, req, "", err, time.Now(), "stream")
	return nil, err
}
