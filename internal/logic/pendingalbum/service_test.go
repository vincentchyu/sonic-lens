package pendingalbum

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/config"
	corelog "github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

func newPendingAlbumServiceTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", name)), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(
		t, db.Exec(`
			CREATE TABLE track (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				artist TEXT NOT NULL,
				album TEXT NOT NULL,
				album_subtitle TEXT,
				track TEXT NOT NULL,
				play_count INTEGER DEFAULT 0,
				is_apple_music_fav BOOLEAN DEFAULT 0,
				is_last_fm_fav BOOLEAN DEFAULT 0,
				version INTEGER DEFAULT 1,
				album_artist TEXT,
				track_number INTEGER,
				disc_number INTEGER DEFAULT 1,
				duration INTEGER,
				genre TEXT,
				composer TEXT,
				release_date TEXT,
				music_brainz_id TEXT,
				source TEXT,
				bundle_id TEXT,
				unique_id TEXT,
				created_at DATETIME,
				updated_at DATETIME
			)
		`).Error,
	)
	require.NoError(
		t,
		db.Exec(`
			CREATE TABLE library_change_log (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				entity_type TEXT NOT NULL,
				entity_id INTEGER NOT NULL,
				operation TEXT NOT NULL,
				created_at DATETIME
			)
		`).Error,
	)
	require.NoError(
		t, db.Exec(`
			CREATE TABLE track_album (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				track_id INTEGER NOT NULL,
				album_id INTEGER NOT NULL,
				track_number INTEGER,
				disc_number INTEGER DEFAULT 1,
				mb_recording_id TEXT,
				track TEXT,
				created_at DATETIME,
				updated_at DATETIME
			)
		`).Error,
	)
	require.NoError(
		t, db.Exec(`
			CREATE TABLE album (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				name_subtitle TEXT,
				title_metadata TEXT,
				artist TEXT NOT NULL,
				release_date TEXT,
				original_release_date TEXT,
				genre TEXT,
				country TEXT,
				status TEXT,
				packaging TEXT,
				barcode TEXT,
				total_discs INTEGER DEFAULT 1,
				disc_infos TEXT,
				sync_status INTEGER DEFAULT 0,
				release_type TEXT,
				cover_art_url TEXT,
				cover_art_mime TEXT,
				cover_art_object_key TEXT,
				play_count INTEGER DEFAULT 0,
				created_at DATETIME,
				updated_at DATETIME
			)
		`).Error,
	)
	require.NoError(
		t, db.Exec(`
			CREATE TABLE genre (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				name_zh TEXT,
				extra TEXT,
				play_count INTEGER DEFAULT 0,
				created_at DATETIME,
				updated_at DATETIME
			)
		`).Error,
	)
	require.NoError(
		t, db.Exec(`
			CREATE TABLE track_play_records (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				artist TEXT NOT NULL,
				album_artist TEXT,
				track TEXT NOT NULL,
				album TEXT NOT NULL,
				album_subtitle TEXT,
				genre TEXT,
				album_id INTEGER DEFAULT 0,
				duration INTEGER,
				play_time DATETIME NOT NULL,
				scrobbled BOOLEAN DEFAULT 0,
				music_brainz_id TEXT,
				track_number INTEGER,
				disc_number INTEGER DEFAULT 1,
				source TEXT NOT NULL,
				release_type TEXT,
				cover_art_path TEXT,
				trace_id TEXT,
				root_span_id TEXT,
				trace_sampled BOOLEAN DEFAULT 0,
				resolved_track_id INTEGER DEFAULT 0,
				resolution_status TEXT NOT NULL DEFAULT 'pending',
				resolution_confidence INTEGER DEFAULT 0,
				library_applied BOOLEAN DEFAULT 0,
				created_at DATETIME,
				updated_at DATETIME
			)
		`).Error,
	)
	require.NoError(
		t, db.Exec(`
			CREATE TABLE track_favorite_event (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				source TEXT NOT NULL,
				provider_favorite BOOLEAN NOT NULL DEFAULT 0,
				artist TEXT NOT NULL,
				album TEXT NOT NULL,
				album_subtitle TEXT,
				track TEXT NOT NULL,
				album_artist TEXT,
				track_number INTEGER,
				disc_number INTEGER DEFAULT 1,
				music_brainz_id TEXT,
				duration INTEGER,
				bundle_id TEXT,
				unique_id TEXT,
				resolved_track_id INTEGER DEFAULT 0,
				resolution_status TEXT NOT NULL DEFAULT 'pending',
				resolution_confidence INTEGER DEFAULT 0,
				applied BOOLEAN NOT NULL DEFAULT 0,
				created_at DATETIME,
				updated_at DATETIME
			)
		`).Error,
	)
	require.NoError(
		t, db.Exec(`
			CREATE TABLE pending_album_work_item (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				artist TEXT NOT NULL,
				album TEXT NOT NULL,
				album_subtitle TEXT,
				album_artist TEXT,
				normalized_identity_key TEXT NOT NULL,
				play_record_ids_json TEXT,
				favorite_event_ids_json TEXT,
				selected_release_mb_id INTEGER DEFAULT 0,
				selected_mbid TEXT,
				staging_draft_json TEXT,
				status TEXT NOT NULL DEFAULT 'open',
				resolved_album_id INTEGER DEFAULT 0,
				last_error TEXT,
				completed_at DATETIME,
				created_at DATETIME,
				updated_at DATETIME
			)
		`).Error,
	)
	require.NoError(
		t, db.Exec(`
			CREATE TABLE release_mb (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				mbid TEXT NOT NULL,
				album_id INTEGER NOT NULL,
				name TEXT,
				json_data TEXT,
				created_at DATETIME,
				updated_at DATETIME
			)
		`).Error,
	)
	require.NoError(
		t, db.Exec(`
			CREATE TABLE album_release_mb (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				album_id INTEGER NOT NULL,
				release_mb_id INTEGER NOT NULL,
				mbid TEXT NOT NULL,
				confirmed BOOLEAN DEFAULT 1,
				created_at DATETIME
			)
		`).Error,
	)

	prevConfig := *config.ConfigObj
	prevMySQL := model.GlobalDBForMysql
	prevLogger := corelog.Logger

	model.GlobalDBForMysql = db
	corelog.Logger = zap.NewNop()

	t.Cleanup(func() {
		*config.ConfigObj = prevConfig
		model.GlobalDBForMysql = prevMySQL
		corelog.Logger = prevLogger
	})
	return db
}

