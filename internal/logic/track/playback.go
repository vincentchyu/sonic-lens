package track

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/lastfm"
	"github.com/vincentchyu/sonic-lens/core/log"
	redisc "github.com/vincentchyu/sonic-lens/core/redis"
	"github.com/vincentchyu/sonic-lens/internal/cache"
	artworklogic "github.com/vincentchyu/sonic-lens/internal/logic/artwork"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

var (
	lastfmPushTrackScrobble     = lastfm.PushTrackScrobble
	lastfmTrackUpdateNowPlaying = lastfm.TrackUpdateNowPlaying
	lastfmIsFavorite            = lastfm.IsFavorite
	artworkEnsureAlbumCover     = artworklogic.EnsureAlbumCover
	cacheGetEnglishGenre        = cache.GetEnglishGenre
	timeNow                     = time.Now
)

const (
	lastfmFavoritePositiveProbeInterval = 15 * time.Second
	lastfmFavoriteNegativeProbeMax      = 2 * time.Minute
	mobileScrobbleDuplicateWindow       = 60 * time.Second
)

func allowsLastfmNowPlaying(playerSource common.PlayerType) bool {
	return playerSource != common.PlayerFoobar2000
}

func allowsLastfmScrobble(playerSource common.PlayerType) bool {
	return playerSource != common.PlayerFoobar2000
}

// PlaybackEventInput 表示播放事件编排所需的标准化输入。
type PlaybackEventInput struct {
	Artist                  string
	AlbumArtist             string
	Album                   string
	AlbumSubtitle           string
	AlbumTitleMetadata      *common.AlbumTitleMetadata
	Track                   string
	TrackNumber             int8
	DiscNumber              int8
	Duration                int64
	MusicBrainzID           string
	Metadata                model.TrackMetadata
	PlayerSource            common.PlayerType
	TrackChanged            bool
	PlaybackStartedAt       time.Time
	ControllerFavoriteKnown bool
	ControllerFavorite      bool
	CoverArtURL             string
	CoverArtMime            string
	CoverArtObjectKey       string
	TraceID                 string
	RootSpanID              string
	TraceSampled            bool
}

// TrackFavoriteProbeResult 表示喜欢状态探测后的结果。
type TrackFavoriteProbeResult struct {
	TrackFavoriteProjection
	Confidence common.TrackMetadataConfidence
}

// PlaybackThresholdResult 表示达到 scrobble 阈值后的处理结果。
type PlaybackThresholdResult struct {
	Scrobbled bool
}

// ExternalPlaybackRecordInput 表示移动端或外部终端采集上报的原始 Apple Music 播放记录。
type ExternalPlaybackRecordInput struct {
	IdempotencyKey  string                         `json:"idempotency_key"`
	Title           string                         `json:"title"`
	Artist          string                         `json:"artist"`
	AlbumArtist     string                         `json:"album_artist"`
	Album           string                         `json:"album"`
	Genre           string                         `json:"genre"`
	Composer        string                         `json:"composer"`
	TrackNumber     int8                           `json:"track_number"`
	DiscNumber      int8                           `json:"disc_number"`
	Duration        int64                          `json:"duration"`
	ReleaseDate     string                         `json:"release_date"`
	ReleaseYear     int                            `json:"release_year"`
	PlayedAt        time.Time                      `json:"played_at"`
	UniqueID        string                         `json:"unique_id"`
	BundleID        string                         `json:"bundle_id"`
	IsAppleMusicFav *bool                          `json:"is_apple_music_fav,omitempty"`
	Confidence      common.TrackMetadataConfidence `json:"confidence,omitzero"`
}

// MobileScrobbleItemResult 表示单条移动端播放记录的接入处理结果。
type MobileScrobbleItemResult struct {
	IdempotencyKey string `json:"idempotency_key"`
	Status         string `json:"status"`
	Scrobbled      bool   `json:"scrobbled,omitzero"`
	Reason         string `json:"reason,omitempty"`
}

// MobileScrobbleBatchResult 表示移动端批量补录的汇总结果。
type MobileScrobbleBatchResult struct {
	AcceptedCount  int                        `json:"accepted_count"`
	DuplicateCount int                        `json:"duplicate_count"`
	ScrobbledCount int                        `json:"scrobbled_count"`
	FailedCount    int                        `json:"failed_count"`
	AcceptedKeys   []string                   `json:"accepted_keys"`
	FailedKeys     []string                   `json:"failed_keys"`
	Results        []MobileScrobbleItemResult `json:"results,omitempty"`
}

