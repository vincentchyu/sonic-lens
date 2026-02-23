package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/cmd/analysis"
	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/core/telemetry"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

func NewMusicAnalysisCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "music-analysis",
		Short: "音乐分析相关命令",
	}

	cmd.AddCommand(newGenerateReportCommand())
	cmd.AddCommand(newScheduleReportCommand())
	cmd.AddCommand(newGenerateRecommendationsCommand())

	return cmd
}

// initTracing 初始化链路跟踪
func initTracing(ctx context.Context, operation string) (context.Context, trace.Span) {
	// 初始化OpenTelemetry链路跟踪
	if err := telemetry.Init(
		config.TelemetryConfig{
			Name: "lastfm-scrobbler-analysis",
		},
	); err != nil {
		log.Error(ctx, "Failed to initialize telemetry", zap.Error(err))
		return ctx, nil
	}

	return telemetry.StartSpan(ctx, operation)
}

func newGenerateReportCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "generate-report",
		Short: "生成音乐偏好分析报告",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 初始化配置和数据库
			config.InitConfig("config/config_bak.yaml")
			logger, _ := log.LogInit(config.ConfigObj.Log.Path, config.ConfigObj.Log.Level, nil)
			if err := model.InitDB(config.ConfigObj.Database.Path, logger); err != nil {
				return err
			}

			ctx := context.Background()
			// 初始化链路跟踪
			ctx, span := initTracing(ctx, "generate-report")
			defer span.End()
			return analysis.GenerateMusicPreferenceReport(ctx)
		},
	}
}

func newScheduleReportCommand() *cobra.Command {
	var interval string

	cmd := &cobra.Command{
		Use:   "schedule-report",
		Short: "定时生成音乐偏好分析报告",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 初始化配置和数据库
			config.InitConfig("config/config_bak.yaml")
			logger, _ := log.LogInit(config.ConfigObj.Log.Path, config.ConfigObj.Log.Level, nil)
			if err := model.InitDB(config.ConfigObj.Database.Path, logger); err != nil {
				return err
			}

			ctx := context.Background()
			// 初始化链路跟踪
			ctx, span := initTracing(ctx, "schedule-report")
			defer span.End()
			duration, err := time.ParseDuration(interval)
			if err != nil {
				return err
			}

			analysis.ScheduleReport(ctx, duration)
			return nil
		},
	}

	cmd.Flags().StringVarP(&interval, "interval", "i", "24h", "报告生成间隔时间 (例如: 1h, 24h, 7d)")

	return cmd
}

func newGenerateRecommendationsCommand() *cobra.Command {
	var limit int
	var artist string

	cmd := &cobra.Command{
		Use:   "generate-recommendations",
		Short: "生成音乐推荐",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 初始化配置和数据库
			config.InitConfig("config/config_bak.yaml")
			logger, _ := log.LogInit(config.ConfigObj.Log.Path, config.ConfigObj.Log.Level, nil)
			if err := model.InitDB(config.ConfigObj.Database.Path, logger); err != nil {
				return err
			}

			ctx := context.Background()
			// 初始化链路跟踪
			ctx, span := initTracing(ctx, "generate-recommendations")
			defer span.End()
			if artist != "" {
				// 生成特定艺术家的推荐
				recommendations, err := analysis.GetArtistRecommendations(ctx, artist, limit)
				if err != nil {
					return err
				}
				analysis.PrintRecommendations(recommendations)
			} else {
				// 生成通用音乐推荐
				recommendations, err := analysis.GenerateMusicRecommendations(ctx, limit)
				if err != nil {
					return err
				}
				analysis.PrintRecommendations(recommendations)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "推荐数量")
	cmd.Flags().StringVarP(&artist, "artist", "a", "", "特定艺术家的推荐")

	return cmd
}
