package scrobbler

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/internal/logic/track"
)

const (
	percentScrobble                   = 0.55
	defaultSleep                      = 3
	longSleep                         = 60 // 休眠间隔六十秒
	checkCount                        = 100
	controllerFavoriteRefreshInterval = 15 * time.Second
)

// BasePlayerChecker 基础播放器检查器结构
type BasePlayerChecker struct {
	controller      common.PlayerController
	source          common.PlayerType
	defaultSleep    time.Duration
	longSleep       time.Duration
	checkCount      int
	percentScrobble float64

	// 状态变量
	scrobbledTracks             map[string]bool
	isLongCheck                 bool
	timer                       *time.Ticker
	previousTrack               string
	previousPosition            float64
	currentTrack                string
	tmpCount                    int
	now                         time.Time
	currentArtURL               string
	currentArtMime              string
	currentArtObjectKey         string
	currentArtTrackKey          string
	currentArtResolved          bool
	currentTrackCtx             context.Context
	currentTrackSpan            trace.Span
	currentTrackSpanKey         string
	currentFavorite             track.TrackFavoriteProjection
	currentFavoriteKey          string
	favoriteKnown               bool
	controllerFavoriteValue     bool
	controllerFavoriteCheckedAt time.Time
	controllerFavoriteTrackKey  string

	trackChanged bool

	// 共享状态
	pushCount           *atomic.Uint32
	atomicPlaying       *atomic.Bool
	currentPlayingCache *sync.Map
	trackService        track.TrackService
}
