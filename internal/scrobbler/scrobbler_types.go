package scrobbler

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/internal/logic/track"
)

const (
	percentScrobble = 0.55
	defaultSleep    = 3
	longSleep       = 60 // 休眠间隔六十秒
	checkCount      = 100
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
	scrobbledTracks map[string]bool
	isLongCheck     bool
	timer           *time.Ticker
	previousTrack   string
	currentTrack    string
	tmpCount        int
	now             time.Time
	currentArtURL   string
	currentArtMime  string

	// 共享状态
	pushCount           *atomic.Uint32
	atomicPlaying       *atomic.Bool
	currentPlayingCache *sync.Map
	trackService        track.TrackService
}
