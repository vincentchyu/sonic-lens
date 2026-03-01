package scrobbler

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/internal/logic/track"
)

const (
	percentScrobble = 0.55
	defaultSleep    = 2
	longSleep       = 60 // 休眠间隔六十秒
	checkCount      = 100
)

// PlayerInfoHandler 定义播放器信息接口
type PlayerInfoHandler interface {
	GetTitle() string
	GetAlbum() string
	GetArtist() string
	GetPosition() float64
	GetDuration() int64
	GetUrl() string // Audirvana 特有

	// 新增的方法以支持Track模型的更多字段
	GetAlbumArtist() string   // 专辑艺术家
	GetTrackNumber() int64    // 曲目编号
	GetGenre() string         // 流派
	GetComposer() string      // 作曲家
	GetReleaseDate() string   // 发布日期
	GetMusicBrainzID() string // MusicBrainz ID
	GetSource() string        // 数据来源
	GetBundleID() string      // 应用标识符
	GetUniqueID() string      // 唯一标识符
}

// PlayerController 定义播放器控制接口
type PlayerController interface {
	IsRunning(ctx context.Context) bool
	IsFavorite(ctx context.Context) bool
	GetState(ctx context.Context) (string, error)
	GetNowPlayingTrackInfo(ctx context.Context) PlayerInfoHandler
	SetFavorite(ctx context.Context) error
}

// PlayerChecker 定义播放器检查器接口
type PlayerChecker interface {
	CheckPlayingTrack(ctx context.Context, stop <-chan struct{})
}

// BasePlayerChecker 基础播放器检查器结构
type BasePlayerChecker struct {
	controller      PlayerController
	source          common.PlayerType
	defaultSleep    time.Duration
	longSleep       time.Duration
	checkCount      int
	percentScrobble float64

	// 状态变量
	mapedTracks   map[string]bool
	isLongCheck   bool
	timer         *time.Ticker
	previousTrack string
	currentTrack  string
	tmpCount      int
	now           time.Time

	// 共享状态
	pushCount           *atomic.Uint32
	atomicPlaying       *atomic.Bool
	currentPlayingCache *sync.Map
	trackService        track.TrackService
}
