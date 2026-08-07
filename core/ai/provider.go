package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/google/uuid"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/log"
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
// 通过该接口，可以为 Gemini、Ollama、Doubao 等不同大模型实现各自的适配器。
type LLMProvider interface {
	AnalyzeTrack(ctx context.Context, req TrackAnalysisRequest) (*TrackAnalysisResult, error)
	AnalyzeAlbum(ctx context.Context, req AlbumAnalysisRequest) (*AlbumAnalysisResult, error)
	// AnalyzeTrackStream 返回流式分析结果
	// Deprecated: 流式接口已废弃，后续不再维护与优化
	AnalyzeTrackStream(ctx context.Context, req TrackAnalysisRequest) (<-chan string, error)
}

// RawChatClient 定义底层的通用 Chat 接口，用于支持多步分析流程
type RawChatClient interface {
	SendChatRequest(
		ctx context.Context, req TrackAnalysisRequest, systemPrompt, userPrompt string, schema map[string]any,
		step string,
	) (string, error)
	SaveCallLog(
		ctx context.Context, req TrackAnalysisRequest, requestJSON string, respJSON string, callErr error,
		startTime time.Time, callType string,
	)
	GetProviderName() string
	GetModelName() string
}

// executeStepWithRetry 封装单步 LLM 请求与 JSON 解析，带有退避重试机制（最多重试 maxRetries 次）
func executeStepWithRetry(
	ctx context.Context, client RawChatClient, req TrackAnalysisRequest,
	systemPrompt, userPrompt string, schema map[string]any, step string, v any,
) error {
	const maxRetries = 2
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*500) * time.Millisecond)
			log.Warn(
				ctx, "AI 单步分析请求失败，触发自动重试",
				zap.Int("attempt", attempt),
				zap.String("step", step),
				zap.Error(lastErr),
			)
		}

		respStr, err := client.SendChatRequest(ctx, req, systemPrompt, userPrompt, schema, step)
		if err != nil {
			lastErr = err
			continue
		}

		if err := parseJSONStep(ctx, respStr, v); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	return lastErr
}

// multiStepAnalyzeTrack 执行多步曲目分析调用流程，包含容错重试、优雅降级与纯音乐检测跳过
func multiStepAnalyzeTrack(ctx context.Context, client RawChatClient, req TrackAnalysisRequest) (
	*TrackAnalysisResult, error,
) {
	// 如果 context 中没有 JobID，我们生成一个临时的 UUID 用于串联多轮对话流水
	if ctx.Value(common.ContextKeyJobID) == nil {
		ctx = context.WithValue(ctx, common.ContextKeyJobID, "multi-step-"+uuid.NewString())
	}

	var lyricsTranslation string
	var appreciateAnalysis string
	isInstrumental := IsInstrumentalOrEmptyLyrics(req)
	isChinese := IsChineseLyrics(req.Lyrics)

	// Step 1: Lyrics Translation
	if isInstrumental {
		log.Info(ctx, "检测到纯音乐/无歌词曲目，自动跳过 Step 1 翻译 LLM 调用", zap.String("track", req.Title))
		lyricsTranslation = "[纯音乐 / 无歌词曲目]"
	} else if isChinese {
		log.Info(ctx, "检测到纯中文歌词，自动跳过 Step 1 翻译 LLM 调用，使用本地格式化结果", zap.String("track", req.Title))
		lyricsTranslation = FormatChineseLyricsTranslation(req.Lyrics)
	} else {
		sys1 := buildTrackInsightStep1SystemPrompt(req)
		schema1 := GetTrackInsightStep1Schema()
		usr1 := buildTrackInsightUserPrompt(req)
		var res1 struct {
			LyricsTranslation string `json:"lyrics_translation"`
		}
		if err := executeStepWithRetry(ctx, client, req, sys1, usr1, schema1, "sync_step1", &res1); err != nil {
			log.Warn(ctx, "Step 1 歌词翻译多次重试后仍失败，触发优雅降级", zap.Error(err))
			lyricsTranslation = "[歌词翻译暂时缺失]"
		} else {
			lyricsTranslation = res1.LyricsTranslation
		}
	}

	// Step 2: Appreciate Analysis
	if isInstrumental {
		log.Info(ctx, "检测到纯音乐/无歌词曲目，自动跳过 Step 2 分段赏析 LLM 调用", zap.String("track", req.Title))
		appreciateAnalysis = "[纯音乐曲目，无需逐句歌词赏析，详见下文风格与编曲分析]"
	} else {
		sys2 := buildTrackInsightStep2SystemPrompt(req, lyricsTranslation)
		schema2 := GetTrackInsightStep2Schema()
		usr2 := buildTrackInsightUserPrompt(req)
		var res2 struct {
			AppreciateAnalysis string `json:"appreciate_analysis"`
		}
		if err := executeStepWithRetry(ctx, client, req, sys2, usr2, schema2, "sync_step2", &res2); err != nil {
			log.Warn(ctx, "Step 2 分段解读多次重试后仍失败，触发优雅降级", zap.Error(err))
			appreciateAnalysis = "[分段解读暂时缺失]"
		} else {
			appreciateAnalysis = res2.AppreciateAnalysis
		}
	}

	// Step 3: Deep Analysis & Metadata
	sys3 := buildTrackInsightStep3SystemPrompt(req, lyricsTranslation, appreciateAnalysis)
	schema3 := GetTrackInsightStep3Schema()
	usr3 := buildTrackInsightUserPrompt(req)
	var res3 TrackAnalysisResult
	if err := executeStepWithRetry(ctx, client, req, sys3, usr3, schema3, "sync_step3", &res3); err != nil {
		log.Warn(ctx, "Step 3 综合评价多次重试后仍失败，尝试降级合成结果", zap.Error(err))
		if lyricsTranslation != "" || appreciateAnalysis != "" {
			res3 = TrackAnalysisResult{
				AnalysisSummary: "已完成歌词与分段解析，综合评价生成超时。",
				AnalysisBySection: map[string]string{
					"literary_analysis": "综合文学分析生成失败或超时。",
				},
			}
		} else {
			return nil, fmt.Errorf("多步分析全部步骤均失败: %w", err)
		}
	}

	// Merge all steps
	finalResult := &res3
	finalResult.LyricsTranslation = lyricsTranslation
	if finalResult.AnalysisBySection == nil {
		finalResult.AnalysisBySection = make(map[string]string)
	}
	finalResult.AnalysisBySection["appreciate_analysis"] = appreciateAnalysis

	// Add Provider Name
	finalResult.LLMProvider = client.GetProviderName() + ":" + client.GetModelName()

	return finalResult, nil
}

// parseJSONStep 解析单步返回的 JSON，并在失败时提取并记录日志
func parseJSONStep(ctx context.Context, raw string, v any) error {
	raw = TrimCodeFence(raw)
	err := json.Unmarshal([]byte(raw), v)
	if err == nil {
		return nil
	}
	extracted := extractJSON(raw)
	if extracted != "" {
		if err2 := json.Unmarshal([]byte(extracted), v); err2 == nil {
			return nil
		}
	}
	log.Error(
		ctx, "JSON parsing failed in step", zap.Error(err), zap.Int("raw_len", len(raw)),
		zap.String("raw_tail", tailString(raw, 200)),
	)
	return err
}

func tailString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

// 公共模型参数（适用于音眸分析的 JSON 结构化输出场景）
const (
	DefaultInsightTemperature = 0.2
	DefaultThinkingBudget     = 8192
)

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
	if start != -1 && end != -1 && end >= start {
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