// NormalizeAppleMusicTitle 统一清洗并简繁归一 Apple Music 曲目标题。
func NormalizeAppleMusicTitle(title string) string {
	return common.ConversionSimplifiedFx(common.UnityFixAll(common.TrackCustomFit(strings.TrimSpace(title))))
}

// ParseAppleMusicAlbumMetadata 统一拆解 Apple Music 专辑主标题、括号版本说明及连字符发行类型后缀。
func ParseAppleMusicAlbumMetadata(album string) (cleanAlbum string, subtitle string, meta common.AlbumTitleMetadata) {
	trimmed := common.UnityFixAll(strings.TrimSpace(album))
	meta = common.ParseAlbumTitleMetadata(trimmed)
	cleanAlbum = common.ConversionSimplifiedFx(common.UnityFixAll(meta.OfficialTitle))
	_, rawSubtitle := common.ParseAlbumTitleAndSubtitle(trimmed)
	subtitle = common.ConversionSimplifiedFx(common.UnityFixAll(rawSubtitle))
	return cleanAlbum, subtitle, meta
}

// NormalizeAppleMusicArtist 统一清洗并简繁归一 Apple Music 曲目艺术家。
func NormalizeAppleMusicArtist(artist string) string {
	return common.ConversionSimplifiedFx(common.ArtistCustomFit(strings.TrimSpace(artist)))
}

// NormalizeAppleMusicAlbumArtist 统一计算 Apple Music 专辑艺术家，缺失时回退曲目艺术家。
func NormalizeAppleMusicAlbumArtist(albumArtist, artist string) string {
	preferred := cmp.Or(
		common.ArtistCustomFit(strings.TrimSpace(albumArtist)),
		common.ArtistCustomFit(strings.TrimSpace(artist)),
	)
	return common.ConversionSimplifiedFx(preferred)
}

// BuildAppleMusicPlaybackEventInput 将外部采集的 Apple Music 原始记录构造为标准播放事件输入。
func BuildAppleMusicPlaybackEventInput(raw ExternalPlaybackRecordInput) PlaybackEventInput {
	cleanTitle := NormalizeAppleMusicTitle(raw.Title)
	cleanAlbum, albumSubtitle, albumMeta := ParseAppleMusicAlbumMetadata(raw.Album)
	cleanArtist := NormalizeAppleMusicArtist(raw.Artist)
	cleanAlbumArtist := NormalizeAppleMusicAlbumArtist(raw.AlbumArtist, raw.Artist)
	cleanGenre := cacheGetEnglishGenre(strings.TrimSpace(raw.Genre))
	cleanComposer := common.ConversionSimplifiedFx(strings.TrimSpace(raw.Composer))

	discNumber := raw.DiscNumber
	if discNumber <= 0 && raw.TrackNumber > 0 {
		discNumber = 1
	}

	confidence := raw.Confidence
	if confidence <= 0 {
		if raw.TrackNumber > 0 && raw.Duration > 0 {
			confidence = common.TrackMetadataConfidenceMedium
		} else {
			confidence = common.TrackMetadataConfidenceLow
		}
	}

	metadata := model.TrackMetadata{
		AlbumArtist:   cleanAlbumArtist,
		AlbumSubtitle: albumSubtitle,
		TrackNumber:   raw.TrackNumber,
		Duration:      raw.Duration,
		Genre:         cleanGenre,
		Composer:      cleanComposer,
		ReleaseDate:   strings.TrimSpace(raw.ReleaseDate),
		ReleaseYear:   raw.ReleaseYear,
		Source:        string(common.PlayerAppleMusic),
		BundleID:      strings.TrimSpace(raw.BundleID),
		UniqueID:      strings.TrimSpace(raw.UniqueID),
		DiscNumber:    discNumber,
		PlayerType:    string(common.PlayerAppleMusic),
		Confidence:    confidence,
	}

	return PlaybackEventInput{
		Artist:                  cleanArtist,
		AlbumArtist:             cleanAlbumArtist,
		Album:                   cleanAlbum,
		AlbumSubtitle:           albumSubtitle,
		AlbumTitleMetadata:      &albumMeta,
		Track:                   cleanTitle,
		TrackNumber:             raw.TrackNumber,
		DiscNumber:              discNumber,
		Duration:                raw.Duration,
		Metadata:                metadata,
		PlayerSource:            common.PlayerAppleMusic,
		TrackChanged:            true,
		PlaybackStartedAt:       raw.PlayedAt,
		ControllerFavoriteKnown: false, // 离线批量补录绝不触发宿主机 macOS Music.app AppleScript 收藏操作
		ControllerFavorite:      false,
	}
}

