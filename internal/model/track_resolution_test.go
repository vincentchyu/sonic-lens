package model

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/config"
	corelog "github.com/vincentchyu/sonic-lens/core/log"
)

func newTrackResolutionTestDB(t *testing.T, name string) *gorm.DB {
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
				album_id INTEGER DEFAULT 0,
				duration INTEGER,
				play_time DATETIME NOT NULL,
				scrobbled BOOLEAN DEFAULT 0,
				music_brainz_id TEXT,
				track_number INTEGER,
				disc_number INTEGER DEFAULT 1,
				source TEXT NOT NULL,
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
				status TEXT NOT NULL DEFAULT 'open',
				resolved_album_id INTEGER DEFAULT 0,
				last_error TEXT,
				completed_at DATETIME,
				created_at DATETIME,
				updated_at DATETIME
			)
		`).Error,
	)

	prevConfig := *config.ConfigObj
	prevSQLite := GlobalDBForSqlLite
	prevMySQL := GlobalDBForMysql
	prevLogger := corelog.Logger

	config.ConfigObj.Database.Type = string(common.DatabaseTypeSQLite)
	GlobalDBForSqlLite = db
	GlobalDBForMysql = nil
	corelog.Logger = zap.NewNop()

	t.Cleanup(
		func() {
			*config.ConfigObj = prevConfig
			GlobalDBForSqlLite = prevSQLite
			GlobalDBForMysql = prevMySQL
			corelog.Logger = prevLogger
		},
	)

	return db
}

func TestResolveTrackIdentityWithOptionsIgnoresDuplicatedUniqueIDHint(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_resolution_unique_hint")

	require.NoError(t, db.Create(&Track{
		ID:          1,
		Artist:      "Radiohead",
		Album:       "OK Computer",
		Track:       "Climbing up the Walls",
		UniqueID:    "32753",
		TrackNumber: 9,
		DiscNumber:  1,
		Version:     1,
	}).Error)
	require.NoError(t, db.Create(&Track{
		ID:          2,
		Artist:      "诺拉·琼斯",
		Album:       "Come Away With Me (Super Deluxe Edition)",
		Track:       "Feelin' The Same Way",
		UniqueID:    "32753",
		TrackNumber: 4,
		DiscNumber:  1,
		Version:     1,
	}).Error)

	identity, existing, err := resolveTrackIdentityWithOptions(
		db,
		"Pink Floyd",
		"Wish You Were Here 50",
		"Shine On You Crazy Diamond (Early Instrumental Version, Rough Mix)",
		TrackMetadata{
			UniqueID:   "32753",
			Confidence: common.TrackMetadataConfidenceLow,
		},
		trackIdentityResolveOptions{
			allowLooseNameFallback: false,
			allowUniqueIDHint:      true,
		},
	)
	require.NoError(t, err)
	require.Nil(t, existing)
	require.Equal(t, "Pink Floyd", identity.Artist)
	require.Equal(t, "Wish You Were Here 50", identity.Album)
}

func TestIncrementTrackPlayCountLowConfidenceDoesNotCreateTrack(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_resolution_low_confidence_no_create")
	ctx := context.Background()

	require.NoError(
		t, IncrementTrackPlayCount(
			IncrementTrackPlayCountParams{
				Ctx:    ctx,
				Artist: "ELTON JOHN",
				Album:  "Too Low For Zero (Bonus Track Version)",
				Track:  "I'm Still Standing",
				TrackMetadata: TrackMetadata{
					Confidence:  common.TrackMetadataConfidenceLow,
					PlayerType:  "Apple Music",
					ReleaseDate: "1983-05-23",
					ReleaseYear: 1983,
				},
			},
		),
	)

	var count int64
	require.NoError(t, db.Model(&Track{}).Count(&count).Error)
	require.Equal(t, int64(0), count)
}

func TestIncrementTrackPlayCountLowConfidenceOnlyIncrementsResolvedTrack(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_resolution_low_confidence_increment")
	ctx := context.Background()

	require.NoError(t, db.Create(&Track{
		ID:          10,
		Artist:      "Radiohead",
		Album:       "OK Computer",
		Track:       "No Surprises",
		TrackNumber: 10,
		DiscNumber:  1,
		PlayCount:   4,
		Version:     1,
		ReleaseDate: "1997-05-21",
	}).Error)

	require.NoError(
		t, IncrementTrackPlayCount(
			IncrementTrackPlayCountParams{
				Ctx:    ctx,
				Artist: "Radiohead",
				Album:  "OK Computer",
				Track:  "No Surprises",
				TrackMetadata: TrackMetadata{
					TrackNumber: 10,
					DiscNumber:  1,
					Confidence:  common.TrackMetadataConfidenceLow,
					ReleaseDate: "2099-01-01",
				},
			},
		),
	)

	var track Track
	require.NoError(t, db.First(&track, 10).Error)
	require.Equal(t, 5, track.PlayCount)
	require.Equal(t, 2, track.Version)
	require.Equal(t, "1997-05-21", track.ReleaseDate)

	var changes []LibraryChangeLog
	require.NoError(t, db.Order("id ASC").Find(&changes).Error)
	require.Len(t, changes, 1)
	require.Equal(t, LibraryEntityTrack, changes[0].EntityType)
	require.Equal(t, int64(10), changes[0].EntityID)
	require.Equal(t, LibraryOpUpsert, changes[0].Operation)
}

func TestIncrementTrackPlayCountHighConfidenceCreatesTrackAndAlbumLink(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_resolution_high_confidence_create")
	ctx := context.Background()

	require.NoError(
		t, IncrementTrackPlayCount(
			IncrementTrackPlayCountParams{
				Ctx:    ctx,
				Artist: "Radiohead",
				Album:  "Kid A",
				Track:  "Everything in Its Right Place",
				TrackMetadata: TrackMetadata{
					TrackNumber: 1,
					DiscNumber:  1,
					Duration:    251,
					Genre:       "Alternative",
					ReleaseDate: "2000-10-02",
					Confidence:  common.TrackMetadataConfidenceHigh,
					Source:      "Audirvana",
				},
			},
		),
	)

	var album Album
	require.NoError(t, db.Where("artist = ? AND name = ?", "Radiohead", "Kid A").First(&album).Error)
	require.Equal(t, "2000-10-02", album.ReleaseDate)

	var track Track
	require.NoError(
		t,
		db.Where("artist = ? AND album = ? AND track = ?", "Radiohead", "Kid A", "Everything in Its Right Place").
			First(&track).Error,
	)
	require.Equal(t, 1, track.PlayCount)
	require.Equal(t, int8(1), track.TrackNumber)
	require.Equal(t, int8(1), track.DiscNumber)

	var link TrackAlbum
	require.NoError(t, db.Where("track_id = ? AND album_id = ?", track.ID, album.ID).First(&link).Error)
	require.Equal(t, int8(1), link.TrackNumber)
	require.Equal(t, int8(1), link.DiscNumber)
}

func TestIncrementTrackPlayCountDoesNotMutateCuratedAlbumBaseFields(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_resolution_curated_album_frozen")
	ctx := context.Background()

	require.NoError(t, db.Create(&Album{
		ID:          910,
		Name:        "Kid A",
		Artist:      "Radiohead",
		ReleaseDate: "2000-10-02",
		Genre:       "Art Rock",
		SyncStatus:  3,
	}).Error)

	require.NoError(
		t, IncrementTrackPlayCount(
			IncrementTrackPlayCountParams{
				Ctx:    ctx,
				Artist: "Radiohead",
				Album:  "Kid A",
				Track:  "Kid A",
				TrackMetadata: TrackMetadata{
					TrackNumber: 2,
					DiscNumber:  1,
					Duration:    285,
					Genre:       "Alt-Pop",
					ReleaseDate: "2099-01-01",
					Confidence:  common.TrackMetadataConfidenceHigh,
					Source:      "Audirvana",
				},
			},
		),
	)

	var album Album
	require.NoError(t, db.First(&album, 910).Error)
	require.Equal(t, "2000-10-02", album.ReleaseDate)
	require.Equal(t, "Art Rock", album.Genre)
	require.Equal(t, 3, album.SyncStatus)
}

func TestResolveTrackPlayRecordMarksResolvedTrack(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_play_record_resolution")
	ctx := context.Background()

	require.NoError(t, db.Create(&Track{
		ID:          20,
		Artist:      "Radiohead",
		Album:       "In Rainbows",
		Track:       "15 Step",
		TrackNumber: 1,
		DiscNumber:  1,
		Version:     1,
	}).Error)
	require.NoError(t, db.Create(&TrackAlbum{
		TrackID:     20,
		AlbumID:     501,
		TrackNumber: 1,
		DiscNumber:  1,
		Track:       "15 Step",
	}).Error)
	record := &TrackPlayRecord{
		Artist:      "Radiohead",
		Album:       "In Rainbows",
		Track:       "15 Step",
		TrackNumber: 1,
		DiscNumber:  1,
		Source:      "Audirvana",
		PlayTime:    modelTestNow,
	}
	require.NoError(t, InsertTrackPlayRecord(ctx, record))

	require.NoError(
		t, ResolveTrackPlayRecord(
			ctx,
			record.ID,
			record.Artist,
			record.Album,
			record.Track,
			TrackMetadata{
				TrackNumber: 1,
				DiscNumber:  1,
				Confidence:  common.TrackMetadataConfidenceHigh,
			},
		),
	)

	var stored TrackPlayRecord
	require.NoError(t, db.First(&stored, record.ID).Error)
	require.Equal(t, int64(20), stored.ResolvedTrackID)
	require.Equal(t, int64(501), stored.AlbumID)
	require.Equal(t, TrackPlayRecordResolutionResolved, stored.ResolutionStatus)
	require.Equal(t, common.TrackMetadataConfidenceHigh, stored.ResolutionConfidence)
}

func TestProcessTrackPlayRecordAppliesLibraryAndResolution(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_play_record_process")
	ctx := context.Background()

	record := &TrackPlayRecord{
		Artist:      "Radiohead",
		Album:       "Kid A",
		Track:       "Everything in Its Right Place",
		TrackNumber: 1,
		DiscNumber:  1,
		Source:      "Audirvana",
		PlayTime:    modelTestNow,
	}
	require.NoError(t, InsertTrackPlayRecord(ctx, record))

	require.NoError(
		t, ProcessTrackPlayRecord(
			ctx,
			record.ID,
			TrackMetadata{
				TrackNumber: 1,
				DiscNumber:  1,
				Duration:    251,
				Genre:       "Alternative",
				ReleaseDate: "2000-10-02",
				Confidence:  common.TrackMetadataConfidenceHigh,
				Source:      "Audirvana",
			},
		),
	)

	var storedRecord TrackPlayRecord
	require.NoError(t, db.First(&storedRecord, record.ID).Error)
	require.True(t, storedRecord.LibraryApplied)
	require.Equal(t, TrackPlayRecordResolutionResolved, storedRecord.ResolutionStatus)
	require.NotZero(t, storedRecord.ResolvedTrackID)
	require.NotZero(t, storedRecord.AlbumID)

	var storedTrack Track
	require.NoError(t, db.First(&storedTrack, storedRecord.ResolvedTrackID).Error)
	require.Equal(t, 1, storedTrack.PlayCount)

	var link TrackAlbum
	require.NoError(
		t, db.Where("track_id = ? AND album_id = ?", storedTrack.ID, storedRecord.AlbumID).First(&link).Error,
	)
}

func TestProcessTrackPlayRecordPromotesAuthoritativeConfidenceForCuratedAlbum(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_play_record_authoritative")
	ctx := context.Background()

	require.NoError(t, db.Create(&Album{
		ID:         700,
		Name:       "Kid A",
		Artist:     "Radiohead",
		SyncStatus: 3,
	}).Error)
	require.NoError(t, db.Create(&Track{
		ID:            701,
		Artist:        "Radiohead",
		Album:         "Kid A",
		Track:         "Everything in Its Right Place",
		TrackNumber:   1,
		DiscNumber:    1,
		MusicBrainzID: "mbid-701",
		Version:       1,
	}).Error)
	require.NoError(t, db.Create(&TrackAlbum{
		TrackID:                701,
		AlbumID:                700,
		TrackNumber:            1,
		DiscNumber:             1,
		Track:                  "Everything in Its Right Place",
		MusicBrainzRecordingID: "mbid-701",
	}).Error)

	record := &TrackPlayRecord{
		Artist:      "Radiohead",
		Album:       "Kid A",
		Track:       "Everything in Its Right Place",
		TrackNumber: 1,
		DiscNumber:  1,
		Source:      "Apple Music",
		PlayTime:    modelTestNow,
	}
	require.NoError(t, InsertTrackPlayRecord(ctx, record))

	require.NoError(
		t, ProcessTrackPlayRecord(
			ctx,
			record.ID,
			TrackMetadata{
				TrackNumber: 1,
				DiscNumber:  1,
				Duration:    251,
				Confidence:  common.TrackMetadataConfidenceMedium,
				Source:      "Apple Music",
			},
		),
	)

	var stored TrackPlayRecord
	require.NoError(t, db.First(&stored, record.ID).Error)
	require.Equal(t, TrackPlayRecordResolutionResolved, stored.ResolutionStatus)
	require.Equal(t, common.TrackMetadataConfidenceAuthoritative, stored.ResolutionConfidence)
	require.Equal(t, int64(701), stored.ResolvedTrackID)
	require.True(t, stored.LibraryApplied)
}

func TestProcessTrackPlayRecordReusesResolvedTrackForSameSource(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_play_record_reuse_same_source")
	ctx := context.Background()

	require.NoError(t, db.Create(&Album{
		ID:         881,
		Name:       "second person",
		Artist:     "Yorushika",
		SyncStatus: 3,
	}).Error)
	require.NoError(t, db.Create(&Track{
		ID:            880,
		Artist:        "Yorushika",
		Album:         "second person",
		Track:         "飞狗",
		TrackNumber:   1,
		DiscNumber:    1,
		MusicBrainzID: "mbid-flydog",
		PlayCount:     5,
		Version:       1,
	}).Error)
	require.NoError(t, db.Create(&TrackAlbum{
		TrackID:     880,
		AlbumID:     881,
		TrackNumber: 1,
		DiscNumber:  1,
		Track:       "飞狗",
	}).Error)
	require.NoError(t, db.Create(&TrackPlayRecord{
		ID:                   900,
		Artist:               "Yorushika",
		Album:                "second person",
		Track:                "飞狗",
		TrackNumber:          1,
		DiscNumber:           1,
		Source:               "Apple Music",
		PlayTime:             modelTestNow.Add(-time.Hour),
		ResolvedTrackID:      880,
		AlbumID:              881,
		ResolutionStatus:     TrackPlayRecordResolutionResolved,
		ResolutionConfidence: common.TrackMetadataConfidenceAuthoritative,
		LibraryApplied:       true,
		MusicBrainzID:        "mbid-flydog",
	}).Error)

	record := &TrackPlayRecord{
		ID:       901,
		Artist:   "Yorushika",
		Album:    "second person",
		Track:    "飞狗",
		Source:   "Apple Music",
		PlayTime: modelTestNow,
	}
	require.NoError(t, InsertTrackPlayRecord(ctx, record))

	require.NoError(
		t, ProcessTrackPlayRecord(
			ctx,
			record.ID,
			TrackMetadata{
				Confidence: common.TrackMetadataConfidenceLow,
				Source:     "Apple Music",
			},
		),
	)

	var storedRecord TrackPlayRecord
	require.NoError(t, db.First(&storedRecord, record.ID).Error)
	require.Equal(t, int64(880), storedRecord.ResolvedTrackID)
	require.Equal(t, TrackPlayRecordResolutionResolved, storedRecord.ResolutionStatus)
	require.True(t, storedRecord.LibraryApplied)
	require.Equal(t, "mbid-flydog", storedRecord.MusicBrainzID)

	var storedTrack Track
	require.NoError(t, db.First(&storedTrack, 880).Error)
	require.Equal(t, 6, storedTrack.PlayCount)
}

func TestProcessTrackPlayRecordKeepsLibraryAppliedFalseWithoutAlbumBinding(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_play_record_without_album_binding")
	ctx := context.Background()

	require.NoError(t, db.Create(&Track{
		ID:          990,
		Artist:      "刺猬",
		Album:       "生之响往",
		Track:       "火车驶向云外, 梦安魂于九霄",
		TrackNumber: 1,
		DiscNumber:  1,
		PlayCount:   2,
		Version:     1,
	}).Error)

	record := &TrackPlayRecord{
		Artist:      "刺猬",
		Album:       "生之响往",
		Track:       "火车驶向云外, 梦安魂于九霄",
		TrackNumber: 1,
		DiscNumber:  1,
		Source:      "Apple Music",
		PlayTime:    modelTestNow,
	}
	require.NoError(t, InsertTrackPlayRecord(ctx, record))

	require.NoError(
		t, ProcessTrackPlayRecord(
			ctx,
			record.ID,
			TrackMetadata{
				TrackNumber: 1,
				DiscNumber:  1,
				Confidence:  common.TrackMetadataConfidenceMedium,
				Source:      "Apple Music",
			},
		),
	)

	var storedRecord TrackPlayRecord
	require.NoError(t, db.First(&storedRecord, record.ID).Error)
	require.Equal(t, TrackPlayRecordResolutionResolved, storedRecord.ResolutionStatus)
	require.Equal(t, int64(990), storedRecord.ResolvedTrackID)
	require.Equal(t, int64(0), storedRecord.AlbumID)
	require.False(t, storedRecord.LibraryApplied)

	var storedTrack Track
	require.NoError(t, db.First(&storedTrack, 990).Error)
	require.Equal(t, 3, storedTrack.PlayCount)
}

func TestGetTrackByMusicBrainzIdentityTxDoesNotCrossAlbumSubtitle(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_resolution_mb_identity_subtitle")

	require.NoError(t, db.Create(&Track{
		ID:            335,
		Artist:        "Pink Floyd",
		Album:         "The Dark Side of the Moon",
		AlbumSubtitle: "",
		Track:         "Breathe (In the Air)",
		TrackNumber:   2,
		DiscNumber:    1,
		MusicBrainzID: "ecbc7c9b-e79d-4ec8-ac77-44e4a7f7f1b8",
		Version:       1,
	}).Error)

	trackObj, err := GetTrackByMusicBrainzIdentityTx(
		db,
		"ecbc7c9b-e79d-4ec8-ac77-44e4a7f7f1b8",
		"Pink Floyd",
		"The Dark Side of the Moon",
		"(50th Anniversary) [Remastered]",
		"Breathe (in the Air)",
		2,
		1,
	)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	require.Nil(t, trackObj)
}

func TestProcessTrackPlayRecordUsesMatchingAlbumBindingForVersionedTrack(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_play_record_versioned_album_binding")
	ctx := context.Background()

	require.NoError(t, db.Create(&Album{
		ID:          177,
		Name:        "The Dark Side of the Moon",
		Artist:      "Pink Floyd",
		ReleaseDate: "2016",
		SyncStatus:  3,
	}).Error)
	require.NoError(t, db.Create(&Album{
		ID:           4675,
		Name:         "The Dark Side of the Moon",
		NameSubtitle: "(50th Anniversary) [Remastered]",
		Artist:       "Pink Floyd",
		ReleaseDate:  "2023-10-13",
		SyncStatus:   3,
	}).Error)
	require.NoError(t, db.Create(&Track{
		ID:            1777,
		Artist:        "Pink Floyd",
		Album:         "The Dark Side of the Moon",
		AlbumSubtitle: "(50th Anniversary) [Remastered]",
		Track:         "Money",
		TrackNumber:   6,
		DiscNumber:    1,
		MusicBrainzID: "7fef22bd-76aa-4803-b56b-93a5d6e70662",
		PlayCount:     3,
		Version:       1,
	}).Error)
	require.NoError(t, db.Create(&TrackAlbum{
		ID:          1749,
		TrackID:     1777,
		AlbumID:     177,
		TrackNumber: 6,
		DiscNumber:  1,
		Track:       "Money",
	}).Error)
	require.NoError(t, db.Create(&TrackAlbum{
		ID:          5881,
		TrackID:     1777,
		AlbumID:     4675,
		TrackNumber: 6,
		DiscNumber:  1,
		Track:       "Money",
	}).Error)

	record := &TrackPlayRecord{
		ID:            6992,
		Artist:        "Pink Floyd",
		Album:         "The Dark Side of the Moon",
		AlbumSubtitle: "(50th Anniversary) [Remastered]",
		Track:         "Money",
		TrackNumber:   6,
		DiscNumber:    1,
		Source:        "Apple Music",
		PlayTime:      modelTestNow,
	}
	require.NoError(t, InsertTrackPlayRecord(ctx, record))

	require.NoError(
		t, ProcessTrackPlayRecord(
			ctx,
			record.ID,
			TrackMetadata{
				AlbumSubtitle: "(50th Anniversary) [Remastered]",
				TrackNumber:   6,
				DiscNumber:    1,
				MusicBrainzID: "7fef22bd-76aa-4803-b56b-93a5d6e70662",
				Confidence:    common.TrackMetadataConfidenceHigh,
				Source:        "Apple Music",
			},
		),
	)

	var storedRecord TrackPlayRecord
	require.NoError(t, db.First(&storedRecord, record.ID).Error)
	require.Equal(t, int64(1777), storedRecord.ResolvedTrackID)
	require.Equal(t, int64(4675), storedRecord.AlbumID)
	require.Equal(t, "(50th Anniversary) [Remastered]", storedRecord.AlbumSubtitle)
	require.True(t, storedRecord.LibraryApplied)
}

func TestProcessTrackPlayRecordBackfillsResolvedTrackFields(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_play_record_backfill_resolved_fields")
	ctx := context.Background()

	require.NoError(t, db.Create(&Album{
		ID:         1200,
		Name:       "冀西南林路行",
		Artist:     "万能青年旅店",
		SyncStatus: 3,
	}).Error)
	require.NoError(t, db.Create(&Track{
		ID:            578,
		Artist:        "万能青年旅店",
		Album:         "冀西南林路行",
		Track:         "山雀",
		TrackNumber:   7,
		DiscNumber:    1,
		MusicBrainzID: "2d051fd1-e7cd-46e0-8a10-2804448ca0e8",
		PlayCount:     9,
		Version:       1,
	}).Error)
	require.NoError(t, db.Create(&TrackAlbum{
		TrackID:     578,
		AlbumID:     1200,
		TrackNumber: 7,
		DiscNumber:  1,
		Track:       "山雀",
	}).Error)
	require.NoError(t, db.Create(&TrackPlayRecord{
		ID:                   577,
		Artist:               "万能青年旅店",
		Album:                "冀西南林路行",
		Track:                "山雀",
		TrackNumber:          0,
		DiscNumber:           0,
		Source:               "Apple Music",
		PlayTime:             modelTestNow.Add(-time.Hour),
		ResolvedTrackID:      578,
		ResolutionStatus:     TrackPlayRecordResolutionResolved,
		ResolutionConfidence: common.TrackMetadataConfidenceAuthoritative,
		LibraryApplied:       true,
	}).Error)

	record := &TrackPlayRecord{
		ID:          578,
		Artist:      "万能青年旅店",
		Album:       "冀西南林路行",
		Track:       "山雀",
		TrackNumber: 0,
		DiscNumber:  0,
		Source:      "Apple Music",
		PlayTime:    modelTestNow,
	}
	require.NoError(t, InsertTrackPlayRecord(ctx, record))

	require.NoError(
		t, ProcessTrackPlayRecord(
			ctx,
			record.ID,
			TrackMetadata{
				Confidence: common.TrackMetadataConfidenceLow,
				Source:     "Apple Music",
			},
		),
	)

	var storedRecord TrackPlayRecord
	require.NoError(t, db.First(&storedRecord, record.ID).Error)
	require.Equal(t, int64(578), storedRecord.ResolvedTrackID)
	require.Equal(t, int8(7), storedRecord.TrackNumber)
	require.Equal(t, int8(1), storedRecord.DiscNumber)
	require.Equal(t, "2d051fd1-e7cd-46e0-8a10-2804448ca0e8", storedRecord.MusicBrainzID)
	require.Equal(t, int64(1200), storedRecord.AlbumID)
	require.True(t, storedRecord.LibraryApplied)
}

func TestProcessTrackPlayRecordDoesNotOverrideExistingResolvedPositions(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_play_record_keep_existing_positions")
	ctx := context.Background()

	require.NoError(t, db.Create(&Album{
		ID:         1300,
		Name:       "生如夏花",
		Artist:     "朴树",
		SyncStatus: 3,
	}).Error)
	require.NoError(t, db.Create(&Track{
		ID:            3313,
		Artist:        "朴树",
		Album:         "生如夏花",
		Track:         "今夜的滋味",
		TrackNumber:   8,
		DiscNumber:    1,
		MusicBrainzID: "mbid-jin-ye",
		PlayCount:     4,
		Version:       1,
	}).Error)
	require.NoError(t, db.Create(&TrackPlayRecord{
		ID:                   1330,
		Artist:               "朴树",
		Album:                "生如夏花",
		Track:                "今夜的滋味",
		TrackNumber:          5,
		DiscNumber:           2,
		Source:               "Apple Music",
		PlayTime:             modelTestNow.Add(-time.Hour),
		ResolvedTrackID:      3313,
		AlbumID:              1300,
		ResolutionStatus:     TrackPlayRecordResolutionResolved,
		ResolutionConfidence: common.TrackMetadataConfidenceMedium,
		LibraryApplied:       true,
	}).Error)

	record := &TrackPlayRecord{
		ID:          1331,
		Artist:      "朴树",
		Album:       "生如夏花",
		Track:       "今夜的滋味",
		TrackNumber: 5,
		DiscNumber:  2,
		Source:      "Apple Music",
		PlayTime:    modelTestNow,
	}
	require.NoError(t, InsertTrackPlayRecord(ctx, record))

	require.NoError(
		t, ProcessTrackPlayRecord(
			ctx,
			record.ID,
			TrackMetadata{
				Confidence: common.TrackMetadataConfidenceLow,
				Source:     "Apple Music",
			},
		),
	)

	var storedRecord TrackPlayRecord
	require.NoError(t, db.First(&storedRecord, record.ID).Error)
	require.Equal(t, int64(3313), storedRecord.ResolvedTrackID)
	require.Equal(t, int8(5), storedRecord.TrackNumber)
	require.Equal(t, int8(2), storedRecord.DiscNumber)
	require.Equal(t, int64(0), storedRecord.AlbumID)
	require.False(t, storedRecord.LibraryApplied)
	require.Equal(t, "mbid-jin-ye", storedRecord.MusicBrainzID)
}

func TestSetAppleMusicFavoriteUnresolvedCreatesPendingEventOnly(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_resolution_favorite_unresolved")
	ctx := context.Background()

	require.NoError(
		t, SetAppleMusicFavorite(
			SetFavoriteParams{
				Ctx:        ctx,
				Artist:     "ELTON JOHN",
				Album:      "Too Low For Zero (Bonus Track Version)",
				Track:      "I'm Still Standing",
				IsFavorite: true,
				TrackMetadata: TrackMetadata{
					TrackNumber: 0,
					DiscNumber:  0,
					Confidence:  common.TrackMetadataConfidenceLow,
					Source:      "Apple Music",
				},
			},
		),
	)

	var trackCount int64
	require.NoError(t, db.Model(&Track{}).Count(&trackCount).Error)
	require.Equal(t, int64(0), trackCount)

	var events []TrackFavoriteEvent
	require.NoError(t, db.Order("id ASC").Find(&events).Error)
	require.Len(t, events, 1)
	require.Equal(t, TrackFavoriteEventResolutionUnresolved, events[0].ResolutionStatus)
	require.False(t, events[0].Applied)
	require.Equal(t, int64(0), events[0].ResolvedTrackID)
}

func TestSetAppleMusicFavoriteUnresolvedIsIdempotentForOpenEvent(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_resolution_favorite_unresolved_idempotent")
	ctx := context.Background()
	params := SetFavoriteParams{
		Ctx:        ctx,
		Artist:     "Yorushika",
		Album:      "second person",
		Track:      "Forget it",
		IsFavorite: true,
		TrackMetadata: TrackMetadata{
			TrackNumber: 9,
			DiscNumber:  1,
			Confidence:  common.TrackMetadataConfidenceMedium,
			Source:      "Apple Music",
			UniqueID:    "40699",
		},
	}

	require.NoError(t, SetAppleMusicFavorite(params))
	require.NoError(t, SetAppleMusicFavorite(params))
	require.NoError(t, SetAppleMusicFavorite(params))

	var count int64
	require.NoError(
		t,
		db.Model(&TrackFavoriteEvent{}).
			Where("source = ? AND artist = ? AND album = ? AND track = ? AND provider_favorite = ?",
				TrackFavoriteEventSourceAppleMusic, "Yorushika", "second person", "Forget it", true).
			Count(&count).Error,
	)
	require.Equal(t, int64(1), count)
}

func TestSetAppleMusicFavoriteResolvedNoopWhenStateAlreadyApplied(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_resolution_favorite_am_resolved_noop")
	ctx := context.Background()

	require.NoError(t, db.Create(&Track{
		ID:              410,
		Artist:          "崔健",
		Album:           "飞狗",
		Track:           "继续",
		TrackNumber:     8,
		DiscNumber:      1,
		IsAppleMusicFav: true,
		Version:         1,
	}).Error)

	params := SetFavoriteParams{
		Ctx:        ctx,
		Artist:     "崔健",
		Album:      "飞狗",
		Track:      "继续",
		IsFavorite: true,
		TrackMetadata: TrackMetadata{
			TrackNumber: 8,
			DiscNumber:  1,
			Source:      "Apple Music",
			UniqueID:    "37490",
		},
	}

	require.NoError(t, SetAppleMusicFavorite(params))
	require.NoError(t, SetAppleMusicFavorite(params))

	var count int64
	require.NoError(
		t,
		db.Model(&TrackFavoriteEvent{}).
			Where("source = ? AND artist = ? AND album = ? AND track = ?",
				TrackFavoriteEventSourceAppleMusic, "崔健", "飞狗", "继续").
			Count(&count).Error,
	)
	require.Equal(t, int64(0), count)
}

func TestProcessTrackPlayRecordResolvesPendingFavoriteEvent(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_resolution_favorite_apply")
	ctx := context.Background()

	require.NoError(
		t, db.Create(&Track{
			ID:          300,
			Artist:      "Radiohead",
			Album:       "OK Computer",
			Track:       "No Surprises",
			TrackNumber: 10,
			DiscNumber:  1,
			PlayCount:   0,
			Version:     1,
		}).Error,
	)
	require.NoError(
		t, db.Create(&TrackFavoriteEvent{
			Source:           "Apple Music",
			ProviderFavorite: true,
			Artist:           "Radiohead",
			Album:            "OK Computer",
			Track:            "No Surprises",
			TrackNumber:      0,
			DiscNumber:       0,
			ResolutionStatus: TrackFavoriteEventResolutionUnresolved,
			Applied:          false,
		}).Error,
	)
	require.NoError(
		t, db.Create(&TrackPlayRecord{
			ID:               9001,
			Artist:           "Radiohead",
			Album:            "OK Computer",
			Track:            "No Surprises",
			TrackNumber:      10,
			DiscNumber:       1,
			Source:           "Audirvana",
			PlayTime:         time.Now(),
			ResolutionStatus: TrackPlayRecordResolutionPending,
		}).Error,
	)

	require.NoError(
		t, ProcessTrackPlayRecord(
			ctx,
			9001,
			TrackMetadata{
				TrackNumber: 10,
				DiscNumber:  1,
				Confidence:  common.TrackMetadataConfidenceHigh,
				Source:      "Audirvana",
				PlayerType:  "Audirvana",
			},
		),
	)

	var track Track
	require.NoError(t, db.Where("id = ?", int64(300)).First(&track).Error)
	require.True(t, track.IsAppleMusicFav)

	var event TrackFavoriteEvent
	require.NoError(
		t,
		db.Where("artist = ? AND album = ? AND track = ?", "Radiohead", "OK Computer", "No Surprises").First(&event).Error,
	)
	require.True(t, event.Applied)
	require.Equal(t, TrackFavoriteEventResolutionResolved, event.ResolutionStatus)
	require.Equal(t, int64(300), event.ResolvedTrackID)
}

func TestSetLastFmFavoriteUnresolvedCreatesPendingEventOnly(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_resolution_lastfm_favorite_unresolved")
	ctx := context.Background()

	require.NoError(
		t, SetLastFmFavorite(
			SetFavoriteParams{
				Ctx:        ctx,
				Artist:     "ELTON JOHN",
				Album:      "Too Low For Zero (Bonus Track Version)",
				Track:      "I'm Still Standing",
				IsFavorite: true,
				TrackMetadata: TrackMetadata{
					TrackNumber: 0,
					DiscNumber:  0,
					Confidence:  common.TrackMetadataConfidenceLow,
					Source:      "Last.fm",
				},
			},
		),
	)

	var trackCount int64
	require.NoError(t, db.Model(&Track{}).Count(&trackCount).Error)
	require.Equal(t, int64(0), trackCount)

	var events []TrackFavoriteEvent
	require.NoError(t, db.Order("id ASC").Find(&events).Error)
	require.Len(t, events, 1)
	require.Equal(t, TrackFavoriteEventSourceLastFm, events[0].Source)
	require.Equal(t, TrackFavoriteEventResolutionUnresolved, events[0].ResolutionStatus)
	require.False(t, events[0].Applied)
	require.Equal(t, int64(0), events[0].ResolvedTrackID)
}

func TestSetLastFmFavoriteResolvedNoopWhenStateAlreadyApplied(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_resolution_favorite_lastfm_resolved_noop")
	ctx := context.Background()

	require.NoError(t, db.Create(&Track{
		ID:          411,
		Artist:      "崔健",
		Album:       "飞狗",
		Track:       "继续",
		TrackNumber: 8,
		DiscNumber:  1,
		IsLastFmFav: true,
		Version:     1,
	}).Error)

	params := SetFavoriteParams{
		Ctx:        ctx,
		Artist:     "崔健",
		Album:      "飞狗",
		Track:      "继续",
		IsFavorite: true,
		TrackMetadata: TrackMetadata{
			TrackNumber: 8,
			DiscNumber:  1,
			Source:      "Last.fm",
			UniqueID:    "37490",
		},
	}

	require.NoError(t, SetLastFmFavorite(params))
	require.NoError(t, SetLastFmFavorite(params))

	var count int64
	require.NoError(
		t,
		db.Model(&TrackFavoriteEvent{}).
			Where("source = ? AND artist = ? AND album = ? AND track = ?",
				TrackFavoriteEventSourceLastFm, "崔健", "飞狗", "继续").
			Count(&count).Error,
	)
	require.Equal(t, int64(0), count)
}

func TestProcessTrackPlayRecordResolvesPendingLastFmFavoriteEvent(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_resolution_lastfm_favorite_apply")
	ctx := context.Background()

	require.NoError(
		t, db.Create(&Track{
			ID:          301,
			Artist:      "Radiohead",
			Album:       "OK Computer",
			Track:       "No Surprises",
			TrackNumber: 10,
			DiscNumber:  1,
			PlayCount:   0,
			Version:     1,
		}).Error,
	)
	require.NoError(
		t, db.Create(&TrackFavoriteEvent{
			Source:           TrackFavoriteEventSourceLastFm,
			ProviderFavorite: true,
			Artist:           "Radiohead",
			Album:            "OK Computer",
			Track:            "No Surprises",
			TrackNumber:      0,
			DiscNumber:       0,
			ResolutionStatus: TrackFavoriteEventResolutionUnresolved,
			Applied:          false,
		}).Error,
	)
	require.NoError(
		t, db.Create(&TrackPlayRecord{
			ID:               9002,
			Artist:           "Radiohead",
			Album:            "OK Computer",
			Track:            "No Surprises",
			TrackNumber:      10,
			DiscNumber:       1,
			Source:           "Audirvana",
			PlayTime:         time.Now(),
			ResolutionStatus: TrackPlayRecordResolutionPending,
		}).Error,
	)

	require.NoError(
		t, ProcessTrackPlayRecord(
			ctx,
			9002,
			TrackMetadata{
				TrackNumber: 10,
				DiscNumber:  1,
				Confidence:  common.TrackMetadataConfidenceHigh,
				Source:      "Audirvana",
				PlayerType:  "Audirvana",
			},
		),
	)

	var track Track
	require.NoError(t, db.Where("id = ?", int64(301)).First(&track).Error)
	require.True(t, track.IsLastFmFav)

	var event TrackFavoriteEvent
	require.NoError(
		t,
		db.Where("source = ? AND artist = ? AND album = ? AND track = ?",
			TrackFavoriteEventSourceLastFm, "Radiohead", "OK Computer", "No Surprises").First(&event).Error,
	)
	require.True(t, event.Applied)
	require.Equal(t, TrackFavoriteEventResolutionResolved, event.ResolutionStatus)
	require.Equal(t, int64(301), event.ResolvedTrackID)
}

func TestReplayTrackPlayRecordsDryRunDoesNotModifyData(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_play_record_replay_dry_run")
	ctx := context.Background()

	record := &TrackPlayRecord{
		Artist:           "Radiohead",
		Album:            "Kid A",
		Track:            "Everything in Its Right Place",
		TrackNumber:      1,
		DiscNumber:       1,
		Duration:         251,
		Source:           "Audirvana",
		PlayTime:         modelTestNow,
		ResolutionStatus: TrackPlayRecordResolutionPending,
		LibraryApplied:   false,
	}
	require.NoError(t, InsertTrackPlayRecord(ctx, record))

	report, err := ReplayTrackPlayRecords(
		ReplayTrackPlayRecordsParams{
			Ctx:    ctx,
			Limit:  10,
			DryRun: true,
		},
	)
	require.NoError(t, err)
	require.Len(t, report.Results, 1)

	var stored TrackPlayRecord
	require.NoError(t, db.First(&stored, record.ID).Error)
	require.False(t, stored.LibraryApplied)
	require.Zero(t, stored.ResolvedTrackID)
	require.Equal(t, TrackPlayRecordResolutionPending, stored.ResolutionStatus)
}

func TestReplayTrackPlayRecordsAppliesPendingRecord(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_play_record_replay_apply")
	ctx := context.Background()

	record := &TrackPlayRecord{
		Artist:           "Radiohead",
		Album:            "Kid A",
		Track:            "Everything in Its Right Place",
		TrackNumber:      1,
		DiscNumber:       1,
		Duration:         251,
		Source:           "Audirvana",
		PlayTime:         modelTestNow,
		ResolutionStatus: TrackPlayRecordResolutionPending,
		LibraryApplied:   false,
	}
	require.NoError(t, InsertTrackPlayRecord(ctx, record))

	report, err := ReplayTrackPlayRecords(
		ReplayTrackPlayRecordsParams{
			Ctx:    ctx,
			Limit:  10,
			DryRun: false,
		},
	)
	require.NoError(t, err)
	require.Len(t, report.Results, 1)
	require.Equal(t, TrackPlayRecordResolutionResolved, report.Results[0].AfterStatus)
	require.True(t, report.Results[0].AfterApplied)
	require.NotZero(t, report.Results[0].ResolvedTrackID)

	var stored TrackPlayRecord
	require.NoError(t, db.First(&stored, record.ID).Error)
	require.True(t, stored.LibraryApplied)
	require.Equal(t, TrackPlayRecordResolutionResolved, stored.ResolutionStatus)
	require.NotZero(t, stored.ResolvedTrackID)
}

func TestReplayTrackPlayRecordsFiltersByRecordID(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_play_record_replay_filter_by_id")
	ctx := context.Background()

	first := &TrackPlayRecord{
		Artist:           "Radiohead",
		Album:            "Kid A",
		Track:            "Everything in Its Right Place",
		TrackNumber:      1,
		DiscNumber:       1,
		Duration:         251,
		Source:           "Audirvana",
		PlayTime:         modelTestNow,
		ResolutionStatus: TrackPlayRecordResolutionPending,
		LibraryApplied:   false,
	}
	second := &TrackPlayRecord{
		Artist:           "Radiohead",
		Album:            "Kid A",
		Track:            "Kid A",
		TrackNumber:      2,
		DiscNumber:       1,
		Duration:         285,
		Source:           "Audirvana",
		PlayTime:         modelTestNow.Add(time.Minute),
		ResolutionStatus: TrackPlayRecordResolutionPending,
		LibraryApplied:   false,
	}
	require.NoError(t, InsertTrackPlayRecord(ctx, first))
	require.NoError(t, InsertTrackPlayRecord(ctx, second))

	report, err := ReplayTrackPlayRecords(
		ReplayTrackPlayRecordsParams{
			Ctx:       ctx,
			RecordIDs: []int64{second.ID},
			DryRun:    true,
		},
	)
	require.NoError(t, err)
	require.Len(t, report.Results, 1)
	require.Equal(t, second.ID, report.Results[0].ID)

	var storedFirst TrackPlayRecord
	require.NoError(t, db.First(&storedFirst, first.ID).Error)
	require.False(t, storedFirst.LibraryApplied)
	require.Equal(t, TrackPlayRecordResolutionPending, storedFirst.ResolutionStatus)
}

func TestReplayTrackPlayRecordsDefaultIgnoresArchivedHistoricalRows(t *testing.T) {
	db := newTrackResolutionTestDB(t, "track_play_record_replay_ignore_archived")
	ctx := context.Background()

	archived := &TrackPlayRecord{
		Artist:           "Radiohead",
		Album:            "The Bends",
		Track:            "Planet Telex",
		TrackNumber:      1,
		DiscNumber:       1,
		Duration:         259,
		Source:           "Audirvana",
		PlayTime:         modelTestNow.Add(-time.Hour),
		ResolutionStatus: TrackPlayRecordResolutionUnresolved,
		LibraryApplied:   true,
	}
	active := &TrackPlayRecord{
		Artist:           "Radiohead",
		Album:            "Kid A",
		Track:            "Everything in Its Right Place",
		TrackNumber:      1,
		DiscNumber:       1,
		Duration:         251,
		Source:           "Audirvana",
		PlayTime:         modelTestNow,
		ResolutionStatus: TrackPlayRecordResolutionPending,
		LibraryApplied:   false,
	}
	require.NoError(t, InsertTrackPlayRecord(ctx, archived))
	require.NoError(t, InsertTrackPlayRecord(ctx, active))

	report, err := ReplayTrackPlayRecords(
		ReplayTrackPlayRecordsParams{
			Ctx:    ctx,
			Limit:  10,
			DryRun: true,
		},
	)
	require.NoError(t, err)
	require.Len(t, report.Results, 1)
	require.Equal(t, active.ID, report.Results[0].ID)

	var storedArchived TrackPlayRecord
	require.NoError(t, db.First(&storedArchived, archived.ID).Error)
	require.True(t, storedArchived.LibraryApplied)
	require.Equal(t, TrackPlayRecordResolutionUnresolved, storedArchived.ResolutionStatus)
}
