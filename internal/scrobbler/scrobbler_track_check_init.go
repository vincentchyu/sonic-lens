package scrobbler

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/lastfm"
	"github.com/vincentchyu/sonic-lens/internal/logic/track"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

var (
	newTrackService = track.NewTrackService()
	one             sync.Once

	// 共享状态变量
	pushCount           = atomic.Uint32{} // 多渠道上报
	atomicPlaying       = atomic.Bool{}   // 并发播放状态
	currentPlayingCache = sync.Map{}      // 本地缓存当前播放信息

	// 播放器检查器实例
	audirvanaChecker  *BasePlayerChecker
	roonChecker       *BasePlayerChecker
	appleMusicChecker *BasePlayerChecker

	playerCheckers map[common.PlayerType]PlayerChecker // playerCheckers 存储所有支持的播放器检查器
)

func Init(
	ctx context.Context, apiKey, apiSecret, userLoginToken string, isMobile bool, userUsername, userPassword string,
	scrobblers []string, c <-chan struct{},
) {
	one.Do(
		func() {
			lastfm.InitLastfmApi(
				ctx,
				apiKey,
				apiSecret,
				userLoginToken,
				isMobile,
				userUsername,
				userPassword,
			)

			// 初始化播放器检查器
			audirvanaChecker = NewBasePlayerChecker(
				&AudirvanaPlayerController{},
				common.PlayerAudirvana,
				&pushCount,
				&atomicPlaying,
				&currentPlayingCache,
				newTrackService,
			)

			roonChecker = NewBasePlayerChecker(
				&RoonPlayerController{},
				common.PlayerRoon,
				&pushCount,
				&atomicPlaying,
				&currentPlayingCache,
				newTrackService,
			)

			appleMusicChecker = NewBasePlayerChecker(
				&AppleMusicPlayerController{},
				common.PlayerAppleMusic,
				&pushCount,
				&atomicPlaying,
				&currentPlayingCache,
				newTrackService,
			)
			playerCheckers = map[common.PlayerType]PlayerChecker{
				common.PlayerAudirvana:  audirvanaChecker,
				common.PlayerRoon:       roonChecker,
				common.PlayerAppleMusic: appleMusicChecker,
			}

			// 初始化检查器
			var playerTypes []common.PlayerType
			for _, player := range scrobblers {
				playerTypes = append(playerTypes, common.PlayerType(player))
			}
			_CheckPlayingTrack(ctx, playerTypes, c)
		},
	)
}

// _CheckPlayingTrack 统一的播放检查函数
func _CheckPlayingTrack(ctx context.Context, playerTypes []common.PlayerType, stop <-chan struct{}) {
	counts, err := model.GetTrackCounts(ctx)
	if err != nil {
		panic(err)
	}

	pushCount.Store(uint32(counts))

	// 为每个playerType启动一个goroutine
	for _, playerType := range playerTypes {
		if checker, exists := playerCheckers[playerType]; exists {
			if checker == nil {
				panic("player checker is nil")
			}
			go checker.CheckPlayingTrack(ctx, stop)
		}
	}
}