type favoriteProbeState struct {
	appleKnown bool
	appleLiked bool
	lastKnown  bool
	lastLiked  bool
}

func (s *TrackServiceImpl) canSafelyLookupCurrentTrack(metadata model.TrackMetadata) bool {
	if metadata.Confidence >= common.TrackMetadataConfidenceHigh {
		return true
	}
	return metadata.TrackNumber > 0 && metadata.DiscNumber > 0
}

// ProbeAndSyncTrackFavorite 统一探测并同步当前曲目的收藏状态。
func (s *TrackServiceImpl) ProbeAndSyncTrackFavorite(
	ctx context.Context,
	input PlaybackEventInput,
) TrackFavoriteProbeResult {
	log.Info(
		ctx,
		"开始探测并同步收藏状态",
		zap.String("artist", input.Artist),
		zap.String("album", input.Album),
		zap.String("track", input.Track),
		zap.Bool("track_changed", input.TrackChanged),
		zap.String("player_source", string(input.PlayerSource)),
	)
	result := TrackFavoriteProbeResult{
		Confidence: input.Metadata.Confidence,
	}
	projectionInput := FavoriteProjectionInput{
		Artist:        input.Artist,
		Album:         input.Album,
		AlbumSubtitle: input.AlbumSubtitle,
		Track:         input.Track,
		TrackNumber:   input.TrackNumber,
		DiscNumber:    input.DiscNumber,
		Metadata:      input.Metadata,
	}
	trackKey := s.buildLikeTrackKey(input)
	cacheVersion := favoriteProjectionVersion.Load()

	if !s.canSafelyLookupCurrentTrack(input.Metadata) {
		if projection, ok := s.getCachedFavoriteProjection(trackKey, cacheVersion, favoriteProbeState{}); ok {
			result.TrackFavoriteProjection = projection
			return result
		}
		projection, err := s.buildFavoriteProjection(ctx, projectionInput)
		if err != nil {
			log.Warn(ctx, "构建收藏态投影视图失败", zap.Error(err))
		} else {
			result.TrackFavoriteProjection = projection
			s.updateFavoriteProjectionCache(trackKey, cacheVersion, favoriteProbeState{}, projection)
		}
		log.Debug(
			ctx,
			"当前元数据置信度不足，跳过收藏探测",
			zap.String("artist", input.Artist),
			zap.String("album", input.Album),
			zap.String("track", input.Track),
			zap.Int8("track_number", input.Metadata.TrackNumber),
			zap.Int8("disc_number", input.Metadata.DiscNumber),
		)
		return result
	}

	probe := favoriteProbeState{}

	if input.ControllerFavoriteKnown {
		probe.appleKnown = true
		probe.appleLiked = input.ControllerFavorite
	}

	lastFmFavorite, lastFmKnown := s.probeLastfmFavorite(ctx, input)
	if lastFmKnown {
		probe.lastKnown = true
		probe.lastLiked = lastFmFavorite
	}

	probeChanged := s.shouldSyncLikeWrite(trackKey, input.TrackChanged, probe)
	if !probeChanged {
		if projection, ok := s.getCachedFavoriteProjection(trackKey, cacheVersion, probe); ok {
			result.TrackFavoriteProjection = projection
			return result
		}
	}

	projection, err := s.buildFavoriteProjection(ctx, projectionInput)
	if err != nil {
		log.Warn(ctx, "构建收藏态投影视图失败", zap.Error(err))
	} else {
		result.TrackFavoriteProjection = projection
	}

	if !probeChanged {
		if projection.FavoriteState != "" {
			s.updateFavoriteProjectionCache(trackKey, cacheVersion, probe, projection)
		}
		return result
	}

	lockKey := s.buildLikeLockKey(input)
	if err := s.withLikeWriteLock(
		ctx, lockKey, func() error {
			currentTrack, _ := modelGetTrackByIdentityWithSubtitle(
				ctx, input.Artist, input.Album, input.AlbumSubtitle, input.Track, input.TrackNumber, input.DiscNumber,
			)
			trackAppleFav := currentTrack != nil && currentTrack.IsAppleMusicFav
			trackLastFav := currentTrack != nil && currentTrack.IsLastFmFav

			if input.PlayerSource == common.PlayerAppleMusic {
				shouldApplyFavorite := (probe.appleKnown && probe.appleLiked) || (probe.lastKnown && probe.lastLiked)
				if shouldApplyFavorite && (!trackAppleFav || !trackLastFav) {
					projection, err := s.SetTrackFavorite(
						ctx,
						input.Artist,
						input.Album,
						input.Track,
						model.TrackFavoriteEventSourceAppleMusic,
						true,
						input.Metadata,
					)
					result.TrackFavoriteProjection = projection
					return err
				}
				return nil
			}

			if probe.lastKnown && probe.lastLiked && !trackLastFav {
				projection, err := s.SetTrackFavorite(
					ctx,
					input.Artist,
					input.Album,
					input.Track,
					model.TrackFavoriteEventSourceLastFm,
					true,
					input.Metadata,
				)
				result.TrackFavoriteProjection = projection
				return err
			}

			return nil
		},
	); err != nil {
		log.Warn(ctx, "ProbeAndSyncTrackFavorite sync favorite err", zap.Error(err))
	}

	if result.TrackFavoriteProjection.FavoriteState != "" {
		s.updateFavoriteProjectionCache(
			trackKey,
			favoriteProjectionVersion.Load(),
			probe,
			result.TrackFavoriteProjection,
		)
	}

	log.Info(
		ctx,
		"完成探测并同步收藏状态",
		zap.String("artist", input.Artist),
		zap.String("album", input.Album),
		zap.String("track", input.Track),
		zap.Bool("apple_music_favorite", result.AppleMusic),
		zap.Bool("last_fm_favorite", result.LastFM),
	)
	return result
}

