package scrobbler

import (
	"context"
	"time"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/internal/logic/track"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

// playingTrackSnapshot 表示 checker 一次采集后得到的标准化播放快照。
type playingTrackSnapshot struct {
	playerInfo               common.PlayerInfoHandler
	metadata                 model.TrackMetadata
	trackKey                 string
	position                 float64
	duration                 int64
	trackChanged             bool
	reachedScrobbleThreshold bool
	coverArtURL              string
	coverArtMime             string
	controllerFavoriteKnown  bool
	controllerFavorite       bool
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
	coverArtURL, coverArtMime := b.resolveArtwork(ctx, playerInfo)
	metadata := b.buildTrackMetadata(playerInfo)

	snapshot := playingTrackSnapshot{
		playerInfo:              playerInfo,
		metadata:                metadata,
		trackKey:                trackKey,
		position:                position,
		duration:                duration,
		trackChanged:            trackChanged,
		coverArtURL:             coverArtURL,
		coverArtMime:            coverArtMime,
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

func (b *BasePlayerChecker) buildCurrentTrackKey(playerInfo common.PlayerInfoHandler) string {
	trackKey := playerInfo.GetTitle()
	if b.source == common.PlayerAudirvana {
		if url := playerInfo.GetUrl(); url != "" {
			trackKey = url + playerInfo.GetTitle()
		}
	}
	return trackKey
}

func (s playingTrackSnapshot) toPlaybackEventInput(source common.PlayerType, startedAt time.Time) track.PlaybackEventInput {
	return track.PlaybackEventInput{
		Artist:                  s.playerInfo.GetArtist(),
		AlbumArtist:             s.playerInfo.GetAlbumArtist(),
		Album:                   s.playerInfo.GetAlbum(),
		Track:                   s.playerInfo.GetTitle(),
		TrackNumber:             int8(s.playerInfo.GetTrackNumber()),
		DiscNumber:              s.playerInfo.GetDiscNumber(),
		Duration:                s.duration,
		MusicBrainzID:           s.playerInfo.GetMusicBrainzID(),
		Metadata:                s.metadata,
		PlayerSource:            source,
		TrackChanged:            s.trackChanged,
		PlaybackStartedAt:       startedAt,
		ControllerFavoriteKnown: s.controllerFavoriteKnown,
		ControllerFavorite:      s.controllerFavorite,
	}
}
