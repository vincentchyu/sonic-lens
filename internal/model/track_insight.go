package model

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// TrackInsight 存储单首歌曲的 AI 歌词解析结果
type TrackInsight struct {
	ID      int64  `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	TrackID int64  `gorm:"column:track_id;type:bigint;index" json:"track_id"` // 关联 track 表
	Artist  string `gorm:"column:artist;type:varchar(255);index:idx_insight_artist_album_track" json:"artist"`
	Album   string `gorm:"column:album;type:varchar(255);index:idx_insight_artist_album_track" json:"album"`
	Track   string `gorm:"column:track;type:varchar(255);index:idx_insight_artist_album_track" json:"track"`
	// 信雅达式中文翻译
	LyricsTranslation string `gorm:"column:lyrics_translation;type:text" json:"lyrics_translation"`
	// 总体摘要性解读
	AnalysisSummary string `gorm:"column:analysis_summary;type:text" json:"analysis_summary"`
	// 按段落/主题的细粒度解析，JSON 字符串
	AnalysisBySection JSONText `gorm:"type:text" json:"analysis_by_section"` // 歌曲或专辑的创作背景
	BackgroundInfo    string   `gorm:"column:background_info;type:text" json:"background_info"`
	// 时代语境、文化/社会背景
	EraContext string `gorm:"column:era_context;type:text" json:"era_context"`
	// 使用的大模型提供方，例如 openai:gpt-4.1、gemini:1.5-pro 等
	LLMProvider string `gorm:"column:llm_provider;type:varchar(255)" json:"llm_provider"`
	// 原文与目标语言
	LangSource string `gorm:"column:lang_source;type:varchar(32)" json:"lang_source"`
	LangTarget string `gorm:"column:lang_target;type:varchar(32)" json:"lang_target"`
	// 额外元信息，例如引用资料链接、检索摘要等，JSON 字符串
	Metadata string `gorm:"column:metadata;type:text" json:"metadata"`

	// 简单的统计字段，便于后期快速过滤
	LikeCount    int64 `gorm:"column:like_count;type:bigint;default:0" json:"like_count"`
	DislikeCount int64 `gorm:"column:dislike_count;type:bigint;default:0" json:"dislike_count"`

	CreatedAt  time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
	LastUsedAt time.Time `gorm:"column:last_used_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"last_used_at"`
}

type JSONText map[string]string

func (j JSONText) Value() (driver.Value, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	err := enc.Encode(j)
	return strings.TrimSpace(buf.String()), err
}

func (j *JSONText) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("invalid scan")
	}
	return json.Unmarshal(b, j)
}

// TableName 自定义表名
func (TrackInsight) TableName() string {
	return "track_insight"
}

// ToSimplifiedJSON 将结果转换为简化版 JSON 字符串，模拟 AI 返回格式
func (ti *TrackInsight) ToSimplifiedJSON() string {
	res := map[string]interface{}{
		"lyrics_translation":  ti.LyricsTranslation,
		"analysis_by_section": ti.AnalysisBySection,
		"analysis_summary":    ti.AnalysisSummary,
		"background_info":     ti.BackgroundInfo,
		"era_context":         ti.EraContext,
		"llm_provider":        ti.LLMProvider,
		"metadata":            ti.Metadata,
	}
	b, _ := json.Marshal(res)
	return string(b)
}

// TrackInsightFeedback 存储用户对某次解析结果的反馈
type TrackInsightFeedback struct {
	ID        int64 `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	InsightID int64 `gorm:"column:insight_id;type:bigint;index" json:"insight_id"`
	// 评分：+1 点赞，-1 点踩
	Score   int    `gorm:"column:score;type:int" json:"score"`
	Comment string `gorm:"column:comment;type:text" json:"comment"`

	CreatedAt time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func CreateTrackInsight(ctx context.Context, insight *TrackInsight) error {
	return GetDB().WithContext(ctx).Create(insight).Error
}

func GetTrackInsight(ctx context.Context, artist, album, track string) (*TrackInsight, error) {
	var insight TrackInsight
	err := GetDB().WithContext(ctx).
		Where("artist = ? AND album = ? AND track = ?", artist, album, track).
		First(&insight).Error
	if err != nil {
		return nil, err
	}
	return &insight, nil
}

func UpdateTrackInsight(ctx context.Context, insight *TrackInsight) error {
	return GetDB().WithContext(ctx).Save(insight).Error
}

func GetTrackInsights(ctx context.Context, artist, album, track string) ([]*TrackInsight, error) {
	var insights []*TrackInsight
	err := GetDB().WithContext(ctx).
		Where("artist = ? AND album = ? AND track = ?", artist, album, track).
		Order("created_at DESC").
		Find(&insights).Error
	if err != nil {
		return nil, err
	}
	return insights, nil
}

func CreateTrackInsightFeedback(ctx context.Context, feedback *TrackInsightFeedback) error {
	return GetDB().WithContext(ctx).Create(feedback).Error
}

func GetInsightsTotalScores(ctx context.Context, insightIDs []int64) (map[int64]int, error) {
	if len(insightIDs) == 0 {
		return make(map[int64]int), nil
	}
	type scoreResult struct {
		InsightID  int64 `gorm:"column:insight_id"`
		TotalScore int   `gorm:"column:total_score"`
	}
	var results []scoreResult
	err := GetDB().WithContext(ctx).
		Model(&TrackInsightFeedback{}).
		Where("insight_id IN ?", insightIDs).
		Select("insight_id, SUM(score) as total_score").
		Group("insight_id").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	scoreMap := make(map[int64]int)
	for _, r := range results {
		scoreMap[r.InsightID] = r.TotalScore
	}
	return scoreMap, nil
}

func GetNegativeFeedbacksByTrack(ctx context.Context, artist, album, track string) ([]*TrackInsightFeedback, error) {
	var feedbacks []*TrackInsightFeedback
	err := GetDB().WithContext(ctx).
		Table("track_insight_feedbacks").
		Joins("JOIN track_insight ON track_insight_feedbacks.insight_id = track_insight.id").
		Where("track_insight.artist = ? AND track_insight.album = ? AND track_insight.track = ?", artist, album, track).
		Where("track_insight_feedbacks.score < 0").
		Order("track_insight_feedbacks.created_at DESC").
		Find(&feedbacks).Error
	if err != nil {
		return nil, err
	}
	return feedbacks, nil
}