// HandleNowPlayingStarted 统一处理新曲开始时的 now playing 上报。
func (s *TrackServiceImpl) HandleNowPlayingStarted(ctx context.Context, input PlaybackEventInput) {
	log.Info(
		ctx,
		"开始上报当前播放曲目",
		zap.String("artist", input.Artist),
		zap.String("album", input.Album),
		zap.String("track", input.Track),
		zap.String("player_source", string(input.PlayerSource)),
		zap.Int64("duration", input.Duration),
	)
	if !allowsLastfmNowPlaying(input.PlayerSource) {
		log.Info(
			ctx,
			"当前来源跳过 Last.fm now playing 上报",
			zap.String("artist", input.Artist),
			zap.String("album", input.Album),
			zap.String("track", input.Track),
			zap.String("player_source", string(input.PlayerSource)),
		)
		return
	}
	req := lastfm.TrackUpdateNowPlayingReq{
		Artist:      input.Artist,
		AlbumArtist: fallbackAlbumArtist(input.AlbumArtist, input.Artist),
		Track:       input.Track,
		Album:       input.Album,
		Duration:    input.Duration,
	}
	if err := lastfmTrackUpdateNowPlaying(ctx, &req); err != nil {
		log.Warn(ctx, "HandleNowPlayingStarted TrackUpdateNowPlaying err", zap.Error(err))
		return
	}
	log.Info(
		ctx,
		"当前播放曲目上报完成",
		zap.String("artist", input.Artist),
		zap.String("album", input.Album),
		zap.String("track", input.Track),
	)
}