func TestManualMaintainPendingAlbumWorkItemCreatesAlbumAndAppliesContext(t *testing.T) {
	db := newPendingAlbumServiceTestDB(t, "pendingalbum_manual_apply")
	ctx := context.Background()
	svc := NewService()
	now := time.Date(2026, 3, 30, 8, 0, 0, 0, time.UTC)

	require.NoError(
		t,
		db.Create(&model.TrackPlayRecord{
			ID:          1,
			Artist:      "崔健",
			AlbumArtist: "崔健",
			Album:       "飞狗",
			Track:       "继续",
			TrackNumber: 1,
			DiscNumber:  1,
			Source:      "Apple Music",
			PlayTime:    now,
		}).Error,
	)
	require.NoError(
		t,
		db.Create(&model.TrackFavoriteEvent{
			ID:               2,
			Source:           model.TrackFavoriteEventSourceAppleMusic,
			ProviderFavorite: true,
			Artist:           "崔健",
			AlbumArtist:      "崔健",
			Album:            "飞狗",
			Track:            "继续",
			TrackNumber:      1,
			DiscNumber:       1,
			ResolutionStatus: model.TrackFavoriteEventResolutionPending,
		}).Error,
	)

	groups, err := model.GetPendingAlbumGroups(ctx, 10)
	require.NoError(t, err)
	require.Len(t, groups, 1)

	item, err := svc.CreateOrGetPendingAlbumWorkItem(ctx, groups[0].IdentityKey)
	require.NoError(t, err)

	report, err := svc.ManualMaintainPendingAlbumWorkItem(
		ctx,
		item.ID,
		ManualPendingAlbumInput{
			ManualAlbum: ManualPendingAlbumAlbumInput{
				Name:          "飞狗",
				AlbumArtist:   "崔健",
				DisplayArtist: "崔健",
				ReleaseDate:   "1994-01-01",
				Genre:         "Rock",
			},
			ManualTracks: []ManualPendingAlbumTrackInput{
				{
					DiscNumber:     1,
					TrackNumber:    1,
					Title:          "继续",
					Artist:         "崔健",
					Duration:       300,
					EvidenceTitles: []string{"继续"},
				},
				{
					DiscNumber:    1,
					TrackNumber:   2,
					Title:         "飞了",
					Artist:        "崔健",
					Duration:      260,
					Composer:      "崔健",
					MusicBrainzID: "manual-mbid-2",
				},
			},
		},
	)
	require.NoError(t, err)
	require.Equal(t, pendingAlbumMaintenanceModeManual, report.Mode)
	require.Positive(t, report.ResolvedAlbumID)
	require.Equal(t, 2, report.TrackAlbumWrites)
	require.Equal(t, 1, report.AppliedPlayRecords)

	var album model.Album
	require.NoError(t, db.First(&album, report.ResolvedAlbumID).Error)
	require.Equal(t, 3, album.SyncStatus)
	require.Equal(t, "1994-01-01", album.ReleaseDate)
	require.Equal(t, 1, album.TotalDiscs)

	var trackAlbums []model.TrackAlbum
	require.NoError(t, db.Where("album_id = ?", album.ID).Order("track_number ASC").Find(&trackAlbums).Error)
	require.Len(t, trackAlbums, 2)
	require.Equal(t, "继续", trackAlbums[0].Track)
	require.Equal(t, "飞了", trackAlbums[1].Track)

	var storedRecord model.TrackPlayRecord
	require.NoError(t, db.First(&storedRecord, 1).Error)
	require.True(t, storedRecord.LibraryApplied)
	require.Positive(t, storedRecord.ResolvedTrackID)

	var storedEvent model.TrackFavoriteEvent
	require.NoError(t, db.First(&storedEvent, 2).Error)
	require.True(t, storedEvent.Applied)
	require.Positive(t, storedEvent.ResolvedTrackID)

	var completedItem model.PendingAlbumWorkItem
	require.NoError(t, db.First(&completedItem, item.ID).Error)
	require.Equal(t, model.PendingAlbumWorkItemStatusCompleted, completedItem.Status)
	require.Equal(t, album.ID, completedItem.ResolvedAlbumID)
}

