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

var playReplaySchedulerOnce sync.Once

// StartTrackPlayReplayScheduler 启动播放流水补归因定时任务。
func StartTrackPlayReplayScheduler(ctx context.Context) {
	playReplaySchedulerOnce.Do(
		func() {
			cfg := config.ConfigObj.PlayReplay
			if !cfg.Enabled {
				log.Info(ctx, "track play replay scheduler is disabled in config")
				return
			}

			interval := 30 * time.Minute
			if cfg.IntervalMinutes > 0 {
				interval = time.Duration(cfg.IntervalMinutes) * time.Minute
			}

			batchSize := 50
			if cfg.BatchSize > 0 {
				batchSize = cfg.BatchSize
			}

			log.Info(
				ctx,
				"track play replay scheduler started",
				zap.Duration("interval", interval),
				zap.Int("batch_size", batchSize),
				zap.Bool("only_unapplied", cfg.OnlyUnapplied),
				zap.Bool("only_unresolved", cfg.OnlyUnresolved),
				zap.Bool("run_on_startup", cfg.RunOnStartup),
			)

			if cfg.RunOnStartup {
				go replayTrackPlayRecordsBatch(ctx, batchSize, cfg.OnlyUnapplied, cfg.OnlyUnresolved)
			}

			go runTrackPlayReplayLoop(ctx, interval, batchSize, cfg.OnlyUnapplied, cfg.OnlyUnresolved)
		},
	)
}

func runTrackPlayReplayLoop(
	ctx context.Context, interval time.Duration, batchSize int, onlyUnapplied, onlyUnresolved bool,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			replayTrackPlayRecordsBatch(ctx, batchSize, onlyUnapplied, onlyUnresolved)
		case <-ctx.Done():
			log.Info(ctx, "track play replay scheduler stopped")
			return
		}
	}
}

func replayTrackPlayRecordsBatch(ctx context.Context, batchSize int, onlyUnapplied, onlyUnresolved bool) {
	log.Info(
		ctx,
		"starting scheduled track play replay",
		zap.Int("batch_size", batchSize),
		zap.Bool("only_unapplied", onlyUnapplied),
		zap.Bool("only_unresolved", onlyUnresolved),
	)

	report, err := model.ReplayTrackPlayRecords(
		model.ReplayTrackPlayRecordsParams{
			Ctx:            ctx,
			Limit:          batchSize,
			DryRun:         false,
			OnlyUnapplied:  onlyUnapplied,
			OnlyUnresolved: onlyUnresolved,
		},
	)
	if err != nil {
		log.Error(ctx, "scheduled track play replay failed", zap.Error(err))
		return
	}

	log.Info(ctx, "scheduled track play replay finished", zap.Int("processed_records", len(report.Results)))
}
