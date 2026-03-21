package track

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/lastfm"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

func TestHandleNowPlayingStarted(t *testing.T) {
	originalTrackUpdate := lastfmTrackUpdateNowPlaying
	t.Cleanup(func() {
		lastfmTrackUpdateNowPlaying = originalTrackUpdate
	})

	var captured lastfm.TrackUpdateNowPlayingReq
	lastfmTrackUpdateNowPlaying = func(ctx context.Context, req *lastfm.TrackUpdateNowPlayingReq) error {
		captured = *req
		return nil
	}

	service := &TrackServiceImpl{}
	service.HandleNowPlayingStarted(context.Background(), PlaybackEventInput{
		Artist:      "Artist",
		AlbumArtist: "Album Artist",
		Album:       "Album",
		Track:       "Track",
		Duration:    321,
	})

	require.Equal(t, "Artist", captured.Artist)
	require.Equal(t, "Album Artist", captured.AlbumArtist)
	require.Equal(t, "Album", captured.Album)
	require.Equal(t, "Track", captured.Track)
	require.Equal(t, int64(321), captured.Duration)
}

func TestHandleTrackPlaybackThresholdProcessesRecord(t *testing.T) {
	originalPush := lastfmPushTrackScrobble
	originalInsert := modelInsertTrackPlayRecord
	originalProcess := modelProcessTrackPlayRecord
	originalAppleSet := appleMusicSetFavorite
	originalModelAppleSet := modelSetAppleMusicFavorite
	originalLastfmSet := lastfmSetFavorite
	originalModelLastSet := modelSetLastFmFavorite
	originalGetApple := modelGetAppleMusicFavoriteByIdentity
	originalGetLast := modelGetLastFmFavoriteByIdentity
	originalGetPending := modelGetPendingTrackFavoriteSnapshot
	t.Cleanup(func() {
		lastfmPushTrackScrobble = originalPush
		modelInsertTrackPlayRecord = originalInsert
		modelProcessTrackPlayRecord = originalProcess
		appleMusicSetFavorite = originalAppleSet
		modelSetAppleMusicFavorite = originalModelAppleSet
		lastfmSetFavorite = originalLastfmSet
		modelSetLastFmFavorite = originalModelLastSet
		modelGetAppleMusicFavoriteByIdentity = originalGetApple
		modelGetLastFmFavoriteByIdentity = originalGetLast
		modelGetPendingTrackFavoriteSnapshot = originalGetPending
	})

	var capturedReq lastfm.PushTrackScrobbleReq
	lastfmPushTrackScrobble = func(ctx context.Context, req *lastfm.PushTrackScrobbleReq) (string, error) {
		capturedReq = *req
		return "ok", nil
	}

	insertCalled := false
	processCalled := false
	modelInsertTrackPlayRecord = func(ctx context.Context, record *model.TrackPlayRecord) error {
		insertCalled = true
		record.ID = 42
		return nil
	}
	modelProcessTrackPlayRecord = func(ctx context.Context, recordID int64, metadata model.TrackMetadata) error {
		processCalled = true
		require.Equal(t, int64(42), recordID)
		require.Equal(t, int8(8), metadata.TrackNumber)
		return nil
	}

	service := &TrackServiceImpl{}
	result := service.HandleTrackPlaybackThreshold(context.Background(), PlaybackEventInput{
		Artist:            "Artist",
		AlbumArtist:       "Album Artist",
		Album:             "Album",
		Track:             "Track",
		TrackNumber:       8,
		DiscNumber:        2,
		Duration:          245,
		MusicBrainzID:     "mbid",
		PlayerSource:      common.PlayerAppleMusic,
		PlaybackStartedAt: time.Unix(1700000000, 0),
		Metadata: model.TrackMetadata{
			TrackNumber: 8,
			DiscNumber:  2,
		},
	})

	require.True(t, result.Scrobbled)
	require.True(t, insertCalled)
	require.True(t, processCalled)
	require.Equal(t, "Artist", capturedReq.Artist)
	require.Equal(t, "Album Artist", capturedReq.AlbumArtist)
	require.Equal(t, "Track", capturedReq.Track)
	require.Equal(t, int64(1700000000), capturedReq.Timestamp)
}

