package model

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetPendingAlbumGroupsAndCreateWorkItem(t *testing.T) {
	db := newTrackResolutionTestDB(t, "pending_album_groups")
	ctx := context.Background()

	now := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)
	require.NoError(
		t,
		db.Create(&TrackPlayRecord{
			ID:          10,
			Artist:      "Yorushika",
			AlbumArtist: "Yorushika",
			Album:       "second person",
			Track:       "Become a cloud before",
			Source:      "Apple Music",
			PlayTime:    now,
		}).Error,
	)
	require.NoError(
		t,
		db.Create(&TrackPlayRecord{
			ID:          11,
			Artist:      "Yorushika",
			AlbumArtist: "Yorushika",
			Album:       "second person",
			Track:       "Forget it",
			Source:      "Apple Music",
			PlayTime:    now.Add(-time.Hour),
		}).Error,
	)
	require.NoError(
		t,
		db.Create(&TrackFavoriteEvent{
			ID:               20,
			Source:           TrackFavoriteEventSourceAppleMusic,
			Artist:           "Yorushika",
			AlbumArtist:      "Yorushika",
			Album:            "second person",
			Track:            "Forget it",
			ProviderFavorite: true,
		}).Error,
	)

	groups, err := GetPendingAlbumGroups(ctx, 10)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, "second person", groups[0].Album)
	require.Equal(t, 2, groups[0].PlayRecordCount)
	require.Equal(t, 1, groups[0].FavoriteEventCount)
	require.ElementsMatch(t, []int64{10, 11}, groups[0].PlayRecordIDs)
	require.ElementsMatch(t, []int64{20}, groups[0].FavoriteEventIDs)

	item, err := CreateOrGetPendingAlbumWorkItem(ctx, groups[0].IdentityKey)
	require.NoError(t, err)
	require.Equal(t, PendingAlbumWorkItemStatusOpen, item.Status)

	detail, err := GetPendingAlbumWorkItemDetail(ctx, item.ID)
	require.NoError(t, err)
	require.Len(t, detail.PlayRecords, 2)
	require.Len(t, detail.FavoriteEvents, 1)
	require.Len(t, detail.ContextTracks, 2)
}

func TestApplyTrackFavoriteEventsByIDs(t *testing.T) {
	db := newTrackResolutionTestDB(t, "pending_album_apply_favorite")
	ctx := context.Background()

	require.NoError(
		t,
		db.Create(&Track{
			ID:          1,
			Artist:      "Yorushika",
			AlbumArtist: "Yorushika",
			Album:       "second person",
			Track:       "Forget it",
			TrackNumber: 8,
			DiscNumber:  1,
			Version:     1,
		}).Error,
	)
	require.NoError(
		t,
		db.Create(&TrackFavoriteEvent{
			ID:               99,
			Source:           TrackFavoriteEventSourceAppleMusic,
			Artist:           "Yorushika",
			AlbumArtist:      "Yorushika",
			Album:            "second person",
			Track:            "Forget it",
			TrackNumber:      8,
			DiscNumber:       1,
			ProviderFavorite: true,
			ResolutionStatus: TrackFavoriteEventResolutionUnresolved,
		}).Error,
	)

	report, err := ApplyTrackFavoriteEventsByIDs(ctx, []int64{99})
	require.NoError(t, err)
	require.Equal(t, 1, report.AppliedCount)

	var event TrackFavoriteEvent
	require.NoError(t, db.First(&event, 99).Error)
	require.True(t, event.Applied)
	require.Equal(t, int64(1), event.ResolvedTrackID)
	require.Equal(t, TrackFavoriteEventResolutionResolved, event.ResolutionStatus)

	var track Track
	require.NoError(t, db.First(&track, 1).Error)
	require.True(t, track.IsAppleMusicFav)
}

func TestGetPendingAlbumGroupsSorting(t *testing.T) {
	db := newTrackResolutionTestDB(t, "pending_album_sorting")
	ctx := context.Background()

	now := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)

	// A: 2 records, Latest: now - 3h, Artist: A
	require.NoError(t, db.Create(&TrackPlayRecord{ID: 1, Artist: "A", Album: "A", AlbumArtist: "A", Track: "T1", PlayTime: now.Add(-time.Hour * 5)}).Error)
	require.NoError(t, db.Create(&TrackPlayRecord{ID: 2, Artist: "A", Album: "A", AlbumArtist: "A", Track: "T2", PlayTime: now.Add(-time.Hour * 3)}).Error)

	// B: 3 records, Latest: now - 5h, Artist: B
	require.NoError(t, db.Create(&TrackPlayRecord{ID: 3, Artist: "B", Album: "B", AlbumArtist: "B", Track: "T1", PlayTime: now.Add(-time.Hour * 7)}).Error)
	require.NoError(t, db.Create(&TrackPlayRecord{ID: 4, Artist: "B", Album: "B", AlbumArtist: "B", Track: "T2", PlayTime: now.Add(-time.Hour * 6)}).Error)
	require.NoError(t, db.Create(&TrackPlayRecord{ID: 5, Artist: "B", Album: "B", AlbumArtist: "B", Track: "T3", PlayTime: now.Add(-time.Hour * 5)}).Error)

	// C: 2 records, Latest: now - 1h, Artist: C
	require.NoError(t, db.Create(&TrackPlayRecord{ID: 6, Artist: "C", Album: "C", AlbumArtist: "C", Track: "T1", PlayTime: now.Add(-time.Hour * 1)}).Error)
	require.NoError(t, db.Create(&TrackPlayRecord{ID: 7, Artist: "C", Album: "C", AlbumArtist: "C", Track: "T2", PlayTime: now.Add(-time.Hour * 4)}).Error)

	// D: 2 records, Latest: now - 1h, Artist: D
	require.NoError(t, db.Create(&TrackPlayRecord{ID: 8, Artist: "D", Album: "D", AlbumArtist: "D", Track: "T1", PlayTime: now.Add(-time.Hour * 1)}).Error)
	require.NoError(t, db.Create(&TrackPlayRecord{ID: 9, Artist: "D", Album: "D", AlbumArtist: "D", Track: "T2", PlayTime: now.Add(-time.Hour * 4)}).Error)

	groups, err := GetPendingAlbumGroups(ctx, 0)
	require.NoError(t, err)

	// 预期顺序:
	// 1. B (3条记录)
	// 2. C (2条记录, 最后播放 -1h, ID: c||c)
	// 3. D (2条记录, 最后播放 -1h, ID: d||d)
	// 4. A (2条记录, 最后播放 -3h, ID: a||a)

	require.Len(t, groups, 4)
	require.Equal(t, "B", groups[0].Artist)
	require.Equal(t, "C", groups[1].Artist)
	require.Equal(t, "D", groups[2].Artist)
	require.Equal(t, "A", groups[3].Artist)
}
