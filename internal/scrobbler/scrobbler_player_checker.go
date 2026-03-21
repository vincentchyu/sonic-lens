package scrobbler

import (
	"context"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/artwork"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/core/objectstorage"
	"github.com/vincentchyu/sonic-lens/core/telemetry"
	"github.com/vincentchyu/sonic-lens/core/websocket"
	"github.com/vincentchyu/sonic-lens/internal/logic/track"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

var resolveArtworkFn = (*BasePlayerChecker).resolveArtwork

// NewBasePlayerChecker 创建基础播放器检查器
func NewBasePlayerChecker(
	controller common.PlayerController,
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
		scrobbledTracks:     make(map[string]bool),
		pushCount:           pushCount,
		atomicPlaying:       atomicPlaying,
		currentPlayingCache: currentPlayingCache,
		trackService:        trackService,
	}
}

func (b *BasePlayerChecker) buildTrackMetadata(playerInfo common.PlayerInfoHandler) model.TrackMetadata {
	metadata := model.TrackMetadata{
		AlbumArtist:   playerInfo.GetAlbumArtist(),
		TrackNumber:   int8(playerInfo.GetTrackNumber()),
		Duration:      playerInfo.GetDuration(),
		Genre:         playerInfo.GetGenre(),
		Composer:      playerInfo.GetComposer(),
		ReleaseDate:   playerInfo.GetReleaseDate(),
		MusicBrainzID: playerInfo.GetMusicBrainzID(),
		Source:        playerInfo.GetSource(),
		BundleID:      playerInfo.GetBundleID(),
		UniqueID:      playerInfo.GetUniqueID(),
		DiscNumber:    playerInfo.GetDiscNumber(),
		PlayerType:    string(b.source),
	}
	if metadata.Source == "" {
		metadata.Source = string(b.source)
	}

	metadata.Confidence = common.TrackMetadataConfidenceMedium
	switch b.source {
	case common.PlayerAudirvana:
		metadata.Confidence = common.TrackMetadataConfidenceHigh
	case common.PlayerRoon:
		metadata.Confidence = common.TrackMetadataConfidenceLow
	case common.PlayerAppleMusic:
		metadata.Confidence = b.appleMusicMetadataConfidence(playerInfo)
	}

	if appleInfo, ok := playerInfo.(*AppleMusicTrackInfoWrapper); ok && appleInfo != nil && appleInfo.TrackInfo != nil {
		metadata.ReleaseYear = appleInfo.Year
	}

	return metadata
}

func (b *BasePlayerChecker) appleMusicMetadataConfidence(
	playerInfo common.PlayerInfoHandler,
) common.TrackMetadataConfidence {
	appleInfo, ok := playerInfo.(*AppleMusicTrackInfoWrapper)
	if !ok || appleInfo == nil || appleInfo.TrackInfo == nil {
		return common.TrackMetadataConfidenceMedium
	}

	kind := strings.TrimSpace(strings.ToLower(appleInfo.Kind))
	switch {
	case strings.Contains(kind, "保真"), strings.Contains(kind, "lossless"), strings.Contains(
		kind, "wav",
	), strings.Contains(kind, "aiff"), strings.Contains(kind, "mpeg"):
		return common.TrackMetadataConfidenceHigh
	case strings.Contains(kind, "apple music aac"):
		return common.TrackMetadataConfidenceMedium
	case kind == "":
		return common.TrackMetadataConfidenceLow
	default:
		return common.TrackMetadataConfidenceMedium
	}
}

// _CheckPlayingTrack 基础播放检查逻辑
func (b *BasePlayerChecker) CheckPlayingTrack(ctx context.Context, stop <-chan struct{}) {
	b.timer = time.NewTicker(b.defaultSleep)
	b.tmpCount = 0
	b.previousTrack = ""
	b.currentTrack = ""
	b.currentArtURL = ""
	b.currentArtMime = ""
	b.currentArtObjectKey = ""
	b.currentArtTrackKey = ""
	b.currentArtResolved = false

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

	// 我觉得这里做的监控就几件事 检查新的信息 信息是什么触发上报，提供上下文数据 仅此而已
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

	var playerInfo common.PlayerInfoHandler
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
		// todo 怎么解耦 现在只是暂时拆去service，后续考虑异步分发
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
			&websocket.WsInfo{
				Type:   "stop",
				Source: string(b.source),
			},
		)
		b.atomicPlaying.Store(false)
	}
}

