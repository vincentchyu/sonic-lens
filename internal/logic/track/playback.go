package track

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/lastfm"
	"github.com/vincentchyu/sonic-lens/core/log"
	redisc "github.com/vincentchyu/sonic-lens/core/redis"
	artworklogic "github.com/vincentchyu/sonic-lens/internal/logic/artwork"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

var (
	lastfmPushTrackScrobble     = lastfm.PushTrackScrobble
	lastfmTrackUpdateNowPlaying = lastfm.TrackUpdateNowPlaying
	lastfmIsFavorite            = lastfm.IsFavorite
	artworkEnsureAlbumCover     = artworklogic.EnsureAlbumCover
)

// PlaybackEventInput 表示播放事件编排所需的标准化输入。
type PlaybackEventInput struct {
	Artist                  string
	AlbumArtist             string
	Album                   string
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
	projection, err := s.buildFavoriteProjection(
		ctx,
		FavoriteProjectionInput{
			Artist:      input.Artist,
			Album:       input.Album,
			Track:       input.Track,
			TrackNumber: input.TrackNumber,
			DiscNumber:  input.DiscNumber,
			Metadata:    input.Metadata,
		},
	)
	if err != nil {
		log.Warn(ctx, "构建收藏态投影视图失败", zap.Error(err))
	} else {
		result.TrackFavoriteProjection = projection
	}

	if !s.canSafelyLookupCurrentTrack(input.Metadata) {
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

	trackKey := s.buildLikeTrackKey(input)
	probe := favoriteProbeState{}

	if input.ControllerFavoriteKnown {
		probe.appleKnown = true
		probe.appleLiked = input.ControllerFavorite
	}

	lastFmFavorite, lastFmErr := lastfmIsFavorite(ctx, input.Artist, input.Track)
	if lastFmErr != nil {
		log.Warn(ctx, "ProbeAndSyncTrackFavorite lastfm favorite err", zap.Error(lastFmErr))
	} else {
		probe.lastKnown = true
		probe.lastLiked = lastFmFavorite
	}

	if !s.shouldSyncLikeWrite(trackKey, input.TrackChanged, probe) {
		return result
	}

	lockKey := s.buildLikeLockKey(input)
	if err := s.withLikeWriteLock(ctx, lockKey, func() error {
		currentTrack, _ := modelGetTrackByIdentity(
			ctx, input.Artist, input.Album, input.Track, input.TrackNumber, input.DiscNumber,
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
	}); err != nil {
		log.Warn(ctx, "ProbeAndSyncTrackFavorite sync favorite err", zap.Error(err))
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

	record := &model.TrackPlayRecord{
		Artist:        req.Artist,
		AlbumArtist:   req.AlbumArtist,
		Track:         req.Track,
		Album:         req.Album,
		Duration:      req.Duration,
		PlayTime:      time.Unix(req.Timestamp, 0),
		Scrobbled:     true,
		MusicBrainzID: req.MusicBrainzTrackID,
		TrackNumber:   input.TrackNumber,
		DiscNumber:    input.DiscNumber,
		Source:        string(input.PlayerSource),
	}

	_, err := lastfmPushTrackScrobble(ctx, req)
	if err != nil {
		log.Warn(ctx, "HandleTrackPlaybackThreshold PushTrackScrobble err", zap.Error(err))
		record.Scrobbled = false
	}

	if insertErr := modelInsertTrackPlayRecord(ctx, record); insertErr != nil {
		log.Warn(ctx, "HandleTrackPlaybackThreshold insert play record err", zap.Error(insertErr))
	} else if processErr := modelProcessTrackPlayRecord(ctx, record.ID, input.Metadata); processErr != nil {
		log.Warn(ctx, "HandleTrackPlaybackThreshold process play record err", zap.Error(processErr))
	} else if input.CoverArtURL != "" || input.CoverArtObjectKey != "" {
		stored, getErr := modelGetTrackPlayRecordByID(ctx, record.ID)
		if getErr != nil {
			log.Warn(ctx, "HandleTrackPlaybackThreshold query play record err", zap.Error(getErr))
		} else if stored != nil && stored.AlbumID > 0 {
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