// HandleTrackPlaybackThreshold 统一处理达到上报阈值时的播放副作用。
func (s *TrackServiceImpl) HandleTrackPlaybackThreshold(
	ctx context.Context,
	input PlaybackEventInput,
) PlaybackThresholdResult {
	log.Info(
		ctx,
		"达到播放阈值，开始处理 scrobble",
		zap.String("artist", input.Artist),
		zap.String("album", input.Album),
		zap.String("track", input.Track),
		zap.String("player_source", string(input.PlayerSource)),
		zap.Int8("track_number", input.TrackNumber),
		zap.Int8("disc_number", input.DiscNumber),
	)
	req := &lastfm.PushTrackScrobbleReq{
		Artist:             input.Artist,
		AlbumArtist:        fallbackAlbumArtist(input.AlbumArtist, input.Artist),
		Track:              input.Track,
		Album:              input.Album,
		TrackNumber:        int64(input.TrackNumber),
		Timestamp:          input.PlaybackStartedAt.UTC().Unix(),
		MusicBrainzTrackID: input.MusicBrainzID,
		Duration:           input.Duration,
	}

	var releaseType string
	if input.AlbumTitleMetadata != nil {
		releaseType = input.AlbumTitleMetadata.ReleaseType
	}

	record := &model.TrackPlayRecord{
		Artist:        req.Artist,
		AlbumArtist:   req.AlbumArtist,
		Track:         req.Track,
		Album:         req.Album,
		AlbumSubtitle: input.AlbumSubtitle,
		Genre:         input.Metadata.Genre,
		Duration:      req.Duration,
		PlayTime:      time.Unix(req.Timestamp, 0),
		Scrobbled:     allowsLastfmScrobble(input.PlayerSource),
		MusicBrainzID: req.MusicBrainzTrackID,
		TrackNumber:   input.TrackNumber,
		DiscNumber:    input.DiscNumber,
		Source:        string(input.PlayerSource),
		ReleaseType:   releaseType,
		CoverArtPath:  model.BuildTrackPlayRecordArtworkPath(input.CoverArtURL, input.CoverArtObjectKey),
		TraceID:       input.TraceID,
		RootSpanID:    input.RootSpanID,
		TraceSampled:  input.TraceSampled,
	}

	if allowsLastfmScrobble(input.PlayerSource) {
		_, err := lastfmPushTrackScrobble(ctx, req)
		if err != nil {
			log.Warn(ctx, "HandleTrackPlaybackThreshold PushTrackScrobble err", zap.Error(err))
			record.Scrobbled = false
		}
	} else {
		log.Info(
			ctx,
			"当前来源跳过 Last.fm scrobble 上报",
			zap.String("artist", input.Artist),
			zap.String("album", input.Album),
			zap.String("track", input.Track),
			zap.String("player_source", string(input.PlayerSource)),
		)
	}

	if isDup, dupErr := modelIsDuplicateTrackPlayRecord(ctx, record, 60*time.Second); dupErr == nil && isDup {
		log.Warn(
			ctx,
			"HandleTrackPlaybackThreshold 阻断 60 秒内同源同曲目的重复听歌流水落库",
			zap.String("artist", record.Artist),
			zap.String("album", record.Album),
			zap.String("track", record.Track),
			zap.String("player_source", record.Source),
			zap.Time("play_time", record.PlayTime),
		)
		return PlaybackThresholdResult{Scrobbled: record.Scrobbled}
	}

	if insertErr := modelInsertTrackPlayRecord(ctx, record); insertErr != nil {
		log.Warn(ctx, "HandleTrackPlaybackThreshold insert play record err", zap.Error(insertErr))
	} else if processErr := modelProcessTrackPlayRecord(ctx, record.ID, input.Metadata); processErr != nil {
		log.Warn(ctx, "HandleTrackPlaybackThreshold process play record err", zap.Error(processErr))
	} else {
		playRecord, err := modelGetTrackPlayRecordByID(ctx, record.ID)
		if err != nil {
			log.Warn(ctx, "HandleTrackPlaybackThreshold GetTrackPlayRecord err", zap.Error(err))
		}
		// 实时增加流派播放次数，确保未归因数据也能计入统计；优先采用落库后已规整或归因的流派，并经 NormalizeGenre 兜底。
		var storedGenre string
		if playRecord != nil {
			storedGenre = playRecord.Genre
		}
		genreName := modelNormalizeGenre(nil, cmp.Or(strings.TrimSpace(storedGenre), strings.TrimSpace(input.Metadata.Genre)))
		if genreName != "" {
			if genreErr := modelIncrementGenrePlayCount(ctx, genreName); genreErr != nil {
				log.Warn(ctx, "HandleTrackPlaybackThreshold increment genre play count err", zap.Error(genreErr))
			}
		}

		telemetryGoOnlySafe(
			ctx, func(goCtx context.Context) {
				websocketBroadcastRecentPlaysUpdated(goCtx)
			},
		)

		stored, getErr := modelGetTrackPlayRecordByID(ctx, record.ID)
		if getErr != nil {
			log.Warn(ctx, "HandleTrackPlaybackThreshold query play record err", zap.Error(getErr))
		} else if stored != nil && stored.AlbumID > 0 {
			if input.CoverArtURL != "" || input.CoverArtObjectKey != "" {
				if coverErr := artworkEnsureAlbumCover(
					ctx,
					artworklogic.EnsureAlbumCoverInput{
						AlbumID:           stored.AlbumID,
						AlbumArtist:       input.AlbumArtist,
						Artist:            input.Artist,
						Album:             input.Album,
						CoverArtURL:       input.CoverArtURL,
						CoverArtMime:      input.CoverArtMime,
						CoverArtObjectKey: input.CoverArtObjectKey,
					},
				); coverErr != nil {
					log.Warn(ctx, "HandleTrackPlaybackThreshold update album cover err", zap.Error(coverErr))
				}
			}

			if metadataErr := modelUpdateAlbumTitleMetadataByID(
				ctx, stored.AlbumID, input.AlbumTitleMetadata,
			); metadataErr != nil {
				log.Warn(ctx, "HandleTrackPlaybackThreshold update album title metadata err", zap.Error(metadataErr))
			}
		}
	}

	if input.PlayerSource == common.PlayerAppleMusic && input.ControllerFavoriteKnown && input.ControllerFavorite {
		if _, favErr := s.SetTrackFavorite(
			ctx,
			input.Artist,
			input.Album,
			input.Track,
			model.TrackFavoriteEventSourceAppleMusic,
			true,
			input.Metadata,
		); favErr != nil {
			log.Warn(ctx, "HandleTrackPlaybackThreshold sync favorite err", zap.Error(favErr))
		}
	}

	log.Info(
		ctx,
		"播放阈值处理完成",
		zap.String("artist", input.Artist),
		zap.String("album", input.Album),
		zap.String("track", input.Track),
		zap.Bool("scrobbled", record.Scrobbled),
		zap.Int64("record_id", record.ID),
	)
	return PlaybackThresholdResult{Scrobbled: record.Scrobbled}
}

