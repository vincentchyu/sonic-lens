package model

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/config"
)

func newAlbumCleanupTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", name)), &gorm.Config{})
	require.NoError(t, err)

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
				created_at DATETIME,
				updated_at DATETIME
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
	require.NoError(
		t, db.Exec(`
			CREATE TABLE library_change_log (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				entity_type TEXT NOT NULL,
				entity_id INTEGER NOT NULL,
				operation TEXT NOT NULL,
				created_at DATETIME
			)
		`).Error,
	)

	prevConfig := *config.ConfigObj
	prevMySQL := GlobalDBForMysql

	config.ConfigObj.Database.Type = string(common.DatabaseTypeSQLite)
	GlobalDBForMysql = db

	t.Cleanup(
		func() {
			*config.ConfigObj = prevConfig
			GlobalDBForMysql = prevMySQL
		},
	)

	return db
}

func TestCleanupDuplicateAlbumsMergesRelationsIntoCanonicalAlbum(t *testing.T) {
	db := newAlbumCleanupTestDB(t, "album_cleanup_merge")
	ctx := context.Background()

	canonical := &Album{
		ID:          10,
		Artist:      "Pink Floyd",
		Name:        "The Dark Side of the Moon",
		ReleaseDate: "2016",
		SyncStatus:  3,
	}
	source := &Album{
		ID:          11,
		Artist:      "Pink Floyd",
		Name:        "The Dark Side of the Moon",
		ReleaseDate: "1973-03-01",
		Genre:       "Progressive Rock",
		SyncStatus:  0,
	}
	placeholderAlbum := &Album{
		ID:         12,
		Artist:     "Pink Floyd",
		Name:       "The Dark Side of the Moon",
		SyncStatus: 0,
	}

	require.NoError(t, db.Create(canonical).Error)
	require.NoError(t, db.Create(source).Error)
	require.NoError(t, db.Create(placeholderAlbum).Error)

	require.NoError(t, db.Create(&TrackAlbum{
		TrackID:     1001,
		AlbumID:     10,
		TrackNumber: 1,
		DiscNumber:  1,
		Track:       "Speak to Me",
	}).Error)
	require.NoError(t, db.Create(&TrackAlbum{
		TrackID:     1002,
		AlbumID:     11,
		TrackNumber: 2,
		DiscNumber:  1,
		Track:       "Breathe (In the Air)",
	}).Error)
	require.NoError(t, db.Create(&TrackAlbum{
		TrackID:     0,
		AlbumID:     12,
		TrackNumber: 3,
		DiscNumber:  1,
		Track:       "On the Run",
	}).Error)

	require.NoError(t, db.Create(&AlbumReleaseMB{
		AlbumID:     11,
		ReleaseMBID: 501,
		MBID:        "release-501",
		Confirmed:   true,
	}).Error)

	report, err := CleanupDuplicateAlbums(
		CleanupDuplicateAlbumsParams{
			Ctx: ctx,
		},
	)
	require.NoError(t, err)
	require.Len(t, report.Groups, 1)
	require.Equal(t, int64(10), report.Groups[0].CanonicalAlbumID)
	require.ElementsMatch(t, []int64{11, 12}, report.Groups[0].MergedAlbumIDs)

	var albums []Album
	require.NoError(t, db.Order("id ASC").Find(&albums).Error)
	require.Len(t, albums, 1)
	require.Equal(t, int64(10), albums[0].ID)
	require.Equal(t, "Progressive Rock", albums[0].Genre)
	require.Equal(t, 3, albums[0].SyncStatus)

	var trackAlbums []TrackAlbum
	require.NoError(t, db.Order("track_id ASC, id ASC").Find(&trackAlbums).Error)
	require.Len(t, trackAlbums, 3)
	for _, row := range trackAlbums {
		require.Equal(t, int64(10), row.AlbumID)
	}

	var releaseLinks []AlbumReleaseMB
	require.NoError(t, db.Find(&releaseLinks).Error)
	require.Len(t, releaseLinks, 1)
	require.Equal(t, int64(10), releaseLinks[0].AlbumID)
	require.Equal(t, int64(501), releaseLinks[0].ReleaseMBID)
}

func TestCleanupDuplicateAlbumsDryRunDoesNotModifyData(t *testing.T) {
	db := newAlbumCleanupTestDB(t, "album_cleanup_dry_run")
	ctx := context.Background()

	require.NoError(
		t, db.Create(&Album{
			ID:          21,
			Artist:      "Radiohead",
			Name:        "The Bends",
			ReleaseDate: "1995",
			SyncStatus:  3,
		}).Error,
	)
	require.NoError(
		t, db.Create(&Album{
			ID:          22,
			Artist:      "Radiohead",
			Name:        "The Bends",
			ReleaseDate: "1995-03-08",
		}).Error,
	)

	report, err := CleanupDuplicateAlbums(
		CleanupDuplicateAlbumsParams{
			Ctx:    ctx,
			DryRun: true,
		},
	)
	require.NoError(t, err)
	require.Len(t, report.Groups, 1)
	require.Equal(t, int64(21), report.Groups[0].CanonicalAlbumID)
	require.Equal(t, []int64{22}, report.Groups[0].MergedAlbumIDs)

	var count int64
	require.NoError(t, db.Model(&Album{}).Count(&count).Error)
	require.Equal(t, int64(2), count)
}