func TestManualMaintainPendingAlbumWorkItemReusesCuratedAlbum(t *testing.T) {
	db := newPendingAlbumServiceTestDB(t, "pendingalbum_manual_reuse_album")
	ctx := context.Background()
	svc := NewService()

	require.NoError(
		t,
		db.Create(&model.Album{
			ID:         88,
			Name:       "飞狗",
			Artist:     "崔健",
			SyncStatus: 3,
			TotalDiscs: 1,
		}).Error,
	)
	require.NoError(
		t,
		db.Create(&model.TrackPlayRecord{
			ID:          11,
			Artist:      "崔健",
			AlbumArtist: "崔健",
			Album:       "飞狗",
			Track:       "继续",
			TrackNumber: 1,
			DiscNumber:  1,
			Source:      "Apple Music",
			PlayTime:    time.Date(2026, 3, 30, 9, 0, 0, 0, time.UTC),
		}).Error,
	)

	groups, err := model.GetPendingAlbumGroups(ctx, 10)
	require.NoError(t, err)
	item, err := svc.CreateOrGetPendingAlbumWorkItem(ctx, groups[0].IdentityKey)
	require.NoError(t, err)

	report, err := svc.ManualMaintainPendingAlbumWorkItem(
		ctx,
		item.ID,
		ManualPendingAlbumInput{
			ManualAlbum: ManualPendingAlbumAlbumInput{
				Name:          "飞狗",
				AlbumArtist:   "崔健",
				DisplayArtist: "崔健",
				Genre:         "Rock",
			},
			ManualTracks: []ManualPendingAlbumTrackInput{
				{
					DiscNumber:     1,
					TrackNumber:    1,
					Title:          "继续",
					Artist:         "崔健",
					EvidenceTitles: []string{"继续"},
				},
			},
		},
	)
	require.NoError(t, err)
	require.Equal(t, int64(88), report.ResolvedAlbumID)

	var albumCount int64
	require.NoError(t, db.Model(&model.Album{}).Where("artist = ? AND name = ?", "崔健", "飞狗").Count(&albumCount).Error)
	require.Equal(t, int64(1), albumCount)
}