// IngestAppleMusicPlaybackBatch 统一接入移动端（iOS/iPadOS）离线采集的 Apple Music 历史播放流水。
func (s *TrackServiceImpl) IngestAppleMusicPlaybackBatch(
	ctx context.Context,
	records []ExternalPlaybackRecordInput,
) (*MobileScrobbleBatchResult, error) {
	result := &MobileScrobbleBatchResult{
		AcceptedKeys: make([]string, 0, len(records)),
		FailedKeys:   make([]string, 0),
		Results:      make([]MobileScrobbleItemResult, 0, len(records)),
	}
	if len(records) == 0 {
		return result, nil
	}

	sorted := slices.Clone(records)
	slices.SortFunc(
		sorted, func(a, b ExternalPlaybackRecordInput) int {
			return a.PlayedAt.Compare(b.PlayedAt)
		},
	)

	for _, raw := range sorted {
		key := strings.TrimSpace(raw.IdempotencyKey)
		if raw.PlayedAt.IsZero() {
			result.FailedCount++
			if key != "" {
				result.FailedKeys = append(result.FailedKeys, key)
			}
			result.Results = append(
				result.Results, MobileScrobbleItemResult{
					IdempotencyKey: key,
					Status:         "failed",
					Reason:         "played_at 不能为空",
				},
			)
			continue
		}

		eventInput := BuildAppleMusicPlaybackEventInput(raw)
		if err := common.ValidateTrackInfo(ctx, eventInput.Artist, eventInput.Album, eventInput.Track); err != nil {
			result.FailedCount++
			if key != "" {
				result.FailedKeys = append(result.FailedKeys, key)
			}
			result.Results = append(
				result.Results, MobileScrobbleItemResult{
					IdempotencyKey: key,
					Status:         "failed",
					Reason:         err.Error(),
				},
			)
			continue
		}

		probeRecord := &model.TrackPlayRecord{
			Artist:   eventInput.Artist,
			Album:    eventInput.Album,
			Track:    eventInput.Track,
			PlayTime: time.Unix(eventInput.PlaybackStartedAt.UTC().Unix(), 0),
			Source:   string(common.PlayerAppleMusic),
		}
		if isDup, dupErr := modelIsDuplicateTrackPlayRecord(ctx, probeRecord, mobileScrobbleDuplicateWindow); dupErr != nil {
			log.Warn(
				ctx,
				"IngestAppleMusicPlaybackBatch 检查重复听歌流水失败",
				zap.String("idempotency_key", key),
				zap.Error(dupErr),
			)
			result.FailedCount++
			if key != "" {
				result.FailedKeys = append(result.FailedKeys, key)
			}
			result.Results = append(
				result.Results, MobileScrobbleItemResult{
					IdempotencyKey: key,
					Status:         "failed",
					Reason:         dupErr.Error(),
				},
			)
			continue
		} else if isDup {
			result.DuplicateCount++
			if key != "" {
				result.AcceptedKeys = append(result.AcceptedKeys, key)
			}
			result.Results = append(
				result.Results, MobileScrobbleItemResult{
					IdempotencyKey: key,
					Status:         "duplicate",
				},
			)
			continue
		}

		thresholdRes := s.HandleTrackPlaybackThreshold(ctx, eventInput)
		result.AcceptedCount++
		if thresholdRes.Scrobbled {
			result.ScrobbledCount++
		}
		if key != "" {
			result.AcceptedKeys = append(result.AcceptedKeys, key)
		}
		result.Results = append(
			result.Results, MobileScrobbleItemResult{
				IdempotencyKey: key,
				Status:         "accepted",
				Scrobbled:      thresholdRes.Scrobbled,
			},
		)
	}

	log.Info(
		ctx,
		"移动端 Apple Music 播放流水批量接入完成",
		zap.Int("total", len(sorted)),
		zap.Int("accepted_count", result.AcceptedCount),
		zap.Int("duplicate_count", result.DuplicateCount),
		zap.Int("scrobbled_count", result.ScrobbledCount),
		zap.Int("failed_count", result.FailedCount),
	)
	return result, nil
}

