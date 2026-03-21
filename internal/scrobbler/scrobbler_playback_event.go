package scrobbler

import (
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
	coverArtObjectKey        string
	controllerFavoriteKnown  bool
	controllerFavorite       bool
}

func (s playingTrackSnapshot) toPlaybackEventInput(
	source common.PlayerType, startedAt time.Time,
) track.PlaybackEventInput {
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
		CoverArtURL:             s.coverArtURL,
		CoverArtMime:            s.coverArtMime,
		CoverArtObjectKey:       s.coverArtObjectKey,
	}
}
