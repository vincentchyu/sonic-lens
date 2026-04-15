package model

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTrackAlbumTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", name)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(
		t,
		db.Exec(
			`
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
		`,
		).Error,
	)
	require.NoError(
		t,
		db.Exec(
			`
			CREATE TABLE album (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				name_subtitle TEXT,
				title_metadata TEXT,
				artist TEXT NOT NULL,
				original_release_date TEXT,
				cover_art_url TEXT,
				cover_art_mime TEXT,
				cover_art_object_key TEXT,
				sync_status INTEGER DEFAULT 0
			)
		`,
		).Error,
	)

	return db
}

func TestUpsertTrackAlbumTxReclaimsPositionalPlaceholder(t *testing.T) {
	db := newTrackAlbumTestDB(t, "track_album_reclaim_placeholder")

	existing := &TrackAlbum{
		TrackID:                5050,
		AlbumID:                4129,
		TrackNumber:            3,
		DiscNumber:             1,
		Track:                  "我講給你一個笑話",
		MusicBrainzRecordingID: "75abcdbc-d0c5-4907-9209-18f24cc80228",
	}
	placeholder := &TrackAlbum{
		TrackID:                0,
		AlbumID:                4129,
		TrackNumber:            3,
		DiscNumber:             2,
		Track:                  "我講給你一個笑話(南京場)",
		MusicBrainzRecordingID: "64bb0ec1-1a74-440b-9a87-74329a2a16a4",
	}

	require.NoError(t, db.Create(existing).Error)
	require.NoError(t, db.Create(placeholder).Error)

	require.NoError(
		t, UpsertTrackAlbumTx(
			db, &TrackAlbum{
				TrackID:                5050,
				AlbumID:                4129,
				TrackNumber:            3,
				DiscNumber:             2,
				Track:                  "我講給你一個笑話(南京場)",
				MusicBrainzRecordingID: "64bb0ec1-1a74-440b-9a87-74329a2a16a4",
			},
			false,
		),
	)

	var rows []TrackAlbum
	require.NoError(t, db.Where("album_id = ?", 4129).Order("id ASC").Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, int64(5050), rows[0].TrackID)
	require.Equal(t, int8(2), rows[0].DiscNumber)
	require.Equal(t, int8(3), rows[0].TrackNumber)
	require.Equal(t, "我講給你一個笑話(南京場)", rows[0].Track)
	require.Equal(t, "64bb0ec1-1a74-440b-9a87-74329a2a16a4", rows[0].MusicBrainzRecordingID)
}

func TestUpsertTrackAlbumTxAllowsInPlaceSaveWhenDuplicateRealRowExists(t *testing.T) {
	db := newTrackAlbumTestDB(t, "track_album_duplicate_real_row")

	require.NoError(
		t, db.Create(
			&TrackAlbum{
				TrackID:     5040,
				AlbumID:     4129,
				TrackNumber: 3,
				DiscNumber:  1,
				Track:       "我講給你一個笑話",
			},
		).Error,
	)
	require.NoError(
		t, db.Create(
			&TrackAlbum{
				TrackID:     5050,
				AlbumID:     4129,
				TrackNumber: 3,
				DiscNumber:  1,
				Track:       "我講給你一個笑話",
			},
		).Error,
	)

	require.NoError(
		t, UpsertTrackAlbumTx(
			db, &TrackAlbum{
				TrackID:                5040,
				AlbumID:                4129,
				TrackNumber:            3,
				DiscNumber:             1,
				Track:                  "我講給你一個笑話",
				MusicBrainzRecordingID: "75abcdbc-d0c5-4907-9209-18f24cc80228",
			},
			false,
		),
	)

	var row TrackAlbum
	require.NoError(t, db.Where("track_id = ? AND album_id = ?", 5040, 4129).First(&row).Error)
	require.Equal(t, int8(1), row.DiscNumber)
	require.Equal(t, int8(3), row.TrackNumber)
	require.Equal(t, "75abcdbc-d0c5-4907-9209-18f24cc80228", row.MusicBrainzRecordingID)
}

func TestUpsertTrackAlbumTxSkipsMutationWhenAlbumLayoutLocked(t *testing.T) {
	db := newTrackAlbumTestDB(t, "track_album_layout_locked")

	require.NoError(
		t,
		db.Exec("INSERT INTO album(id, name, artist, sync_status) VALUES(?, ?, ?, ?)", 6001, "Kid A", "Radiohead", 3).
			Error,
	)
	require.NoError(
		t,
		db.Create(
			&TrackAlbum{
				TrackID:                701,
				AlbumID:                6001,
				TrackNumber:            1,
				DiscNumber:             1,
				Track:                  "Everything in Its Right Place",
				MusicBrainzRecordingID: "mbid-old",
			},
		).Error,
	)

	require.NoError(
		t, UpsertTrackAlbumTx(
			db, &TrackAlbum{
				TrackID:                701,
				AlbumID:                6001,
				TrackNumber:            2,
				DiscNumber:             1,
				Track:                  "Everything in Its Right Place",
				MusicBrainzRecordingID: "mbid-new",
			},
			false,
		),
	)

	var row TrackAlbum
	require.NoError(t, db.Where("track_id = ? AND album_id = ?", 701, 6001).First(&row).Error)
	require.Equal(t, int8(1), row.TrackNumber)
	require.Equal(t, int8(1), row.DiscNumber)
	require.Equal(t, "mbid-old", row.MusicBrainzRecordingID)
}
