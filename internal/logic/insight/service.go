package insight

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/ai"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/core/lyrics"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

// Service 定义歌词解析与深度分析服务接口
type Service interface {
	// GetOrCreateInsight 获取或创建某首歌的解析结果，第二个返回值表示是否命中缓存
	GetOrCreateInsight(
		ctx context.Context, artist, album, track string, force bool, modelType string,
	) ([]*model.TrackInsight, bool, error)
	// RecordFeedback 记录用户点赞/点踩反馈
	RecordFeedback(ctx context.Context, insightID int64, score int, comment string) error
	// GetOrCreateInsightStream 获取大模型流式解析结果，第二个返回值表示是否命中缓存
	GetOrCreateInsightStream(
		ctx context.Context, artist, album, track string, force bool, modelType string,
	) (<-chan string, bool, error)
	// GetAvailableAIProviders 获取当前系统可用的 AI 服务模型
	GetAvailableAIProviders() []string
	// GetInsightOnly 仅从数据库获取已有的解析结果，不触发 AI 分析
	GetInsightOnly(ctx context.Context, artist, album, track string) ([]*InsightWithScore, error)
}

type serviceImpl struct {
	llmCache  map[string]ai.LLMProvider
	providers []lyrics.LyricsProvider
}

type InsightWithScore struct {
	*model.TrackInsight
	TotalScore int `json:"total_score"`
}

// NewService 创建 Insight Service 实例
func NewService() (Service, error) {
	return &serviceImpl{
		llmCache: make(map[string]ai.LLMProvider),
		providers: []lyrics.LyricsProvider{
			lyrics.NewLrcAPIProvider(),
			lyrics.NewMusixmatchProvider(),
		},
	}, nil
}

// getLLMProvider 获取指定的 Provider，如果不存在则实例化并缓存
func (s *serviceImpl) getLLMProvider(modelType string) (ai.LLMProvider, error) {
	if modelType == "" {
		// 默认使用配置中的默认 Provider
		return ai.NewProviderFromConfig()
	}
	if p, ok := s.llmCache[modelType]; ok {
		return p, nil
	}
	p, err := ai.NewProviderByName(modelType)
	if err != nil {
		return nil, err
	}
	s.llmCache[modelType] = p
	return p, nil
}

// GetAvailableAIProviders 获取当前配置中可用的 AI 模型列表
func (s *serviceImpl) GetAvailableAIProviders() []string {
	return config.ConfigObj.AI.GetAvailableProviders()
}

// GetInsightOnly 仅从数据库获取已有的解析结果
func (s *serviceImpl) GetInsightOnly(ctx context.Context, artist, album, track string) ([]*InsightWithScore, error) {
	artist = strings.TrimSpace(artist)
	album = strings.TrimSpace(album)
	track = strings.TrimSpace(track)

	insights, err := model.GetTrackInsights(ctx, artist, album, track)
	if err != nil {
		return nil, err
	}

	ids := make([]int64, len(insights))
	for i, ins := range insights {
		ids[i] = ins.ID
	}

	scoreMap, err := model.GetInsightsTotalScores(ctx, ids)
	if err != nil {
		return nil, err
	}

	result := make([]*InsightWithScore, len(insights))
	for i, ins := range insights {
		result[i] = &InsightWithScore{
			TrackInsight: ins,
			TotalScore:   scoreMap[ins.ID],
		}
	}
	return result, nil
}

