package scrobbler

import (
	"context"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
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
		AlbumArtist:         playerInfo.GetAlbumArtist(),
		AlbumSubtitle:       playerInfo.GetAlbumSubtitle(),
		TrackNumber:         int8(playerInfo.GetTrackNumber()),
		Duration:            playerInfo.GetDuration(),
		Genre:               playerInfo.GetGenre(),
		Composer:            playerInfo.GetComposer(),
		ReleaseDate:         playerInfo.GetReleaseDate(),
		OriginalReleaseDate: playerInfo.GetOriginalReleaseDate(),
		MusicBrainzID:       playerInfo.GetMusicBrainzID(),
		Source:              playerInfo.GetSource(),
		BundleID:            playerInfo.GetBundleID(),
		UniqueID:            playerInfo.GetUniqueID(),
		DiscNumber:          playerInfo.GetDiscNumber(),
		PlayerType:          string(b.source),
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
	defer b.timer.Stop()
	defer b.endCurrentTrackPlaybackSpan()
	b.tmpCount = 0
	b.previousTrack = ""
	b.previousPosition = 0
	b.currentTrack = ""
	b.currentArtURL = ""
	b.currentArtMime = ""
	b.currentArtObjectKey = ""
	b.currentArtTrackKey = ""
	b.currentArtResolved = false
	b.currentTrackCtx = nil
	b.currentTrackSpan = nil
	b.currentTrackSpanKey = ""
	b.currentFavorite = track.TrackFavoriteProjection{}
	b.currentFavoriteKey = ""
	b.favoriteKnown = false
	b.controllerFavoriteValue = false
	b.controllerFavoriteCheckedAt = time.Time{}
	b.controllerFavoriteTrackKey = ""
	b.trackChanged = true // 第一次启动
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
	/*	ctx, span := telemetry.StartSpanForTracerName(
			ctx, _TracerName, string(b.source)+"_CheckPlayingTrack",
		)
		defer span.End()*/

	log.Debug(ctx, string(b.source)+" Checking playing track..."+time.Now().String())

	// 我觉得这里做的监控就几件事 检查新的信息 信息是什么触发上报，提供上下文数据 仅此而已
	b.tmpCount++
	if b.tmpCount > b.checkCount && !b.isLongCheck { // 检查100次依旧没有播放检查轮训放大到60秒
		b.timer.Reset(b.longSleep)
		b.isLongCheck = true
		log.Info(
			ctx, string(b.source)+"检查100次依旧没有播放检查轮训放大到60秒",
			zap.Uint32("共计上传歌曲标记", b.pushCount.Load()),
		)
	}
	if b.isLongCheck {
		log.Info(ctx, string(b.source)+"60秒检查", zap.Uint32("共计上传歌曲标记", b.pushCount.Load()))
	}

	running := b.controller.IsRunning(ctx)
	log.Debug(ctx, string(b.source)+" 程序运行是否运行", zap.Bool("running", running))

	var playerInfo common.PlayerInfoHandler
	if running {
		playerInfo = nil
		state, _ := b.controller.GetState(ctx)
		log.Debug(ctx, string(b.source)+" 播放状态", zap.Any("state", state))
		if state == common.PlayerStatePlaying {
			if b.tmpCount > b.checkCount {
				b.isLongCheck = false
				b.timer.Reset(b.defaultSleep)
			}
			b.tmpCount = 0
			playerInfo = b.controller.GetNowPlayingTrackInfo(ctx)
			if playerInfo == nil {
				log.Warn(ctx, string(b.source)+" 当前处于播放态但未获取到曲目信息")
			}
		} else {
			if _, ok := b.currentPlayingCache.Load(b.source); ok {
				log.Info(ctx, string(b.source)+" 检测到播放停止或暂停", zap.String("state", string(state)))
				b.currentPlayingCache.Delete(b.source)
				b.handleStopEvent(ctx)
			}
		}
	}

	if playerInfo != nil {
		// todo 怎么解耦 现在只是暂时拆去service，后续考虑异步分发
		b.processPlayingTrack(ctx, playerInfo)
	}
}

