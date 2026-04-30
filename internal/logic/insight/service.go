package insight

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/ai"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/core/lyrics"
	"github.com/vincentchyu/sonic-lens/core/telemetry"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

// Service 定义歌词解析与深度分析服务接口
type Service interface {
	// GetOrCreateTrackInsight 获取或创建某首歌的解析结果，第二个返回值表示是否命中缓存
	GetOrCreateTrackInsight(
		ctx context.Context, artist, album, track string, trackNumber, discNumber int8, force bool,
		provider, model, legacyModelType string,
	) ([]*model.TrackInsight, bool, error)
	// RecordFeedback 记录曲目音眸的点赞/点踩反馈。
	RecordFeedback(ctx context.Context, insightID int64, req FeedbackRecordRequest) error
	// RecordAlbumFeedback 记录专辑音眸的点赞/点踩反馈。
	RecordAlbumFeedback(ctx context.Context, insightID int64, req FeedbackRecordRequest) error
	// GetOrCreateInsightStream 获取大模型流式解析结果，第二个返回值表示是否命中缓存
	GetOrCreateInsightStream(
		ctx context.Context, artist, album, track string, trackNumber, discNumber int8, force bool,
		provider, model, legacyModelType string,
	) (<-chan string, bool, error)
	// GetAvailableAIPlatforms 获取当前系统可用的 AI 平台。
	GetAvailableAIPlatforms() []ai.PlatformOption
	// GetPlatformModels 获取某个平台可用的模型目录。
	GetPlatformModels(ctx context.Context, platform common.AIModelPlatform) ([]ai.ModelOption, error)
	// GetInsightOnly 仅从数据库获取已有的解析结果，不触发 AI 分析
	GetInsightOnly(ctx context.Context, artist, album, track string, trackNumber, discNumber int8) (
		[]*InsightWithScore, error,
	)
	// GetAlbumInsightOnly 仅从数据库获取已有的专辑解析结果，不触发 AI 分析
	GetAlbumInsightOnly(ctx context.Context, albumID int64) ([]*model.AlbumInsight, error)
	// GetOrCreateAlbumInsight 获取或创建某张专辑的解析结果，第二个返回值表示是否命中缓存
	GetOrCreateAlbumInsight(
		ctx context.Context, albumID int64, force bool, provider, model, legacyModelType string,
	) ([]*model.AlbumInsight, bool, error)
	// CreateInsightJob 创建音眸异步任务，供客户端后台等待与 WS 同步使用。
	CreateInsightJob(ctx context.Context, req CreateInsightJobRequest) (*model.InsightJob, bool, error)
	// GetInsightJob 获取单个音眸任务当前状态。
	GetInsightJob(ctx context.Context, jobID string) (*model.InsightJob, error)
	// GetInsightJobCallLogs 获取指定音眸任务关联的调用流水。
	GetInsightJobCallLogs(ctx context.Context, jobID string) ([]*model.LLMCallLog, error)
	// ListInsightJobs 分页获取音眸任务，供管理端查看与筛选。
	ListInsightJobs(ctx context.Context, query model.InsightJobListQuery) ([]*model.InsightJob, int64, error)
	// UpdateInsightJobLiveActivityToken 更新任务关联的 Live Activity push token。
	UpdateInsightJobLiveActivityToken(ctx context.Context, jobID, token string) (*model.InsightJob, error)
	// CancelInsightJob 取消指定音眸任务，供管理端手动维护生命周期。
	CancelInsightJob(ctx context.Context, jobID string) (*model.InsightJob, error)
	// RetryInsightJob 基于既有任务重新排队，便于管理端重试失败任务。
	RetryInsightJob(ctx context.Context, jobID string) (*model.InsightJob, bool, error)
	// DeleteInsightJob 删除失败或已取消的音眸任务及其调用流水。
	DeleteInsightJob(ctx context.Context, jobID string) error
	// GetAllInsights 分页获取所有解析记录
	GetAllInsights(
		ctx context.Context, limit, offset int, keyword string, targetType common.AnalysisTargetType,
	) ([]*model.InsightListItem, int64, error)
	// GetInsightDetail 按主键和对象类型获取解析详情，避免上层直接区分表结构。
	GetInsightDetail(ctx context.Context, targetType common.AnalysisTargetType, id int64) (any, error)
	// ToggleInsightStatus 切换解析记录的禁用状态，并按对象类型路由到正确的数据表。
	ToggleInsightStatus(ctx context.Context, targetType common.AnalysisTargetType, id int64) error
	// GetTrackCallLogs 获取某曲目的 LLM 调用流水
	GetTrackCallLogs(ctx context.Context, artist, album, track string) ([]*model.LLMCallLog, error)
	// GetAlbumCallLogs 获取某专辑的 LLM 调用流水
	GetAlbumCallLogs(ctx context.Context, albumID int64) ([]*model.LLMCallLog, error)
	// GetInsightCallLogs 按对象类型获取调用流水。
	GetInsightCallLogs(ctx context.Context, targetType common.AnalysisTargetType, id int64) ([]*model.LLMCallLog, error)
	// DeleteInsight 按对象类型删除解析记录及其关联流水。
	DeleteInsight(ctx context.Context, targetType common.AnalysisTargetType, id int64) error
	// GetTrackInsightFeedbacks 获取曲目音眸的关联反馈。
	GetTrackInsightFeedbacks(ctx context.Context, insightID int64) ([]*model.TrackInsightFeedback, error)
	// GetAlbumInsightFeedbacks 获取专辑音眸的关联反馈。
	GetAlbumInsightFeedbacks(ctx context.Context, insightID int64) ([]*model.AlbumInsightFeedback, error)
	// GetInsightFeedbackSummary 获取单条音眸的反馈摘要，供详情和列表轻量状态展示。
	GetInsightFeedbackSummary(
		ctx context.Context, targetType common.AnalysisTargetType, insightID int64,
	) (*InsightFeedbackSummary, error)
	// GetInsightFeedbackHistory 获取单条音眸的最近反馈历史，默认按时间倒序。
	GetInsightFeedbackHistory(
		ctx context.Context, targetType common.AnalysisTargetType, insightID int64, limit int,
	) ([]*InsightFeedbackHistoryItem, error)
	// GetInsightHistory 获取单条音眸的历史版本摘要，供详情页的历史 tab 懒加载使用。
	GetInsightHistory(
		ctx context.Context, targetType common.AnalysisTargetType, insightID int64, limit int,
	) ([]*model.InsightListItem, error)
	// GetLyrics 获取歌词内容，缺失时自动回源并写入缓存。
	GetLyrics(ctx context.Context, artist, album, track string, trackNumber, discNumber int8) (
		*model.TrackLyrics, error,
	)
}

type serviceImpl struct {
	llmCache  map[string]ai.LLMProvider
	providers []lyrics.Provider
}

type InsightWithScore struct {
	*model.TrackInsight
	TotalScore int `json:"total_score"`
}

type FeedbackRecordRequest struct {
	Score          int
	Comment        string
	ReasonCodes    []string
	SectionKey     string
	SourcePlatform string
}