// GetOrCreateInsight 获取或创建某首歌的解析结果
func (s *serviceImpl) GetOrCreateInsight(
	ctx context.Context, artist, album, track string, force bool, modelType string,
) ([]*model.TrackInsight, bool, error) {
	artist = strings.TrimSpace(artist)
	album = strings.TrimSpace(album)
	track = strings.TrimSpace(track)

	if artist == "" || album == "" || track == "" {
		return nil, false, errors.New("artist, album, track 不能为空")
	}

	// 先尝试从数据库中获取已存在的解析
	insights, err := model.GetTrackInsights(ctx, artist, album, track)
	if err == nil && len(insights) > 0 && !force {
		// 命中缓存，更新最近一条的使用时间
		insights[0].LastUsedAt = time.Now()
		_ = model.UpdateTrackInsight(ctx, insights[0])
		return insights, true, nil
	}
	// 如果强制刷新或没找到且出错不是 NotFound，返回错误
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	// 准备歌词
	lyrics, err := s.getOrFetchLyrics(ctx, artist, album, track)
	if err != nil {
		log.Warn(
			ctx, "获取歌词失败，将使用空歌词进行分析",
			zap.String("artist", artist),
			zap.String("track", track),
			zap.Error(err),
		)
	}

	// 查询历史差评反馈，用于改进分析质量
	feedbackCtx := ""
	if negativeFeedbacks, fbErr := model.GetNegativeFeedbacksByTrack(
		ctx, artist, album, track,
	); fbErr == nil && len(negativeFeedbacks) > 0 {
		var feedbackComments []string
		for _, fb := range negativeFeedbacks {
			if fb.Comment != "" {
				feedbackComments = append(feedbackComments, fb.Comment)
			}
		}
		if len(feedbackComments) > 0 {
			feedbackCtx = "用户对之前分析的主要反馈意见（请避免重复这些问题）：\n- " + strings.Join(
				feedbackComments, "\n- ",
			)
			log.Info(ctx, "检测到历史差评反馈，将注入到分析上下文", zap.String("feedback", feedbackCtx))
		}
	}

	// 调用大模型进行翻译与解析
	llmReq := ai.TrackAnalysisRequest{
		Title:           track,
		Artist:          artist,
		Album:           album,
		Lyrics:          ai.CleanLyrics(lyrics),
		LangSource:      "auto",
		LangTarget:      "zh-CN",
		FeedbackContext: feedbackCtx,
	}

	llm, err := s.getLLMProvider(modelType)
	if err != nil {
		return nil, false, err
	}
	llmResp, err := llm.AnalyzeTrack(ctx, llmReq)
	if err != nil {
		log.Error(
			ctx, "调用大模型进行歌词解析失败",
			zap.String("artist", artist),
			zap.String("album", album),
			zap.String("track", track),
			zap.Error(err),
		)
		return nil, false, err
	}

	// 将结果落库
	newInsight := &model.TrackInsight{
		Artist:            artist,
		Album:             album,
		Track:             track,
		LyricsTranslation: llmResp.LyricsTranslation,
		AnalysisSummary:   llmResp.AnalysisSummary,
		BackgroundInfo:    llmResp.BackgroundInfo,
		EraContext:        llmResp.EraContext,
		LLMProvider:       llmResp.LLMProvider,
		LangSource:        llmReq.LangSource,
		LangTarget:        llmReq.LangTarget,
		LastUsedAt:        time.Now(),
	}

	if llmResp.AnalysisBySection != nil {
		if serialized, serErr := json.Marshal(llmResp.AnalysisBySection); serErr == nil {
			newInsight.AnalysisBySection = string(serialized)
		}
	}
	if llmResp.Metadata != nil {
		if serialized, serErr := json.Marshal(llmResp.Metadata); serErr == nil {
			newInsight.Metadata = string(serialized)
		}
	}

	if err := model.CreateTrackInsight(ctx, newInsight); err != nil {
		return nil, false, err
	}

	// 重新获取完整列表
	insights, err = model.GetTrackInsights(ctx, artist, album, track)
	if err != nil {
		// 降级：只返回新创建的
		return []*model.TrackInsight{newInsight}, false, nil
	}

	return insights, false, nil
}

