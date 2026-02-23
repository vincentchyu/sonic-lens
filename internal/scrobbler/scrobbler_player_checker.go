package scrobbler

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/lastfm"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/core/telemetry"
	"github.com/vincentchyu/sonic-lens/core/websocket"
	"github.com/vincentchyu/sonic-lens/internal/logic/track"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

// NewBasePlayerChecker 创建基础播放器检查器
func NewBasePlayerChecker(
	controller PlayerController,
	source common.PlayerType,
	pushCount *atomic.Uint32,
	atomicPlaying *atomic.Bool,
	currentPlayingCache *sync.Map,
	trackService track.TrackService,
) *BasePlayerChecker {
	return &BasePlayerChecker{
		controller:          controller,
		source:              source,
		defaultSleep:        time.Second * defaultSleep,
		longSleep:           time.Second * longSleep,
		checkCount:          checkCount,
		percentScrobble:     percentScrobble,
		mapedTracks:         make(map[string]bool),
		pushCount:           pushCount,
		atomicPlaying:       atomicPlaying,
		currentPlayingCache: currentPlayingCache,
		trackService:        trackService,
	}
}

// _CheckPlayingTrack 基础播放检查逻辑
func (b *BasePlayerChecker) CheckPlayingTrack(ctx context.Context, stop <-chan struct{}) {
	b.timer = time.NewTicker(b.defaultSleep)
	b.tmpCount = 0
	b.previousTrack = ""
	b.currentTrack = ""

	for {
		select {
		case <-b.timer.C:
			b.checkCycle(ctx)
		case <-stop:
			log.Info(ctx, string(b.source)+" check playing track exit")
			return
		}
	}
}

// checkCycle 执行一次检查周期
func (b *BasePlayerChecker) checkCycle(ctx context.Context) {
	// Start a new span for this check cycle
	checkCtx, span := telemetry.StartSpanForTracerName(
		ctx, _TracerName, string(b.source)+"_CheckPlayingTrack",
	)
	defer span.End()

	log.Debug(checkCtx, string(b.source)+" Checking playing track..."+time.Now().String())

	b.tmpCount++
	if b.tmpCount > b.checkCount && !b.isLongCheck { // 检查100次依旧没有播放检查轮训放大到60秒
		b.timer.Reset(b.longSleep)
		b.isLongCheck = true
		log.Info(
			checkCtx, string(b.source)+"检查100次依旧没有播放检查轮训放大到60秒",
			zap.Uint32("共计上传歌曲标记", b.pushCount.Load()),
		)
	}
	if b.isLongCheck {
		log.Info(checkCtx, string(b.source)+"60秒检查", zap.Uint32("共计上传歌曲标记", b.pushCount.Load()))
	}

	running := b.controller.IsRunning(checkCtx)
	log.Debug(checkCtx, string(b.source)+" 程序运行是否运行", zap.Bool("running", running))

	var playerInfo PlayerInfoHandler
	if running {
		playerInfo = nil
		state, _ := b.controller.GetState(checkCtx)
		log.Debug(checkCtx, string(b.source)+" 播放状态", zap.Any("state", state))
		if state == common.PlayerStatePlaying {
			if b.tmpCount > b.checkCount {
				b.isLongCheck = false
				b.timer.Reset(b.defaultSleep)
			}
			b.tmpCount = 0
			playerInfo = b.controller.GetNowPlayingTrackInfo(checkCtx)
		} else {
			if _, ok := b.currentPlayingCache.Load(b.source); ok {
				b.currentPlayingCache.Delete(b.source)
				b.handleStopEvent(checkCtx)
			}
		}
	}

	if playerInfo != nil {
		b.processPlayingTrack(checkCtx, playerInfo)
	}
}

// handleStopEvent 处理停止事件
func (b *BasePlayerChecker) handleStopEvent(ctx context.Context) {
	// 检查是否还有其他播放器在播放
	_, audirvanaPlaying := b.currentPlayingCache.Load(common.PlayerAudirvana)
	_, roonPlaying := b.currentPlayingCache.Load(common.PlayerRoon)
	_, appleMusicPlaying := b.currentPlayingCache.Load(common.PlayerAppleMusic)

	// 如果没有其他播放器在播放，则发送停止消息
	shouldStop := false
	switch b.source {
	case common.PlayerAudirvana:
		shouldStop = !roonPlaying && !appleMusicPlaying
	case common.PlayerRoon:
		shouldStop = !audirvanaPlaying && !appleMusicPlaying
	case common.PlayerAppleMusic:
		shouldStop = !audirvanaPlaying && !roonPlaying
	}

	if shouldStop {
		websocket.BroadcastMessage(
			ctx,
			&websocket.WsTrackInfo{
				Type:   "stop",
				Source: string(b.source),
			},
		)
		b.atomicPlaying.Store(false)
	}
}

