package model

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/common"
)

// InsightJob 记录客户端发起的音眸异步任务，用于 WS 与客户端恢复。
type InsightJob struct {
	ID                    string                    `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	AnalysisTargetType    common.AnalysisTargetType `gorm:"column:analysis_target_type;type:varchar(32);not null;index:idx_insight_job_target_type" json:"analysis_target_type"`
	Status                common.InsightJobPhase    `gorm:"column:status;type:varchar(32);not null;index:idx_insight_job_status" json:"status"`
	TargetKey             string                    `gorm:"column:target_key;type:varchar(512);not null;index:idx_insight_job_target_key" json:"target_key"`
	AlbumID               int64                     `gorm:"column:album_id;type:bigint;index:idx_insight_job_album_id" json:"album_id"`
	Artist                string                    `gorm:"column:artist;type:varchar(255)" json:"artist"`
	Album                 string                    `gorm:"column:album;type:varchar(255)" json:"album"`
	Track                 string                    `gorm:"column:track;type:varchar(255)" json:"track"`
	TrackNumber           int8                      `gorm:"column:track_number;type:tinyint" json:"track_number"`
	DiscNumber            int8                      `gorm:"column:disc_number;type:tinyint" json:"disc_number"`
	Provider              string                    `gorm:"column:provider;type:varchar(64);not null" json:"provider"`
	Model                 string                    `gorm:"column:model;type:varchar(255);not null" json:"model"`
	ProviderDisplayName   string                    `gorm:"column:provider_display_name;type:varchar(128)" json:"provider_display_name"`
	ModelDisplayName      string                    `gorm:"column:model_display_name;type:varchar(255)" json:"model_display_name"`
	ClientPlatform        string                    `gorm:"column:client_platform;type:varchar(64)" json:"client_platform"`
	LiveActivityPushToken string                    `gorm:"column:live_activity_push_token;type:text" json:"-"`
	CoverArtURL           string                    `gorm:"column:cover_art_url;type:varchar(1024)" json:"cover_art_url"`
	ResultInsightID       *int64                    `gorm:"column:result_insight_id;type:bigint;index:idx_insight_job_result_insight_id" json:"result_insight_id"`
	ResultAvailable       bool                      `gorm:"column:result_available;type:tinyint(1);not null;default:0" json:"result_available"`
	ErrorMessage          string                    `gorm:"column:error_message;type:text" json:"error_message"`
	StartedAt             *time.Time                `gorm:"column:started_at;type:timestamp null" json:"started_at"`
	FinishedAt            *time.Time                `gorm:"column:finished_at;type:timestamp null" json:"finished_at"`
	CreatedAt             time.Time                 `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt             time.Time                 `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

// InsightJobListQuery 定义任务列表查询条件。
type InsightJobListQuery struct {
	Limit         int
	Offset        int
	Keyword       string
	Status        common.InsightJobPhase
	HasStatus     bool
	TargetType    common.AnalysisTargetType
	HasTargetType bool
}

// TableName 返回音眸任务表名。
func (InsightJob) TableName() string {
	return "insight_job"
}

// BuildTrackInsightJobTargetKey 构建曲目任务唯一键，便于幂等复用同目标任务。
func BuildTrackInsightJobTargetKey(artist, album, track string, trackNumber, discNumber int8) string {
	return fmt.Sprintf(
		"track::%s::%s::%s::%d::%d",
		strings.TrimSpace(artist),
		strings.TrimSpace(album),
		strings.TrimSpace(track),
		trackNumber,
		discNumber,
	)
}

// BuildAlbumInsightJobTargetKey 构建专辑任务唯一键。
func BuildAlbumInsightJobTargetKey(albumID int64) string {
	return fmt.Sprintf("album::%d", albumID)
}

// CreateInsightJob 创建新的音眸任务记录。
func CreateInsightJob(ctx context.Context, job *InsightJob) error {
	return GetDB().WithContext(ctx).Create(job).Error
}

// GetInsightJobByID 按主键读取任务。
func GetInsightJobByID(ctx context.Context, jobID string) (*InsightJob, error) {
	var job InsightJob
	if err := GetDB().WithContext(ctx).Where("id = ?", strings.TrimSpace(jobID)).First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// ListInsightJobs 分页查询音眸任务，供管理端列表使用。
func ListInsightJobs(ctx context.Context, query InsightJobListQuery) ([]*InsightJob, int64, error) {
	limit := query.Limit
	switch {
	case limit <= 0:
		limit = 20
	case limit > 200:
		limit = 200
	}

	offset := query.Offset
	if offset < 0 {
		offset = 0
	}

	db := GetDB().WithContext(ctx).Model(&InsightJob{})
	if query.HasStatus {
		db = db.Where("status = ?", query.Status)
	}
	if query.HasTargetType {
		db = db.Where("analysis_target_type = ?", query.TargetType)
	}
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where(
			`id LIKE ? OR target_key LIKE ? OR artist LIKE ? OR album LIKE ? OR track LIKE ? OR provider LIKE ? OR model LIKE ?`,
			like,
			like,
			like,
			like,
			like,
			like,
			like,
		)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var jobs []*InsightJob
	if err := db.Order("updated_at DESC").Order("created_at DESC").Limit(limit).Offset(offset).Find(&jobs).Error; err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

// GetLatestActiveInsightJobByTarget 查询同目标、同模型的最新活跃任务，用于避免重复创建。
func GetLatestActiveInsightJobByTarget(
	ctx context.Context,
	targetKey, provider, modelName string,
) (*InsightJob, error) {
	var job InsightJob
	err := GetDB().WithContext(ctx).
		Where(
			"target_key = ? AND provider = ? AND model = ? AND status IN ?",
			targetKey,
			strings.TrimSpace(provider),
			strings.TrimSpace(modelName),
			[]common.InsightJobPhase{common.InsightJobPhaseQueued, common.InsightJobPhaseRunning},
		).
		Order("created_at DESC").
		First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// UpdateInsightJobFieldsTx 在事务内更新任务字段，避免上层直接散落更新 SQL。
func UpdateInsightJobFieldsTx(tx *gorm.DB, jobID string, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}
	return tx.Model(&InsightJob{}).Where("id = ?", strings.TrimSpace(jobID)).Updates(fields).Error
}

// UpdateInsightJobFields 更新任务字段集合。
func UpdateInsightJobFields(ctx context.Context, jobID string, fields map[string]interface{}) error {
	return UpdateInsightJobFieldsTx(GetDB().WithContext(ctx), jobID, fields)
}

// UpdateInsightJobLiveActivityToken 更新任务关联的 Live Activity push token。
func UpdateInsightJobLiveActivityToken(ctx context.Context, jobID, token string) error {
	return UpdateInsightJobFields(
		ctx,
		jobID,
		map[string]interface{}{
			"live_activity_push_token": strings.TrimSpace(token),
		},
	)
}

// CancelInsightJob 将任务标记为取消，供管理端手动维护生命周期。
func CancelInsightJob(ctx context.Context, jobID, reason string) (*InsightJob, error) {
	job, err := GetInsightJobByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status == common.InsightJobPhaseCanceled {
		return job, nil
	}

	now := time.Now()
	fields := map[string]interface{}{
		"status":            common.InsightJobPhaseCanceled,
		"finished_at":       now,
		"error_message":     strings.TrimSpace(reason),
		"result_available":  false,
		"result_insight_id": nil,
	}
	if err := UpdateInsightJobFields(ctx, jobID, fields); err != nil {
		return nil, err
	}

	job.Status = common.InsightJobPhaseCanceled
	job.FinishedAt = &now
	job.ErrorMessage = strings.TrimSpace(reason)
	job.ResultAvailable = false
	job.ResultInsightID = nil
	job.UpdatedAt = now
	return job, nil
}

// DeleteInsightJob 删除失败或已取消的音眸任务，并同步清理其关联调用流水。
func DeleteInsightJob(ctx context.Context, jobID string) error {
	return InTx(
		ctx, func(tx *gorm.DB) error {
			var job InsightJob
			if err := tx.First(&job, "id = ?", strings.TrimSpace(jobID)).Error; err != nil {
				return err
			}
			if job.Status != common.InsightJobPhaseFailed && job.Status != common.InsightJobPhaseCanceled {
				return errors.New("仅允许删除失败或已取消的任务")
			}

			legacyTrackInfo := ""
			switch job.AnalysisTargetType {
			case common.AnalysisTargetTypeAlbum:
				legacyTrackInfo = strings.TrimSpace(job.Artist) + " - " + strings.TrimSpace(job.Album)
			default:
				legacyTrackInfo = strings.TrimSpace(job.Artist) + " - " + strings.TrimSpace(job.Track)
			}
			if err := deleteLLMCallLogsByTargetTx(
				tx,
				job.AnalysisTargetType,
				job.TargetKey,
				legacyTrackInfo,
			); err != nil {
				return err
			}

			return tx.Delete(&job).Error
		},
	)
}
