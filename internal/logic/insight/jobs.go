package insight

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/ai"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/core/telemetry"
	"github.com/vincentchyu/sonic-lens/core/websocket"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

// CreateInsightJobRequest 描述客户端发起的音眸异步任务。
type CreateInsightJobRequest struct {
	TargetType      common.AnalysisTargetType
	Artist          string
	Album           string
	Track           string
	TrackNumber     int8
	DiscNumber      int8
	AlbumID         int64
	Provider        string
	Model           string
	LegacyModelType string
	ClientPlatform  string
}

// CreateInsightJob 创建异步分析任务并在后台执行，便于客户端通过 WS/轮询恢复状态。
func (s *serviceImpl) CreateInsightJob(
	ctx context.Context, req CreateInsightJobRequest,
) (*model.InsightJob, bool, error) {
	selection, err := ai.ResolveProviderSelection(
		ai.ProviderSelectionInput{
			Provider:        req.Provider,
			Model:           req.Model,
			LegacyModelType: req.LegacyModelType,
		},
	)
	if err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(req.Model) != "" {
		if err := ai.ValidateModelAvailability(ctx, selection.Platform, selection.Model); err != nil {
			return nil, false, err
		}
	}

	job, err := s.buildInsightJob(ctx, req, selection)
	if err != nil {
		return nil, false, err
	}

	existing, err := model.GetLatestActiveInsightJobByTarget(ctx, job.TargetKey, job.Provider, job.Model)
	if err == nil && existing != nil {
		log.Info(
			ctx,
			"复用进行中的音眸任务",
			zap.String("job_id", existing.ID),
			zap.String("target_key", existing.TargetKey),
			zap.String("provider", existing.Provider),
			zap.String("model", existing.Model),
		)
		return existing, true, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	if err := model.CreateInsightJob(ctx, job); err != nil {
		return nil, false, err
	}

	log.Info(
		ctx,
		"创建音眸异步任务成功",
		zap.String("job_id", job.ID),
		zap.String("target_type", string(job.AnalysisTargetType)),
		zap.String("target_key", job.TargetKey),
		zap.String("provider", job.Provider),
		zap.String("model", job.Model),
	)
	websocket.BroadcastInsightJobUpdate(ctx, buildInsightJobWSData(job))

	telemetry.GoSafeDetached(
		ctx, "insight.job.process", func(asyncCtx context.Context) {
			s.processInsightJob(asyncCtx, job)
		},
	)

	return job, false, nil
}

// GetInsightJob 获取单个任务状态。
func (s *serviceImpl) GetInsightJob(ctx context.Context, jobID string) (*model.InsightJob, error) {
	return model.GetInsightJobByID(ctx, jobID)
}

// GetInsightJobCallLogs 获取指定任务关联的调用流水，便于管理端在任务详情中直接排障。
func (s *serviceImpl) GetInsightJobCallLogs(ctx context.Context, jobID string) ([]*model.LLMCallLog, error) {
	job, err := model.GetInsightJobByID(ctx, jobID)
	if err != nil {
		return nil, err
	}

	switch job.AnalysisTargetType {
	case common.AnalysisTargetTypeAlbum:
		return s.GetAlbumCallLogs(ctx, job.AlbumID)
	default:
		return s.GetTrackCallLogs(ctx, job.Artist, job.Album, job.Track)
	}
}

// ListInsightJobs 获取音眸任务列表，供管理端筛选和分页使用。
func (s *serviceImpl) ListInsightJobs(
	ctx context.Context,
	query model.InsightJobListQuery,
) ([]*model.InsightJob, int64, error) {
	return model.ListInsightJobs(ctx, query)
}

// UpdateInsightJobLiveActivityToken 记录客户端上报的 push token，便于后续终态推送。
func (s *serviceImpl) UpdateInsightJobLiveActivityToken(
	ctx context.Context, jobID, token string,
) (*model.InsightJob, error) {
	if strings.TrimSpace(jobID) == "" {
		return nil, errors.New("jobID 不能为空")
	}
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("token 不能为空")
	}

	if err := model.UpdateInsightJobLiveActivityToken(ctx, jobID, token); err != nil {
		return nil, err
	}
	return model.GetInsightJobByID(ctx, jobID)
}