func TestProbeAndSyncTrackFavoriteAppliesAppleMusicFavorite(t *testing.T) {
	originalGetTrack := modelGetTrackByIdentity
	originalIsFavorite := lastfmIsFavorite
	originalAppleSet := appleMusicSetFavorite
	originalModelAppleSet := modelSetAppleMusicFavorite
	originalLastfmSet := lastfmSetFavorite
	originalModelLastSet := modelSetLastFmFavorite
	originalGetApple := modelGetAppleMusicFavoriteByIdentity
	originalGetLast := modelGetLastFmFavoriteByIdentity
	originalGetPending := modelGetPendingTrackFavoriteSnapshot
	t.Cleanup(func() {
		modelGetTrackByIdentity = originalGetTrack
		lastfmIsFavorite = originalIsFavorite
		appleMusicSetFavorite = originalAppleSet
		modelSetAppleMusicFavorite = originalModelAppleSet
		lastfmSetFavorite = originalLastfmSet
		modelSetLastFmFavorite = originalModelLastSet
		modelGetAppleMusicFavoriteByIdentity = originalGetApple
		modelGetLastFmFavoriteByIdentity = originalGetLast
		modelGetPendingTrackFavoriteSnapshot = originalGetPending
	})

	appleApplied := false
	lastApplied := false
	modelGetTrackByIdentity = func(ctx context.Context, artist, album, track string, trackNumber, discNumber int8) (*model.Track, error) {
		return &model.Track{
			Artist:          artist,
			Album:           album,
			Track:           track,
			TrackNumber:     trackNumber,
			DiscNumber:      discNumber,
			IsAppleMusicFav: appleApplied,
			IsLastFmFav:     lastApplied,
		}, nil
	}
	lastfmIsFavorite = func(ctx context.Context, artist, track string) (bool, error) {
		return false, nil
	}
	appleCalls := 0
	lastfmCalls := 0
	appleMusicSetFavorite = func(ctx context.Context, favorited bool) error {
		appleCalls++
		require.True(t, favorited)
		return nil
	}
	modelSetAppleMusicFavorite = func(params model.SetFavoriteParams) error {
		appleApplied = params.IsFavorite
		return nil
	}
	lastfmSetFavorite = func(ctx context.Context, artist, track string, favorited bool) error {
		lastfmCalls++
		require.True(t, favorited)
		return nil
	}
	modelSetLastFmFavorite = func(params model.SetFavoriteParams) error {
		lastApplied = params.IsFavorite
		return nil
	}
	modelGetAppleMusicFavoriteByIdentity = func(ctx context.Context, artist, album, track string, trackNumber, discNumber int8) (bool, error) {
		return true, nil
	}
	modelGetLastFmFavoriteByIdentity = func(ctx context.Context, artist, album, track string, trackNumber, discNumber int8) (bool, error) {
		return true, nil
	}
	modelGetPendingTrackFavoriteSnapshot = func(ctx context.Context, identity model.TrackIdentity) (*model.TrackFavoritePendingSnapshot, error) {
		return &model.TrackFavoritePendingSnapshot{}, nil
	}

	service := &TrackServiceImpl{}
	result := service.ProbeAndSyncTrackFavorite(context.Background(), PlaybackEventInput{
		Artist:                  "Artist",
		Album:                   "Album",
		Track:                   "Track",
		TrackNumber:             1,
		DiscNumber:              1,
		PlayerSource:            common.PlayerAppleMusic,
		TrackChanged:            true,
		ControllerFavoriteKnown: true,
		ControllerFavorite:      true,
		Metadata: model.TrackMetadata{
			Confidence:  common.TrackMetadataConfidenceHigh,
			TrackNumber: 1,
			DiscNumber:  1,
		},
	})

	require.True(t, result.AppleMusic)
	require.True(t, result.LastFM)
	require.Equal(t, common.TrackFavoriteStateFavorited, result.AppleMusicState)
	require.Equal(t, common.TrackFavoriteStateFavorited, result.LastFMState)
	require.Equal(t, 1, appleCalls)
	require.Equal(t, 1, lastfmCalls)
}

func TestProbeAndSyncTrackFavoriteSkipsUnsafeLookup(t *testing.T) {
	originalGetTrack := modelGetTrackByIdentity
	originalIsFavorite := lastfmIsFavorite
	originalGetPending := modelGetPendingTrackFavoriteSnapshot
	t.Cleanup(func() {
		modelGetTrackByIdentity = originalGetTrack
		lastfmIsFavorite = originalIsFavorite
		modelGetPendingTrackFavoriteSnapshot = originalGetPending
	})

	modelGetTrackByIdentity = func(ctx context.Context, artist, album, track string, trackNumber, discNumber int8) (*model.Track, error) {
		return nil, errors.New("should not be called")
	}
	lastfmIsFavorite = func(ctx context.Context, artist, track string) (bool, error) {
		return false, errors.New("should not be called")
	}
	modelGetPendingTrackFavoriteSnapshot = func(ctx context.Context, identity model.TrackIdentity) (*model.TrackFavoritePendingSnapshot, error) {
		return &model.TrackFavoritePendingSnapshot{}, nil
	}

	service := &TrackServiceImpl{}
	result := service.ProbeAndSyncTrackFavorite(context.Background(), PlaybackEventInput{
		Artist:   "Artist",
		Album:    "Album",
		Track:    "Track",
		Metadata: model.TrackMetadata{Confidence: common.TrackMetadataConfidenceLow},
	})

	require.False(t, result.AppleMusic)
	require.False(t, result.LastFM)
	require.Equal(t, common.TrackFavoriteStateNotFavorited, result.FavoriteState)
	require.Equal(t, common.TrackMetadataConfidenceLow, result.Confidence)
}