// GetOrCreateInsightStream 获取流式解析结果
func (s *serviceImpl) GetOrCreateInsightStream(
	ctx context.Context, artist, album, track string, force bool, modelType string,
) (<-chan string, bool, error) {
	artist = strings.TrimSpace(artist)
	album = strings.TrimSpace(album)
	track = strings.TrimSpace(track)

	// 先尝试从数据库中获取已存在的解析
	insights, err := model.GetTrackInsights(ctx, artist, album, track)
	if err == nil && len(insights) > 0 && !force {
		// 命中缓存，模拟流式输出，返回整个列表的 JSON
		out := make(chan string, 1)
		b, _ := json.Marshal(insights)
		// 转换为简化 JSON 格式 (这里直接返回完整对象的 JSON array 也可以，前端需要适配)
		// 为了保持一致性，我们这里直接返回 insights 的 JSON
		out <- string(b)
		close(out)
		return out, true, nil
	}

	// 准备歌词
	lyrics, err := s.getOrFetchLyrics(ctx, artist, album, track)
	if err != nil {
		log.Warn(ctx, "获取歌词失败，流式分析使用空歌词", zap.Error(err))
	}

	// 查询历史差评反馈，用于改进分析质量
	feedbackCtx := ""
	if negativeFeedbacks, fbErr := model.GetNegativeFeedbacksByTrack(
		ctx, artist, album, track,
	); fbErr == nil && len(negativeFeedbacks) > 0 {
		var feedbackComments []string
		for _, fb := range negativeFeedbacks {
			if fb.Comment != "" {
				feedbackComments = append(feedbackComments, fb.Comment)
			}
		}
		if len(feedbackComments) > 0 {
			feedbackCtx = "用户对之前分析的主要反馈意见（请避免重复这些问题）：\n- " + strings.Join(
				feedbackComments, "\n- ",
			)
			log.Info(ctx, "检测到历史差评反馈，将注入到流式分析上下文", zap.String("feedback", feedbackCtx))
		}
	}

	llmReq := ai.TrackAnalysisRequest{
		Title:           track,
		Artist:          artist,
		Album:           album,
		Lyrics:          ai.CleanLyrics(lyrics),
		LangSource:      "auto",
		LangTarget:      "zh-CN",
		FeedbackContext: feedbackCtx,
	}

	llm, err := s.getLLMProvider(modelType)
	if err != nil {
		return nil, false, err
	}
	ch, err := llm.AnalyzeTrackStream(ctx, llmReq)
	if err != nil {
		return nil, false, err
	}

	// 我们需要一个中间层来聚合结果并存入数据库，同时转发给前端
	out := make(chan string, 10)
	// 使用一个新的 context，避免主请求取消导致流中断
	// 但实际上 SSE 应该随主请求结束而结束，这里的 out 发送应该监听 ctx.Done()
	go func() {
		defer close(out)
		var fullContent strings.Builder
		for {
			select {
			case <-ctx.Done():
				return
			case chunk, ok := <-ch:
				if !ok {
					// 流自然结束，执行存入数据库逻辑
					go func(content string) {
						if content == "" {
							return
						}
						raw := ai.TrimCodeFence(content)
						var llmResp ai.TrackAnalysisResult
						if err := json.Unmarshal([]byte(raw), &llmResp); err != nil {
							log.Error(context.Background(), "流式结果解析 JSON 失败，无法存入缓存", zap.Error(err))
							return
						}
						// 修复：将字面量 \n 转换为实际换行符，避免前端显示异常
						llmResp.LyricsTranslation = strings.ReplaceAll(llmResp.LyricsTranslation, "\\n", "\n")

						// 保存搜索到的结果
						newInsight := &model.TrackInsight{
							Artist: artist,
							Album:  album,
							Track:  track,
							// LyricsOriginal:    lyrics, // 移除
							LyricsTranslation: llmResp.LyricsTranslation,
							AnalysisSummary:   llmResp.AnalysisSummary,
							BackgroundInfo:    llmResp.BackgroundInfo,
							EraContext:        llmResp.EraContext,
							LLMProvider:       llmResp.LLMProvider,
							LangSource:        llmReq.LangSource,
							LangTarget:        llmReq.LangTarget,
							LastUsedAt:        time.Now(),
						}
						if llmResp.AnalysisBySection != nil {
							if serialized, serErr := json.Marshal(llmResp.AnalysisBySection); serErr == nil {
								newInsight.AnalysisBySection = string(serialized)
							}
						}

						// 检查是否已存在记录，存在则更新
						if existing, eErr := model.GetTrackInsight(
							context.Background(), artist, album, track,
						); eErr == nil {
							newInsight.ID = existing.ID
							_ = model.UpdateTrackInsight(context.Background(), newInsight)
						} else {
							_ = model.CreateTrackInsight(context.Background(), newInsight)
						}
					}(fullContent.String())
					return
				}
				out <- chunk
				fullContent.WriteString(chunk)
			}
		}
	}()

	return out, false, nil
}