// CancelInsightJob 将进行中的任务标记为取消，避免管理端只能被动等待终态。
func (s *serviceImpl) CancelInsightJob(ctx context.Context, jobID string) (*model.InsightJob, error) {
	if strings.TrimSpace(jobID) == "" {
		return nil, errors.New("jobID 不能为空")
	}

	job, err := model.GetInsightJobByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status == common.InsightJobPhaseCompleted || job.Status == common.InsightJobPhaseFailed {
		return nil, errors.New("终态任务不能取消")
	}

	updated, err := model.CancelInsightJob(ctx, jobID, "管理员在控制台取消任务")
	if err != nil {
		return nil, err
	}
	websocket.BroadcastInsightJobUpdate(ctx, buildInsightJobWSData(updated))
	return updated, nil
}

// RetryInsightJob 基于既有任务元数据重新创建任务，便于管理端手动补救失败链路。
func (s *serviceImpl) RetryInsightJob(ctx context.Context, jobID string) (*model.InsightJob, bool, error) {
	if strings.TrimSpace(jobID) == "" {
		return nil, false, errors.New("jobID 不能为空")
	}

	job, err := model.GetInsightJobByID(ctx, jobID)
	if err != nil {
		return nil, false, err
	}
	if job.Status == common.InsightJobPhaseQueued || job.Status == common.InsightJobPhaseRunning {
		return job, true, nil
	}

	return s.CreateInsightJob(
		ctx,
		CreateInsightJobRequest{
			TargetType:     job.AnalysisTargetType,
			Artist:         job.Artist,
			Album:          job.Album,
			Track:          job.Track,
			TrackNumber:    job.TrackNumber,
			DiscNumber:     job.DiscNumber,
			AlbumID:        job.AlbumID,
			Provider:       job.Provider,
			Model:          job.Model,
			ClientPlatform: job.ClientPlatform,
		},
	)
}

// DeleteInsightJob 删除失败或已取消的任务及其调用流水，便于管理端清理无效记录。
func (s *serviceImpl) DeleteInsightJob(ctx context.Context, jobID string) error {
	if strings.TrimSpace(jobID) == "" {
		return errors.New("jobID 不能为空")
	}

	return model.DeleteInsightJob(ctx, jobID)
}

func (s *serviceImpl) buildInsightJob(
	ctx context.Context,
	req CreateInsightJobRequest,
	selection ai.ResolvedProviderSelection,
) (*model.InsightJob, error) {
	providerDisplayName := s.resolvePlatformDisplayName(selection.Platform)
	modelDisplayName, err := s.resolveModelDisplayName(ctx, selection.Platform, selection.Model)
	if err != nil {
		return nil, err
	}

	job := &model.InsightJob{
		ID:                  uuid.NewString(),
		AnalysisTargetType:  req.TargetType,
		Status:              common.InsightJobPhaseQueued,
		Provider:            string(selection.Platform),
		Model:               selection.Model,
		ProviderDisplayName: providerDisplayName,
		ModelDisplayName:    modelDisplayName,
		ClientPlatform:      strings.TrimSpace(req.ClientPlatform),
	}

	switch req.TargetType {
	case common.AnalysisTargetTypeAlbum:
		if req.AlbumID <= 0 {
			return nil, errors.New("album_id 不能为空")
		}
		detail, err := model.GetAlbumWithTracks(ctx, req.AlbumID)
		if err != nil {
			return nil, err
		}
		job.TargetKey = model.BuildAlbumInsightJobTargetKey(req.AlbumID)
		job.AlbumID = req.AlbumID
		job.Artist = detail.Artist
		job.Album = detail.Name
	default:
		artist := strings.TrimSpace(req.Artist)
		track := strings.TrimSpace(req.Track)
		if artist == "" || track == "" {
			return nil, errors.New("artist 和 track 不能为空")
		}
		job.TargetKey = model.BuildTrackInsightJobTargetKey(
			artist,
			strings.TrimSpace(req.Album),
			track,
			req.TrackNumber,
			req.DiscNumber,
		)
		job.Artist = artist
		job.Album = strings.TrimSpace(req.Album)
		job.Track = track
		job.TrackNumber = req.TrackNumber
		job.DiscNumber = req.DiscNumber
	}

	return job, nil
}