type InsightFeedbackHistoryItem struct {
	ID             int64     `json:"id"`
	InsightID      int64     `json:"insight_id"`
	Score          int       `json:"score"`
	Comment        string    `json:"comment"`
	ReasonCodes    []string  `json:"reason_codes"`
	SectionKey     string    `json:"section_key,omitempty"`
	SourcePlatform string    `json:"source_platform,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type InsightFeedbackSummary struct {
	InsightID              int64                       `json:"insight_id"`
	AnalysisTargetType     common.AnalysisTargetType   `json:"analysis_target_type"`
	LikeCount              int64                       `json:"like_count"`
	DislikeCount           int64                       `json:"dislike_count"`
	HasFeedback            bool                        `json:"has_feedback"`
	LatestFeedback         *InsightFeedbackHistoryItem `json:"latest_feedback,omitempty"`
	LatestNegativeFeedback *InsightFeedbackHistoryItem `json:"latest_negative_feedback,omitempty"`
	TopReasonCodes         []string                    `json:"top_reason_codes"`
}

// NewService 创建 Insight Service 实例
func NewService() (Service, error) {
	return &serviceImpl{
		llmCache: make(map[string]ai.LLMProvider),
		providers: []lyrics.Provider{
			lyrics.NewLrcAPIProvider(config.ConfigObj.Lyrics.LrcAPI.BaseURL, config.ConfigObj.Lyrics.LrcAPI.Token),
			lyrics.NewMxmProvider(),
		},
	}, nil
}

// getLLMProvider 获取指定的 Provider，如果不存在则实例化并缓存
func (s *serviceImpl) getLLMProvider(selection ai.ResolvedProviderSelection) (ai.LLMProvider, error) {
	cacheKey := string(selection.Platform) + "::" + selection.Model
	if p, ok := s.llmCache[cacheKey]; ok {
		return p, nil
	}
	p, err := ai.NewProviderBySelection(selection.Platform, selection.Model)
	if err != nil {
		return nil, err
	}
	s.llmCache[cacheKey] = p
	return p, nil
}

// GetAvailableAIPlatforms 获取当前配置中可用的 AI 平台列表。
func (s *serviceImpl) GetAvailableAIPlatforms() []ai.PlatformOption {
	return ai.GetConfiguredPlatforms()
}

// GetPlatformModels 获取某个平台可用的模型目录。
func (s *serviceImpl) GetPlatformModels(ctx context.Context, platform common.AIModelPlatform) (
	[]ai.ModelOption, error,
) {
	return ai.GetModelsByPlatform(ctx, platform)
}

// GetInsightOnly 仅从数据库获取已有的解析结果
func (s *serviceImpl) GetInsightOnly(
	ctx context.Context, artist, album, track string, trackNumber, discNumber int8,
) ([]*InsightWithScore, error) {
	artist = strings.TrimSpace(artist)
	album = strings.TrimSpace(album)
	track = strings.TrimSpace(track)

	trackObj, _ := model.GetTrackByIdentity(ctx, artist, album, track, trackNumber, discNumber)
	lookup := model.TrackInsightLookup{Artist: artist, Album: album, Track: track}
	if trackObj != nil {
		lookup.TrackID = trackObj.ID
	}
	insights, err := model.GetTrackInsightsByLookup(ctx, lookup)
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
	// 排序：高分优先；同分时，按创建时间降序（最新优先）
	sort.Slice(
		result, func(i, j int) bool {
			if result[i].TotalScore != result[j].TotalScore {
				return result[i].TotalScore > result[j].TotalScore
			}
			return result[i].CreatedAt.After(result[j].CreatedAt)
		},
	)
	return result, nil
}

// GetAlbumInsightOnly 仅从数据库获取已有的专辑解析结果。
func (s *serviceImpl) GetAlbumInsightOnly(ctx context.Context, albumID int64) ([]*model.AlbumInsight, error) {
	if albumID <= 0 {
		return nil, errors.New("albumID 不能为空")
	}

	detail, err := model.GetAlbumWithTracks(ctx, albumID)
	if err != nil {
		return nil, err
	}

	insights, err := model.GetAlbumInsightsByLookup(
		ctx,
		model.AlbumInsightLookup{
			AlbumID: albumID,
			Artist:  detail.Artist,
			Album:   detail.Name,
		},
	)
	if err != nil {
		return nil, err
	}
	if err := sortAlbumInsightsByTotalScore(ctx, insights); err != nil {
		return nil, err
	}
	return insights, nil
}

// GetOrCreateAlbumInsight 获取或创建某张专辑的解析结果。
func (s *serviceImpl) GetOrCreateAlbumInsight(
	ctx context.Context, albumID int64, force bool, provider, modelName, legacyModelType string,
) ([]*model.AlbumInsight, bool, error) {
	selection, err := ai.ResolveProviderSelection(
		ai.ProviderSelectionInput{
			Provider:        provider,
			Model:           modelName,
			LegacyModelType: legacyModelType,
		},
	)
	if err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(modelName) != "" {
		if err := ai.ValidateModelAvailability(ctx, selection.Platform, selection.Model); err != nil {
			return nil, false, err
		}
	}

	log.Info(
		ctx,
		"开始获取或创建专辑解析",
		zap.Int64("album_id", albumID),
		zap.Bool("force", force),
		zap.String("provider", string(selection.Platform)),
		zap.String("model", selection.Model),
	)

	if albumID <= 0 {
		return nil, false, errors.New("albumID 不能为空")
	}

	detail, err := model.GetAlbumWithTracks(ctx, albumID)
	if err != nil {
		return nil, false, err
	}

	lookup := model.AlbumInsightLookup{
		AlbumID: albumID,
		Artist:  detail.Artist,
		Album:   detail.Name,
	}
	insights, err := model.GetAlbumInsightsByLookup(ctx, lookup)
	if err == nil && len(insights) > 0 && !force {
		if err := sortAlbumInsightsByTotalScore(ctx, insights); err != nil {
			return nil, false, err
		}
		insights[0].LastUsedAt = time.Now()
		_ = model.UpdateAlbumInsight(ctx, insights[0])
		log.Info(
			ctx,
			"专辑解析命中缓存",
			zap.Int64("album_id", albumID),
			zap.String("artist", detail.Artist),
			zap.String("album", detail.Name),
			zap.Int("insight_count", len(insights)),
		)
		return insights, true, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	trackContexts, selectedInsightIDs, totalTracks, err := s.buildAlbumTrackContexts(ctx, detail)
	if err != nil {
		return nil, false, err
	}
	if len(trackContexts) == 0 {
		return nil, false, errors.New("当前专辑下暂无可用的曲目音眸分析，请先生成曲目分析")
	}

	feedbackCtx := ""
	if negativeFeedbacks, fbErr := model.GetNegativeAlbumFeedbacksByLookup(
		ctx, lookup,
	); fbErr == nil && len(negativeFeedbacks) > 0 {
		feedbackCtx = buildAlbumNegativeFeedbackContext(negativeFeedbacks)
		if feedbackCtx != "" {
			log.Info(
				ctx,
				"检测到历史差评反馈，将注入到专辑分析上下文",
				zap.String("feedback", feedbackCtx),
			)
		}
	}

	llmReq := ai.AlbumAnalysisRequest{
		AlbumID:           detail.ID,
		Artist:            detail.Artist,
		Album:             detail.Name,
		ReleaseDate:       detail.ReleaseDate,
		Genre:             detail.Genre,
		TrackCount:        totalTracks,
		AnalyzedTracks:    len(trackContexts),
		TrackContexts:     trackContexts,
		FeedbackContext:   feedbackCtx,
		RequestedProvider: selection.RequestedProvider,
		RequestedModel:    selection.RequestedModel,
	}

	llm, err := s.getLLMProvider(selection)
	if err != nil {
		log.Error(
			ctx,
			"获取专辑分析模型失败",
			zap.Int64("album_id", albumID),
			zap.String("artist", detail.Artist),
			zap.String("album", detail.Name),
			zap.Error(err),
		)
		return nil, false, err
	}

	llmResp, err := llm.AnalyzeAlbum(ctx, llmReq)
	if err != nil {
		log.Error(
			ctx,
			"调用大模型进行专辑分析失败",
			zap.Int64("album_id", albumID),
			zap.String("artist", detail.Artist),
			zap.String("album", detail.Name),
			zap.Error(err),
		)
		return nil, false, err
	}

	newInsight := &model.AlbumInsight{
		AlbumID:           detail.ID,
		Artist:            detail.Artist,
		Album:             detail.Name,
		AnalysisSummary:   llmResp.AnalysisSummary,
		AnalysisBySection: llmResp.AnalysisBySection,
		BackgroundInfo:    llmResp.BackgroundInfo,
		EraContext:        llmResp.EraContext,
		LLMProvider:       llmResp.LLMProvider,
		LastUsedAt:        time.Now(),
	}

	metadata := llmResp.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["album_id"] = detail.ID
	metadata["total_tracks"] = totalTracks
	metadata["analyzed_tracks"] = len(trackContexts)
	metadata["selected_track_insight_ids"] = selectedInsightIDs
	if serialized, serErr := json.Marshal(metadata); serErr == nil {
		newInsight.Metadata = string(serialized)
	}

	if err := model.CreateAlbumInsight(ctx, newInsight); err != nil {
		return nil, false, err
	}

	insights, err = model.GetAlbumInsightsByLookup(ctx, lookup)
	if err != nil {
		log.Warn(
			ctx,
			"重新读取专辑解析失败，直接返回新建结果",
			zap.Int64("album_id", albumID),
			zap.Error(err),
		)
		return []*model.AlbumInsight{newInsight}, false, nil
	}
	if err := sortAlbumInsightsByTotalScore(ctx, insights); err != nil {
		return nil, false, err
	}

	log.Info(
		ctx,
		"专辑解析获取完成",
		zap.Int64("album_id", albumID),
		zap.String("artist", detail.Artist),
		zap.String("album", detail.Name),
		zap.Bool("cache_hit", false),
		zap.Int("insight_count", len(insights)),
		zap.Int("analyzed_tracks", len(trackContexts)),
	)
	return insights, false, nil
}

// GetOrCreateTrackInsight 获取或创建某首歌的解析结果
func (s *serviceImpl) GetOrCreateTrackInsight(
	ctx context.Context, artist, album, track string, trackNumber, discNumber int8, force bool,
	provider, modelName, legacyModelType string,
) ([]*model.TrackInsight, bool, error) {
	artist = strings.TrimSpace(artist)
	album = strings.TrimSpace(album)
	track = strings.TrimSpace(track)

	selection, err := ai.ResolveProviderSelection(
		ai.ProviderSelectionInput{
			Provider:        provider,
			Model:           modelName,
			LegacyModelType: legacyModelType,
		},
	)
	if err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(modelName) != "" {
		if err := ai.ValidateModelAvailability(ctx, selection.Platform, selection.Model); err != nil {
			return nil, false, err
		}
	}

	log.Info(
		ctx,
		"开始获取或创建歌曲解析",
		zap.String("artist", artist),
		zap.String("album", album),
		zap.String("track", track),
		zap.Bool("force", force),
		zap.String("provider", string(selection.Platform)),
		zap.String("model", selection.Model),
	)

	if artist == "" || album == "" || track == "" {
		return nil, false, errors.New("artist, album, track 不能为空")
	}

	// 先尝试从数据库中获取已存在的解析
	trackObj, _ := model.GetTrackByIdentity(ctx, artist, album, track, trackNumber, discNumber)
	lookup := model.TrackInsightLookup{Artist: artist, Album: album, Track: track}
	if trackObj != nil {
		lookup.TrackID = trackObj.ID
	}

	insights, err := model.GetTrackInsightsByLookup(ctx, lookup)
	if err == nil && len(insights) > 0 && !force {
		if err := sortTrackInsightsByTotalScore(ctx, insights); err != nil {
			return nil, false, err
		}
		// 命中缓存，更新最近一条的使用时间
		insights[0].LastUsedAt = time.Now()
		_ = model.UpdateTrackInsight(ctx, insights[0])
		log.Info(
			ctx,
			"歌曲解析命中缓存",
			zap.String("artist", artist),
			zap.String("album", album),
			zap.String("track", track),
			zap.Int("insight_count", len(insights)),
		)
		return insights, true, nil
	}
	// 如果强制刷新或没找到且出错不是 NotFound，返回错误
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	// 准备歌词
	lyrics, err := s.getOrFetchLyrics(ctx, artist, album, track, trackNumber, discNumber)
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
	if negativeFeedbacks, fbErr := model.GetNegativeFeedbacksByLookup(
		ctx, lookup,
	); fbErr == nil && len(negativeFeedbacks) > 0 {
		feedbackCtx = buildTrackNegativeFeedbackContext(negativeFeedbacks)
		if feedbackCtx != "" {
			log.Info(ctx, "检测到历史差评反馈，将注入到分析上下文", zap.String("feedback", feedbackCtx))
		}
	}

	// 调用大模型进行翻译与解析
	llmReq := ai.TrackAnalysisRequest{
		Title:             track,
		Artist:            artist,
		Album:             album,
		Lyrics:            ai.CleanLyrics(lyrics),
		LangSource:        "auto",
		LangTarget:        "zh-CN",
		FeedbackContext:   feedbackCtx,
		RequestedProvider: selection.RequestedProvider,
		RequestedModel:    selection.RequestedModel,
	}

	llm, err := s.getLLMProvider(selection)
	if err != nil {
		log.Error(
			ctx,
			"获取分析模型失败",
			zap.String("artist", artist),
			zap.String("album", album),
			zap.String("track", track),
			zap.Error(err),
		)
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
		TrackID:           lookup.TrackID,
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
		AnalysisBySection: llmResp.AnalysisBySection,
		LastUsedAt:        time.Now(),
	}
	if llmResp.Metadata != nil {
		if serialized, serErr := json.Marshal(llmResp.Metadata); serErr == nil {
			newInsight.Metadata = string(serialized)
		}
	}

	if err := model.CreateTrackInsight(ctx, newInsight); err != nil {
		return nil, false, err
	}
	log.Info(
		ctx,
		"歌曲解析已创建并落库",
		zap.String("artist", artist),
		zap.String("album", album),
		zap.String("track", track),
	)

	// 重新获取完整列表
	insights, err = model.GetTrackInsightsByLookup(ctx, lookup)
	if err != nil {
		// 降级：只返回新创建的
		log.Warn(
			ctx,
			"重新读取歌曲解析失败，直接返回新建结果",
			zap.String("artist", artist),
			zap.String("album", album),
			zap.String("track", track),
			zap.Error(err),
		)
		return []*model.TrackInsight{newInsight}, false, nil
	}
	if err := sortTrackInsightsByTotalScore(ctx, insights); err != nil {
		return nil, false, err
	}

	log.Info(
		ctx,
		"歌曲解析获取完成",
		zap.String("artist", artist),
		zap.String("album", album),
		zap.String("track", track),
		zap.Bool("cache_hit", false),
		zap.Int("insight_count", len(insights)),
	)
	return insights, false, nil
}

// GetOrCreateInsightStream 获取流式解析结果
func (s *serviceImpl) GetOrCreateInsightStream(
	ctx context.Context, artist, album, track string, trackNumber, discNumber int8, force bool,
	provider, modelName, legacyModelType string,
) (<-chan string, bool, error) {
	artist = strings.TrimSpace(artist)
	album = strings.TrimSpace(album)
	track = strings.TrimSpace(track)

	selection, err := ai.ResolveProviderSelection(
		ai.ProviderSelectionInput{
			Provider:        provider,
			Model:           modelName,
			LegacyModelType: legacyModelType,
		},
	)
	if err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(modelName) != "" {
		if err := ai.ValidateModelAvailability(ctx, selection.Platform, selection.Model); err != nil {
			return nil, false, err
		}
	}

	log.Info(
		ctx,
		"开始获取歌曲流式解析",
		zap.String("artist", artist),
		zap.String("album", album),
		zap.String("track", track),
		zap.Bool("force", force),
		zap.String("provider", string(selection.Platform)),
		zap.String("model", selection.Model),
	)

	// 先尝试从数据库中获取已存在的解析
	trackObj, _ := model.GetTrackByIdentity(ctx, artist, album, track, trackNumber, discNumber)
	lookup := model.TrackInsightLookup{Artist: artist, Album: album, Track: track}
	if trackObj != nil {
		lookup.TrackID = trackObj.ID
	}
	insights, err := model.GetTrackInsightsByLookup(ctx, lookup)
	if err == nil && len(insights) > 0 && !force {
		// 命中缓存，模拟流式输出，返回整个列表的 JSON
		out := make(chan string, 1)
		b, _ := json.Marshal(insights)
		// 转换为简化 JSON 格式 (这里直接返回完整对象的 JSON array 也可以，前端需要适配)
		// 为了保持一致性，我们这里直接返回 insights 的 JSON
		out <- string(b)
		close(out)
		log.Info(
			ctx,
			"歌曲流式解析命中缓存",
			zap.String("artist", artist),
			zap.String("album", album),
			zap.String("track", track),
			zap.Int("insight_count", len(insights)),
		)
		return out, true, nil
	}

	// 准备歌词
	lyrics, err := s.getOrFetchLyrics(ctx, artist, album, track, trackNumber, discNumber)
	if err != nil {
		log.Warn(ctx, "获取歌词失败，流式分析使用空歌词", zap.Error(err))
	}

	// 查询历史差评反馈，用于改进分析质量
	feedbackCtx := ""
	if negativeFeedbacks, fbErr := model.GetNegativeFeedbacksByLookup(
		ctx, lookup,
	); fbErr == nil && len(negativeFeedbacks) > 0 {
		feedbackCtx = buildTrackNegativeFeedbackContext(negativeFeedbacks)
		if feedbackCtx != "" {
			log.Info(ctx, "检测到历史差评反馈，将注入到流式分析上下文", zap.String("feedback", feedbackCtx))
		}
	}

	llmReq := ai.TrackAnalysisRequest{
		Title:             track,
		Artist:            artist,
		Album:             album,
		Lyrics:            ai.CleanLyrics(lyrics),
		LangSource:        "auto",
		LangTarget:        "zh-CN",
		FeedbackContext:   feedbackCtx,
		RequestedProvider: selection.RequestedProvider,
		RequestedModel:    selection.RequestedModel,
	}

	llm, err := s.getLLMProvider(selection)
	if err != nil {
		log.Error(
			ctx,
			"获取流式分析模型失败",
			zap.String("artist", artist),
			zap.String("album", album),
			zap.String("track", track),
			zap.Error(err),
		)
		return nil, false, err
	}
	ch, err := llm.AnalyzeTrackStream(ctx, llmReq)
	if err != nil {
		log.Error(
			ctx,
			"启动歌曲流式解析失败",
			zap.String("artist", artist),
			zap.String("album", album),
			zap.String("track", track),
			zap.Error(err),
		)
		return nil, false, err
	}

	// 我们需要一个中间层来聚合结果并存入数据库，同时转发给前端
	out := make(chan string, 10)
	// 使用一个新的 context，避免主请求取消导致流中断
	// 但实际上 SSE 应该随主请求结束而结束，这里的 out 发送应该监听 ctx.Done()
	telemetry.GoSafe(
		ctx, "insight.stream.forward_result", func(asyncCtx context.Context) {
			defer close(out)
			var fullContent strings.Builder
			for {
				select {
				case <-asyncCtx.Done():
					return
				case chunk, ok := <-ch:
					if !ok {
						// 流自然结束，执行存入数据库逻辑
						content := fullContent.String()
						telemetry.GoSafeDetached(
							asyncCtx, "insight.stream.persist_result", func(persistCtx context.Context) {
								if content == "" {
									return
								}
								raw := ai.TrimCodeFence(content)
								var llmResp ai.TrackAnalysisResult
								if err := json.Unmarshal([]byte(raw), &llmResp); err != nil {
									log.Error(persistCtx, "流式结果解析 JSON 失败，无法存入缓存", zap.Error(err))
									return
								}
								// 修复：将字面量 \n 转换为实际换行符，避免前端显示异常
								llmResp.LyricsTranslation = strings.ReplaceAll(llmResp.LyricsTranslation, "\\n", "\n")

								// 保存搜索到的结果
								newInsight := &model.TrackInsight{
									TrackID: lookup.TrackID,
									Artist:  artist,
									Album:   album,
									Track:   track,
									// LyricsOriginal:    lyrics, // 移除
									LyricsTranslation: llmResp.LyricsTranslation,
									AnalysisBySection: llmResp.AnalysisBySection,
									AnalysisSummary:   llmResp.AnalysisSummary,
									BackgroundInfo:    llmResp.BackgroundInfo,
									EraContext:        llmResp.EraContext,
									LLMProvider:       llmResp.LLMProvider,
									LangSource:        llmReq.LangSource,
									LangTarget:        llmReq.LangTarget,
									LastUsedAt:        time.Now(),
								}

								// 检查是否已存在记录，存在则更新
								if existing, eErr := model.GetTrackInsightByLookup(persistCtx, lookup); eErr == nil {
									newInsight.ID = existing.ID
									_ = model.UpdateTrackInsight(persistCtx, newInsight)
								} else {
									_ = model.CreateTrackInsight(persistCtx, newInsight)
								}
								log.Info(
									persistCtx,
									"歌曲流式解析结果已落库",
									zap.String("artist", artist),
									zap.String("album", album),
									zap.String("track", track),
								)
							},
						)
						return
					}
					out <- chunk
					fullContent.WriteString(chunk)
				}
			}
		},
	)

	log.Info(
		ctx,
		"歌曲流式解析开始输出",
		zap.String("artist", artist),
		zap.String("album", album),
		zap.String("track", track),
	)
	return out, false, nil
}

// RecordFeedback 记录用户点赞/点踩反馈
func (s *serviceImpl) RecordFeedback(
	ctx context.Context, insightID int64, req FeedbackRecordRequest,
) error {
	if req.Score != 1 && req.Score != -1 {
		return errors.New("score 只能为 1 或 -1")
	}

	feedback := &model.TrackInsightFeedback{
		InsightID:      insightID,
		Score:          req.Score,
		Comment:        strings.TrimSpace(req.Comment),
		ReasonCodes:    normalizeReasonCodes(req.ReasonCodes),
		SectionKey:     strings.TrimSpace(req.SectionKey),
		SourcePlatform: strings.TrimSpace(req.SourcePlatform),
		CreatedAt:      time.Now(),
	}
	if err := model.CreateTrackInsightFeedback(ctx, feedback); err != nil {
		return err
	}
	log.Info(
		ctx,
		"已记录歌曲解析反馈",
		zap.Int64("insight_id", insightID),
		zap.Int("score", req.Score),
	)

	// 简单累加统计，避免每次都跑聚合
	insight, err := getInsightByID(ctx, insightID)
	if err != nil {
		return err
	}
	if req.Score == 1 {
		insight.LikeCount++
	} else if req.Score == -1 {
		insight.DislikeCount++
	}
	return model.UpdateTrackInsight(ctx, insight)
}

// RecordAlbumFeedback 记录专辑音眸的点赞/点踩反馈。
func (s *serviceImpl) RecordAlbumFeedback(
	ctx context.Context, insightID int64, req FeedbackRecordRequest,
) error {
	if req.Score != 1 && req.Score != -1 {
		return errors.New("score 只能为 1 或 -1")
	}

	feedback := &model.AlbumInsightFeedback{
		InsightID:      insightID,
		Score:          req.Score,
		Comment:        strings.TrimSpace(req.Comment),
		ReasonCodes:    normalizeReasonCodes(req.ReasonCodes),
		SectionKey:     strings.TrimSpace(req.SectionKey),
		SourcePlatform: strings.TrimSpace(req.SourcePlatform),
		CreatedAt:      time.Now(),
	}
	if err := model.CreateAlbumInsightFeedback(ctx, feedback); err != nil {
		return err
	}
	log.Info(
		ctx,
		"已记录专辑解析反馈",
		zap.Int64("insight_id", insightID),
		zap.Int("score", req.Score),
	)

	insight, err := model.GetAlbumInsightByID(ctx, insightID)
	if err != nil {
		return err
	}
	if req.Score == 1 {
		insight.LikeCount++
	} else if req.Score == -1 {
		insight.DislikeCount++
	}
	return model.UpdateAlbumInsight(ctx, insight)
}

func getInsightByID(ctx context.Context, id int64) (*model.TrackInsight, error) {
	return model.GetTrackInsightByID(ctx, id)
}

// getOrFetchLyrics 优先从数据库获取歌词，如果没有则调用 provider 获取并入库
func (s *serviceImpl) getOrFetchLyrics(
	ctx context.Context, artist, album, track string, trackNumber, discNumber int8,
) (string, error) {
	// 1. 先查询歌词表
	trackObj, _ := model.GetTrackByIdentity(ctx, artist, album, track, trackNumber, discNumber)
	lookup := model.TrackLyricsLookup{Artist: artist, Album: album, Track: track}
	if trackObj != nil {
		lookup.TrackID = trackObj.ID
	}
	lyricsRecord, err := model.GetTrackLyricsByLookup(ctx, lookup)
	if err == nil && lyricsRecord.LyricsOriginal != "" {
		return lyricsRecord.LyricsOriginal, nil
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
	synced := lyrics.IsSyncedLRC(lyricsText)

	newLyrics := &model.TrackLyrics{
		TrackID:        lookup.TrackID,
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

// GetAllInsights 分页获取所有解析记录
func (s *serviceImpl) GetAllInsights(
	ctx context.Context, limit, offset int, keyword string, targetType common.AnalysisTargetType,
) ([]*model.InsightListItem, int64, error) {
	return model.GetAllInsightSummaries(ctx, targetType, limit, offset, keyword)
}

// GetInsightDetail 按主键和对象类型获取解析详情。
func (s *serviceImpl) GetInsightDetail(
	ctx context.Context,
	targetType common.AnalysisTargetType,
	id int64,
) (any, error) {
	switch targetType {
	case common.AnalysisTargetTypeAlbum:
		return model.GetAlbumInsightByID(ctx, id)
	default:
		return getInsightByID(ctx, id)
	}
}

// ToggleInsightStatus 切换解析记录的禁用状态，并按对象类型更新正确的数据表。
func (s *serviceImpl) ToggleInsightStatus(
	ctx context.Context,
	targetType common.AnalysisTargetType,
	id int64,
) error {
	switch targetType {
	case common.AnalysisTargetTypeAlbum:
		insight, err := model.GetAlbumInsightByID(ctx, id)
		if err != nil {
			return err
		}
		_, err = model.UpdateAlbumInsightDisabled(ctx, id, !insight.IsDisabled)
		return err
	default:
		insight, err := getInsightByID(ctx, id)
		if err != nil {
			return err
		}
		insight.IsDisabled = !insight.IsDisabled
		return model.UpdateTrackInsight(ctx, insight)
	}
}

// GetTrackCallLogs 获取某曲目的 LLM 调用流水。
func (s *serviceImpl) GetTrackCallLogs(ctx context.Context, artist, album, track string) ([]*model.LLMCallLog, error) {
	primaryLogs, err := model.GetLLMCallLogsByTrack(ctx, artist, album, track, 50)
	if err != nil {
		return nil, err
	}

	legacyLogs, err := model.GetLLMCallLogsByTrackInfo(
		ctx,
		common.AnalysisTargetTypeTrack,
		artist+" - "+track,
		50,
	)
	if err != nil {
		return nil, err
	}

	return mergeCallLogs(50, primaryLogs, legacyLogs), nil
}

// GetAlbumCallLogs 获取某专辑的 LLM 调用流水。
func (s *serviceImpl) GetAlbumCallLogs(ctx context.Context, albumID int64) ([]*model.LLMCallLog, error) {
	primaryLogs, err := model.GetLLMCallLogsByAlbumID(ctx, albumID, 50)
	if err != nil {
		return nil, err
	}

	albumDetail, err := model.GetAlbumWithTracks(ctx, albumID)
	if err != nil {
		return nil, err
	}

	legacyLogs, err := model.GetLLMCallLogsByTrackInfo(
		ctx,
		common.AnalysisTargetTypeAlbum,
		albumDetail.Artist+" - "+albumDetail.Name,
		50,
	)
	if err != nil {
		return nil, err
	}

	return mergeCallLogs(50, primaryLogs, legacyLogs), nil
}

// GetInsightCallLogs 按对象类型获取某次解析关联的调用流水。
func (s *serviceImpl) GetInsightCallLogs(
	ctx context.Context,
	targetType common.AnalysisTargetType,
	id int64,
) ([]*model.LLMCallLog, error) {
	switch targetType {
	case common.AnalysisTargetTypeAlbum:
		insight, err := model.GetAlbumInsightByID(ctx, id)
		if err != nil {
			return nil, err
		}
		return s.GetAlbumCallLogs(ctx, insight.AlbumID)
	default:
		insight, err := getInsightByID(ctx, id)
		if err != nil {
			return nil, err
		}
		return s.GetTrackCallLogs(ctx, insight.Artist, insight.Album, insight.Track)
	}
}

func mergeCallLogs(limit int, groups ...[]*model.LLMCallLog) []*model.LLMCallLog {
	seen := make(map[int64]struct{})
	if limit <= 0 {
		limit = 50
	}
	merged := make([]*model.LLMCallLog, 0, limit)

	for _, group := range groups {
		for _, logItem := range group {
			if logItem == nil {
				continue
			}
			if _, ok := seen[logItem.ID]; ok {
				continue
			}
			seen[logItem.ID] = struct{}{}
			merged = append(merged, logItem)
		}
	}

	sort.Slice(
		merged, func(i, j int) bool {
			return merged[i].CreatedAt.After(merged[j].CreatedAt)
		},
	)
	if len(merged) > limit {
		merged = merged[:limit]
	}
	return merged
}

// DeleteInsight 按对象类型删除解析记录。
func (s *serviceImpl) DeleteInsight(ctx context.Context, targetType common.AnalysisTargetType, id int64) error {
	switch targetType {
	case common.AnalysisTargetTypeAlbum:
		return model.DeleteAlbumInsight(ctx, uint64(id))
	default:
		return model.DeleteTrackInsight(ctx, uint64(id))
	}
}

// GetTrackInsightFeedbacks 获取曲目反馈记录。
func (s *serviceImpl) GetTrackInsightFeedbacks(
	ctx context.Context, insightID int64,
) ([]*model.TrackInsightFeedback, error) {
	return model.GetTrackInsightFeedbacks(ctx, insightID)
}

// GetAlbumInsightFeedbacks 获取专辑反馈记录。
func (s *serviceImpl) GetAlbumInsightFeedbacks(
	ctx context.Context, insightID int64,
) ([]*model.AlbumInsightFeedback, error) {
	return model.GetAlbumInsightFeedbacks(ctx, insightID)
}

// GetInsightFeedbackSummary 获取单条音眸的反馈摘要。
func (s *serviceImpl) GetInsightFeedbackSummary(
	ctx context.Context,
	targetType common.AnalysisTargetType,
	insightID int64,
) (*InsightFeedbackSummary, error) {
	switch targetType {
	case common.AnalysisTargetTypeAlbum:
		insight, err := model.GetAlbumInsightByID(ctx, insightID)
		if err != nil {
			return nil, err
		}
		feedbacks, err := model.GetAlbumInsightFeedbacks(ctx, insightID)
		if err != nil {
			return nil, err
		}
		return buildFeedbackSummaryFromAlbum(
			insightID, targetType, insight.LikeCount, insight.DislikeCount, feedbacks,
		), nil
	default:
		insight, err := getInsightByID(ctx, insightID)
		if err != nil {
			return nil, err
		}
		feedbacks, err := model.GetTrackInsightFeedbacks(ctx, insightID)
		if err != nil {
			return nil, err
		}
		return buildFeedbackSummaryFromTrack(
			insightID, targetType, insight.LikeCount, insight.DislikeCount, feedbacks,
		), nil
	}
}

// GetInsightFeedbackHistory 获取单条音眸的最近反馈历史。
func (s *serviceImpl) GetInsightFeedbackHistory(
	ctx context.Context,
	targetType common.AnalysisTargetType,
	insightID int64,
	limit int,
) ([]*InsightFeedbackHistoryItem, error) {
	switch targetType {
	case common.AnalysisTargetTypeAlbum:
		feedbacks, err := model.GetAlbumInsightFeedbacksLimited(ctx, insightID, limit)
		if err != nil {
			return nil, err
		}
		return convertAlbumFeedbackHistory(feedbacks), nil
	default:
		feedbacks, err := model.GetTrackInsightFeedbacksLimited(ctx, insightID, limit)
		if err != nil {
			return nil, err
		}
		return convertTrackFeedbackHistory(feedbacks), nil
	}
}

// GetInsightHistory 获取单条音眸对应的历史版本摘要。
func (s *serviceImpl) GetInsightHistory(
	ctx context.Context,
	targetType common.AnalysisTargetType,
	insightID int64,
	limit int,
) ([]*model.InsightListItem, error) {
	if limit <= 0 {
		limit = 20
	}

	switch targetType {
	case common.AnalysisTargetTypeAlbum:
		current, err := model.GetAlbumInsightByID(ctx, insightID)
		if err != nil {
			return nil, err
		}
		insights, err := model.GetAlbumInsightsByLookup(
			ctx,
			model.AlbumInsightLookup{
				AlbumID: current.AlbumID,
				Artist:  current.Artist,
				Album:   current.Album,
			},
		)
		if err != nil {
			return nil, err
		}
		ids := make([]int64, 0, len(insights))
		for _, insight := range insights {
			ids = append(ids, insight.ID)
		}
		totalScoreMap, err := model.GetAlbumInsightsTotalScores(ctx, ids)
		if err != nil {
			return nil, err
		}
		sortAlbumInsightsWithScoreMap(insights, totalScoreMap)
		if len(insights) > limit {
			insights = insights[:limit]
		}
		scoreMap, err := model.GetAlbumInsightLatestFeedbackScores(ctx, ids)
		if err != nil {
			return nil, err
		}
		return convertAlbumInsightHistoryList(insights, scoreMap), nil
	default:
		current, err := getInsightByID(ctx, insightID)
		if err != nil {
			return nil, err
		}
		insights, err := model.GetTrackInsightsByLookup(
			ctx,
			model.TrackInsightLookup{
				TrackID: current.TrackID,
				Artist:  current.Artist,
				Album:   current.Album,
				Track:   current.Track,
			},
		)
		if err != nil {
			return nil, err
		}
		ids := make([]int64, 0, len(insights))
		for _, insight := range insights {
			ids = append(ids, insight.ID)
		}
		totalScoreMap, err := model.GetInsightsTotalScores(ctx, ids)
		if err != nil {
			return nil, err
		}
		sortTrackInsightsWithScoreMap(insights, totalScoreMap)
		if len(insights) > limit {
			insights = insights[:limit]
		}
		scoreMap, err := model.GetTrackInsightLatestFeedbackScores(ctx, ids)
		if err != nil {
			return nil, err
		}
		return convertTrackInsightHistoryList(insights, scoreMap), nil
	}
}

// GetLyrics 获取歌词内容，缺失时自动回源并复用歌词缓存写入逻辑。
func (s *serviceImpl) GetLyrics(
	ctx context.Context, artist, album, track string, trackNumber, discNumber int8,
) (*model.TrackLyrics, error) {
	artist = strings.TrimSpace(artist)
	album = strings.TrimSpace(album)
	track = strings.TrimSpace(track)

	if artist == "" || track == "" {
		return nil, errors.New("artist 和 track 不能为空")
	}

	lyricsText, err := s.getOrFetchLyrics(ctx, artist, album, track, trackNumber, discNumber)
	if err != nil {
		log.Warn(
			ctx, "获取歌词失败",
			zap.String("artist", artist),
			zap.String("track", track),
			zap.String("album", album),
			zap.Error(err),
		)
		return nil, err
	}

	trackObj, _ := model.GetTrackByIdentity(ctx, artist, album, track, trackNumber, discNumber)
	lookup := model.TrackLyricsLookup{Artist: artist, Album: album, Track: track}
	if trackObj != nil {
		lookup.TrackID = trackObj.ID
	}

	lyricsData, lookupErr := model.GetTrackLyricsByLookup(ctx, lookup)
	if lookupErr == nil {
		normalizedSynced := lyrics.IsSyncedLRC(lyricsData.LyricsOriginal)
		if lyricsData.Synced != normalizedSynced {
			lyricsData.Synced = normalizedSynced
			if err := model.UpdateTrackLyrics(ctx, lyricsData); err != nil {
				log.Warn(ctx, "修正歌词同步标记失败", zap.Error(err))
			}
		}
		log.Info(
			ctx,
			"获取歌词完成",
			zap.String("artist", artist),
			zap.String("album", album),
			zap.String("track", track),
			zap.Bool("cache_hit", true),
		)
		return lyricsData, nil
	}

	log.Info(
		ctx,
		"获取歌词完成",
		zap.String("artist", artist),
		zap.String("album", album),
		zap.String("track", track),
		zap.Bool("cache_hit", false),
	)
	return &model.TrackLyrics{
		TrackID:        lookup.TrackID,
		Artist:         artist,
		Album:          album,
		Track:          track,
		LyricsOriginal: lyricsText,
		Synced:         lyrics.IsSyncedLRC(lyricsText),
	}, nil
}

func (s *serviceImpl) buildAlbumTrackContexts(
	ctx context.Context, detail *model.AlbumDetail,
) ([]ai.AlbumTrackContext, []int64, int, error) {
	tracksByID := make(map[int64]*model.Track, len(detail.Tracks))
	for _, track := range detail.Tracks {
		tracksByID[track.ID] = track
	}

	totalTracks := 0
	trackContexts := make([]ai.AlbumTrackContext, 0, len(detail.TrackAlbums))
	selectedInsightIDs := make([]int64, 0, len(detail.TrackAlbums))

	for _, trackAlbum := range detail.TrackAlbums {
		if trackAlbum.TrackID <= 0 {
			continue
		}
		totalTracks++

		trackObj := tracksByID[trackAlbum.TrackID]
		title := trackAlbum.Track
		artist := detail.Artist
		albumName := detail.Name
		if trackObj != nil {
			if strings.TrimSpace(trackObj.Track) != "" {
				title = trackObj.Track
			}
			if strings.TrimSpace(trackObj.Artist) != "" {
				artist = trackObj.Artist
			}
			if strings.TrimSpace(trackObj.Album) != "" {
				albumName = trackObj.Album
			}
		}

		insights, err := model.GetTrackInsightsByLookup(
			ctx,
			model.TrackInsightLookup{
				TrackID: trackAlbum.TrackID,
				Artist:  artist,
				Album:   albumName,
				Track:   title,
			},
		)
		if err != nil {
			return nil, nil, 0, err
		}

		selectedInsight, totalScore, err := selectBestTrackInsight(ctx, insights)
		if err != nil {
			return nil, nil, 0, err
		}
		if selectedInsight == nil {
			continue
		}

		sections := make(map[string]string)
		for key, value := range selectedInsight.AnalysisBySection {
			if key == "appreciate_analysis" {
				continue
			}
			if trimmed := truncatePromptText(value, 700); trimmed != "" {
				sections[key] = trimmed
			}
		}

		trackContexts = append(
			trackContexts, ai.AlbumTrackContext{
				TrackID:          trackAlbum.TrackID,
				DiscNumber:       trackAlbum.DiscNumber,
				TrackNumber:      trackAlbum.TrackNumber,
				Title:            title,
				InsightID:        selectedInsight.ID,
				InsightScore:     totalScore,
				AnalysisSummary:  truncatePromptText(selectedInsight.AnalysisSummary, 1000),
				BackgroundInfo:   truncatePromptText(selectedInsight.BackgroundInfo, 700),
				EraContext:       truncatePromptText(selectedInsight.EraContext, 700),
				AnalysisSections: sections,
			},
		)
		selectedInsightIDs = append(selectedInsightIDs, selectedInsight.ID)
	}

	return trackContexts, selectedInsightIDs, totalTracks, nil
}

func selectBestTrackInsight(
	ctx context.Context, insights []*model.TrackInsight,
) (*model.TrackInsight, int, error) {
	if len(insights) == 0 {
		return nil, 0, nil
	}

	ids := make([]int64, 0, len(insights))
	for _, insight := range insights {
		ids = append(ids, insight.ID)
	}

	scoreMap, err := model.GetInsightsTotalScores(ctx, ids)
	if err != nil {
		return nil, 0, err
	}

	selected := pickBestTrackInsightWithScores(insights, scoreMap)
	if selected == nil {
		return nil, 0, nil
	}
	return selected, scoreMap[selected.ID], nil
}

func pickBestTrackInsightWithScores(
	insights []*model.TrackInsight, scoreMap map[int64]int,
) *model.TrackInsight {
	if len(insights) == 0 {
		return nil
	}

	sort.SliceStable(
		insights, func(i, j int) bool {
			leftScore := scoreMap[insights[i].ID]
			rightScore := scoreMap[insights[j].ID]
			if leftScore != rightScore {
				return leftScore > rightScore
			}
			return insights[i].CreatedAt.After(insights[j].CreatedAt)
		},
	)

	return insights[0]
}

func sortTrackInsightsByTotalScore(ctx context.Context, insights []*model.TrackInsight) error {
	ids := make([]int64, 0, len(insights))
	for _, insight := range insights {
		if insight == nil {
			continue
		}
		ids = append(ids, insight.ID)
	}
	scoreMap, err := model.GetInsightsTotalScores(ctx, ids)
	if err != nil {
		return err
	}
	sortTrackInsightsWithScoreMap(insights, scoreMap)
	return nil
}

func sortAlbumInsightsByTotalScore(ctx context.Context, insights []*model.AlbumInsight) error {
	ids := make([]int64, 0, len(insights))
	for _, insight := range insights {
		if insight == nil {
			continue
		}
		ids = append(ids, insight.ID)
	}
	scoreMap, err := model.GetAlbumInsightsTotalScores(ctx, ids)
	if err != nil {
		return err
	}
	sortAlbumInsightsWithScoreMap(insights, scoreMap)
	return nil
}

func sortTrackInsightsWithScoreMap(insights []*model.TrackInsight, scoreMap map[int64]int) {
	sort.SliceStable(
		insights, func(i, j int) bool {
			leftScore := scoreMap[insights[i].ID]
			rightScore := scoreMap[insights[j].ID]
			if leftScore != rightScore {
				return leftScore > rightScore
			}
			return insights[i].CreatedAt.After(insights[j].CreatedAt)
		},
	)
}

func sortAlbumInsightsWithScoreMap(insights []*model.AlbumInsight, scoreMap map[int64]int) {
	sort.SliceStable(
		insights, func(i, j int) bool {
			leftScore := scoreMap[insights[i].ID]
			rightScore := scoreMap[insights[j].ID]
			if leftScore != rightScore {
				return leftScore > rightScore
			}
			return insights[i].CreatedAt.After(insights[j].CreatedAt)
		},
	)
}

func truncatePromptText(text string, limit int) string {
	text = strings.TrimSpace(text)
	if text == "" || limit <= 0 {
		return ""
	}

	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return strings.TrimSpace(string(runes[:limit])) + "..."
}

func normalizeReasonCodes(reasonCodes []string) model.StringArray {
	return model.StringArray(common.NormalizeInsightFeedbackReasons(reasonCodes))
}

func convertTrackFeedbackHistory(feedbacks []*model.TrackInsightFeedback) []*InsightFeedbackHistoryItem {
	items := make([]*InsightFeedbackHistoryItem, 0, len(feedbacks))
	for _, feedback := range feedbacks {
		if feedback == nil {
			continue
		}
		items = append(
			items, &InsightFeedbackHistoryItem{
				ID:             feedback.ID,
				InsightID:      feedback.InsightID,
				Score:          feedback.Score,
				Comment:        feedback.Comment,
				ReasonCodes:    normalizeReasonCodes(feedback.ReasonCodes),
				SectionKey:     feedback.SectionKey,
				SourcePlatform: feedback.SourcePlatform,
				CreatedAt:      feedback.CreatedAt,
			},
		)
	}
	return items
}

func convertAlbumFeedbackHistory(feedbacks []*model.AlbumInsightFeedback) []*InsightFeedbackHistoryItem {
	items := make([]*InsightFeedbackHistoryItem, 0, len(feedbacks))
	for _, feedback := range feedbacks {
		if feedback == nil {
			continue
		}
		items = append(
			items, &InsightFeedbackHistoryItem{
				ID:             feedback.ID,
				InsightID:      feedback.InsightID,
				Score:          feedback.Score,
				Comment:        feedback.Comment,
				ReasonCodes:    normalizeReasonCodes(feedback.ReasonCodes),
				SectionKey:     feedback.SectionKey,
				SourcePlatform: feedback.SourcePlatform,
				CreatedAt:      feedback.CreatedAt,
			},
		)
	}
	return items
}

func buildFeedbackSummaryFromTrack(
	insightID int64,
	targetType common.AnalysisTargetType,
	likeCount, dislikeCount int64,
	feedbacks []*model.TrackInsightFeedback,
) *InsightFeedbackSummary {
	history := convertTrackFeedbackHistory(feedbacks)
	return buildFeedbackSummary(insightID, targetType, likeCount, dislikeCount, history)
}

func buildFeedbackSummaryFromAlbum(
	insightID int64,
	targetType common.AnalysisTargetType,
	likeCount, dislikeCount int64,
	feedbacks []*model.AlbumInsightFeedback,
) *InsightFeedbackSummary {
	history := convertAlbumFeedbackHistory(feedbacks)
	return buildFeedbackSummary(insightID, targetType, likeCount, dislikeCount, history)
}

func buildFeedbackSummary(
	insightID int64,
	targetType common.AnalysisTargetType,
	likeCount, dislikeCount int64,
	history []*InsightFeedbackHistoryItem,
) *InsightFeedbackSummary {
	summary := &InsightFeedbackSummary{
		InsightID:          insightID,
		AnalysisTargetType: targetType,
		LikeCount:          likeCount,
		DislikeCount:       dislikeCount,
		HasFeedback:        len(history) > 0,
		TopReasonCodes:     []string{},
	}
	if len(history) == 0 {
		return summary
	}

	summary.LatestFeedback = history[0]
	reasonCounts := make(map[string]int)
	for _, item := range history {
		if item == nil {
			continue
		}
		if summary.LatestNegativeFeedback == nil && item.Score < 0 {
			summary.LatestNegativeFeedback = item
		}
		for _, reason := range item.ReasonCodes {
			if trimmed := strings.TrimSpace(reason); trimmed != "" {
				reasonCounts[trimmed]++
			}
		}
	}
	if len(reasonCounts) == 0 {
		return summary
	}

	type reasonCount struct {
		Reason string
		Count  int
	}
	pairs := make([]reasonCount, 0, len(reasonCounts))
	for reason, count := range reasonCounts {
		pairs = append(pairs, reasonCount{Reason: reason, Count: count})
	}
	sort.Slice(
		pairs, func(i, j int) bool {
			if pairs[i].Count != pairs[j].Count {
				return pairs[i].Count > pairs[j].Count
			}
			return pairs[i].Reason < pairs[j].Reason
		},
	)
	limit := 3
	if len(pairs) < limit {
		limit = len(pairs)
	}
	summary.TopReasonCodes = make([]string, 0, limit)
	for _, pair := range pairs[:limit] {
		summary.TopReasonCodes = append(summary.TopReasonCodes, pair.Reason)
	}

	return summary
}

func buildTrackNegativeFeedbackContext(feedbacks []*model.TrackInsightFeedback) string {
	return buildNegativeFeedbackContext(convertTrackFeedbackHistory(feedbacks))
}

func buildAlbumNegativeFeedbackContext(feedbacks []*model.AlbumInsightFeedback) string {
	return buildNegativeFeedbackContext(convertAlbumFeedbackHistory(feedbacks))
}

func buildNegativeFeedbackContext(history []*InsightFeedbackHistoryItem) string {
	if len(history) == 0 {
		return ""
	}

	reasonCounts := make(map[string]int)
	sectionCounts := make(map[string]int)
	commentLines := make([]string, 0, 3)
	for _, item := range history {
		if item == nil || item.Score >= 0 {
			continue
		}
		for _, reason := range item.ReasonCodes {
			if trimmed := strings.TrimSpace(reason); trimmed != "" {
				reasonCounts[trimmed]++
			}
		}
		if section := strings.TrimSpace(item.SectionKey); section != "" {
			sectionCounts[section]++
		}
		if comment := truncatePromptText(item.Comment, 180); comment != "" && len(commentLines) < 3 {
			commentLines = append(commentLines, comment)
		}
	}
	if len(reasonCounts) == 0 && len(sectionCounts) == 0 && len(commentLines) == 0 {
		return ""
	}

	lines := []string{"用户之前对分析提出过这些负反馈，请在本次结果中尽量避免重复这些问题："}
	if topReasons := topCountKeys(reasonCounts, 4); len(topReasons) > 0 {
		lines = append(lines, "高频问题标签："+strings.Join(topReasons, "、"))
	}
	if sections := topCountKeys(sectionCounts, 3); len(sections) > 0 {
		lines = append(lines, "重点被指出的问题分区："+strings.Join(sections, "、"))
	}
	for index, comment := range commentLines {
		lines = append(lines, fmt.Sprintf("最近负反馈 %d：%s", index+1, comment))
	}

	return strings.Join(lines, "\n")
}

func topCountKeys(counts map[string]int, limit int) []string {
	if len(counts) == 0 || limit <= 0 {
		return nil
	}
	type pair struct {
		Key   string
		Count int
	}
	pairs := make([]pair, 0, len(counts))
	for key, count := range counts {
		pairs = append(pairs, pair{Key: key, Count: count})
	}
	sort.Slice(
		pairs, func(i, j int) bool {
			if pairs[i].Count != pairs[j].Count {
				return pairs[i].Count > pairs[j].Count
			}
			return pairs[i].Key < pairs[j].Key
		},
	)
	if len(pairs) < limit {
		limit = len(pairs)
	}
	keys := make([]string, 0, limit)
	for _, item := range pairs[:limit] {
		keys = append(keys, item.Key)
	}
	return keys
}

func convertTrackInsightHistoryList(
	insights []*model.TrackInsight,
	scoreMap map[int64]int,
) []*model.InsightListItem {
	items := make([]*model.InsightListItem, 0, len(insights))
	for _, insight := range insights {
		if insight == nil {
			continue
		}
		items = append(
			items, &model.InsightListItem{
				ID:                  insight.ID,
				AnalysisTargetType:  common.AnalysisTargetTypeTrack,
				TrackID:             insight.TrackID,
				AlbumID:             0,
				Artist:              insight.Artist,
				Album:               insight.Album,
				Track:               insight.Track,
				AnalysisSummary:     insight.AnalysisSummary,
				LLMProvider:         insight.LLMProvider,
				LikeCount:           insight.LikeCount,
				DislikeCount:        insight.DislikeCount,
				LatestFeedbackScore: scoreMap[insight.ID],
				CreatedAt:           insight.CreatedAt,
				IsDisabled:          insight.IsDisabled,
			},
		)
	}
	return items
}

func convertAlbumInsightHistoryList(
	insights []*model.AlbumInsight,
	scoreMap map[int64]int,
) []*model.InsightListItem {
	items := make([]*model.InsightListItem, 0, len(insights))
	for _, insight := range insights {
		if insight == nil {
			continue
		}
		items = append(
			items, &model.InsightListItem{
				ID:                  insight.ID,
				AnalysisTargetType:  common.AnalysisTargetTypeAlbum,
				TrackID:             0,
				AlbumID:             insight.AlbumID,
				Artist:              insight.Artist,
				Album:               insight.Album,
				Track:               "",
				AnalysisSummary:     insight.AnalysisSummary,
				LLMProvider:         insight.LLMProvider,
				LikeCount:           insight.LikeCount,
				DislikeCount:        insight.DislikeCount,
				LatestFeedbackScore: scoreMap[insight.ID],
				CreatedAt:           insight.CreatedAt,
				IsDisabled:          insight.IsDisabled,
			},
		)
	}
	return items
}