func TestManualMaintainPendingAlbumWorkItemRejectsDuplicatedTrackPosition(t *testing.T) {
	db := newPendingAlbumServiceTestDB(t, "pendingalbum_manual_duplicate_position")
	ctx := context.Background()
	svc := NewService()

	require.NoError(
		t,
		db.Create(&model.TrackPlayRecord{
			ID:          21,
			Artist:      "崔健",
			AlbumArtist: "崔健",
			Album:       "飞狗",
			Track:       "继续",
			Source:      "Apple Music",
			PlayTime:    time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC),
		}).Error,
	)

	groups, err := model.GetPendingAlbumGroups(ctx, 10)
	require.NoError(t, err)
	item, err := svc.CreateOrGetPendingAlbumWorkItem(ctx, groups[0].IdentityKey)
	require.NoError(t, err)

	_, err = svc.ManualMaintainPendingAlbumWorkItem(
		ctx,
		item.ID,
		ManualPendingAlbumInput{
			ManualAlbum: ManualPendingAlbumAlbumInput{
				Name:        "飞狗",
				AlbumArtist: "崔健",
			},
			ManualTracks: []ManualPendingAlbumTrackInput{
				{DiscNumber: 1, TrackNumber: 1, Title: "继续"},
				{DiscNumber: 1, TrackNumber: 1, Title: "飞了"},
			},
		},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicated disc/track position")

	var albumCount int64
	require.NoError(t, db.Model(&model.Album{}).Where("artist = ? AND name = ?", "崔健", "飞狗").Count(&albumCount).Error)
	require.Zero(t, albumCount)

	var storedItem model.PendingAlbumWorkItem
	require.NoError(t, db.First(&storedItem, item.ID).Error)
	require.Equal(t, model.PendingAlbumWorkItemStatusOpen, storedItem.Status)
}

func TestManualMaintainPendingAlbumWorkItemUsesManualTitleWhenReusingResolvedTrack(t *testing.T) {
	db := newPendingAlbumServiceTestDB(t, "pendingalbum_manual_reuse_track_title")
	ctx := context.Background()
	svc := NewService()
	now := time.Date(2026, 3, 30, 11, 0, 0, 0, time.UTC)

	require.NoError(
		t,
		db.Create(&model.Track{
			ID:          31,
			Artist:      "达达乐队",
			Album:       "黄金时代",
			Track:       "黄金时代",
			AlbumArtist: "达达乐队",
			TrackNumber: 2,
			DiscNumber:  1,
			Version:     1,
		}).Error,
	)
	require.NoError(
		t,
		db.Create(&model.TrackPlayRecord{
			ID:               41,
			Artist:           "达达乐队",
			AlbumArtist:      "达达乐队",
			Album:            "黄金时代",
			Track:            "黄金时代",
			TrackNumber:      2,
			DiscNumber:       1,
			Source:           "Apple Music",
			PlayTime:         now,
			ResolvedTrackID:  31,
			ResolutionStatus: model.TrackPlayRecordResolutionResolved,
		}).Error,
	)

	groups, err := model.GetPendingAlbumGroups(ctx, 10)
	require.NoError(t, err)
	require.Len(t, groups, 1)

	item, err := svc.CreateOrGetPendingAlbumWorkItem(ctx, groups[0].IdentityKey)
	require.NoError(t, err)

	report, err := svc.ManualMaintainPendingAlbumWorkItem(
		ctx,
		item.ID,
		ManualPendingAlbumInput{
			ManualAlbum: ManualPendingAlbumAlbumInput{
				Name:        "黄金时代",
				AlbumArtist: "达达乐队",
				Genre:       "pop rock",
			},
			ManualTracks: []ManualPendingAlbumTrackInput{
				{
					DiscNumber:     1,
					TrackNumber:    2,
					Title:          "从巴巴罗萨到浮出水面",
					Artist:         "达达乐队",
					EvidenceTitles: []string{"黄金时代"},
				},
			},
		},
	)
	require.NoError(t, err)
	require.Equal(t, 1, report.ReusedHeardTracks)

	var storedTrack model.Track
	require.NoError(t, db.First(&storedTrack, 31).Error)
	require.Equal(t, "从巴巴罗萨到浮出水面", storedTrack.Track)
	require.Equal(t, int8(2), storedTrack.TrackNumber)
	require.Equal(t, int8(1), storedTrack.DiscNumber)

	var storedTrackAlbum model.TrackAlbum
	require.NoError(
		t,
		db.Where("album_id = ? AND track_number = ? AND disc_number = ?", report.ResolvedAlbumID, 2, 1).
			First(&storedTrackAlbum).Error,
	)
	require.Equal(t, "从巴巴罗萨到浮出水面", storedTrackAlbum.Track)
}