func (s *serviceImpl) processInsightJob(ctx context.Context, job *model.InsightJob) {
	if job == nil {
		return
	}

	startedAt := time.Now()
	if err := model.UpdateInsightJobFields(
		ctx,
		job.ID,
		map[string]interface{}{
			"status":        common.InsightJobPhaseRunning,
			"started_at":    startedAt,
			"error_message": "",
		},
	); err != nil {
		log.Error(ctx, "更新音眸任务运行态失败", zap.String("job_id", job.ID), zap.Error(err))
		return
	}

	job.Status = common.InsightJobPhaseRunning
	job.StartedAt = &startedAt
	job.ErrorMessage = ""
	job.UpdatedAt = startedAt
	websocket.BroadcastInsightJobUpdate(ctx, buildInsightJobWSData(job))

	var resultInsightID *int64
	var resultAvailable bool
	var runErr error
	switch job.AnalysisTargetType {
	case common.AnalysisTargetTypeAlbum:
		var insights []*model.AlbumInsight
		insights, _, runErr = s.GetOrCreateAlbumInsight(ctx, job.AlbumID, true, job.Provider, job.Model, "")
		resultAvailable = len(insights) > 0
		if len(insights) > 0 {
			resultInsightID = &insights[0].ID
		}
	default:
		var insights []*model.TrackInsight
		insights, _, runErr = s.GetOrCreateTrackInsight(
			ctx,
			job.Artist,
			job.Album,
			job.Track,
			job.TrackNumber,
			job.DiscNumber,
			true,
			job.Provider,
			job.Model,
			"",
		)
		resultAvailable = len(insights) > 0
		if len(insights) > 0 {
			resultInsightID = &insights[0].ID
		}
	}

	finishedAt := time.Now()
	fields := map[string]interface{}{
		"finished_at":       finishedAt,
		"result_available":  resultAvailable,
		"result_insight_id": resultInsightID,
	}
	latestJob, latestErr := model.GetInsightJobByID(ctx, job.ID)
	if latestErr == nil && latestJob.Status == common.InsightJobPhaseCanceled {
		log.Info(ctx, "音眸任务已被手动取消，跳过终态覆盖", zap.String("job_id", job.ID))
		return
	}
	if runErr != nil {
		fields["status"] = common.InsightJobPhaseFailed
		fields["error_message"] = runErr.Error()
		job.Status = common.InsightJobPhaseFailed
		job.ErrorMessage = runErr.Error()
		job.ResultInsightID = nil
		job.ResultAvailable = false
		log.Error(ctx, "音眸异步任务执行失败", zap.String("job_id", job.ID), zap.Error(runErr))
	} else {
		fields["status"] = common.InsightJobPhaseCompleted
		fields["error_message"] = ""
		job.Status = common.InsightJobPhaseCompleted
		job.ErrorMessage = ""
		job.ResultInsightID = resultInsightID
		job.ResultAvailable = resultAvailable
		log.Info(
			ctx,
			"音眸异步任务执行完成",
			zap.String("job_id", job.ID),
			zap.Int64p("result_insight_id", resultInsightID),
			zap.Bool("result_available", resultAvailable),
		)
	}

	job.FinishedAt = &finishedAt
	job.UpdatedAt = finishedAt
	if err := model.UpdateInsightJobFields(ctx, job.ID, fields); err != nil {
		log.Error(ctx, "更新音眸任务终态失败", zap.String("job_id", job.ID), zap.Error(err))
	}

	websocket.BroadcastInsightJobUpdate(ctx, buildInsightJobWSData(job))
	s.pushLiveActivityIfNeeded(ctx, job)
}

func (s *serviceImpl) resolvePlatformDisplayName(platform common.AIModelPlatform) string {
	for _, option := range s.GetAvailableAIPlatforms() {
		if option.ID == platform {
			return option.DisplayName
		}
	}
	return string(platform)
}

