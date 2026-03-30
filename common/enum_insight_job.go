package common

import "strings"

// InsightJobPhase 定义音眸异步任务状态。
type InsightJobPhase string

const (
	InsightJobPhaseQueued    InsightJobPhase = "queued"
	InsightJobPhaseRunning   InsightJobPhase = "running"
	InsightJobPhaseCompleted InsightJobPhase = "completed"
	InsightJobPhaseFailed    InsightJobPhase = "failed"
	InsightJobPhaseCanceled  InsightJobPhase = "canceled"
)

// ParseInsightJobPhase 将外部输入归一为已知任务状态。
func ParseInsightJobPhase(value string) InsightJobPhase {
	switch InsightJobPhase(strings.ToLower(strings.TrimSpace(value))) {
	case InsightJobPhaseRunning:
		return InsightJobPhaseRunning
	case InsightJobPhaseCompleted:
		return InsightJobPhaseCompleted
	case InsightJobPhaseFailed:
		return InsightJobPhaseFailed
	case InsightJobPhaseCanceled:
		return InsightJobPhaseCanceled
	default:
		return InsightJobPhaseQueued
	}
}

// IsTerminal 判断任务是否已进入终态。
func (p InsightJobPhase) IsTerminal() bool {
	switch p {
	case InsightJobPhaseCompleted, InsightJobPhaseFailed, InsightJobPhaseCanceled:
		return true
	default:
		return false
	}
}