func TestManualMaintainPendingAlbumWorkItemUsesManualTitleForCreatedTrackAndBindsEvidence(t *testing.T) {
	db := newPendingAlbumServiceTestDB(t, "pendingalbum_manual_create_track_title")
	ctx := context.Background()
	svc := NewService()
	now := time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC)

	require.NoError(
		t,
		db.Create(&model.TrackPlayRecord{
			ID:          51,
			Artist:      "达达乐队",
			AlbumArtist: "达达乐队",
			Album:       "黄金时代",
			Track:       "黄金时代",
			TrackNumber: 2,
			DiscNumber:  1,
			Source:      "Apple Music",
			PlayTime:    now,
		}).Error,
	)
	require.NoError(
		t,
		db.Create(&model.TrackFavoriteEvent{
			ID:               52,
			Source:           model.TrackFavoriteEventSourceAppleMusic,
			ProviderFavorite: true,
			Artist:           "达达乐队",
			AlbumArtist:      "达达乐队",
			Album:            "黄金时代",
			Track:            "黄金时代",
			TrackNumber:      2,
			DiscNumber:       1,
			ResolutionStatus: model.TrackFavoriteEventResolutionPending,
		}).Error,
	)

	groups, err := model.GetPendingAlbumGroups(ctx, 10)
	require.NoError(t, err)
	require.Len(t, groups, 1)

	item, err := svc.CreateOrGetPendingAlbumWorkItem(ctx, groups[0].IdentityKey)
	require.NoError(t, err)

	report, err := svc.ManualMaintainPendingAlbumWorkItem(
		ctx,
		item.ID,
		ManualPendingAlbumInput{
			ManualAlbum: ManualPendingAlbumAlbumInput{
				Name:        "黄金时代",
				AlbumArtist: "达达乐队",
				Genre:       "pop rock",
			},
			ManualTracks: []ManualPendingAlbumTrackInput{
				{
					DiscNumber:     1,
					TrackNumber:    2,
					Title:          "巴巴罗萨",
					Artist:         "达达乐队",
					EvidenceTitles: []string{"黄金时代"},
				},
			},
		},
	)
	require.NoError(t, err)
	require.Equal(t, 1, report.CreatedTracks)
	require.Equal(t, 1, report.AppliedPlayRecords)
	require.Equal(t, 1, report.AppliedFavoriteEvents)

	var storedTrack model.Track
	require.NoError(
		t,
		db.Where("artist = ? AND album = ? AND track_number = ? AND disc_number = ?", "达达乐队", "黄金时代", 2, 1).
			First(&storedTrack).Error,
	)
	require.Equal(t, "巴巴罗萨", storedTrack.Track)
	require.Equal(t, 1, storedTrack.PlayCount)
	require.True(t, storedTrack.IsAppleMusicFav)

	var storedRecord model.TrackPlayRecord
	require.NoError(t, db.First(&storedRecord, 51).Error)
	require.True(t, storedRecord.LibraryApplied)
	require.Equal(t, storedTrack.ID, storedRecord.ResolvedTrackID)

	var storedEvent model.TrackFavoriteEvent
	require.NoError(t, db.First(&storedEvent, 52).Error)
	require.True(t, storedEvent.Applied)
	require.Equal(t, storedTrack.ID, storedEvent.ResolvedTrackID)

	var storedTrackAlbum model.TrackAlbum
	require.NoError(
		t,
		db.Where("album_id = ? AND track_number = ? AND disc_number = ?", report.ResolvedAlbumID, 2, 1).
			First(&storedTrackAlbum).Error,
	)
	require.Equal(t, "巴巴罗萨", storedTrackAlbum.Track)
}

