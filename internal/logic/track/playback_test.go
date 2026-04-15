package track

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/lastfm"
	artworklogic "github.com/vincentchyu/sonic-lens/internal/logic/artwork"
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
	originalGoSafe := telemetryGoOnlySafe
	originalBroadcastRecentPlaysUpdated := websocketBroadcastRecentPlaysUpdated
	originalGetTrackPlayRecordByID := modelGetTrackPlayRecordByID
	originalUpdateAlbumTitleMetadataByID := modelUpdateAlbumTitleMetadataByID
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
		telemetryGoOnlySafe = originalGoSafe
		websocketBroadcastRecentPlaysUpdated = originalBroadcastRecentPlaysUpdated
		modelGetTrackPlayRecordByID = originalGetTrackPlayRecordByID
		modelUpdateAlbumTitleMetadataByID = originalUpdateAlbumTitleMetadataByID
	})

	var capturedReq lastfm.PushTrackScrobbleReq
	lastfmPushTrackScrobble = func(ctx context.Context, req *lastfm.PushTrackScrobbleReq) (string, error) {
		capturedReq = *req
		return "ok", nil
	}

	insertCalled := false
	processCalled := false
	var insertedRecord model.TrackPlayRecord
	modelInsertTrackPlayRecord = func(ctx context.Context, record *model.TrackPlayRecord) error {
		insertCalled = true
		insertedRecord = *record
		record.ID = 42
		return nil
	}
	modelProcessTrackPlayRecord = func(ctx context.Context, recordID int64, metadata model.TrackMetadata) error {
		processCalled = true
		require.Equal(t, int64(42), recordID)
		require.Equal(t, int8(8), metadata.TrackNumber)
		return nil
	}
	recentPlaysUpdated := false
	telemetryGoOnlySafe = func(ctx context.Context, fn func(context.Context)) {
		fn(ctx)
	}
	websocketBroadcastRecentPlaysUpdated = func(ctx context.Context) {
		recentPlaysUpdated = true
	}
	modelGetTrackPlayRecordByID = func(ctx context.Context, id int64) (*model.TrackPlayRecord, error) {
		require.Equal(t, int64(42), id)
		return &model.TrackPlayRecord{ID: id}, nil
	}
	modelUpdateAlbumTitleMetadataByID = func(ctx context.Context, albumID int64, metadata *common.AlbumTitleMetadata) error {
		require.Fail(t, "should not update album title metadata when album_id is missing")
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
		TraceID:           "0123456789abcdef0123456789abcdef",
		RootSpanID:        "89abcdef01234567",
		TraceSampled:      true,
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
	require.Equal(t, "0123456789abcdef0123456789abcdef", insertedRecord.TraceID)
	require.Equal(t, "89abcdef01234567", insertedRecord.RootSpanID)
	require.True(t, insertedRecord.TraceSampled)
	require.True(t, recentPlaysUpdated)
}

