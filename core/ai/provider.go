package ai

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/config"
)

// TrackAnalysisRequest 表示发送给大模型的歌曲分析请求
type TrackAnalysisRequest struct {
	Title             string `json:"title"`
	Artist            string `json:"artist"`
	Album             string `json:"album"`
	Lyrics            string `json:"lyrics"`
	LangSource        string `json:"lang_source"`
	LangTarget        string `json:"lang_target"`
	FeedbackContext   string `json:"feedback_context"`
	RequestedProvider string `json:"-"`
	RequestedModel    string `json:"-"`
}

// AlbumTrackContext 表示专辑分析时每首曲目的聚合上下文。
type AlbumTrackContext struct {
	TrackID          int64             `json:"track_id"`
	DiscNumber       int8              `json:"disc_number"`
	TrackNumber      int8              `json:"track_number"`
	Title            string            `json:"title"`
	InsightID        int64             `json:"insight_id"`
	InsightScore     int               `json:"insight_score"`
	AnalysisSummary  string            `json:"analysis_summary"`
	BackgroundInfo   string            `json:"background_info"`
	EraContext       string            `json:"era_context"`
	AnalysisSections map[string]string `json:"analysis_sections"`
}

// AlbumAnalysisRequest 表示发送给大模型的专辑分析请求。
type AlbumAnalysisRequest struct {
	AlbumID           int64               `json:"album_id"`
	Artist            string              `json:"artist"`
	Album             string              `json:"album"`
	ReleaseDate       string              `json:"release_date"`
	Genre             string              `json:"genre"`
	TrackCount        int                 `json:"track_count"`
	AnalyzedTracks    int                 `json:"analyzed_tracks"`
	TrackContexts     []AlbumTrackContext `json:"track_contexts"`
	FeedbackContext   string              `json:"feedback_context"`
	RequestedProvider string              `json:"-"`
	RequestedModel    string              `json:"-"`
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

// AlbumAnalysisResult 表示大模型返回的专辑结构化分析结果。
type AlbumAnalysisResult struct {
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
	AnalyzeAlbum(ctx context.Context, req AlbumAnalysisRequest) (*AlbumAnalysisResult, error)
	// AnalyzeTrackStream 返回流式分析结果
	AnalyzeTrackStream(ctx context.Context, req TrackAnalysisRequest) (<-chan string, error)
}

// NewProviderFromConfig 根据全局配置选择并初始化默认的大模型 Provider。
func NewProviderFromConfig() (LLMProvider, error) {
	aiCfg := config.ConfigObj.AI
	selection, err := ResolveProviderSelection(ProviderSelectionInput{Provider: aiCfg.Provider})
	if err != nil {
		return nil, err
	}
	return NewProviderBySelection(selection.Platform, selection.Model)
}

// NewProviderByName 根据 Provider 名称从配置中初始化并返回对应的 Provider。
func NewProviderByName(name string) (LLMProvider, error) {
	platform := common.ParseAIModelPlatform(name)
	if !platform.IsValid() {
		return nil, errors.New("不支持的 AI provider: " + name)
	}
	factory, err := getProviderFactory(platform)
	if err != nil {
		return nil, err
	}
	return factory.Create("")
}

// NewProviderBySelection 根据平台和模型创建对应的 Provider。
func NewProviderBySelection(platform common.AIModelPlatform, model string) (LLMProvider, error) {
	factory, err := getProviderFactory(platform)
	if err != nil {
		return nil, err
	}
	return factory.Create(model)
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

// repairPrematureTopLevelObjectClosure 修复模型偶发返回的“顶层对象提前闭合”问题。
// 典型场景是输出形如 `{"a":1},"b":2}`，这里会移除多余的那个 `}`。
func repairPrematureTopLevelObjectClosure(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	var builder strings.Builder
	builder.Grow(len(s))

	depth := 0
	inString := false
	escaped := false
	changed := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inString {
			builder.WriteByte(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
			builder.WriteByte(ch)
		case '{':
			depth++
			builder.WriteByte(ch)
		case '}':
			if depth == 1 && nextNonSpaceByte(s, i+1) == ',' {
				changed = true
				continue
			}
			if depth > 0 {
				depth--
			}
			builder.WriteByte(ch)
		default:
			builder.WriteByte(ch)
		}
	}

	if !changed {
		return s
	}
	return builder.String()
}

func nextNonSpaceByte(s string, start int) byte {
	for i := start; i < len(s); i++ {
		switch s[i] {
		case ' ', '\n', '\r', '\t':
			continue
		default:
			return s[i]
		}
	}
	return 0
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