func TestApplyPendingAlbumStructureTxKeepsMusicBrainzLinkingBehavior(t *testing.T) {
	db := newPendingAlbumServiceTestDB(t, "pendingalbum_mb_structure")

	report := &DeepMaintainPendingAlbumReport{Mode: pendingAlbumMaintenanceModeMusicBrainz}
	detail := &model.PendingAlbumWorkItemDetail{
		PlayRecords:    []*model.TrackPlayRecord{},
		FavoriteEvents: []*model.TrackFavoriteEvent{},
	}
	material := &pendingAlbumMaintenanceMaterial{
		Mode: pendingAlbumMaintenanceModeMusicBrainz,
		AlbumCandidate: &model.Album{
			Name:        "黄金时代",
			Artist:      "达达乐队",
			ReleaseDate: "2003-10-29",
			Genre:       "pop rock",
			TotalDiscs:  1,
			DiscInfos:   `{"1":11}`,
		},
		TrackDrafts: []pendingAlbumTrackDraft{
			{
				Title:       "不经意间",
				TrackArtist: "达达乐队",
				AlbumArtist: "达达乐队",
				DiscNumber:  1,
				TrackNumber: 1,
				Duration:    240,
			},
		},
		ReleaseMB: &model.ReleaseMB{
			ID:       7001,
			MBID:     "mb-release-1",
			Name:     "黄金时代",
			JSONData: `{"id":"mb-release-1"}`,
		},
	}

	require.NoError(
		t,
		db.Transaction(func(tx *gorm.DB) error {
			return applyPendingAlbumStructureTx(tx, detail, material, report)
		}),
	)
	require.Positive(t, report.ResolvedAlbumID)
	require.Equal(t, 1, report.CreatedTracks)
	require.Equal(t, 1, report.TrackAlbumWrites)

	var release model.ReleaseMB
	require.NoError(t, db.Where("mbid = ?", "mb-release-1").First(&release).Error)
	require.Equal(t, report.ResolvedAlbumID, release.AlbumID)

	var link model.AlbumReleaseMB
	require.NoError(
		t,
		db.Where("album_id = ? AND release_mb_id = ?", report.ResolvedAlbumID, release.ID).First(&link).Error,
	)
	require.Equal(t, "mb-release-1", link.MBID)

	var album model.Album
	require.NoError(t, db.First(&album, report.ResolvedAlbumID).Error)
	require.Equal(t, 3, album.SyncStatus)

	var track model.Track
	require.NoError(
		t,
		db.Where("artist = ? AND album = ? AND track_number = ? AND disc_number = ?", "达达乐队", "黄金时代", 1, 1).
			First(&track).Error,
	)
	require.Equal(t, "不经意间", track.Track)
}