// processPlayingTrack 处理正在播放的曲目
func (b *BasePlayerChecker) processPlayingTrack(ctx context.Context, playerInfo common.PlayerInfoHandler) {
	snapshot := b.buildPlayingTrackSnapshot(ctx, playerInfo)
	b.currentArtURL = snapshot.coverArtURL
	b.currentArtMime = snapshot.coverArtMime
	b.currentArtObjectKey = snapshot.coverArtObjectKey

	// 统一探测并同步当前曲目的收藏状态
	favoriteState := b.trackService.ProbeAndSyncTrackFavorite(
		ctx,
		snapshot.toPlaybackEventInput(b.source, b.now),
	)

	wti := &websocket.WsInfo{
		Type:   "now_playing",
		Source: string(b.source),
		Data: websocket.WsTrackData{
			Title:             playerInfo.GetTitle(),
			Album:             playerInfo.GetAlbum(),
			Artist:            playerInfo.GetArtist(),
			AppleMusic:        favoriteState.AppleMusic,
			LastFM:            favoriteState.LastFM,
			AppleMusicState:   favoriteState.AppleMusicState,
			LastFMState:       favoriteState.LastFMState,
			FavoriteState:     favoriteState.FavoriteState,
			Duration:          snapshot.duration,
			Position:          int64(snapshot.position),
			PositionMs:        int64(math.Round(snapshot.position * 1000)),
			TrackNumber:       int8(playerInfo.GetTrackNumber()),
			DiscNumber:        int8(playerInfo.GetDiscNumber()),
			CoverArtURL:       b.currentArtURL,
			CoverArtMime:      b.currentArtMime,
			Confidence:        favoriteState.Confidence,
			PlayerInfoHandler: playerInfo,
		},
	}
	// 向WebSocket客户端广播播放信息、将播放信息写入本地缓存
	b.currentPlayingCache.Store(b.source, wti)
	b.atomicPlaying.Store(true)
	websocket.BroadcastMessage(ctx, wti)

	if snapshot.trackChanged {
		b.handleNewTrack(ctx, snapshot)
	}
	if snapshot.reachedScrobbleThreshold {
		b.handleTrackScrobble(ctx, snapshot)
	}

	b.previousTrack = snapshot.trackKey
}

func (b *BasePlayerChecker) resolveArtwork(
	ctx context.Context,
	playerInfo common.PlayerInfoHandler,
) (coverURL, coverMime, objectKey string) {
	provider, ok := playerInfo.(common.ArtworkProvider)
	if !ok {
		return "", "", ""
	}
	albumSeed := artwork.BuildAlbumArtworkSeed(
		playerInfo.GetAlbumArtist(), playerInfo.GetArtist(), playerInfo.GetAlbum(),
	)

	obj := objectstorage.Get()
	if obj != nil {
		objectKey = obj.BuildOriginalObjectKey(albumSeed)
		exists, contentType, existsErr := obj.CheckObjectExists(ctx, objectKey)
		if existsErr != nil {
			log.Warn(ctx, string(b.source)+"resolveArtwork check object err", zap.Error(existsErr))
		} else if exists {
			return obj.GetObjectCDNURL(objectKey), contentType, objectKey
		}
	}

	art, err := provider.GetArtwork(ctx)
	if err != nil {
		log.Warn(ctx, string(b.source)+"resolveArtwork get artwork err", zap.Error(err))
		return "", "", ""
	}
	if art == nil || len(art.Data) == 0 {
		log.Warn(ctx, string(b.source)+"resolveArtwork art == nil")
		return "", "", ""
	}

	if obj != nil {
		if objectKey == "" {
			objectKey = obj.BuildOriginalObjectKey(albumSeed)
		}
		if uploadErr := obj.UploadBytesToObject(ctx, objectKey, art.Data, art.MimeType); uploadErr != nil {
			log.Warn(ctx, string(b.source)+"resolveArtwork upload object err", zap.Error(uploadErr))
		} else {
			return obj.GetObjectCDNURL(objectKey), art.MimeType, objectKey
		}
	}
	// 缓存兜底
	seed := artwork.DefaultStore.GetKeyForSeed(provider.GetArtworkKey(ctx))
	if e, ok := artwork.DefaultStore.Get(seed); ok {
		return artwork.URLForKey(seed), e.MimeType, ""
	}

	key := artwork.DefaultStore.Put(art.CacheKey, art.Data, art.MimeType)
	return artwork.URLForKey(key), art.MimeType, ""
}

