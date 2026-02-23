package analysis

import (
	"context"
	"time"

	"github.com/vincentchyu/sonic-lens/internal/logic/analysis"
)

// GenerateMusicPreferenceReport 生成音乐偏好分析报告
func GenerateMusicPreferenceReport(ctx context.Context) error {
	// 初始化分析服务
	service := analysis.NewMusicAnalysisService()

	// 调用逻辑层接口生成报告
	_, err := service.GenerateMusicPreferenceReport(ctx)
	return err
}

// ScheduleReport 定时生成报告
func ScheduleReport(ctx context.Context, interval time.Duration) {
	// 初始化分析服务
	service := analysis.NewMusicAnalysisService()

	// 调用逻辑层接口定时生成报告
	// 需要启动一个 goroutine 来运行定时任务
	go service.ScheduleReport(ctx, interval)
}