func TestBuildAlbumMergeUpdatesSkipsEmptyReleaseDate(t *testing.T) {
	canonical := &Album{
		ID:         2,
		Artist:     "Erik Truffaz Quartet",
		Name:       "El tiempo de la revolución",
		SyncStatus: 1,
	}
	source := &Album{
		ID:         4132,
		Artist:     "Erik Truffaz Quartet",
		Name:       "El tiempo de la revolución",
		SyncStatus: 0,
	}

	fields := buildAlbumMergeUpdates(canonical, source)
	require.Empty(t, fields["release_date"])
	require.NotContains(t, fields, "release_date")
	require.NotContains(t, fields, "sync_status")
}

func TestCleanupDuplicateAlbumsPromotesReleaseDateAfterSourceDeletion(t *testing.T) {
	db := newAlbumCleanupTestDB(t, "album_cleanup_release_date_promotion")
	ctx := context.Background()

	require.NoError(
		t, db.Create(&Album{
			ID:         11,
			Artist:     "郭顶",
			Name:       "飞行器的执行周期",
			SyncStatus: 0,
		}).Error,
	)
	require.NoError(
		t, db.Create(&Album{
			ID:          4201,
			Artist:      "郭顶",
			Name:        "飞行器的执行周期",
			ReleaseDate: "2016-11-18",
			SyncStatus:  0,
		}).Error,
	)

	require.NoError(t, db.Create(&TrackAlbum{TrackID: 1001, AlbumID: 11, TrackNumber: 1, DiscNumber: 1}).Error)
	require.NoError(t, db.Create(&TrackAlbum{TrackID: 1003, AlbumID: 11, TrackNumber: 3, DiscNumber: 1}).Error)
	require.NoError(t, db.Create(&TrackAlbum{TrackID: 1002, AlbumID: 4201, TrackNumber: 2, DiscNumber: 1}).Error)

	report, err := CleanupDuplicateAlbums(CleanupDuplicateAlbumsParams{Ctx: ctx})
	require.NoError(t, err)
	require.Len(t, report.Groups, 1)
	require.Equal(t, int64(11), report.Groups[0].CanonicalAlbumID)
	require.Equal(t, []int64{4201}, report.Groups[0].MergedAlbumIDs)

	var albums []Album
	require.NoError(t, db.Order("id ASC").Find(&albums).Error)
	require.Len(t, albums, 1)
	require.Equal(t, int64(11), albums[0].ID)
	require.Equal(t, "2016-11-18", albums[0].ReleaseDate)
}

func TestCleanupDuplicateAlbumsContinueOnErrorSkipsConflictedGroup(t *testing.T) {
	db := newAlbumCleanupTestDB(t, "album_cleanup_continue_on_error")
	ctx := context.Background()

	require.NoError(t, db.Create(&Album{ID: 309, Artist: "万能青年旅店", Name: "冀西南林路行", ReleaseDate: "2020-12-22", SyncStatus: 3}).Error)
	require.NoError(t, db.Create(&Album{ID: 4097, Artist: "万能青年旅店", Name: "冀西南林路行", ReleaseDate: "2022-03-10", SyncStatus: 3}).Error)
	require.NoError(t, db.Create(&TrackAlbum{TrackID: 574, AlbumID: 309, TrackNumber: 1, DiscNumber: 1, Track: "早"}).Error)
	require.NoError(t, db.Create(&TrackAlbum{TrackID: 3022, AlbumID: 4097, TrackNumber: 1, DiscNumber: 1, Track: "冀西南林路行"}).Error)

	require.NoError(t, db.Create(&Album{ID: 5000, Artist: "Radiohead", Name: "The Bends", ReleaseDate: "1995", SyncStatus: 3}).Error)
	require.NoError(t, db.Create(&Album{ID: 5001, Artist: "Radiohead", Name: "The Bends", ReleaseDate: "1995-03-08"}).Error)
	require.NoError(t, db.Create(&TrackAlbum{TrackID: 7001, AlbumID: 5000, TrackNumber: 1, DiscNumber: 1, Track: "Planet Telex"}).Error)
	require.NoError(t, db.Create(&TrackAlbum{TrackID: 7002, AlbumID: 5001, TrackNumber: 2, DiscNumber: 1, Track: "The Bends"}).Error)

	report, err := CleanupDuplicateAlbums(
		CleanupDuplicateAlbumsParams{
			Ctx:             ctx,
			ContinueOnError: true,
		},
	)
	require.NoError(t, err)
	require.Len(t, report.Groups, 1)
	require.Equal(t, "Radiohead", report.Groups[0].Artist)
	require.Equal(t, "The Bends", report.Groups[0].Name)
	require.Len(t, report.Skipped, 1)
	require.Equal(t, "万能青年旅店", report.Skipped[0].Artist)
	require.Equal(t, "冀西南林路行", report.Skipped[0].Name)

	var bendsCount int64
	require.NoError(t, db.Model(&Album{}).Where("artist = ? AND name = ?", "Radiohead", "The Bends").Count(&bendsCount).Error)
	require.Equal(t, int64(1), bendsCount)

	var omnipotentCount int64
	require.NoError(t, db.Model(&Album{}).Where("artist = ? AND name = ?", "万能青年旅店", "冀西南林路行").Count(&omnipotentCount).Error)
	require.Equal(t, int64(2), omnipotentCount)
}