func fallbackAlbumArtist(albumArtist, artist string) string {
	if strings.TrimSpace(albumArtist) != "" {
		return albumArtist
	}
	return artist
}

func (s *TrackServiceImpl) buildLikeTrackKey(input PlaybackEventInput) string {
	return fmt.Sprintf(
		"%s|%s|%s|%d|%d",
		strings.ToLower(strings.TrimSpace(input.Artist)),
		strings.ToLower(strings.TrimSpace(input.Album)),
		strings.ToLower(strings.TrimSpace(input.Track)),
		input.DiscNumber,
		input.TrackNumber,
	)
}

func (s *TrackServiceImpl) buildLikeLockKey(input PlaybackEventInput) string {
	return "lock:favorite-sync:" + s.buildLikeTrackKey(input)
}

func (s *TrackServiceImpl) shouldSyncLikeWrite(
	trackKey string,
	trackChanged bool,
	current favoriteProbeState,
) bool {
	s.favoriteProbeMu.Lock()
	defer s.favoriteProbeMu.Unlock()

	changed := trackChanged || s.lastLikeKey == "" || s.lastLikeKey != trackKey
	if s.lastLikeKey == trackKey {
		if current.appleKnown && (!s.lastLikeProbe.appleKnown || s.lastLikeProbe.appleLiked != current.appleLiked) {
			changed = true
		}
		if current.lastKnown && (!s.lastLikeProbe.lastKnown || s.lastLikeProbe.lastLiked != current.lastLiked) {
			changed = true
		}
	}

	next := favoriteProbeState{}
	if s.lastLikeKey == trackKey {
		next = s.lastLikeProbe
	}
	if current.appleKnown {
		next.appleKnown = true
		next.appleLiked = current.appleLiked
	}
	if current.lastKnown {
		next.lastKnown = true
		next.lastLiked = current.lastLiked
	}
	s.lastLikeKey = trackKey
	s.lastLikeProbe = next

	return changed
}

func (s *TrackServiceImpl) getCachedFavoriteProjection(
	trackKey string,
	version uint64,
	probe favoriteProbeState,
) (TrackFavoriteProjection, bool) {
	s.favoriteProjectionMu.Lock()
	defer s.favoriteProjectionMu.Unlock()

	cache := s.favoriteProjectionCache
	if !cache.valid || cache.key != trackKey || cache.version != version {
		return TrackFavoriteProjection{}, false
	}
	if cache.probe != probe {
		return TrackFavoriteProjection{}, false
	}
	return cache.projection, true
}

func (s *TrackServiceImpl) updateFavoriteProjectionCache(
	trackKey string,
	version uint64,
	probe favoriteProbeState,
	projection TrackFavoriteProjection,
) {
	s.favoriteProjectionMu.Lock()
	defer s.favoriteProjectionMu.Unlock()

	s.favoriteProjectionCache = cachedFavoriteProjection{
		key:        trackKey,
		probe:      probe,
		projection: projection,
		version:    version,
		valid:      true,
	}
}