// processPlayingTrack 处理正在播放的曲目
func (b *BasePlayerChecker) processPlayingTrack(ctx context.Context, playerInfo PlayerInfoHandler) {
	// 根据播放器类型生成 track 标识
	tmpTrack := playerInfo.GetTitle()
	// 对于 Audirvana，使用 URL + Title 防止 cue 文件问题
	if b.source == common.PlayerAudirvana {
		if url := playerInfo.GetUrl(); url != "" {
			tmpTrack = url + playerInfo.GetTitle()
		}
	}

	b.currentTrack = tmpTrack
	position := playerInfo.GetPosition()
	duration := playerInfo.GetDuration()

	// 查询数据库获取喜欢标志 检查喜欢状态并处理
	appleMusicFav, lastFmFav := b.trackLikeCheckAndHandle(ctx, playerInfo)

	wti := &websocket.WsTrackInfo{
		Type:   "now_playing",
		Source: string(b.source),
		Data: struct {
			Title      string `json:"title"`
			Album      string `json:"album"`
			Artist     string `json:"artist"`
			AppleMusic bool   `json:"apple_music"`
			LastFM     bool   `json:"lastfm"`
			Duration   int64  `json:"duration"` // 歌曲时长，单位秒
			Position   int64  `json:"position"` // 歌曲当前播放位置，单位秒
		}{
			Title:      playerInfo.GetTitle(),
			Album:      playerInfo.GetAlbum(),
			Artist:     playerInfo.GetArtist(),
			AppleMusic: appleMusicFav,
			LastFM:     lastFmFav,
			Duration:   duration,
			Position:   int64(position),
		},
	}
	// 向WebSocket客户端广播播放信息
	// 将播放信息写入本地缓存
	b.currentPlayingCache.Store(b.source, wti)
	b.atomicPlaying.Store(true)
	websocket.BroadcastMessage(ctx, wti)

	// 检查是否需要标记听歌完成
	if position/float64(duration) > b.percentScrobble && !b.mapedTracks[b.currentTrack] {
		b.handleTrackScrobble(ctx, playerInfo)
	}

	// 检查是否是新歌曲
	if b.currentTrack != b.previousTrack {
		b.handleNewTrack(ctx, playerInfo)
	}

	b.previousTrack = tmpTrack
}