func TestHandleTrackPlaybackThresholdUpdatesAlbumTitleMetadata(t *testing.T) {
	originalPush := lastfmPushTrackScrobble
	originalInsert := modelInsertTrackPlayRecord
	originalProcess := modelProcessTrackPlayRecord
	originalGetTrackPlayRecordByID := modelGetTrackPlayRecordByID
	originalUpdateAlbumTitleMetadataByID := modelUpdateAlbumTitleMetadataByID
	originalEnsureAlbumCover := artworkEnsureAlbumCover
	originalGoSafe := telemetryGoOnlySafe
	originalBroadcastRecentPlaysUpdated := websocketBroadcastRecentPlaysUpdated
	t.Cleanup(func() {
		lastfmPushTrackScrobble = originalPush
		modelInsertTrackPlayRecord = originalInsert
		modelProcessTrackPlayRecord = originalProcess
		modelGetTrackPlayRecordByID = originalGetTrackPlayRecordByID
		modelUpdateAlbumTitleMetadataByID = originalUpdateAlbumTitleMetadataByID
		artworkEnsureAlbumCover = originalEnsureAlbumCover
		telemetryGoOnlySafe = originalGoSafe
		websocketBroadcastRecentPlaysUpdated = originalBroadcastRecentPlaysUpdated
	})

	lastfmPushTrackScrobble = func(ctx context.Context, req *lastfm.PushTrackScrobbleReq) (string, error) {
		return "ok", nil
	}
	modelInsertTrackPlayRecord = func(ctx context.Context, record *model.TrackPlayRecord) error {
		record.ID = 99
		return nil
	}
	modelProcessTrackPlayRecord = func(ctx context.Context, recordID int64, metadata model.TrackMetadata) error {
		require.Equal(t, int64(99), recordID)
		return nil
	}
	modelGetTrackPlayRecordByID = func(ctx context.Context, id int64) (*model.TrackPlayRecord, error) {
		require.Equal(t, int64(99), id)
		return &model.TrackPlayRecord{ID: id, AlbumID: 123}, nil
	}
	artworkEnsureAlbumCover = func(ctx context.Context, input artworklogic.EnsureAlbumCoverInput) error {
		require.Fail(t, "should not update album cover when cover art is missing")
		return nil
	}
	telemetryGoOnlySafe = func(ctx context.Context, fn func(context.Context)) {
		fn(ctx)
	}
	websocketBroadcastRecentPlaysUpdated = func(ctx context.Context) {}

	metadataCalled := false
	modelUpdateAlbumTitleMetadataByID = func(ctx context.Context, albumID int64, metadata *common.AlbumTitleMetadata) error {
		metadataCalled = true
		require.Equal(t, int64(123), albumID)
		require.NotNil(t, metadata)
		require.Equal(t, "Album", metadata.SourceDisplayTitle)
		require.Equal(t, "Album", metadata.OfficialTitle)
		require.Len(t, metadata.TitleVersions, 1)
		require.Equal(t, "Deluxe Edition", metadata.TitleVersions[0].Text)
		return nil
	}

	service := &TrackServiceImpl{}
	service.HandleTrackPlaybackThreshold(context.Background(), PlaybackEventInput{
		Artist:            "Artist",
		AlbumArtist:       "Album Artist",
		Album:             "Album",
		Track:             "Track",
		Duration:          245,
		PlayerSource:      common.PlayerAppleMusic,
		PlaybackStartedAt: time.Unix(1700000000, 0),
		AlbumTitleMetadata: &common.AlbumTitleMetadata{
			SourceDisplayTitle: "Album",
			OfficialTitle:      "Album",
			TitleVersions: []common.AlbumTitleVersion{
				{
					Text: "Deluxe Edition",
					Type: common.AlbumTitleVersionTypeEdition,
				},
			},
			NormalizedDisplayTitle: "Album (Deluxe Edition)",
		},
		Metadata: model.TrackMetadata{},
	})

	require.True(t, metadataCalled)
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

func TestProbeAndSyncTrackFavoriteReusesProjectionCacheWhenProbeUnchanged(t *testing.T) {
	originalGetTrack := modelGetTrackByIdentity
	originalIsFavorite := lastfmIsFavorite
	originalGetPending := modelGetPendingTrackFavoriteSnapshot
	originalVersion := favoriteProjectionVersion.Load()
	favoriteProjectionVersion.Store(0)
	t.Cleanup(func() {
		modelGetTrackByIdentity = originalGetTrack
		lastfmIsFavorite = originalIsFavorite
		modelGetPendingTrackFavoriteSnapshot = originalGetPending
		favoriteProjectionVersion.Store(originalVersion)
	})

	trackLookups := 0
	pendingLookups := 0
	modelGetTrackByIdentity = func(ctx context.Context, artist, album, track string, trackNumber, discNumber int8) (*model.Track, error) {
		trackLookups++
		return &model.Track{
			Artist:      artist,
			Album:       album,
			Track:       track,
			TrackNumber: trackNumber,
			DiscNumber:  discNumber,
		}, nil
	}
	lastfmIsFavorite = func(ctx context.Context, artist, track string) (bool, error) {
		return false, nil
	}
	modelGetPendingTrackFavoriteSnapshot = func(ctx context.Context, identity model.TrackIdentity) (*model.TrackFavoritePendingSnapshot, error) {
		pendingLookups++
		return &model.TrackFavoritePendingSnapshot{}, nil
	}

	service := &TrackServiceImpl{}
	input := PlaybackEventInput{
		Artist:                  "Artist",
		Album:                   "Album",
		Track:                   "Track",
		TrackNumber:             1,
		DiscNumber:              1,
		PlayerSource:            common.PlayerAppleMusic,
		TrackChanged:            true,
		ControllerFavoriteKnown: true,
		ControllerFavorite:      false,
		Metadata: model.TrackMetadata{
			Confidence:  common.TrackMetadataConfidenceHigh,
			TrackNumber: 1,
			DiscNumber:  1,
		},
	}

	first := service.ProbeAndSyncTrackFavorite(context.Background(), input)
	input.TrackChanged = false
	second := service.ProbeAndSyncTrackFavorite(context.Background(), input)

	require.Equal(t, first.TrackFavoriteProjection, second.TrackFavoriteProjection)
	require.Equal(t, 2, trackLookups)
	require.Equal(t, 1, pendingLookups)
}

func TestProbeAndSyncTrackFavoriteInvalidatesProjectionCacheOnVersionChange(t *testing.T) {
	originalGetTrack := modelGetTrackByIdentity
	originalIsFavorite := lastfmIsFavorite
	originalGetPending := modelGetPendingTrackFavoriteSnapshot
	originalVersion := favoriteProjectionVersion.Load()
	favoriteProjectionVersion.Store(0)
	t.Cleanup(func() {
		modelGetTrackByIdentity = originalGetTrack
		lastfmIsFavorite = originalIsFavorite
		modelGetPendingTrackFavoriteSnapshot = originalGetPending
		favoriteProjectionVersion.Store(originalVersion)
	})

	trackLookups := 0
	pendingLookups := 0
	modelGetTrackByIdentity = func(ctx context.Context, artist, album, track string, trackNumber, discNumber int8) (*model.Track, error) {
		trackLookups++
		return &model.Track{
			Artist:      artist,
			Album:       album,
			Track:       track,
			TrackNumber: trackNumber,
			DiscNumber:  discNumber,
		}, nil
	}
	lastfmIsFavorite = func(ctx context.Context, artist, track string) (bool, error) {
		return false, nil
	}
	modelGetPendingTrackFavoriteSnapshot = func(ctx context.Context, identity model.TrackIdentity) (*model.TrackFavoritePendingSnapshot, error) {
		pendingLookups++
		return &model.TrackFavoritePendingSnapshot{}, nil
	}

	service := &TrackServiceImpl{}
	input := PlaybackEventInput{
		Artist:                  "Artist",
		Album:                   "Album",
		Track:                   "Track",
		TrackNumber:             1,
		DiscNumber:              1,
		PlayerSource:            common.PlayerAppleMusic,
		TrackChanged:            true,
		ControllerFavoriteKnown: true,
		ControllerFavorite:      false,
		Metadata: model.TrackMetadata{
			Confidence:  common.TrackMetadataConfidenceHigh,
			TrackNumber: 1,
			DiscNumber:  1,
		},
	}

	service.ProbeAndSyncTrackFavorite(context.Background(), input)
	input.TrackChanged = false
	favoriteProjectionVersion.Add(1)
	service.ProbeAndSyncTrackFavorite(context.Background(), input)

	require.Equal(t, 3, trackLookups)
	require.Equal(t, 2, pendingLookups)
}

func TestProbeAndSyncTrackFavoriteUsesLastfmHotCacheAndBackoff(t *testing.T) {
	originalGetTrack := modelGetTrackByIdentity
	originalIsFavorite := lastfmIsFavorite
	originalGetPending := modelGetPendingTrackFavoriteSnapshot
	originalNow := timeNow
	originalVersion := favoriteProjectionVersion.Load()
	favoriteProjectionVersion.Store(0)
	now := time.Date(2026, 3, 29, 16, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return now }
	t.Cleanup(func() {
		modelGetTrackByIdentity = originalGetTrack
		lastfmIsFavorite = originalIsFavorite
		modelGetPendingTrackFavoriteSnapshot = originalGetPending
		timeNow = originalNow
		favoriteProjectionVersion.Store(originalVersion)
	})

	modelGetTrackByIdentity = func(ctx context.Context, artist, album, track string, trackNumber, discNumber int8) (*model.Track, error) {
		return &model.Track{
			Artist:      artist,
			Album:       album,
			Track:       track,
			TrackNumber: trackNumber,
			DiscNumber:  discNumber,
		}, nil
	}
	modelGetPendingTrackFavoriteSnapshot = func(ctx context.Context, identity model.TrackIdentity) (*model.TrackFavoritePendingSnapshot, error) {
		return &model.TrackFavoritePendingSnapshot{}, nil
	}

	lastfmCalls := 0
	lastfmIsFavorite = func(ctx context.Context, artist, track string) (bool, error) {
		lastfmCalls++
		return false, nil
	}

	service := NewTrackService().(*TrackServiceImpl)
	input := PlaybackEventInput{
		Artist:                  "Artist",
		Album:                   "Album",
		Track:                   "Track",
		TrackNumber:             1,
		DiscNumber:              1,
		PlayerSource:            common.PlayerAppleMusic,
		TrackChanged:            true,
		ControllerFavoriteKnown: true,
		ControllerFavorite:      false,
		Metadata: model.TrackMetadata{
			Confidence:  common.TrackMetadataConfidenceHigh,
			TrackNumber: 1,
			DiscNumber:  1,
		},
	}

	service.ProbeAndSyncTrackFavorite(context.Background(), input)
	input.TrackChanged = false
	service.ProbeAndSyncTrackFavorite(context.Background(), input)
	require.Equal(t, 1, lastfmCalls)

	now = now.Add(16 * time.Second)
	service.ProbeAndSyncTrackFavorite(context.Background(), input)
	require.Equal(t, 2, lastfmCalls)

	now = now.Add(20 * time.Second)
	service.ProbeAndSyncTrackFavorite(context.Background(), input)
	require.Equal(t, 2, lastfmCalls)

	now = now.Add(11 * time.Second)
	service.ProbeAndSyncTrackFavorite(context.Background(), input)
	require.Equal(t, 3, lastfmCalls)
}

func TestProbeAndSyncTrackFavoriteInvalidatesLastfmHotCacheOnVersionChange(t *testing.T) {
	originalGetTrack := modelGetTrackByIdentity
	originalIsFavorite := lastfmIsFavorite
	originalGetPending := modelGetPendingTrackFavoriteSnapshot
	originalNow := timeNow
	originalVersion := favoriteProjectionVersion.Load()
	favoriteProjectionVersion.Store(0)
	now := time.Date(2026, 3, 29, 16, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return now }
	t.Cleanup(func() {
		modelGetTrackByIdentity = originalGetTrack
		lastfmIsFavorite = originalIsFavorite
		modelGetPendingTrackFavoriteSnapshot = originalGetPending
		timeNow = originalNow
		favoriteProjectionVersion.Store(originalVersion)
	})

	modelGetTrackByIdentity = func(ctx context.Context, artist, album, track string, trackNumber, discNumber int8) (*model.Track, error) {
		return &model.Track{
			Artist:      artist,
			Album:       album,
			Track:       track,
			TrackNumber: trackNumber,
			DiscNumber:  discNumber,
		}, nil
	}
	modelGetPendingTrackFavoriteSnapshot = func(ctx context.Context, identity model.TrackIdentity) (*model.TrackFavoritePendingSnapshot, error) {
		return &model.TrackFavoritePendingSnapshot{}, nil
	}

	lastfmCalls := 0
	lastfmIsFavorite = func(ctx context.Context, artist, track string) (bool, error) {
		lastfmCalls++
		return false, nil
	}

	service := NewTrackService().(*TrackServiceImpl)
	input := PlaybackEventInput{
		Artist:                  "Artist",
		Album:                   "Album",
		Track:                   "Track",
		TrackNumber:             1,
		DiscNumber:              1,
		PlayerSource:            common.PlayerAppleMusic,
		TrackChanged:            true,
		ControllerFavoriteKnown: true,
		ControllerFavorite:      false,
		Metadata: model.TrackMetadata{
			Confidence:  common.TrackMetadataConfidenceHigh,
			TrackNumber: 1,
			DiscNumber:  1,
		},
	}

	service.ProbeAndSyncTrackFavorite(context.Background(), input)
	input.TrackChanged = false
	service.ProbeAndSyncTrackFavorite(context.Background(), input)
	require.Equal(t, 1, lastfmCalls)

	favoriteProjectionVersion.Add(1)
	service.ProbeAndSyncTrackFavorite(context.Background(), input)
	require.Equal(t, 2, lastfmCalls)
}
