package model

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
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

func TestPendingAlbumWorkItemDetailRefreshesStaleContext(t *testing.T) {
	db := newTrackResolutionTestDB(t, "pending_album_detail_refresh")
	ctx := context.Background()

	baseTime := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)
	require.NoError(
		t,
		db.Create(&TrackPlayRecord{
			ID:          10,
			Artist:      "Yorushika",
			AlbumArtist: "Yorushika",
			Album:       "second person",
			Track:       "Become a cloud before",
			Source:      "Apple Music",
			PlayTime:    baseTime,
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
			PlayTime:    baseTime.Add(-time.Hour),
		}).Error,
	)

	groups, err := GetPendingAlbumGroups(ctx, 10)
	require.NoError(t, err)
	require.Len(t, groups, 1)

	item, err := CreateOrGetPendingAlbumWorkItem(ctx, groups[0].IdentityKey)
	require.NoError(t, err)

	require.NoError(
		t,
		db.Create(&TrackPlayRecord{
			ID:          12,
			Artist:      "Yorushika",
			AlbumArtist: "Yorushika",
			Album:       "second person",
			Track:       "If I were a cloud",
			Source:      "Last.fm",
			PlayTime:    baseTime.Add(time.Minute),
		}).Error,
	)

	detail, err := GetPendingAlbumWorkItemDetail(ctx, item.ID)
	require.NoError(t, err)
	require.True(t, detail.ContextStale)
	require.NotNil(t, detail.LiveGroup)
	require.Equal(t, 3, detail.LiveGroup.PlayRecordCount)
	require.Len(t, detail.PlayRecords, 2)

	refreshed, err := RefreshPendingAlbumWorkItemContext(ctx, item.ID)
	require.NoError(t, err)
	require.Equal(t, PendingAlbumWorkItemStatusOpen, refreshed.Status)

	freshDetail, err := GetPendingAlbumWorkItemDetail(ctx, item.ID)
	require.NoError(t, err)
	require.False(t, freshDetail.ContextStale)
	require.NotNil(t, freshDetail.LiveGroup)
	require.Equal(t, 3, freshDetail.LiveGroup.PlayRecordCount)
	require.Len(t, freshDetail.PlayRecords, 3)
}

func TestPendingAlbumWorkItemDetailDoesNotFlagCompletedAsStale(t *testing.T) {
	db := newTrackResolutionTestDB(t, "pending_album_completed_detail")
	ctx := context.Background()

	baseTime := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)
	require.NoError(
		t,
		db.Create(&TrackPlayRecord{
			ID:          30,
			Artist:      "Yorushika",
			AlbumArtist: "Yorushika",
			Album:       "second person",
			Track:       "Become a cloud before",
			Source:      "Apple Music",
			PlayTime:    baseTime,
		}).Error,
	)

	groups, err := GetPendingAlbumGroups(ctx, 10)
	require.NoError(t, err)
	require.Len(t, groups, 1)

	item, err := CreateOrGetPendingAlbumWorkItem(ctx, groups[0].IdentityKey)
	require.NoError(t, err)

	require.NoError(t, UpdatePendingAlbumWorkItemProgress(ctx, item.ID, PendingAlbumWorkItemStatusCompleted, 123, ""))

	require.NoError(
		t,
		db.Create(&TrackPlayRecord{
			ID:          31,
			Artist:      "Yorushika",
			AlbumArtist: "Yorushika",
			Album:       "second person",
			Track:       "If I were a cloud",
			Source:      "Last.fm",
			PlayTime:    baseTime.Add(time.Minute),
		}).Error,
	)

	detail, err := GetPendingAlbumWorkItemDetail(ctx, item.ID)
	require.NoError(t, err)
	require.False(t, detail.ContextStale)
	require.Nil(t, detail.LiveGroup)
	require.Len(t, detail.PlayRecords, 1)
}