func TestSavePendingAlbumWorkItemStagingDraftAndPreview(t *testing.T) {
	db := newPendingAlbumServiceTestDB(t, t.Name())
	ctx := context.Background()

	require.NoError(
		t, db.Create(
			&model.PendingAlbumWorkItem{
				ID:                    101,
				Artist:                "周杰伦",
				Album:                 "范特西",
				NormalizedIdentityKey: "周杰伦||范特西||",
				Status:                model.PendingAlbumWorkItemStatusOpen,
			},
		).Error,
	)

	draftJSON := `{"work_item_id":101,"diff_track_count":1}`
	err := model.SavePendingAlbumWorkItemStagingDraft(ctx, 101, 202, "mb-fantasty-1", draftJSON)
	require.NoError(t, err)

	item, err := model.GetPendingAlbumWorkItemByID(ctx, 101)
	require.NoError(t, err)
	require.Equal(t, model.PendingAlbumWorkItemStatusStaged, item.Status)
	require.Equal(t, int64(202), item.SelectedReleaseMBID)
	require.Equal(t, "mb-fantasty-1", item.SelectedMBID)
	require.Equal(t, draftJSON, item.StagingDraftJSON)
}

func TestApplyAlbumMBMaintenanceUpdatesTrackAlbumMBRecordingID(t *testing.T) {
	db := newPendingAlbumServiceTestDB(t, t.Name())
	ctx := context.Background()

	album := &model.Album{
		ID:          4743,
		Name:        "My Life Will",
		Artist:      "张悬",
		Genre:       "C-Pop",
		ReleaseDate: "2006-06-09",
		SyncStatus:  2,
	}
	require.NoError(t, db.Create(album).Error)

	track := &model.Track{
		ID:          6006,
		Artist:      "张悬",
		AlbumArtist: "张悬",
		Track:       "Scream",
		Album:       "My Life Will",
		Genre:       "C-Pop",
	}
	require.NoError(t, db.Create(track).Error)

	ta := &model.TrackAlbum{
		ID:          6563,
		TrackID:     6006,
		AlbumID:     4743,
		DiscNumber:  1,
		TrackNumber: 1,
		Track:       "Scream",
	}
	require.NoError(t, db.Create(ta).Error)

	svc := NewService()
	input := &ManualPendingAlbumInput{
		ManualAlbum: ManualPendingAlbumAlbumInput{
			Name:        "My Life Will",
			AlbumArtist: "张悬",
			Genre:       "C-Pop",
			ReleaseDate: "2006-06-09",
		},
		ManualTracks: []ManualPendingAlbumTrackInput{
			{
				DiscNumber:    1,
				TrackNumber:   1,
				Title:         "Scream",
				Artist:        "张悬",
				Genre:         "C-Pop",
				MusicBrainzID: "b72e4585-c536-4a4f-969e-8067c4b5bb99",
			},
		},
	}

	err := svc.ApplyAlbumMBMaintenance(ctx, 4743, input)
	require.NoError(t, err)

	var updatedTA model.TrackAlbum
	require.NoError(t, db.Where("id = ?", 6563).First(&updatedTA).Error)
	require.Equal(t, "b72e4585-c536-4a4f-969e-8067c4b5bb99", updatedTA.MusicBrainzRecordingID)

	var updatedTrack model.Track
	require.NoError(t, db.Where("id = ?", 6006).First(&updatedTrack).Error)
	require.Equal(t, "b72e4585-c536-4a4f-969e-8067c4b5bb99", updatedTrack.MusicBrainzID)
}

