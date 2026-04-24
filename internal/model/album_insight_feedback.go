package model

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// AlbumInsightFeedback 存储用户对专辑音眸结果的反馈，便于和曲目反馈链路独立演进。
type AlbumInsightFeedback struct {
	ID             int64       `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	InsightID      int64       `gorm:"column:insight_id;type:bigint;not null;index" json:"insight_id"`
	Score          int         `gorm:"column:score;type:int;not null" json:"score"`
	Comment        string      `gorm:"column:comment;type:text" json:"comment"`
	ReasonCodes    StringArray `gorm:"column:reason_codes;type:longtext" json:"reason_codes"`
	SectionKey     string      `gorm:"column:section_key;type:varchar(128);not null;default:''" json:"section_key"`
	SourcePlatform string      `gorm:"column:source_platform;type:varchar(64);not null;default:''" json:"source_platform"`

	CreatedAt time.Time `gorm:"column:created_at;type:timestamp;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

// TableName 自定义表名，确保专辑反馈不和曲目反馈混表。
func (AlbumInsightFeedback) TableName() string {
	return "album_insight_feedbacks"
}

// CreateAlbumInsightFeedback 写入专辑反馈记录。
func CreateAlbumInsightFeedback(ctx context.Context, feedback *AlbumInsightFeedback) error {
	return GetDB().WithContext(ctx).Create(feedback).Error
}

// GetAlbumInsightFeedbacks 获取某次专辑解析的全部反馈，按时间倒序返回。
func GetAlbumInsightFeedbacks(ctx context.Context, insightID int64) ([]*AlbumInsightFeedback, error) {
	var feedbacks []*AlbumInsightFeedback
	err := GetDB().WithContext(ctx).
		Where("insight_id = ?", insightID).
		Order("created_at DESC, id DESC").
		Find(&feedbacks).Error
	return feedbacks, err
}

// GetAlbumInsightFeedbacksLimited 获取某次专辑解析的最近若干条反馈，默认按时间倒序返回。
func GetAlbumInsightFeedbacksLimited(ctx context.Context, insightID int64, limit int) ([]*AlbumInsightFeedback, error) {
	if limit <= 0 {
		return GetAlbumInsightFeedbacks(ctx, insightID)
	}

	var feedbacks []*AlbumInsightFeedback
	err := GetDB().WithContext(ctx).
		Where("insight_id = ?", insightID).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&feedbacks).Error
	return feedbacks, err
}

// GetNegativeAlbumFeedbacksByLookup 获取某张专辑对应的差评反馈，供下次分析注入上下文。
func GetNegativeAlbumFeedbacksByLookup(ctx context.Context, lookup AlbumInsightLookup) ([]*AlbumInsightFeedback, error) {
	var feedbacks []*AlbumInsightFeedback
	base := GetDB().WithContext(ctx).
		Table("album_insight_feedbacks").
		Joins("JOIN album_insight ON album_insight_feedbacks.insight_id = album_insight.id").
		Where("album_insight_feedbacks.score < 0").
		Order("album_insight_feedbacks.created_at DESC")

	if lookup.AlbumID > 0 {
		err := base.Session(&gorm.Session{}).
			Where("album_insight.album_id = ?", lookup.AlbumID).
			Find(&feedbacks).Error
		if err == nil && len(feedbacks) > 0 {
			return feedbacks, nil
		}
		if err != nil {
			return nil, err
		}
	}

	err := base.Session(&gorm.Session{}).
		Where("album_insight.artist = ? AND album_insight.album = ?", lookup.Artist, lookup.Album).
		Find(&feedbacks).Error
	if err != nil {
		return nil, err
	}
	return feedbacks, nil
}

// GetAlbumInsightLatestFeedbackScores 获取指定专辑音眸的最新反馈分值，供列表轻量状态展示。
func GetAlbumInsightLatestFeedbackScores(ctx context.Context, insightIDs []int64) (map[int64]int, error) {
	result := make(map[int64]int)
	if len(insightIDs) == 0 {
		return result, nil
	}

	type latestFeedbackRow struct {
		InsightID int64 `gorm:"column:insight_id"`
		Score     int   `gorm:"column:score"`
	}

	var rows []latestFeedbackRow
	err := GetDB().WithContext(ctx).
		Model(&AlbumInsightFeedback{}).
		Select("insight_id, score").
		Where("insight_id IN ?", insightIDs).
		Order("created_at DESC, id DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		if _, ok := result[row.InsightID]; ok {
			continue
		}
		result[row.InsightID] = row.Score
	}

	return result, nil
}

// GetAlbumInsightsTotalScores 获取指定专辑音眸的累计反馈分值，供推荐版本排序使用。
func GetAlbumInsightsTotalScores(ctx context.Context, insightIDs []int64) (map[int64]int, error) {
	result := make(map[int64]int)
	if len(insightIDs) == 0 {
		return result, nil
	}

	type totalScoreRow struct {
		InsightID  int64 `gorm:"column:insight_id"`
		TotalScore int   `gorm:"column:total_score"`
	}

	var rows []totalScoreRow
	err := GetDB().WithContext(ctx).
		Model(&AlbumInsightFeedback{}).
		Select("insight_id, SUM(score) AS total_score").
		Where("insight_id IN ?", insightIDs).
		Group("insight_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		result[row.InsightID] = row.TotalScore
	}

	return result, nil
}