func TestResolveCanonicalAlbumForPendingContextTxKeepsVersionedAlbumSeparate(t *testing.T) {
	db := newTrackResolutionTestDB(t, "pending_album_canonical_versioned_album")

	require.NoError(
		t,
		db.Create(&Album{
			ID:          177,
			Name:        "The Dark Side of the Moon",
			Artist:      "Pink Floyd",
			ReleaseDate: "2016",
			SyncStatus:  3,
			TotalDiscs:  1,
			DiscInfos:   `{"1":10}`,
		}).Error,
	)

	var resolved *Album
	require.NoError(
		t,
		db.Transaction(func(tx *gorm.DB) error {
			var err error
			resolved, err = ResolveCanonicalAlbumForPendingContextTx(
				tx,
				&Album{
					Name:         "The Dark Side of the Moon",
					NameSubtitle: "(50th Anniversary) [Remastered]",
					Artist:       "Pink Floyd",
					ReleaseDate:  "2023-10-13",
					TotalDiscs:   1,
					DiscInfos:    `{"1":10}`,
				},
			)
			return err
		}),
	)

	require.NotNil(t, resolved)
	require.NotEqual(t, int64(177), resolved.ID)
	require.Equal(t, "(50th Anniversary) [Remastered]", resolved.NameSubtitle)
	require.Equal(t, "2023-10-13", resolved.ReleaseDate)

	var original Album
	require.NoError(t, db.First(&original, 177).Error)
	require.Equal(t, "", original.NameSubtitle)
	require.Equal(t, "2016", original.ReleaseDate)
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

	// A: 0 favorites, 2 play records, Latest: now - 3h, Artist: A
	require.NoError(t, db.Create(&TrackPlayRecord{ID: 1, Artist: "A", Album: "A", AlbumArtist: "A", Track: "T1", PlayTime: now.Add(-time.Hour * 5)}).Error)
	require.NoError(t, db.Create(&TrackPlayRecord{ID: 2, Artist: "A", Album: "A", AlbumArtist: "A", Track: "T2", PlayTime: now.Add(-time.Hour * 3)}).Error)

	// B: 0 favorites, 3 play records, Latest: now - 5h, Artist: B
	require.NoError(t, db.Create(&TrackPlayRecord{ID: 3, Artist: "B", Album: "B", AlbumArtist: "B", Track: "T1", PlayTime: now.Add(-time.Hour * 7)}).Error)
	require.NoError(t, db.Create(&TrackPlayRecord{ID: 4, Artist: "B", Album: "B", AlbumArtist: "B", Track: "T2", PlayTime: now.Add(-time.Hour * 6)}).Error)
	require.NoError(t, db.Create(&TrackPlayRecord{ID: 5, Artist: "B", Album: "B", AlbumArtist: "B", Track: "T3", PlayTime: now.Add(-time.Hour * 5)}).Error)

	// C: 0 favorites, 2 play records, Latest: now - 1h, Artist: C
	require.NoError(t, db.Create(&TrackPlayRecord{ID: 6, Artist: "C", Album: "C", AlbumArtist: "C", Track: "T1", PlayTime: now.Add(-time.Hour * 1)}).Error)
	require.NoError(t, db.Create(&TrackPlayRecord{ID: 7, Artist: "C", Album: "C", AlbumArtist: "C", Track: "T2", PlayTime: now.Add(-time.Hour * 4)}).Error)

	// D: 0 favorites, 2 play records, Latest: now - 1h, Artist: D
	require.NoError(t, db.Create(&TrackPlayRecord{ID: 8, Artist: "D", Album: "D", AlbumArtist: "D", Track: "T1", PlayTime: now.Add(-time.Hour * 1)}).Error)
	require.NoError(t, db.Create(&TrackPlayRecord{ID: 9, Artist: "D", Album: "D", AlbumArtist: "D", Track: "T2", PlayTime: now.Add(-time.Hour * 4)}).Error)

	// E: 1 favorite, 1 play record, Artist: E
	require.NoError(t, db.Create(&TrackPlayRecord{ID: 10, Artist: "E", Album: "E", AlbumArtist: "E", Track: "T1", PlayTime: now.Add(-time.Hour * 2)}).Error)
	require.NoError(t, db.Create(&TrackFavoriteEvent{ID: 1, Artist: "E", Album: "E", AlbumArtist: "E", Track: "T1", ResolutionStatus: TrackFavoriteEventResolutionPending, Applied: false}).Error)

	// F: 2 favorites, 0 play records, Artist: F
	require.NoError(t, db.Create(&TrackFavoriteEvent{ID: 2, Artist: "F", Album: "F", AlbumArtist: "F", Track: "T1", ResolutionStatus: TrackFavoriteEventResolutionPending, Applied: false}).Error)
	require.NoError(t, db.Create(&TrackFavoriteEvent{ID: 3, Artist: "F", Album: "F", AlbumArtist: "F", Track: "T2", ResolutionStatus: TrackFavoriteEventResolutionPending, Applied: false}).Error)

	groups, err := GetPendingAlbumGroups(ctx, 0)
	require.NoError(t, err)

	// 预期顺序:
	// 1. F (2条点赞事件, 0条播放)
	// 2. E (1条点赞事件, 1条播放)
	// 3. B (0条点赞, 3条播放)
	// 4. C (0条点赞, 2条播放, 最后播放 -1h, ID: c||c)
	// 5. D (0条点赞, 2条播放, 最后播放 -1h, ID: d||d)
	// 6. A (0条点赞, 2条播放, 最后播放 -3h, ID: a||a)

	require.Len(t, groups, 6)
	require.Equal(t, "F", groups[0].Artist)
	require.Equal(t, "E", groups[1].Artist)
	require.Equal(t, "B", groups[2].Artist)
	require.Equal(t, "C", groups[3].Artist)
	require.Equal(t, "D", groups[4].Artist)
	require.Equal(t, "A", groups[5].Artist)
}

