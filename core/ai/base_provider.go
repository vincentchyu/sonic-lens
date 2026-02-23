package ai

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

// BaseProvider 是所有 LLMProvider 实现的父结构体，提供通用的调用流水日志记录能力。
// 子结构体应通过 Go 的嵌入（embedding）方式"继承"该结构体。
type BaseProvider struct {
	ProviderName string // 提供方名称，如 doubao, ollama, openai
	ModelName    string // 模型名称
}

// SaveCallLog 将大模型的请求和响应全文 JSON 异步保存到调用流水表中，用于未来排查和恢复现场。
// 该方法通过 goroutine 异步执行，不阻塞主请求流程。
//
// 参数说明：
//   - ctx: 上下文（仅用于日志，存储使用 background context 避免请求取消导致丢失）
//   - req: 原始请求对象（TrackAnalysisRequest）
//   - respJSON: 响应体全文 JSON 字符串
//   - callErr: 调用过程中的错误（如有）
//   - startTime: 调用开始时间，用于计算耗时
//   - callType: 调用类型，"sync" 或 "stream"
func (b *BaseProvider) SaveCallLog(
	ctx context.Context,
	req TrackAnalysisRequest,
	respJSON string,
	callErr error,
	startTime time.Time,
	callType string,
) {
	go func() {
		// 序列化请求体
		reqBytes, _ := json.Marshal(req)

		// 确定状态和错误信息
		status := "success"
		errMsg := ""
		if callErr != nil {
			status = "error"
			errMsg = callErr.Error()
		}

		// 构建曲目信息
		trackInfo := req.Artist + " - " + req.Title

		// 计算耗时
		durationMs := time.Since(startTime).Milliseconds()

		callLog := &model.LLMCallLog{
			Provider:     b.ProviderName,
			Model:        b.ModelName,
			RequestJSON:  string(reqBytes),
			ResponseJSON: respJSON,
			Status:       status,
			ErrorMsg:     errMsg,
			DurationMs:   durationMs,
			TrackInfo:    trackInfo,
			CallType:     callType,
			CreatedAt:    time.Now(),
		}

		if err := model.CreateLLMCallLog(context.Background(), callLog); err != nil {
			log.Error(ctx, "保存大模型调用流水日志失败", zap.Error(err))
		}
	}()
}
