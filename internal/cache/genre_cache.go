package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/internal/logic/genre"
)

// GenreCache represents a cache for genre c2E
type GenreCache struct {
	c2E   map[string]string // Chinese genre name to English genre name mapping
	muC2E sync.RWMutex

	e2C   map[string]string // 英文 -> 中文
	muE2C sync.RWMutex

	lastUpdate time.Time
	ticker     *time.Ticker
	cancel     context.CancelFunc

	genreService genre.GenreService
}

// NewGenreCache creates a new genre cache
func NewGenreCache() *GenreCache {
	return &GenreCache{
		c2E:          make(map[string]string),
		e2C:          make(map[string]string),
		genreService: genre.NewGenreService(),
	}
}

// GetC2E retrieves the English genre name for a Chinese genre name
func (gc *GenreCache) GetC2E(chineseGenre string) (string, bool) {
	gc.muC2E.RLock()
	defer gc.muC2E.RUnlock()

	englishGenre, exists := gc.c2E[chineseGenre]
	return englishGenre, exists
}

func (gc *GenreCache) GetE2C(chineseGenre string) (string, bool) {
	gc.muE2C.RLock()
	defer gc.muE2C.RUnlock()

	englishGenre, exists := gc.e2C[chineseGenre]
	return englishGenre, exists
}

// Set updates the cache with a new genre mapping
/*func (gc *GenreCache) Set(chineseGenre, englishGenre string) {
	gc.muC2E.Lock()
	defer gc.muC2E.Unlock()

	gc.c2E[chineseGenre] = englishGenre
}*/

// SetAll updates the cache with all genre mappings
func (gc *GenreCache) SetAll(c2EMap, e2CMap map[string]string) {
	func() {
		gc.muC2E.Lock()
		defer gc.muC2E.Unlock()

		gc.c2E = make(map[string]string, len(c2EMap))
		for k, v := range c2EMap {
			gc.c2E[k] = v
		}
	}()

	func() {
		gc.muE2C.Lock()
		defer gc.muE2C.Unlock()

		gc.e2C = make(map[string]string, len(e2CMap))
		for k, v := range e2CMap {
			gc.e2C[k] = v
		}
	}()

	gc.lastUpdate = time.Now()
}

// GetAll returns all genre mappings
func (gc *GenreCache) GetAll() map[string]string {
	gc.muC2E.RLock()
	defer gc.muC2E.RUnlock()

	// Create a copy to avoid race conditions
	result := make(map[string]string, len(gc.c2E))
	for k, v := range gc.c2E {
		result[k] = v
	}
	return result
}

// GetLastUpdate returns the last update time of the cache
func (gc *GenreCache) GetLastUpdate() time.Time {
	gc.muC2E.RLock()
	defer gc.muC2E.RUnlock()

	return gc.lastUpdate
}

// RefreshFromDB refreshes the cache with c2E from the database
func (gc *GenreCache) RefreshFromDB(ctx context.Context) error {
	count, err := gc.genreService.GetGenreCount(ctx)
	if err != nil {
		return err
	}
	genres, err := gc.genreService.GetAllGenres(ctx, int(count), 0)
	if err != nil {
		log.Warn(ctx, "refresh genres from db failed", zap.Error(err))
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

	go func() {
		defer gc.ticker.Stop()

		// Refresh immediately on startup
		if err := gc.RefreshFromDB(timerCtx); err != nil {
			log.Error(timerCtx, "Failed to refresh genre cache on startup", zap.Error(err))
		}

		for {
			select {
			case <-gc.ticker.C:
				if err := gc.RefreshFromDB(timerCtx); err != nil {
					log.Error(timerCtx, "Failed to refresh genre cache", zap.Error(err))
				} else {
					log.Info(timerCtx, "Genre cache refreshed successfully")
				}
			case <-timerCtx.Done():
				log.Info(timerCtx, "Genre cache exit")
				return
			}
		}
	}()

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

// GetEnglishGenre retrieves the English genre name for a Chinese genre name from cache
func GetEnglishGenre(genre string) string {
	if ok := common.IsExistsChineseSimplified(genre); ok {
		normalized := common.NormalizeChineseGenre(genre)
		if english, ok := globalGenreCache.GetC2E(normalized); ok {
			return english
		} else {
			// todo 说明数据库中没有这个这个中文分类
			return genre // 如果没有对应的英文分类，返回原始分类
		}
	}
	// 说明是纯英文
	if e2cGenre, ok := globalGenreCache.GetE2C(genre); ok {
		fmt.Println(e2cGenre)
		return genre // 并且说明命中缓存
	} else {
		// todo 说明数据库中没有这个这个英文
	}
	return genre // 如果不是简体中文，直接返回原始分类
}
