package cache

import (
	"context"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/core/telemetry"
	"github.com/vincentchyu/sonic-lens/internal/logic/genre"
)

// genreData 存储 GenreCache 的核心数据，用于原子替换 (COW)
type genreData struct {
	c2E        map[string]string // 中文 -> 英文
	e2C        map[string]string // 英文 -> 中文
	lastUpdate time.Time
}

// GenreCache represents a cache for genre c2E
type GenreCache struct {
	data atomic.Pointer[genreData]

	ticker *time.Ticker
	cancel context.CancelFunc

	genreService genre.GenreService
}

// NewGenreCache creates a new genre cache
func NewGenreCache() *GenreCache {
	gc := &GenreCache{
		genreService: genre.NewGenreService(),
	}
	// 初始化一个空数据对象，避免 Load() 返回 nil
	gc.data.Store(&genreData{
		c2E: make(map[string]string),
		e2C: make(map[string]string),
	})
	return gc
}

// GetC2E retrieves the English genre name for a Chinese genre name
func (gc *GenreCache) GetC2E(chineseGenre string) (string, bool) {
	d := gc.data.Load()
	if d == nil {
		return "", false
	}
	englishGenre, exists := d.c2E[chineseGenre]
	return englishGenre, exists
}

func (gc *GenreCache) GetE2C(englishGenre string) (string, bool) {
	d := gc.data.Load()
	if d == nil {
		return "", false
	}
	chineseGenre, exists := d.e2C[englishGenre]
	return chineseGenre, exists
}

// Set updates the cache with a new genre mapping
/*func (gc *GenreCache) Set(chineseGenre, englishGenre string) {
	gc.muC2E.Lock()
	defer gc.muC2E.Unlock()

	gc.c2E[chineseGenre] = englishGenre
}*/

// SetAll updates the cache with all genre mappings
func (gc *GenreCache) SetAll(c2EMap, e2CMap map[string]string) {
	newData := &genreData{
		c2E:        c2EMap,
		e2C:        e2CMap,
		lastUpdate: time.Now(),
	}
	gc.data.Store(newData)

	// 记录日志，说明更新成功及条目数
	log.Info(context.Background(), "Genre 映射缓存已原子更新",
		zap.Int("c2e_count", len(c2EMap)),
		zap.Int("e2c_count", len(e2CMap)),
		zap.Time("last_update", newData.lastUpdate))
}

// GetAll returns all genre mappings (copy for safety)
func (gc *GenreCache) GetAll() map[string]string {
	d := gc.data.Load()
	if d == nil {
		return make(map[string]string)
	}

	// Create a copy to avoid race conditions
	result := make(map[string]string, len(d.c2E))
	for k, v := range d.c2E {
		result[k] = v
	}
	return result
}

// GetLastUpdate returns the last update time of the cache
func (gc *GenreCache) GetLastUpdate() time.Time {
	d := gc.data.Load()
	if d == nil {
		return time.Time{}
	}
	return d.lastUpdate
}

// RefreshFromDB refreshes the cache with c2E from the database
func (gc *GenreCache) RefreshFromDB(ctx context.Context) error {
	count, err := gc.genreService.GetGenreCount(ctx)
	if err != nil {
		return err
	}
	genres, err := gc.genreService.GetAllGenres(ctx, int(count), 0)
	if err != nil {
		log.Warn(ctx, "从数据库刷新 Genre 失败", zap.Error(err))
		return err
	}
	c2e := make(map[string]string, len(genres))
	e2c := make(map[string]string, len(genres))
	for _, genreDB := range genres {
		if genreDB.NameZh != "" {
			c2e[genreDB.NameZh] = genreDB.Name
		}
		e2c[genreDB.Name] = genreDB.NameZh
	}
	gc.SetAll(c2e, e2c)
	log.Info(ctx, "从数据库刷新 Genre 成功", zap.Int("数量", len(genres)))
	return nil
}

// StartRefreshTimer starts a timer to refresh the genre cache every 6 hours
func (gc *GenreCache) StartRefreshTimer(ctx context.Context) context.CancelFunc {
	// Cancel any existing timer
	if gc.cancel != nil {
		gc.cancel()
	}

	// Create a new context for the timer
	timerCtx, cancel := context.WithCancel(ctx)
	gc.cancel = cancel

	// Create a ticker for 6 hours
	gc.ticker = time.NewTicker(1 * time.Hour)

	telemetry.GoOnlySafe(
		timerCtx, func(asyncCtx context.Context) {
			defer gc.ticker.Stop()

			// Refresh immediately on startup
			if err := gc.RefreshFromDB(asyncCtx); err != nil {
				log.Error(asyncCtx, "Failed to refresh genre cache on startup", zap.Error(err))
			}

			for {
				select {
				case <-gc.ticker.C:
					if err := gc.RefreshFromDB(asyncCtx); err != nil {
						log.Error(asyncCtx, "刷新 Genre 缓存失败", zap.Error(err))
					} else {
						log.Info(asyncCtx, "Genre 缓存定时刷新成功")
					}
				case <-asyncCtx.Done():
					log.Info(asyncCtx, "Genre 缓存刷新任务退出")
					return
				}
			}
		},
	)

	return gc.cancel
}

// StopRefreshTimer stops the refresh timer
func (gc *GenreCache) StopRefreshTimer() {
	if gc.cancel != nil {
		gc.cancel()
		gc.cancel = nil
	}
}

// Global genre cache instance
var globalGenreCache = NewGenreCache()

// InitializeGenreCache initializes the global genre cache with a refresh timer
func InitializeGenreCache(ctx context.Context) context.CancelFunc {
	return globalGenreCache.StartRefreshTimer(ctx)
}

// GetGenreCache returns the global genre cache instance
func GetGenreCache() *GenreCache {
	return globalGenreCache
}

// GetEnglishGenre 从缓存中获取中文风格对应的英文名称
func GetEnglishGenre(genre string) string {
	ctx := context.Background()
	if ok := common.IsExistsChineseSimplified(genre); ok {
		normalized := common.NormalizeChineseGenre(genre)
		if english, ok := globalGenreCache.GetC2E(normalized); ok {
			return english
		}
		log.Debug(ctx, "数据库中未找到对应的中文 Genre 映射", zap.String("genre", genre), zap.String("normalized", normalized))
		return genre
	}

	// 说明是纯英文
	if e2cGenre, ok := globalGenreCache.GetE2C(genre); ok {
		log.Debug(ctx, "Genre 命中英文缓存", zap.String("genre", genre), zap.String("对应中文", e2cGenre))
		return genre
	}

	log.Debug(ctx, "数据库中未找到该英文 Genre", zap.String("genre", genre))
	return genre
}
