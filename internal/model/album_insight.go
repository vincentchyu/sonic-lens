package model

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// AlbumInsight 存储专辑级 AI 深度分析结果。
type AlbumInsight struct {
	ID      int64  `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	AlbumID int64  `gorm:"column:album_id;type:bigint;index" json:"album_id"`
	Artist  string `gorm:"column:artist;type:varchar(255);index:idx_album_insight_artist_album" json:"artist"`
	Album   string `gorm:"column:album;type:varchar(255);index:idx_album_insight_artist_album" json:"album"`

	AnalysisSummary   string   `gorm:"column:analysis_summary;type:text" json:"analysis_summary"`
	AnalysisBySection JSONText `gorm:"type:text" json:"analysis_by_section"`
	BackgroundInfo    string   `gorm:"column:background_info;type:text" json:"background_info"`
	EraContext        string   `gorm:"column:era_context;type:text" json:"era_context"`
	LLMProvider       string   `gorm:"column:llm_provider;type:varchar(255)" json:"llm_provider"`
	Metadata          string   `gorm:"column:metadata;type:text" json:"metadata,omitempty"`

	LikeCount    int64 `gorm:"column:like_count;type:bigint;default:0" json:"like_count"`
	DislikeCount int64 `gorm:"column:dislike_count;type:bigint;default:0" json:"dislike_count"`

	CreatedAt  time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
	LastUsedAt time.Time `gorm:"column:last_used_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"last_used_at"`

	IsDisabled bool `gorm:"column:is_disabled;type:tinyint(1);default:0;index" json:"is_disabled"`
}

type AlbumInsightLookup struct {
	AlbumID int64
	Artist  string
	Album   string
}

// TableName 自定义表名。
func (AlbumInsight) TableName() string {
	return "album_insight"
}

func CreateAlbumInsight(ctx context.Context, insight *AlbumInsight) error {
	return GetDB().WithContext(ctx).Create(insight).Error
}

func GetAlbumInsightByLookup(ctx context.Context, lookup AlbumInsightLookup) (*AlbumInsight, error) {
	var insight AlbumInsight
	base := GetDB().WithContext(ctx).Where("is_disabled = ?", false)

	if lookup.AlbumID > 0 {
		err := base.Session(&gorm.Session{}).
			Where("album_id = ?", lookup.AlbumID).
			Order("created_at DESC").
			First(&insight).Error
		if err == nil {
			return &insight, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	err := base.Session(&gorm.Session{}).
		Where("artist = ? AND album = ?", lookup.Artist, lookup.Album).
		Order("created_at DESC").
		First(&insight).Error
	if err != nil {
		return nil, err
	}
	return &insight, nil
}

// GetAlbumInsightsByLookup 获取指定专辑下的全部分析结果，默认最新优先。
func GetAlbumInsightsByLookup(ctx context.Context, lookup AlbumInsightLookup) ([]*AlbumInsight, error) {
	var insights []*AlbumInsight
	base := GetDB().WithContext(ctx).Where("is_disabled = ?", false).Order("created_at DESC")

	if lookup.AlbumID > 0 {
		err := base.Session(&gorm.Session{}).Where("album_id = ?", lookup.AlbumID).Find(&insights).Error
		if err == nil && len(insights) > 0 {
			return insights, nil
		}
		if err != nil {
			return nil, err
		}
	}

	err := base.Session(&gorm.Session{}).
		Where("artist = ? AND album = ?", lookup.Artist, lookup.Album).
		Find(&insights).Error
	if err != nil {
		return nil, err
	}
	return insights, nil
}

func UpdateAlbumInsight(ctx context.Context, insight *AlbumInsight) error {
	return GetDB().WithContext(ctx).Save(insight).Error
}
