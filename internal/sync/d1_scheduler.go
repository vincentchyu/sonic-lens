package d1sync

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/core/telemetry"
)

var (
	schedulerOnce sync.Once
	d1Client      *D1Client
)

// StartD1SyncScheduler 启动 D1 同步定时器
func StartD1SyncScheduler(ctx context.Context) {
	schedulerOnce.Do(
		func() {
			cfg := config.ConfigObj.Cloudflare
			// 检查是否启用同步
			if !cfg.SyncEnabled {
				log.Info(ctx, "D1 sync is disabled in config")
				return
			}
			if cfg.AccountID == "" || cfg.APIToken == "" || cfg.D1DatabaseID == "" {
				log.Warn(ctx, "D1 sync config incomplete, skip scheduler")
				return
			}

			// 创建 D1 客户端
			var err error
			d1Client, err = NewD1Client(&cfg)
			if err != nil {
				log.Error(ctx, "Failed to create D1 client", zap.Error(err))
				return
			}

			// 确保在上下文取消时关闭客户端
			telemetry.GoOnlySafe(
				ctx, func(asyncCtx context.Context) {
					<-asyncCtx.Done()
					if d1Client != nil {
						if err := d1Client.Close(); err != nil {
							log.Error(asyncCtx, "Failed to close D1 client", zap.Error(err))
						}
					}
				},
			)

			// 首次启动时先完成一次同步，避免定时器与首轮同步并发抢跑。
			log.Info(ctx, "Performing initial D1 sync")
			if err := d1Client.SyncAll(ctx, true); err != nil {
				if !errors.Is(err, ErrD1SyncAlreadyRunning) {
					log.Error(ctx, "Initial D1 sync failed", zap.Error(err))
				}
			}

			// 首轮同步结束后再启动定时同步，后续由单飞保护兜底。
			telemetry.GoOnlySafe(
				ctx, func(asyncCtx context.Context) {
					runSyncLoop(asyncCtx, &cfg)
				},
			)
		},
	)
}

// runSyncLoop 运行同步循环
func runSyncLoop(ctx context.Context, cfg *config.CloudflareConfig) {
	// 默认同步间隔为 24 小时
	syncInterval := 24 * time.Hour
	if cfg.SyncInterval > 0 {
		syncInterval = time.Duration(cfg.SyncInterval) * time.Hour
	}

	log.Info(ctx, "D1 sync scheduler started", zap.Duration("interval", syncInterval))

	ticker := time.NewTicker(syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Info(ctx, "Starting scheduled D1 sync")
			// 定时同步使用增量模式
			if err := d1Client.SyncAll(ctx, true); err != nil {
				if errors.Is(err, ErrD1SyncAlreadyRunning) {
					log.Warn(ctx, "Skip scheduled D1 sync because another sync is still running")
					continue
				}
				log.Error(ctx, "Scheduled D1 sync failed", zap.Error(err))
			} else {
				log.Info(ctx, "Scheduled D1 sync completed successfully")
			}
		case <-ctx.Done():
			log.Info(ctx, "D1 sync scheduler stopped")
			return
		}
	}
}

// TriggerManualSync 手动触发同步(用于测试或管理)
func TriggerManualSync(ctx context.Context, fullSync bool) error {
	if d1Client == nil {
		return ErrD1ClientNotInitialized
	}

	log.Info(ctx, "Manual D1 sync triggered", zap.Bool("full_sync", fullSync))
	return d1Client.SyncAll(ctx, !fullSync)
}

var ErrD1ClientNotInitialized = fmt.Errorf("D1 client not initialized")