// handleStopEvent 处理停止事件
func (b *BasePlayerChecker) handleStopEvent(ctx context.Context) {
	stopCtx := ctx
	if b.currentTrackCtx != nil {
		stopCtx = b.currentTrackCtx
	}

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
		addTrackPlaybackEvent(
			b.currentTrackSpan,
			"player_stopped",
			attribute.String("player.source", string(b.source)),
		)
		websocket.BroadcastMessage(
			stopCtx,
			&websocket.WsInfo{
				Type:   "stop",
				Source: string(b.source),
			},
		)
		b.atomicPlaying.Store(false)
		log.Info(ctx, string(b.source)+" 已广播停止播放事件")
	}

	b.endCurrentTrackPlaybackSpan()
}

// processPlayingTrack 处理正在播放的曲目
func (b *BasePlayerChecker) processPlayingTrack(ctx context.Context, playerInfo common.PlayerInfoHandler) {
	trackKey := b.buildCurrentTrackKey(playerInfo)
	trackRestarted := b.didTrackRestart(playerInfo, trackKey)
	trackChanged := trackKey != b.previousTrack || trackRestarted || !b.hasActiveTrackPlaybackSpan(trackKey)
	trackCtx := b.ensureTrackPlaybackContext(ctx, trackKey, trackChanged)

	var snapshot playingTrackSnapshot
	if trackChanged {
		resolveCtx, resolveSpan := startPlayerTrackStageSpan(trackCtx, b.source, "resolve_now_playing")
		snapshot = b.buildPlayingTrackSnapshot(resolveCtx, playerInfo, trackKey, trackChanged)
		resolveSpan.SetAttributes(
			attribute.Bool("track.changed", snapshot.trackChanged),
			attribute.Bool("track.cover_art_present", snapshot.coverArtURL != ""),
			attribute.Float64("track.position_sec", snapshot.position),
		)
		resolveSpan.End()
	} else {
		snapshot = b.buildPlayingTrackSnapshot(trackCtx, playerInfo, trackKey, trackChanged)
	}

	setTrackPlaybackSpanAttributes(b.currentTrackSpan, b.source, snapshot)
	b.currentArtURL = snapshot.coverArtURL
	b.currentArtMime = snapshot.coverArtMime
	b.currentArtObjectKey = snapshot.coverArtObjectKey

	// 统一探测并同步当前曲目的收藏状态
	favoriteState := b.probeFavoriteState(trackCtx, snapshot)

	wti := &websocket.WsInfo{
		Type:   "now_playing",
		Source: string(b.source),
		Data: websocket.WsTrackData{
			Title:             playerInfo.GetTitle(),
			Album:             playerInfo.GetAlbum(),
			AlbumSubtitle:     playerInfo.GetAlbumSubtitle(),
			Artist:            playerInfo.GetArtist(),
			AppleMusic:        favoriteState.AppleMusic,
			LastFM:            favoriteState.LastFM,
			AppleMusicState:   favoriteState.AppleMusicState,
			LastFMState:       favoriteState.LastFMState,
			FavoriteState:     favoriteState.FavoriteState,
			Duration:          snapshot.duration,
			Position:          int64(snapshot.position),
			PositionMs:        int64(math.Round(snapshot.position * 1000)),
			SampleRate:        playerInfo.GetSampleRate(),
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
	websocket.BroadcastMessage(trackCtx, wti)

	if snapshot.trackChanged {
		addTrackPlaybackEvent(
			b.currentTrackSpan,
			"track_started",
			attribute.String("track.title", snapshot.playerInfo.GetTitle()),
			attribute.String("track.artist", snapshot.playerInfo.GetArtist()),
			attribute.String("track.album", snapshot.playerInfo.GetAlbum()),
		)
		b.handleNewTrack(trackCtx, snapshot)
	}
	if snapshot.reachedScrobbleThreshold {
		thresholdCtx, thresholdSpan := startPlayerTrackStageSpan(trackCtx, b.source, "handle_playback_threshold")
		b.handleTrackScrobble(thresholdCtx, snapshot)
		thresholdSpan.SetAttributes(
			attribute.Float64("track.position_sec", snapshot.position),
			attribute.Int64("track.duration_sec", snapshot.duration),
		)
		thresholdSpan.End()
	}

	b.previousTrack = snapshot.trackKey
	b.previousPosition = snapshot.position
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
	addTrackPlaybackEvent(
		b.currentTrackSpan,
		"scrobble_threshold_reached",
		attribute.Bool("track.scrobbled", result.Scrobbled),
		attribute.Float64("track.position_sec", snapshot.position),
		attribute.Int64("track.duration_sec", snapshot.duration),
	)
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
	trackKey string,
	trackChanged bool,
) playingTrackSnapshot {
	if audirvanaInfo, ok := playerInfo.(*AudirvanaTrackInfoWrapper); ok {
		audirvanaInfo.LogResolvedPosition(ctx)
	}

	b.currentTrack = trackKey

	position := playerInfo.GetPosition()
	duration := playerInfo.GetDuration()
	b.trackChanged = trackChanged
	coverArtURL, coverArtMime, coverArtObjectKey := b.resolveArtworkForSnapshot(
		ctx, playerInfo, trackKey, b.trackChanged,
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
		trackChanged:            b.trackChanged,
		coverArtURL:             coverArtURL,
		coverArtMime:            coverArtMime,
		coverArtObjectKey:       coverArtObjectKey,
		controllerFavoriteKnown: b.source == common.PlayerAppleMusic,
		// albumTitleMetadata:      playerInfo.GetAlbumTitleMetadata(),
	}
	if snapshot.controllerFavoriteKnown {
		snapshot.controllerFavorite = b.resolveControllerFavorite(ctx, trackKey, trackChanged)
	}
	if duration > 0 {
		snapshot.reachedScrobbleThreshold = position/float64(duration) > b.percentScrobble &&
			!b.scrobbledTracks[trackKey]
	}
	snapshot.traceID, snapshot.rootSpanID, snapshot.traceSampled = b.currentTrackTraceContext()

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

func (b *BasePlayerChecker) didTrackRestart(
	playerInfo common.PlayerInfoHandler,
	trackKey string,
) bool {
	if trackKey == "" || trackKey != b.previousTrack {
		return false
	}

	duration := playerInfo.GetDuration()
	if duration <= 0 {
		return false
	}

	position := playerInfo.GetPosition()
	if position >= b.previousPosition {
		return false
	}

	previousProgress := b.previousPosition / float64(duration)
	currentProgress := position / float64(duration)

	return previousProgress >= b.percentScrobble && currentProgress <= 0.2
}

func (b *BasePlayerChecker) hasActiveTrackPlaybackSpan(trackKey string) bool {
	return b.currentTrackSpan != nil && b.currentTrackCtx != nil && b.currentTrackSpanKey == trackKey
}

func (b *BasePlayerChecker) probeFavoriteState(
	trackCtx context.Context,
	snapshot playingTrackSnapshot,
) track.TrackFavoriteProbeResult {
	input := snapshot.toPlaybackEventInput(b.source, b.now)
	if snapshot.trackChanged {
		favoriteCtx, favoriteSpan := startPlayerTrackStageSpan(trackCtx, b.source, "sync_favorite_state")
		favoriteState := b.trackService.ProbeAndSyncTrackFavorite(favoriteCtx, input)
		setFavoriteStateSpanAttributes(
			favoriteSpan,
			favoriteState.FavoriteState,
			favoriteState.AppleMusic,
			favoriteState.LastFM,
		)
		favoriteSpan.End()
		b.updateFavoriteProjection(snapshot.trackKey, favoriteState.TrackFavoriteProjection)
		return favoriteState
	}

	favoriteState := b.trackService.ProbeAndSyncTrackFavorite(trackCtx, input)
	if b.favoriteProjectionChanged(snapshot.trackKey, favoriteState.TrackFavoriteProjection) {
		_, favoriteSpan := startPlayerTrackStageSpan(trackCtx, b.source, "sync_favorite_state")
		setFavoriteStateSpanAttributes(
			favoriteSpan,
			favoriteState.FavoriteState,
			favoriteState.AppleMusic,
			favoriteState.LastFM,
		)
		favoriteSpan.End()
		addTrackPlaybackEvent(
			b.currentTrackSpan,
			"favorite_state_changed",
			attribute.String("track.favorite_state", string(favoriteState.FavoriteState)),
			attribute.Bool("track.favorite.apple_music", favoriteState.AppleMusic),
			attribute.Bool("track.favorite.lastfm", favoriteState.LastFM),
		)
	}
	b.updateFavoriteProjection(snapshot.trackKey, favoriteState.TrackFavoriteProjection)
	return favoriteState
}

func (b *BasePlayerChecker) ensureTrackPlaybackContext(
	ctx context.Context,
	trackKey string,
	trackChanged bool,
) context.Context {
	if trackChanged {
		b.endCurrentTrackPlaybackSpan()
		trackCtx, span := telemetry.StartSpanForTracerName(
			ctx, _TracerName, string(b.source)+"_TrackPlayback",
		)
		b.currentTrackCtx = trackCtx
		b.currentTrackSpan = span
		b.currentTrackSpanKey = trackKey
		return trackCtx
	}
	if b.currentTrackCtx != nil {
		return b.currentTrackCtx
	}
	return ctx
}

func (b *BasePlayerChecker) endCurrentTrackPlaybackSpan() {
	if b.currentTrackSpan != nil {
		b.currentTrackSpan.End()
	}
	b.currentTrackCtx = nil
	b.currentTrackSpan = nil
	b.currentTrackSpanKey = ""
	b.previousPosition = 0
	b.currentFavorite = track.TrackFavoriteProjection{}
	b.currentFavoriteKey = ""
	b.favoriteKnown = false
	b.controllerFavoriteValue = false
	b.controllerFavoriteCheckedAt = time.Time{}
	b.controllerFavoriteTrackKey = ""
}

func (b *BasePlayerChecker) favoriteProjectionChanged(
	trackKey string,
	projection track.TrackFavoriteProjection,
) bool {
	if !b.favoriteKnown || b.currentFavoriteKey != trackKey {
		return true
	}
	return b.currentFavorite != projection
}

func (b *BasePlayerChecker) updateFavoriteProjection(
	trackKey string,
	projection track.TrackFavoriteProjection,
) {
	b.currentFavorite = projection
	b.currentFavoriteKey = trackKey
	b.favoriteKnown = true
}

func (b *BasePlayerChecker) resolveControllerFavorite(
	ctx context.Context,
	trackKey string,
	trackChanged bool,
) bool {
	if trackChanged ||
		b.controllerFavoriteTrackKey != trackKey ||
		b.controllerFavoriteCheckedAt.IsZero() ||
		time.Since(b.controllerFavoriteCheckedAt) >= controllerFavoriteRefreshInterval {
		b.controllerFavoriteValue = b.controller.IsFavorite(ctx)
		b.controllerFavoriteCheckedAt = time.Now()
		b.controllerFavoriteTrackKey = trackKey
	}
	return b.controllerFavoriteValue
}

func (b *BasePlayerChecker) currentTrackTraceContext() (string, string, bool) {
	if b.currentTrackSpan == nil {
		return "", "", false
	}
	spanCtx := b.currentTrackSpan.SpanContext()
	if !spanCtx.IsValid() {
		return "", "", false
	}
	return spanCtx.TraceID().String(), spanCtx.SpanID().String(), spanCtx.IsSampled()
}
