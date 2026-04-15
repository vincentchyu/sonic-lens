package model

import (
	"context"
	"strings"
	"time"

	"github.com/vincentchyu/sonic-lens/common"
)

// InsightListItem 是音眸列表的统一摘要结构，兼容曲目与专辑两类分析对象。
type InsightListItem struct {
	ID                  int64                     `json:"id"`
	AnalysisTargetType  common.AnalysisTargetType `json:"analysis_target_type"`
	TrackID             int64                     `json:"track_id,omitempty"`
	AlbumID             int64                     `json:"album_id,omitempty"`
	Artist              string                    `json:"artist"`
	Album               string                    `json:"album"`
	Track               string                    `json:"track"`
	AnalysisSummary     string                    `json:"analysis_summary"`
	LLMProvider         string                    `json:"llm_provider"`
	LikeCount           int64                     `json:"like_count"`
	DislikeCount        int64                     `json:"dislike_count"`
	LatestFeedbackScore int                       `json:"latest_feedback_score"`
	CreatedAt           time.Time                 `json:"created_at"`
	IsDisabled          bool                      `json:"is_disabled"`
}

// GetAllInsightSummaries 按分析对象类型获取统一的音眸列表摘要。
func GetAllInsightSummaries(
	ctx context.Context,
	targetType common.AnalysisTargetType,
	limit, offset int,
	keyword string,
) ([]*InsightListItem, int64, error) {
	switch targetType {
	case common.AnalysisTargetTypeAlbum:
		return getAllAlbumInsightSummaries(ctx, limit, offset, keyword)
	default:
		return getAllTrackInsightSummaries(ctx, limit, offset, keyword)
	}
}

func getAllTrackInsightSummaries(
	ctx context.Context,
	limit, offset int,
	keyword string,
) ([]*InsightListItem, int64, error) {
	var insights []*InsightListItem
	var total int64
	like := "%" + strings.TrimSpace(keyword) + "%"
	db := GetDB().WithContext(ctx).Model(&TrackInsight{})

	if trimmed := strings.TrimSpace(keyword); trimmed != "" {
		db = db.Where(
			"artist LIKE ? OR album LIKE ? OR track LIKE ?",
			like, like, like,
		)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.
		Select(
			"id, track_id, 0 AS album_id, artist, album, track, analysis_summary, llm_provider, like_count, dislike_count, created_at, is_disabled",
		).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&insights).Error
	if err != nil {
		return nil, 0, err
	}

	ids := make([]int64, 0, len(insights))
	for _, insight := range insights {
		ids = append(ids, insight.ID)
	}
	scoreMap, err := GetTrackInsightLatestFeedbackScores(ctx, ids)
	if err != nil {
		return nil, 0, err
	}

	for _, insight := range insights {
		insight.AnalysisTargetType = common.AnalysisTargetTypeTrack
		insight.LatestFeedbackScore = scoreMap[insight.ID]
	}
	return insights, total, nil
}

func getAllAlbumInsightSummaries(
	ctx context.Context,
	limit, offset int,
	keyword string,
) ([]*InsightListItem, int64, error) {
	var insights []*InsightListItem
	var total int64
	like := "%" + strings.TrimSpace(keyword) + "%"
	db := GetDB().WithContext(ctx).Model(&AlbumInsight{})

	if trimmed := strings.TrimSpace(keyword); trimmed != "" {
		db = db.Where(
			"artist LIKE ? OR album LIKE ?",
			like, like,
		)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.
		Select(
			"id, 0 AS track_id, album_id, artist, album, '' AS track, analysis_summary, llm_provider, like_count, dislike_count, created_at, is_disabled",
		).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&insights).Error
	if err != nil {
		return nil, 0, err
	}

	ids := make([]int64, 0, len(insights))
	for _, insight := range insights {
		ids = append(ids, insight.ID)
	}
	scoreMap, err := GetAlbumInsightLatestFeedbackScores(ctx, ids)
	if err != nil {
		return nil, 0, err
	}

	for _, insight := range insights {
		insight.AnalysisTargetType = common.AnalysisTargetTypeAlbum
		insight.LatestFeedbackScore = scoreMap[insight.ID]
	}
	return insights, total, nil
}