func (s *TrackServiceImpl) probeLastfmFavorite(
	ctx context.Context,
	input PlaybackEventInput,
) (bool, bool) {
	cacheKey := s.buildLastfmFavoriteCacheKey(input.Artist, input.Track)
	cacheVersion := favoriteProjectionVersion.Load()
	now := timeNow()

	if favorited, ok := s.getCachedLastfmFavorite(cacheKey, cacheVersion, input.TrackChanged, now); ok {
		return favorited, true
	}

	favorited, err := lastfmIsFavorite(ctx, input.Artist, input.Track)
	if err != nil {
		log.Warn(ctx, "ProbeAndSyncTrackFavorite lastfm favorite err", zap.Error(err))
		if cached, ok := s.getLastfmFavoriteSnapshot(cacheKey, cacheVersion); ok {
			return cached, true
		}
		return false, false
	}

	s.updateLastfmFavoriteCache(cacheKey, cacheVersion, favorited, now)
	return favorited, true
}

func (s *TrackServiceImpl) getCachedLastfmFavorite(
	cacheKey string,
	version uint64,
	trackChanged bool,
	now time.Time,
) (bool, bool) {
	s.lastfmFavoriteMu.Lock()
	defer s.lastfmFavoriteMu.Unlock()

	if trackChanged {
		return false, false
	}
	if s.lastfmFavoriteCache == nil {
		return false, false
	}
	cached, ok := s.lastfmFavoriteCache[cacheKey]
	if !ok || cached.version != version || now.After(cached.nextProbeAt) {
		return false, false
	}
	return cached.favorited, true
}

func (s *TrackServiceImpl) getLastfmFavoriteSnapshot(cacheKey string, version uint64) (bool, bool) {
	s.lastfmFavoriteMu.Lock()
	defer s.lastfmFavoriteMu.Unlock()

	if s.lastfmFavoriteCache == nil {
		return false, false
	}
	cached, ok := s.lastfmFavoriteCache[cacheKey]
	if !ok || cached.version != version {
		return false, false
	}
	return cached.favorited, true
}

func (s *TrackServiceImpl) updateLastfmFavoriteCache(
	cacheKey string,
	version uint64,
	favorited bool,
	now time.Time,
) {
	s.lastfmFavoriteMu.Lock()
	defer s.lastfmFavoriteMu.Unlock()

	if s.lastfmFavoriteCache == nil {
		s.lastfmFavoriteCache = make(map[string]cachedLastfmFavorite)
	}
	cached := s.lastfmFavoriteCache[cacheKey]
	if cached.version != version {
		cached = cachedLastfmFavorite{}
	}
	if favorited {
		cached.negativeStreak = 0
		cached.nextProbeAt = now.Add(lastfmFavoritePositiveProbeInterval)
	} else {
		cached.negativeStreak++
		cached.nextProbeAt = now.Add(lastfmFavoriteNegativeProbeInterval(cached.negativeStreak))
	}
	cached.favorited = favorited
	cached.version = version
	s.lastfmFavoriteCache[cacheKey] = cached
}

func (s *TrackServiceImpl) buildLastfmFavoriteCacheKey(artist, track string) string {
	return strings.ToLower(strings.TrimSpace(artist)) + "|" + strings.ToLower(strings.TrimSpace(track))
}

func lastfmFavoriteNegativeProbeInterval(streak int) time.Duration {
	if streak <= 1 {
		return lastfmFavoritePositiveProbeInterval
	}

	interval := lastfmFavoritePositiveProbeInterval
	for i := 1; i < streak; i++ {
		interval *= 2
		if interval >= lastfmFavoriteNegativeProbeMax {
			return lastfmFavoriteNegativeProbeMax
		}
	}
	return interval
}

func (s *TrackServiceImpl) withLikeWriteLock(ctx context.Context, lockKey string, fn func() error) error {
	client := redisc.GetRedisClient()
	if client == nil {
		return fn()
	}

	token := fmt.Sprintf("%d", time.Now().UnixNano())
	ok, err := client.SetNX(ctx, lockKey, token, 10*time.Second).Result()
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	defer func() {
		_ = client.Eval(
			ctx,
			"if redis.call('get', KEYS[1]) == ARGV[1] then return redis.call('del', KEYS[1]) else return 0 end",
			[]string{lockKey},
			token,
		).Err()
	}()

	err = fn()
	if err != nil && errors.Is(err, goredis.Nil) {
		return nil
	}
	return err
}