// RecordFeedback 记录用户点赞/点踩反馈
func (s *serviceImpl) RecordFeedback(
	ctx context.Context, insightID int64, score int, comment string,
) error {
	if score != 1 && score != -1 {
		return errors.New("score 只能为 1 或 -1")
	}

	feedback := &model.TrackInsightFeedback{
		InsightID: insightID,
		Score:     score,
		Comment:   strings.TrimSpace(comment),
		CreatedAt: time.Now(),
	}
	if err := model.CreateTrackInsightFeedback(ctx, feedback); err != nil {
		return err
	}

	// 简单累加统计，避免每次都跑聚合
	insight, err := getInsightByID(ctx, insightID)
	if err != nil {
		return err
	}
	if score == 1 {
		insight.LikeCount++
	} else if score == -1 {
		insight.DislikeCount++
	}
	return model.UpdateTrackInsight(ctx, insight)
}

func getInsightByID(ctx context.Context, id int64) (*model.TrackInsight, error) {
	var insight model.TrackInsight
	err := model.GetDB().WithContext(ctx).First(&insight, id).Error
	if err != nil {
		return nil, err
	}
	return &insight, nil
}

// getOrFetchLyrics 优先从数据库获取歌词，如果没有则调用 provider 获取并入库
func (s *serviceImpl) getOrFetchLyrics(ctx context.Context, artist, album, track string) (string, error) {
	// 1. 先查询歌词表
	lyrics, err := model.GetTrackLyrics(ctx, artist, album, track)
	if err == nil && lyrics.LyricsOriginal != "" {
		return lyrics.LyricsOriginal, nil
	}

	// 2. 如果没有，遍历 provider 获取
	var fetchErr error
	lyricsText := ""
	source := ""
	for _, p := range s.providers {
		l, lErr := p.GetLyrics(ctx, artist, album, track)
		if lErr != nil {
			log.Warn(
				ctx, "从提供者获取歌词失败",
				zap.String("provider", p.GetName()),
				zap.Error(lErr),
			)
			fetchErr = lErr
			continue
		}
		if l != "" {
			lyricsText = l
			source = p.GetName()
			log.Info(
				ctx, "成功从 Provider 获取歌词",
				zap.String("provider", source),
			)
			break
		}
	}

	if lyricsText == "" {
		if fetchErr != nil {
			return "", fetchErr
		}
		return "", errors.New("lyrics not found")
	}

	// 3. 保存到歌词表
	// 简单的语言检测逻辑（实际可换为库）
	langCode := detectLanguage(lyricsText)
	synced := strings.Contains(lyricsText, "[") && strings.Contains(lyricsText, "]")

	newLyrics := &model.TrackLyrics{
		Artist:         artist,
		Album:          album,
		Track:          track,
		LyricsOriginal: lyricsText,
		LyricsSource:   source,
		LangCode:       langCode,
		Synced:         synced,
	}

	// 使用 GetOrCreate 避免并发冲突
	if _, err := model.GetOrCreateTrackLyrics(ctx, newLyrics); err != nil {
		log.Warn(ctx, "保存歌词失败", zap.Error(err))
	}

	return lyricsText, nil
}

func detectLanguage(text string) string {
	// 简单 heuristic
	for _, r := range text {
		if r > 0x4e00 && r < 0x9fff {
			return "zh"
		}
	}
	return "en"
}