func (b *BasePlayerChecker) trackLikeCheckAndHandle(ctx context.Context, playerInfo PlayerInfoHandler) (bool, bool) {
	appleMusicFav := false
	lastFmFav := false
	if getTrack, _ := b.trackService.GetTrack(
		ctx, playerInfo.GetArtist(), playerInfo.GetAlbum(), playerInfo.GetTitle(),
	); getTrack != nil {
		appleMusicFav = getTrack.IsAppleMusicFav
		lastFmFav = getTrack.IsLastFmFav
		// 当前歌曲没有标记为苹果喜欢状态
		if !getTrack.IsAppleMusicFav && !getTrack.IsLastFmFav {
			switch b.source {
			case common.PlayerAppleMusic:
				// 检查Apple Music喜欢状态
				favorite := b.controller.IsFavorite(ctx)
				appleMusicFav = favorite
				if favorite {
					err := b.trackService.SetAppleMusicFavorite(
						model.SetFavoriteParams{
							Ctx:           ctx,
							Artist:        getTrack.Artist,
							Album:         getTrack.Album,
							Track:         getTrack.Track,
							IsFavorite:    true,
							TrackMetadata: model.TrackMetadata{},
						},
					)
					if err != nil {
						log.Warn(
							ctx, string(b.source)+" processPlayingTrack SetAppleMusicFavorite err", zap.Error(err),
						)
					}
				}
			case common.PlayerAudirvana:
			}

			// 更新lastfm 同步（因为苹果喜欢lastfm 也必须喜欢）
			// dbLastFm喜欢状态 没有被设置过，检查lastfm是否喜欢
			favorite, err := lastfm.IsFavorite(ctx, getTrack.Artist, getTrack.Track)
			if err != nil {
				log.Warn(
					ctx, string(b.source)+" processPlayingTrack lastfm Favorite err", zap.Error(err),
				)
			}
			lastFmFav = favorite
			if favorite {
				err := b.trackService.SetLastFmFavorite(
					model.SetFavoriteParams{
						Ctx:           ctx,
						Artist:        getTrack.Artist,
						Album:         getTrack.Album,
						Track:         getTrack.Track,
						IsFavorite:    true,
						TrackMetadata: model.TrackMetadata{},
					},
				)
				if err != nil {
					log.Warn(
						ctx, string(b.source)+" processPlayingTrack SetLastFmFavorite err", zap.Error(err),
					)
				}

			}
		} else if getTrack.IsLastFmFav && !getTrack.IsAppleMusicFav {
			// 曾经在auridrvana 标记喜欢的歌曲，切换到Apple Music 播放，自动标记为Apple Music喜欢
			if b.source == common.PlayerAppleMusic {
				err := b.trackService.SetAppleMusicFavorite(
					model.SetFavoriteParams{
						Ctx:           ctx,
						Artist:        getTrack.Artist,
						Album:         getTrack.Album,
						Track:         getTrack.Track,
						IsFavorite:    true,
						TrackMetadata: model.TrackMetadata{},
					},
				)
				if err != nil {
					log.Warn(
						ctx, string(b.source)+" processPlayingTrack SetAppleMusicFavorite err", zap.Error(err),
					)
				}
				appleMusicFav = true
			}
		} else if getTrack.IsAppleMusicFav {
			if b.source == common.PlayerAppleMusic {
				favorite := b.controller.IsFavorite(ctx)
				appleMusicFav = favorite
				if favorite && !getTrack.IsLastFmFav {
					err := b.trackService.SetLastFmFavorite(
						model.SetFavoriteParams{
							Ctx:           ctx,
							Artist:        getTrack.Artist,
							Album:         getTrack.Album,
							Track:         getTrack.Track,
							IsFavorite:    true,
							TrackMetadata: model.TrackMetadata{},
						},
					)
					if err != nil {
						log.Warn(
							ctx, string(b.source)+" processPlayingTrack SetLastFmFavorite err", zap.Error(err),
						)
					}
				}
			}
		} else if !getTrack.IsAppleMusicFav {
			if b.source == common.PlayerAppleMusic {
				favorite := b.controller.IsFavorite(ctx)
				appleMusicFav = favorite
				if favorite {
					err := b.trackService.SetAppleMusicFavorite(
						model.SetFavoriteParams{
							Ctx:           ctx,
							Artist:        getTrack.Artist,
							Album:         getTrack.Album,
							Track:         getTrack.Track,
							IsFavorite:    true,
							TrackMetadata: model.TrackMetadata{},
						},
					)
					if err != nil {
						log.Warn(
							ctx, string(b.source)+" processPlayingTrack SetAppleMusicFavorite err", zap.Error(err),
						)
					}
				}
			}
		}
	} else {
		// 检查Apple Music喜欢状态
		if b.source == common.PlayerAppleMusic {
			appleMusicFavorite := b.controller.IsFavorite(ctx)
			lastfmFavorite, err := lastfm.IsFavorite(ctx, playerInfo.GetArtist(), playerInfo.GetTitle())
			if err != nil {
				log.Warn(
					ctx, string(b.source)+" processPlayingTrack lastfm Favorite err", zap.Error(err),
				)
			}
			appleMusicFav = appleMusicFavorite
			lastfmFavorite = lastfmFavorite
			if appleMusicFavorite {
				err := b.trackService.SetAppleMusicFavorite(
					model.SetFavoriteParams{
						Ctx:           ctx,
						Artist:        playerInfo.GetArtist(),
						Album:         playerInfo.GetAlbum(),
						Track:         playerInfo.GetTitle(),
						IsFavorite:    true,
						TrackMetadata: model.TrackMetadata{},
					},
				)
				if err != nil {
					log.Warn(
						ctx, string(b.source)+" processPlayingTrack SetAppleMusicFavorite err", zap.Error(err),
					)
				}
				_ = b.trackService.SetLastFmFavorite(
					model.SetFavoriteParams{
						Ctx:           ctx,
						Artist:        playerInfo.GetArtist(),
						Album:         playerInfo.GetAlbum(),
						Track:         playerInfo.GetTitle(),
						IsFavorite:    true,
						TrackMetadata: model.TrackMetadata{},
					},
				)
			}
			if lastfmFavorite {
				err := b.trackService.SetAppleMusicFavorite(
					model.SetFavoriteParams{
						Ctx:           ctx,
						Artist:        playerInfo.GetArtist(),
						Album:         playerInfo.GetAlbum(),
						Track:         playerInfo.GetTitle(),
						IsFavorite:    true,
						TrackMetadata: model.TrackMetadata{},
					},
				)
				if err != nil {
					log.Warn(
						ctx, string(b.source)+" processPlayingTrack SetAppleMusicFavorite err", zap.Error(err),
					)
				}
				_ = b.trackService.SetLastFmFavorite(
					model.SetFavoriteParams{
						Ctx:           ctx,
						Artist:        playerInfo.GetArtist(),
						Album:         playerInfo.GetAlbum(),
						Track:         playerInfo.GetTitle(),
						IsFavorite:    true,
						TrackMetadata: model.TrackMetadata{},
					},
				)
			}
		} else {
			// dbLastFm喜欢状态 没有被设置过，检查lastfm是否喜欢
			favorite, err := lastfm.IsFavorite(ctx, playerInfo.GetArtist(), playerInfo.GetTitle())
			if err != nil {
				log.Warn(
					ctx, string(b.source)+" processPlayingTrack lastfm Favorite err", zap.Error(err),
				)
			}
			lastFmFav = favorite
			if favorite {
				err := b.trackService.SetLastFmFavorite(
					model.SetFavoriteParams{
						Ctx:           ctx,
						Artist:        playerInfo.GetArtist(),
						Album:         playerInfo.GetAlbum(),
						Track:         playerInfo.GetTitle(),
						IsFavorite:    true,
						TrackMetadata: model.TrackMetadata{},
					},
				)
				if err != nil {
					log.Warn(
						ctx, string(b.source)+" processPlayingTrack SetLastFmFavorite err", zap.Error(err),
					)
				}
			}
		}
	}
	return appleMusicFav, lastFmFav
}