func (s *serviceImpl) resolveModelDisplayName(
	ctx context.Context,
	platform common.AIModelPlatform,
	modelName string,
) (string, error) {
	models, err := s.GetPlatformModels(ctx, platform)
	if err != nil {
		return "", err
	}
	for _, option := range models {
		if option.ID == modelName {
			return option.DisplayName, nil
		}
	}
	return modelName, nil
}

func (s *serviceImpl) pushLiveActivityIfNeeded(ctx context.Context, job *model.InsightJob) {
	if job == nil {
		return
	}
	if strings.TrimSpace(job.LiveActivityPushToken) == "" {
		return
	}
	/*
		todo
			还差一个明确的尾巴：服务端“真正调用 APNs 去远程更新 Live Activity 终态”还没做完，现在在 jobs.go (line 292) 只是保存 token 并留了终态 hook。所以当前版本已经有“任务化 + WS + 本地 Live Activity + 深链回流”，但“App 被系统挂起后，服务端直接把完成态推上灵动岛”这一段还需要后续接 APNs 凭据和发送链路。
			APNs是什么意思是要连接苹果服务的意思吗？还是本地局域网就可以
			`APNs` 是 Apple Push Notification service，也就是苹果官方的推送服务。
			这里的意思是：
			- 不是局域网直连 iPhone
			- 不是你本地服务和手机在同一个 Wi‑Fi 下就能直接把灵动岛更新过去
			- 而是你的服务端需要通过互联网请求苹果的推送网关
			- 再由苹果把 Live Activity 的远程更新送到那台 iPhone
			所以如果要做到“App 已经挂起，任务完成后灵动岛自己变成完成态”，通常必须走这条链路：
			1. iPhone 创建 Live Activity，拿到 `push token`
			2. App 把这个 token 发给你的后端
			3. 后端任务完成后，请求苹果 APNs 接口
			4. 苹果把终态更新投递给手机上的 Live Activity
			结论很直接：
			- 只靠本地局域网，不够
			- 只靠 WebSocket，不够
			- 要做后台可靠终态更新，基本就是要接苹果 APNs
			你现在这版为什么“前台可以，后台还差一点”，本质原因就在这：
			前台时 App 活着，`WS` 和本地 `ActivityKit update` 都能工作；
			后台时 App 被挂起，服务端如果不经苹果，就没法稳定把终态送进灵动岛。
			如果你愿意，我下一步可以继续帮你把这部分也设计完整：
			- 后端需要哪些 APNs 凭据
			- SonicLens 现在这种 Go 服务怎么发 Live Activity remote update
			- 开发环境 / 真机 / 生产环境分别怎么配
			- 以及这套链路该怎么落到你现在的 `jobs.go` 里
	*/
	log.Info(
		ctx,
		"检测到 Live Activity token，当前版本先记录终态等待后续远程推送接入",
		zap.String("job_id", job.ID),
		zap.String("status", string(job.Status)),
	)
}

func buildInsightJobWSData(job *model.InsightJob) *websocket.WsInsightJobData {
	if job == nil {
		return nil
	}

	data := &websocket.WsInsightJobData{
		ID:                  job.ID,
		AnalysisTargetType:  job.AnalysisTargetType,
		Status:              job.Status,
		AlbumID:             job.AlbumID,
		Artist:              job.Artist,
		Album:               job.Album,
		Track:               job.Track,
		TrackNumber:         job.TrackNumber,
		DiscNumber:          job.DiscNumber,
		Provider:            job.Provider,
		Model:               job.Model,
		ProviderDisplayName: job.ProviderDisplayName,
		ModelDisplayName:    job.ModelDisplayName,
		ClientPlatform:      job.ClientPlatform,
		ResultInsightID:     job.ResultInsightID,
		ResultAvailable:     job.ResultAvailable,
		ErrorMessage:        job.ErrorMessage,
	}
	if job.StartedAt != nil {
		data.StartedAt = job.StartedAt.Format(time.RFC3339Nano)
	}
	if job.FinishedAt != nil {
		data.FinishedAt = job.FinishedAt.Format(time.RFC3339Nano)
	}
	if !job.UpdatedAt.IsZero() {
		data.UpdatedAt = job.UpdatedAt.Format(time.RFC3339Nano)
	}
	return data
}
