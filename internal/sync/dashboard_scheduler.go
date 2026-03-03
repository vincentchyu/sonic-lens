package d1sync

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

var dashboardSchedulerOnce sync.Once

func StartDashboardStatScheduler(ctx context.Context) {
	dashboardSchedulerOnce.Do(
		func() {
			runtimeCfg := model.GetDashboardStatRuntimeConfig()
			if !runtimeCfg.Enabled && !isDashboardConfigUnset(config.ConfigObj.Dashboard) {
				log.Info(ctx, "dashboard stat scheduler is disabled in config")
				return
			}

			lightInterval := time.Duration(runtimeCfg.IntervalMinutes) * time.Minute
			heavyInterval := time.Duration(runtimeCfg.HeavyIntervalMinutes) * time.Minute
			log.Info(
				ctx, "dashboard stat scheduler started",
				zap.Duration("light_interval", lightInterval),
				zap.Duration("heavy_interval", heavyInterval),
				zap.Bool("heavy_only_on_new_play", runtimeCfg.HeavyOnlyOnNewPlay),
				zap.Int("top_n", runtimeCfg.TopN),
				zap.Int("trend_days", runtimeCfg.TrendDays),
				zap.Int("hourly_trend_days", runtimeCfg.HourlyTrendDays),
			)

			go runDashboardStatLoop(ctx, lightInterval, heavyInterval, runtimeCfg.HeavyOnlyOnNewPlay)
		},
	)
}

func runDashboardStatLoop(ctx context.Context, lightInterval, heavyInterval time.Duration, heavyOnlyOnNewPlay bool) {
	log.Info(ctx, "running initial dashboard light aggregation")
	if err := model.RefreshDashboardStatsLight(ctx); err != nil {
		log.Error(ctx, "initial dashboard light aggregation failed", zap.Error(err))
	}

	log.Info(ctx, "running initial dashboard heavy aggregation")
	if err := model.RefreshDashboardStatsHeavy(ctx); err != nil {
		log.Error(ctx, "initial dashboard heavy aggregation failed", zap.Error(err))
	}

	lastHeavyRun := time.Now()
	_, lastSeenPlayUpdateAt, err := model.HasNewTrackPlayRecordsSince(ctx, time.Time{})
	if err != nil {
		log.Warn(ctx, "failed to load initial play record update watermark", zap.Error(err))
	}

	ticker := time.NewTicker(lightInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Info(ctx, "starting scheduled dashboard light aggregation")
			if err := model.RefreshDashboardStatsLight(ctx); err != nil {
				log.Error(ctx, "scheduled dashboard light aggregation failed", zap.Error(err))
			}

			if time.Since(lastHeavyRun) < heavyInterval {
				continue
			}

			if heavyOnlyOnNewPlay {
				changed, latestUpdateAt, checkErr := model.HasNewTrackPlayRecordsSince(ctx, lastSeenPlayUpdateAt)
				if checkErr != nil {
					log.Error(ctx, "failed to check play record changes for heavy aggregation", zap.Error(checkErr))
					lastHeavyRun = time.Now()
					continue
				}
				if !changed {
					log.Info(ctx, "skip dashboard heavy aggregation because no new play records")
					lastHeavyRun = time.Now()
					continue
				}
				lastSeenPlayUpdateAt = latestUpdateAt
			}

			log.Info(ctx, "starting scheduled dashboard heavy aggregation")
			if err := model.RefreshDashboardStatsHeavy(ctx); err != nil {
				log.Error(ctx, "scheduled dashboard heavy aggregation failed", zap.Error(err))
			}
			lastHeavyRun = time.Now()
		case <-ctx.Done():
			log.Info(ctx, "dashboard stat scheduler stopped")
			return
		}
	}
}

func isDashboardConfigUnset(cfg config.DashboardConfig) bool {
	return !cfg.StatRefreshEnabled &&
		cfg.StatRefreshIntervalMinutes == 0 &&
		cfg.HeavyStatRefreshIntervalMinutes == 0 &&
		!cfg.HeavyStatOnlyOnNewPlay &&
		cfg.TopN == 0 &&
		cfg.TrendDays == 0 &&
		cfg.HourlyTrendDays == 0
}
