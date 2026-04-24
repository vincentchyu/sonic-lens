package model

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/common"
)

// LLMCallLog 大模型调用流水表，记录每次请求/响应的完整 JSON 数据，用于排查和恢复现场
type LLMCallLog struct {
	ID                 int64                     `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	Provider           string                    `gorm:"column:provider;type:varchar(64);index" json:"provider"`                                                                           // 提供方：doubao/ollama/openai
	Model              string                    `gorm:"column:model;type:varchar(128)" json:"model"`                                                                                      // 模型名称
	RequestJSON        string                    `gorm:"column:request_json;type:longtext" json:"request_json"`                                                                            // 请求体全文 JSON
	ResponseJSON       string                    `gorm:"column:response_json;type:longtext" json:"response_json"`                                                                          // 响应体全文 JSON
	Status             string                    `gorm:"column:status;type:varchar(32);index" json:"status"`                                                                               // 调用状态：success/error
	ErrorMsg           string                    `gorm:"column:error_msg;type:text" json:"error_msg"`                                                                                      // 错误信息
	DurationMs         int64                     `gorm:"column:duration_ms;type:bigint" json:"duration_ms"`                                                                                // 调用耗时（毫秒）
	AnalysisTargetType common.AnalysisTargetType `gorm:"column:analysis_target_type;type:varchar(32);not null;default:'track';index:idx_llm_logs_target_type" json:"analysis_target_type"` // 分析对象类型：track/album
	TargetKey          string                    `gorm:"column:target_key;type:varchar(512);not null;default:'';index:idx_llm_logs_target_key" json:"target_key"`                          // 对象唯一标识
	TargetMetadata     string                    `gorm:"column:target_metadata;type:longtext" json:"target_metadata"`                                                                      // 对象元数据 JSON
	TrackInfo          string                    `gorm:"column:track_info;type:varchar(512);index" json:"track_info"`                                                                      // 兼容展示字段
	CallType           string                    `gorm:"column:call_type;type:varchar(32)" json:"call_type"`                                                                               // 调用类型：sync/stream
	CreatedAt          time.Time                 `gorm:"column:created_at;type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
}

// TableName 自定义表名
func (LLMCallLog) TableName() string {
	return "llm_call_logs"
}

// CreateLLMCallLog 插入一条调用流水记录
func CreateLLMCallLog(ctx context.Context, log *LLMCallLog) error {
	return GetDB().WithContext(ctx).Create(log).Error
}

// BuildLLMCallLogTrackKey 构造曲目日志的统一查询键，避免只靠 track 名称误伤同名曲目。
func BuildLLMCallLogTrackKey(artist, album, track string) string {
	return strings.TrimSpace(artist) + "||" + strings.TrimSpace(album) + "||" + strings.TrimSpace(track)
}

// BuildLLMCallLogAlbumKey 构造专辑日志的统一查询键。
func BuildLLMCallLogAlbumKey(albumID int64) string {
	return fmt.Sprintf("%d", albumID)
}

// GetLLMCallLogsByTarget 按对象类型与查询键获取流水日志。
func GetLLMCallLogsByTarget(ctx context.Context, targetType common.AnalysisTargetType, targetKey string, limit int) ([]*LLMCallLog, error) {
	var logs []*LLMCallLog
	err := GetDB().WithContext(ctx).
		Where("analysis_target_type = ? AND target_key = ?", targetType, targetKey).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// GetLLMCallLogsByTrackInfo 按兼容展示键查询流水日志。
// 主要用于旧数据回放：早期日志没有 target_key 时，只能依赖 track_info 兜底查询。
func GetLLMCallLogsByTrackInfo(ctx context.Context, targetType common.AnalysisTargetType, trackInfo string, limit int) ([]*LLMCallLog, error) {
	var logs []*LLMCallLog
	trackInfo = strings.TrimSpace(trackInfo)
	if trackInfo == "" {
		return logs, nil
	}

	err := GetDB().WithContext(ctx).
		Where("analysis_target_type = ? AND track_info = ?", targetType, trackInfo).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// GetLLMCallLogsByTrack 按曲目查询流水日志。
func GetLLMCallLogsByTrack(ctx context.Context, artist, album, track string, limit int) ([]*LLMCallLog, error) {
	return GetLLMCallLogsByTarget(ctx, common.AnalysisTargetTypeTrack, BuildLLMCallLogTrackKey(artist, album, track), limit)
}

// GetLLMCallLogsByAlbumID 按专辑 ID 查询流水日志。
func GetLLMCallLogsByAlbumID(ctx context.Context, albumID int64, limit int) ([]*LLMCallLog, error) {
	return GetLLMCallLogsByTarget(ctx, common.AnalysisTargetTypeAlbum, BuildLLMCallLogAlbumKey(albumID), limit)
}

// DeleteLLMCallLogsByTarget 删除某个解析对象关联的所有调用流水。
func DeleteLLMCallLogsByTarget(ctx context.Context, targetType common.AnalysisTargetType, targetKey, legacyTrackInfo string) error {
	return deleteLLMCallLogsByTargetTx(GetDB().WithContext(ctx), targetType, targetKey, legacyTrackInfo)
}

func deleteLLMCallLogsByTargetTx(tx *gorm.DB, targetType common.AnalysisTargetType, targetKey, legacyTrackInfo string) error {
	if tx == nil {
		return nil
	}

	targetKey = strings.TrimSpace(targetKey)
	legacyTrackInfo = strings.TrimSpace(legacyTrackInfo)
	db := tx.Session(&gorm.Session{NewDB: true, SkipHooks: true})

	if targetKey != "" {
		if err := db.Where("analysis_target_type = ? AND target_key = ?", targetType, targetKey).Delete(&LLMCallLog{}).Error; err != nil {
			return err
		}
	}

	if legacyTrackInfo != "" && legacyTrackInfo != targetKey {
		if err := db.Where("analysis_target_type = ? AND track_info = ?", targetType, legacyTrackInfo).Delete(&LLMCallLog{}).Error; err != nil {
			return err
		}
	}

	return nil
}
