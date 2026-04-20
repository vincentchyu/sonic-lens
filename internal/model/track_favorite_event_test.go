package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetPendingTrackFavoriteSnapshotReturnsLatestBySource(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_favorite_pending_snapshot")
	prevSQLite := GlobalDBForSqlLite
	prevMySQL := GlobalDBForMysql
	GlobalDBForSqlLite = db
	GlobalDBForMysql = nil
	t.Cleanup(func() {
		GlobalDBForSqlLite = prevSQLite
		GlobalDBForMysql = prevMySQL
	})

	ctx := context.Background()
	events := []*TrackFavoriteEvent{
		{
			Source:           TrackFavoriteEventSourceAppleMusic,
			ProviderFavorite: true,
			Artist:           "Artist",
			Album:            "Album",
			AlbumSubtitle:    "Standard",
			Track:            "Track",
			TrackNumber:      0,
			DiscNumber:       0,
			ResolutionStatus: TrackFavoriteEventResolutionPending,
			Applied:          false,
		},
		{
			Source:           TrackFavoriteEventSourceLastFm,
			ProviderFavorite: true,
			Artist:           "Artist",
			Album:            "Album",
			AlbumSubtitle:    "Deluxe",
			Track:            "Track",
			TrackNumber:      3,
			DiscNumber:       1,
			ResolutionStatus: TrackFavoriteEventResolutionUnresolved,
			Applied:          false,
		},
		{
			Source:           TrackFavoriteEventSourceAppleMusic,
			ProviderFavorite: false,
			Artist:           "Artist",
			Album:            "Album",
			AlbumSubtitle:    "Deluxe",
			Track:            "Track",
			TrackNumber:      3,
			DiscNumber:       1,
			ResolutionStatus: TrackFavoriteEventResolutionAmbiguous,
			Applied:          false,
		},
		{
			Source:           TrackFavoriteEventSourceAppleMusic,
			ProviderFavorite: true,
			Artist:           "Artist",
			Album:            "Album",
			AlbumSubtitle:    "Deluxe",
			Track:            "Track",
			TrackNumber:      3,
			DiscNumber:       1,
			ResolutionStatus: TrackFavoriteEventResolutionResolved,
			Applied:          true,
		},
	}
	for _, event := range events {
		require.NoError(t, CreateTrackFavoriteEvent(ctx, event))
	}

	snapshot, err := GetPendingTrackFavoriteSnapshot(ctx, TrackIdentity{
		Artist:        "Artist",
		Album:         "Album",
		AlbumSubtitle: "Deluxe",
		Track:         "Track",
		TrackNumber:   3,
		DiscNumber:    1,
	})

	require.NoError(t, err)
	require.True(t, snapshot.AppleMusicKnown)
	require.False(t, snapshot.AppleMusicFavorite)
	require.True(t, snapshot.LastFmKnown)
	require.True(t, snapshot.LastFmFavorite)
}