func TestManualMaintainPendingAlbumWorkItemPreservesAlbumSubtitleAndReleaseType(t *testing.T) {
	db := newPendingAlbumServiceTestDB(t, t.Name())
	ctx := context.Background()
	svc := NewService()
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)

	require.NoError(
		t,
		db.Create(&model.TrackPlayRecord{
			ID:            101,
			Artist:        "Dire Straits",
			AlbumArtist:   "Dire Straits",
			Album:         "Communiqué",
			AlbumSubtitle: "Remastered",
			Track:         "Once Upon a Time in the West",
			TrackNumber:   1,
			DiscNumber:    1,
			Source:        "Apple Music",
			PlayTime:      now,
		}).Error,
	)

	groups, err := model.GetPendingAlbumGroups(ctx, 10)
	require.NoError(t, err)
	require.Len(t, groups, 1)

	item, err := svc.CreateOrGetPendingAlbumWorkItem(ctx, groups[0].IdentityKey)
	require.NoError(t, err)
	require.Equal(t, "Remastered", item.AlbumSubtitle)

	report, err := svc.ManualMaintainPendingAlbumWorkItem(
		ctx,
		item.ID,
		ManualPendingAlbumInput{
			ManualAlbum: ManualPendingAlbumAlbumInput{
				Name:          "Communiqué",
				AlbumSubtitle: "Remastered",
				ReleaseType:   "album",
				AlbumArtist:   "Dire Straits",
				DisplayArtist: "Dire Straits",
				ReleaseDate:   "2013-02-20",
				Genre:         "Rock",
			},
			ManualTracks: []ManualPendingAlbumTrackInput{
				{
					DiscNumber:     1,
					TrackNumber:    1,
					Title:          "Once Upon a Time in the West",
					Artist:         "Dire Straits",
					Duration:       325,
					EvidenceTitles: []string{"Once Upon a Time in the West"},
				},
			},
		},
	)
	require.NoError(t, err)
	require.Equal(t, 1, report.CreatedTracks)

	var album model.Album
	require.NoError(t, db.Where("id = ?", report.ResolvedAlbumID).First(&album).Error)
	require.Equal(t, "Communiqué", album.Name)
	require.Equal(t, "Remastered", album.NameSubtitle)
	require.Equal(t, "album", album.ReleaseType)
	require.Equal(t, int64(1), album.PlayCount)

	var track model.Track
	require.NoError(t, db.Where("album = ? AND track_number = 1", "Communiqué").First(&track).Error)
	require.Equal(t, "Remastered", track.AlbumSubtitle)

	var playRecord model.TrackPlayRecord
	require.NoError(t, db.Where("id = 101").First(&playRecord).Error)
	require.Equal(t, "Remastered", playRecord.AlbumSubtitle)
	require.Equal(t, "album", playRecord.ReleaseType)
	require.Equal(t, album.ID, playRecord.AlbumID)
}

func TestManualMaintainPendingAlbumWorkItemInheritsSubtitleAndParsesSuffix(t *testing.T) {
	db := newPendingAlbumServiceTestDB(t, t.Name())
	ctx := context.Background()
	svc := NewService()
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)

	require.NoError(
		t,
		db.Create(&model.TrackPlayRecord{
			ID:            201,
			Artist:        "Artist X",
			AlbumArtist:   "Artist X",
			Album:         "Flowers",
			AlbumSubtitle: "Deluxe",
			Track:         "Track A",
			TrackNumber:   1,
			DiscNumber:    1,
			Source:        "Apple Music",
			PlayTime:      now,
		}).Error,
	)

	groups, err := model.GetPendingAlbumGroups(ctx, 10)
	require.NoError(t, err)
	require.Len(t, groups, 1)

	item, err := svc.CreateOrGetPendingAlbumWorkItem(ctx, groups[0].IdentityKey)
	require.NoError(t, err)

	// 手动维护时输入了 "Flowers - EP"，且未显式传 AlbumSubtitle
	report, err := svc.ManualMaintainPendingAlbumWorkItem(
		ctx,
		item.ID,
		ManualPendingAlbumInput{
			ManualAlbum: ManualPendingAlbumAlbumInput{
				Name:          "Flowers - EP",
				AlbumArtist:   "Artist X",
				DisplayArtist: "Artist X",
				ReleaseDate:   "2020-01-01",
				Genre:         "Pop",
			},
			ManualTracks: []ManualPendingAlbumTrackInput{
				{
					DiscNumber:     1,
					TrackNumber:    1,
					Title:          "Track A",
					Artist:         "Artist X",
					Duration:       200,
					EvidenceTitles: []string{"Track A"},
				},
			},
		},
	)
	require.NoError(t, err)

	var album model.Album
	require.NoError(t, db.Where("id = ?", report.ResolvedAlbumID).First(&album).Error)
	require.Equal(t, "Flowers", album.Name)
	require.Equal(t, "Deluxe", album.NameSubtitle)
	require.Equal(t, "ep", album.ReleaseType)

	var track model.Track
	require.NoError(t, db.Where("album = ? AND track_number = 1", "Flowers").First(&track).Error)
	require.Equal(t, "Deluxe", track.AlbumSubtitle)
}