// handleTrackScrobble 处理曲目标记
func (b *BasePlayerChecker) handleTrackScrobble(ctx context.Context, snapshot playingTrackSnapshot) {
	result := b.trackService.HandleTrackPlaybackThreshold(ctx, snapshot.toPlaybackEventInput(b.source, b.now))
	b.scrobbledTracks[snapshot.trackKey] = true
	b.pushCount.Add(1)
	log.Info(
		ctx, string(b.source)+"标记听歌完成",
		zap.String("track", snapshot.playerInfo.GetTitle()),
		zap.Bool("scrobbled", result.Scrobbled),
	)
}

// handleNewTrack 处理新曲目
func (b *BasePlayerChecker) handleNewTrack(ctx context.Context, snapshot playingTrackSnapshot) {
	delete(b.scrobbledTracks, b.previousTrack)
	b.now = time.Now()
	log.Info(
		ctx, string(b.source)+"NowPlayingTrackInfo", zap.Any("playerInfo", snapshot.playerInfo),
	)
	b.trackService.HandleNowPlayingStarted(ctx, snapshot.toPlaybackEventInput(b.source, b.now))
}

func (b *BasePlayerChecker) buildPlayingTrackSnapshot(
	ctx context.Context,
	playerInfo common.PlayerInfoHandler,
) playingTrackSnapshot {
	if audirvanaInfo, ok := playerInfo.(*AudirvanaTrackInfoWrapper); ok {
		audirvanaInfo.LogResolvedPosition(ctx)
	}

	trackKey := b.buildCurrentTrackKey(playerInfo)
	b.currentTrack = trackKey

	position := playerInfo.GetPosition()
	duration := playerInfo.GetDuration()
	trackChanged := b.currentTrack != b.previousTrack
	coverArtURL, coverArtMime, coverArtObjectKey := b.resolveArtworkForSnapshot(
		ctx, playerInfo, trackKey, trackChanged,
	)
	metadata := b.buildTrackMetadata(playerInfo)
	metadata.CoverArtURL = coverArtURL
	metadata.CoverArtMime = coverArtMime
	metadata.CoverArtObjectKey = coverArtObjectKey

	snapshot := playingTrackSnapshot{
		playerInfo:              playerInfo,
		metadata:                metadata,
		trackKey:                trackKey,
		position:                position,
		duration:                duration,
		trackChanged:            trackChanged,
		coverArtURL:             coverArtURL,
		coverArtMime:            coverArtMime,
		coverArtObjectKey:       coverArtObjectKey,
		controllerFavoriteKnown: b.source == common.PlayerAppleMusic,
	}
	if snapshot.controllerFavoriteKnown {
		snapshot.controllerFavorite = b.controller.IsFavorite(ctx)
	}
	if duration > 0 {
		snapshot.reachedScrobbleThreshold = position/float64(duration) > b.percentScrobble &&
			!b.scrobbledTracks[trackKey]
	}

	return snapshot
}

func (b *BasePlayerChecker) resolveArtworkForSnapshot(
	ctx context.Context,
	playerInfo common.PlayerInfoHandler,
	trackKey string,
	trackChanged bool,
) (string, string, string) {
	if !trackChanged && b.currentArtTrackKey == trackKey && b.currentArtResolved {
		return b.currentArtURL, b.currentArtMime, b.currentArtObjectKey
	}

	coverArtURL, coverArtMime, coverArtObjectKey := resolveArtworkFn(b, ctx, playerInfo)
	b.currentArtTrackKey = trackKey
	b.currentArtResolved = coverArtURL != ""
	if b.currentArtResolved {
		b.currentArtURL = coverArtURL
		b.currentArtMime = coverArtMime
		b.currentArtObjectKey = coverArtObjectKey
	} else {
		b.currentArtURL = ""
		b.currentArtMime = ""
		b.currentArtObjectKey = ""
	}
	return coverArtURL, coverArtMime, coverArtObjectKey
}

func (b *BasePlayerChecker) buildCurrentTrackKey(playerInfo common.PlayerInfoHandler) string {
	trackKey := playerInfo.GetTitle()
	if b.source == common.PlayerAudirvana {
		if url := playerInfo.GetUrl(); url != "" {
			trackKey = url + playerInfo.GetTitle()
		}
	}
	return trackKey
}