// handleTrackScrobble 处理曲目标记
func (b *BasePlayerChecker) handleTrackScrobble(ctx context.Context, playerInfo PlayerInfoHandler) {
	// 标记听歌完成
	pushTrackScrobbleReq := &lastfm.PushTrackScrobbleReq{
		Artist:      playerInfo.GetArtist(),
		AlbumArtist: playerInfo.GetArtist(),
		Track:       playerInfo.GetTitle(),
		Album:       playerInfo.GetAlbum(),
		Duration:    playerInfo.GetDuration(),
		Timestamp:   b.now.UTC().Unix(),
	}

	// Save to database
	record := &model.TrackPlayRecord{
		Artist:        pushTrackScrobbleReq.Artist,
		AlbumArtist:   pushTrackScrobbleReq.AlbumArtist,
		Track:         pushTrackScrobbleReq.Track,
		Album:         pushTrackScrobbleReq.Album,
		Duration:      pushTrackScrobbleReq.Duration,
		PlayTime:      time.Unix(pushTrackScrobbleReq.Timestamp, 0),
		Scrobbled:     true,
		MusicBrainzID: pushTrackScrobbleReq.MusicBrainzTrackID,
		TrackNumber:   int8(pushTrackScrobbleReq.TrackNumber),
		Source:        string(b.source),
	}
	_, err := lastfm.PushTrackScrobble(ctx, pushTrackScrobbleReq)
	if err != nil {
		log.Warn(ctx, string(b.source)+" handleTrackScrobble err", zap.Error(err))
		record.Scrobbled = false
	}

	if err := b.trackService.InsertTrackPlayRecord(ctx, record); err != nil {
		log.Warn(ctx, string(b.source)+" Failed to insert track play record", zap.Error(err))
	}

	// Update track play count
	incrementTrackPlayCountParams := model.IncrementTrackPlayCountParams{
		Ctx:    ctx,
		Artist: playerInfo.GetArtist(),
		Album:  playerInfo.GetAlbum(),
		Track:  playerInfo.GetTitle(),
		TrackMetadata: model.TrackMetadata{
			AlbumArtist:   playerInfo.GetAlbumArtist(),
			TrackNumber:   int8(playerInfo.GetTrackNumber()),
			Duration:      playerInfo.GetDuration(),
			Genre:         playerInfo.GetGenre(),
			Composer:      playerInfo.GetComposer(),
			ReleaseDate:   playerInfo.GetReleaseDate(),
			MusicBrainzID: playerInfo.GetMusicBrainzID(),
			Source:        playerInfo.GetUrl(), // 优先地址
			BundleID:      playerInfo.GetBundleID(),
			UniqueID:      playerInfo.GetUniqueID(),
		},
	}
	if incrementTrackPlayCountParams.TrackMetadata.Source == "" {
		incrementTrackPlayCountParams.TrackMetadata.Source = playerInfo.GetSource()
	} else {
		incrementTrackPlayCountParams.TrackMetadata.BundleID = playerInfo.GetSource()
	}
	if err := b.trackService.IncrementTrackPlayCount(
		incrementTrackPlayCountParams,
	); err != nil {
		log.Warn(ctx, string(b.source)+" Failed to increment track play count", zap.Error(err))
	}

	go func() {
		if getTrack, _ := b.trackService.GetTrack(ctx, record.Artist, record.Artist, record.Track); getTrack != nil {
			// 暂时重点关注 PlayerAppleMusic
			if b.source == common.PlayerAppleMusic {
				if !getTrack.IsAppleMusicFav {
					favorite := b.controller.IsFavorite(ctx)
					if favorite {
						err := b.trackService.SetAppleMusicFavorite(
							model.SetFavoriteParams{
								Ctx:           ctx,
								Artist:        getTrack.Artist,
								Album:         getTrack.Album,
								Track:         getTrack.Track,
								IsFavorite:    true,
								TrackMetadata: model.TrackMetadata{
									// AlbumArtist:   playerInfo.GetAlbumArtist(),
									// TrackNumber:   playerInfo.GetTrackNumber(),
									// Duration:      playerInfo.GetDuration(),
									// Genre:         playerInfo.GetGenre(),
									// Composer:      playerInfo.GetComposer(),
									// ReleaseDate:   playerInfo.GetReleaseDate(),
									// MusicBrainzID: playerInfo.GetMusicBrainzID(),
									// Source:        playerInfo.GetUrl(),
									// BundleID:      playerInfo.GetBundleID(),
									// UniqueID:      playerInfo.GetUniqueID(),
								},
							},
						)
						if err != nil {
							log.Warn(
								ctx, string(b.source)+" handleTrackScrobble SetAppleMusicFavorite err", zap.Error(err),
							)
						}
						isFavorite, err := lastfm.IsFavorite(ctx, getTrack.Album, getTrack.Track)
						if err != nil {
							log.Warn(ctx, string(b.source)+" handleTrackScrobble lastfm IsFavorite err", zap.Error(err))
						}
						if isFavorite {
							_ = lastfm.SetFavorite(ctx, getTrack.Album, getTrack.Track, true)
						}
					}
				}
			}
		}
	}()

	b.mapedTracks[b.currentTrack] = true
	b.pushCount.Add(1)
	log.Info(
		ctx, string(b.source)+"标记听歌完成", zap.String("track", pushTrackScrobbleReq.Track),
		zap.Bool("scrobbled", record.Scrobbled),
	)
}

// handleNewTrack 处理新曲目
func (b *BasePlayerChecker) handleNewTrack(ctx context.Context, playerInfo PlayerInfoHandler) {
	// 产生新歌曲
	delete(b.mapedTracks, b.previousTrack)
	b.now = time.Now()
	playingReq := lastfm.TrackUpdateNowPlayingReq{
		Artist:      playerInfo.GetArtist(),
		AlbumArtist: playerInfo.GetArtist(),
		Track:       playerInfo.GetTitle(),
		Album:       playerInfo.GetAlbum(),
		Duration:    playerInfo.GetDuration(),
	}

	log.Info(
		ctx, string(b.source)+"NowPlayingTrackInfo", zap.Any("playerInfo", playerInfo),
	)
	err := lastfm.TrackUpdateNowPlaying(ctx, &playingReq)
	if err != nil {
		log.Warn(ctx, string(b.source)+" TrackUpdateNowPlaying err", zap.Error(err))
	}
}
