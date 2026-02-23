package ai

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/vincentchyu/sonic-lens/config"
)

// TrackAnalysisRequest 表示发送给大模型的歌曲分析请求
type TrackAnalysisRequest struct {
	Title           string `json:"title"`
	Artist          string `json:"artist"`
	Album           string `json:"album"`
	Lyrics          string `json:"lyrics"`
	LangSource      string `json:"lang_source"`
	LangTarget      string `json:"lang_target"`
	FeedbackContext string `json:"feedback_context"`
}

// TrackAnalysisResult 表示大模型返回的结构化分析结果
type TrackAnalysisResult struct {
	LyricsTranslation string                 `json:"lyrics_translation"`
	AnalysisSummary   string                 `json:"analysis_summary"`
	AnalysisBySection map[string]string      `json:"analysis_by_section"`
	BackgroundInfo    string                 `json:"background_info"`
	EraContext        string                 `json:"era_context"`
	Metadata          map[string]interface{} `json:"metadata"`
	LLMProvider       string                 `json:"llm_provider"`
}

// LLMProvider 抽象大模型提供方
// 通过该接口，可以为 OpenAI、Gemini、Ollama、Doubao 等不同大模型实现各自的适配器。
type LLMProvider interface {
	AnalyzeTrack(ctx context.Context, req TrackAnalysisRequest) (*TrackAnalysisResult, error)
	// AnalyzeTrackStream 返回流式分析结果
	AnalyzeTrackStream(ctx context.Context, req TrackAnalysisRequest) (<-chan string, error)
}

// NewProviderFromConfig 根据全局配置选择并初始化默认的大模型 Provider。
func NewProviderFromConfig() (LLMProvider, error) {
	aiCfg := config.ConfigObj.AI
	provider := aiCfg.Provider
	if provider == "" {
		provider = "openai"
	}
	return NewProviderByName(provider)
}

// NewProviderByName 根据 Provider 名称从配置中初始化并返回对应的 Provider。
func NewProviderByName(name string) (LLMProvider, error) {
	aiCfg := config.ConfigObj.AI

	switch name {
	case "openai":
		return newOpenAIProviderFromConfigOrEnv(aiCfg.OpenAI)
	case "gemini":
		return nil, errors.New("Gemini Provider 暂未实现，请先使用 openai provider")
	case "ollama":
		return newOllamaProvider(aiCfg.Ollama)
	case "doubao":
		return newDoubaoProvider(aiCfg.Doubao)
	default:
		return nil, errors.New("不支持的 AI provider: " + name)
	}
}

// TrimCodeFence 去掉可能存在的 ```json ``` 包裹
func TrimCodeFence(s string) string {
	if len(s) == 0 {
		return s
	}
	s = strings.TrimSpace(s)
	// 粗略处理即可，避免引入额外依赖
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

// extractJSON 尝试从杂乱文本中提取第一个 JSON 块
func extractJSON(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start != -1 && end != -1 && end > start {
		return s[start : end+1]
	}
	return ""
}

// CleanLyrics 清洗歌词，去除 LRC 时间戳和元数据（如 [ar: artist], [ti: title], [00:12.34]）
func CleanLyrics(lyrics string) string {
	// 匹配 LRC 时间戳，如 [00:12.34], [00:12.345], [01:02.03]
	reTimestamp := regexp.MustCompile(`\[\d{2}:\d{2}(?:\.\d{2,3})?\]`)
	// 匹配 LRC 元数据标签，如 [ar:artist], [ti:title], [al:album], [by:creator], [offset:0]
	reTag := regexp.MustCompile(`^\[[a-z]{1,10}:.*\]$`)
	// 匹配段落标记，如 [Verse], [Chorus], [Bridge]
	reSection := regexp.MustCompile(`^\[[A-Za-z\s]+\]$`)

	lines := strings.Split(lyrics, "\n")
	var cleanedLines []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			cleanedLines = append(cleanedLines, line)
			continue
		}

		// 如果整行是元数据标签或段落标记，记录但不包含在核心歌词中（或者根据需求保留空行）
		if reTag.MatchString(trimmedLine) || reSection.MatchString(trimmedLine) {
			continue
		}

		// 去除行内的所有时间戳
		cleanedLine := reTimestamp.ReplaceAllString(line, "")
		cleanedLine = strings.TrimSpace(cleanedLine)

		if cleanedLine != "" {
			cleanedLines = append(cleanedLines, cleanedLine)
		}
	}

	return strings.TrimSpace(strings.Join(cleanedLines, "\n"))
}