func TestGetPendingAlbumGroupsWithOptions_Filter(t *testing.T) {
	db := newTrackResolutionTestDB(t, "pending_album_filter_fill_limit")
	ctx := context.Background()

	now := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)

	// 构造 10 个专辑 (Album01 ~ Album10)，播放量从高到低 (10 次 ~ 1 次)
	for i := 1; i <= 10; i++ {
		albumName := fmt.Sprintf("Album%02d", i)
		for p := 0; p < (11 - i); p++ {
			require.NoError(t, db.Create(&TrackPlayRecord{
				Artist:      "ArtistX",
				AlbumArtist: "ArtistX",
				Album:       albumName,
				Track:       "Track1",
				PlayTime:    now.Add(time.Duration(-p) * time.Minute),
			}).Error)
		}
	}

	// 为前 6 个专辑 (Album01 ~ Album06) 创建活跃工单 (status = open)
	for i := 1; i <= 6; i++ {
		albumName := fmt.Sprintf("Album%02d", i)
		identityKey := normalizePendingAlbumIdentity("ArtistX", "ArtistX", albumName, "")
		require.NoError(t, db.Create(&PendingAlbumWorkItem{
			Artist:                "ArtistX",
			AlbumArtist:           "ArtistX",
			Album:                 albumName,
			NormalizedIdentityKey: identityKey,
			Status:                PendingAlbumWorkItemStatusOpen,
		}).Error)
	}

	// 测试 1: 请求 filter = "uncreated", limit = 3
	// 前 6 个热门专辑都已建单，过滤后应从后 4 个未建单专辑中精准返回 3 个 (Album07, Album08, Album09)
	uncreatedGroups, err := GetPendingAlbumGroupsWithOptions(ctx, PendingAlbumGroupQueryOptions{
		Filter: "uncreated",
		Limit:  3,
	})
	require.NoError(t, err)
	require.Len(t, uncreatedGroups, 3, "未建单过滤必须填满请求的 limit 数量")
	for _, g := range uncreatedGroups {
		require.Equal(t, int64(0), g.OpenWorkItemID, "返回的分组必须未建单")
	}
	require.Equal(t, "Album07", uncreatedGroups[0].Album)
	require.Equal(t, "Album08", uncreatedGroups[1].Album)
	require.Equal(t, "Album09", uncreatedGroups[2].Album)

	// 测试 2: 请求 filter = "created", limit = 4
	createdGroups, err := GetPendingAlbumGroupsWithOptions(ctx, PendingAlbumGroupQueryOptions{
		Filter: "created",
		Limit:  4,
	})
	require.NoError(t, err)
	require.Len(t, createdGroups, 4, "已建单过滤必须返回 4 条")
	for _, g := range createdGroups {
		require.Greater(t, g.OpenWorkItemID, int64(0), "返回的分组必须已建单")
	}
	require.Equal(t, "Album01", createdGroups[0].Album)
	require.Equal(t, "Album02", createdGroups[1].Album)
}
