package ai

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/core/telemetry"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

// BaseProvider 是所有 LLMProvider 实现的父结构体，提供通用的调用流水日志记录能力。
// 子结构体应通过 GoSafe 的嵌入（embedding）方式"继承"该结构体。
type BaseProvider struct {
	ProviderName string // 提供方名称，如 doubao, ollama, openai
	ModelName    string // 模型名称
}

func (b *BaseProvider) GetProviderName() string {
	return b.ProviderName
}

func (b *BaseProvider) GetModelName() string {
	return b.ModelName
}

// SaveCallLog 将大模型的出站请求和响应全文 JSON 异步保存到调用流水表中，用于未来排查和恢复现场。
// 该方法通过 goroutine 异步执行，不阻塞主请求流程。
//
// 参数说明：
//   - ctx: 上下文（仅用于日志，存储使用 background context 避免请求取消导致丢失）
//   - req: 原始业务请求对象，用于生成对象维度和元数据
//   - requestJSON: 实际发给模型商的请求 JSON，必须包含完整 prompt
//   - respJSON: 响应体全文 JSON 字符串
//   - callErr: 调用过程中的错误（如有）
//   - startTime: 调用开始时间，用于计算耗时
//   - callType: 调用类型，"sync" 或 "stream"
func (b *BaseProvider) SaveCallLog(
	ctx context.Context,
	req TrackAnalysisRequest,
	requestJSON string,
	respJSON string,
	callErr error,
	startTime time.Time,
	callType string,
) {
	telemetry.GoSafeDetached(
		ctx, "ai.save_track_call_log", func(asyncCtx context.Context) {
			targetMetadata := map[string]any{
				"artist":             req.Artist,
				"album":              req.Album,
				"title":              req.Title,
				"requested_provider": req.RequestedProvider,
				"requested_model":    req.RequestedModel,
				"effective_provider": b.ProviderName,
				"effective_model":    b.ModelName,
			}

			// 确定状态和错误信息
			status := "success"
			errMsg := ""
			if callErr != nil {
				status = "error"
				errMsg = callErr.Error()
			}

			// 构建曲目信息
			trackInfo := req.Artist + " - " + req.Title
			targetKey := model.BuildLLMCallLogTrackKey(req.Artist, req.Album, req.Title)
			targetMetadataBytes, _ := json.Marshal(targetMetadata)

			// 计算耗时
			durationMs := time.Since(startTime).Milliseconds()

			jobID, _ := asyncCtx.Value(common.ContextKeyJobID).(string)

			callLog := &model.LLMCallLog{
				JobID:              jobID,
				Provider:           b.ProviderName,
				Model:              b.ModelName,
				RequestJSON:        requestJSON,
				ResponseJSON:       respJSON,
				Status:             status,
				ErrorMsg:           errMsg,
				DurationMs:         durationMs,
				AnalysisTargetType: common.AnalysisTargetTypeTrack,
				TargetKey:          targetKey,
				TargetMetadata:     string(targetMetadataBytes),
				TrackInfo:          trackInfo,
				CallType:           callType,
				CreatedAt:          time.Now(),
			}

			if err := model.CreateLLMCallLog(asyncCtx, callLog); err != nil {
				log.Error(asyncCtx, "保存大模型调用流水日志失败", zap.Error(err))
			}
		},
	)
}

// SaveAlbumCallLog 将专辑级出站请求与响应异步保存到调用流水表。
func (b *BaseProvider) SaveAlbumCallLog(
	ctx context.Context,
	req AlbumAnalysisRequest,
	requestJSON string,
	respJSON string,
	callErr error,
	startTime time.Time,
	callType string,
) {
	telemetry.GoSafeDetached(
		ctx, "ai.save_album_call_log", func(asyncCtx context.Context) {
			targetMetadata := map[string]any{
				"album_id":           req.AlbumID,
				"artist":             req.Artist,
				"album":              req.Album,
				"release_date":       req.ReleaseDate,
				"genre":              req.Genre,
				"track_count":        req.TrackCount,
				"analyzed_tracks":    req.AnalyzedTracks,
				"requested_provider": req.RequestedProvider,
				"requested_model":    req.RequestedModel,
				"effective_provider": b.ProviderName,
				"effective_model":    b.ModelName,
			}
			if len(req.TrackContexts) > 0 {
				targetMetadata["track_context_count"] = len(req.TrackContexts)
			}

			status := "success"
			errMsg := ""
			if callErr != nil {
				status = "error"
				errMsg = callErr.Error()
			}

			trackInfo := req.Artist + " - " + req.Album
			targetKey := model.BuildLLMCallLogAlbumKey(req.AlbumID)
			targetMetadataBytes, _ := json.Marshal(targetMetadata)
			durationMs := time.Since(startTime).Milliseconds()

			jobID, _ := asyncCtx.Value(common.ContextKeyJobID).(string)

			callLog := &model.LLMCallLog{
				JobID:              jobID,
				Provider:           b.ProviderName,
				Model:              b.ModelName,
				RequestJSON:        requestJSON,
				ResponseJSON:       respJSON,
				Status:             status,
				ErrorMsg:           errMsg,
				DurationMs:         durationMs,
				AnalysisTargetType: common.AnalysisTargetTypeAlbum,
				TargetKey:          targetKey,
				TargetMetadata:     string(targetMetadataBytes),
				TrackInfo:          trackInfo,
				CallType:           callType,
				CreatedAt:          time.Now(),
			}

			if err := model.CreateLLMCallLog(asyncCtx, callLog); err != nil {
				log.Error(asyncCtx, "保存专辑大模型调用流水日志失败", zap.Error(err))
			}
		},
	)
}
