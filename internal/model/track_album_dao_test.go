package model

import (
	"context"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestGetTrackAlbumsByAlbumTx(t *testing.T) {
	_, mock := newModelTestDB(t)

	rows := sqlmock.NewRows(
		[]string{"id", "track_id", "album_id", "track_number", "disc_number", "mb_recording_id", "track", "created_at", "updated_at"},
	).AddRow(1, 101, 51, 1, 1, "mbid-1", "In the Flesh?", modelTestNow, modelTestNow).
		AddRow(2, 102, 51, 2, 1, "mbid-2", "The Thin Ice", modelTestNow, modelTestNow)

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT * FROM `track_album` WHERE album_id = ? ORDER BY disc_number ASC, track_number ASC, id ASC",
	)).
		WithArgs(int64(51)).
		WillReturnRows(rows)

	results, err := GetTrackAlbumsByAlbumTx(GetDB(), 51)
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, int64(101), results[0].TrackID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTrackAlbumByTrackID(t *testing.T) {
	_, mock := newModelTestDB(t)
	ctx := context.Background()

	rows := sqlmock.NewRows(
		[]string{"id", "track_id", "album_id", "track_number", "disc_number", "mb_recording_id", "track", "created_at", "updated_at"},
	).AddRow(3, 105, 55, 5, 1, "mbid-55", "Goodbye Blue Sky", modelTestNow, modelTestNow)

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT * FROM `track_album` WHERE track_id = ? ORDER BY disc_number ASC, track_number ASC, id ASC,`track_album`.`id` LIMIT ?",
	)).
		WithArgs(int64(105), 1).
		WillReturnRows(rows)

	result, err := GetTrackAlbumByTrackID(ctx, 105)
	require.NoError(t, err)
	require.Equal(t, int64(55), result.AlbumID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveTrackAlbumTx(t *testing.T) {
	_, mock := newModelTestDB(t)

	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE `track_album` SET `track_id`=?,`album_id`=?,`track_number`=?,`disc_number`=?,`mb_recording_id`=?,`track`=?,`created_at`=?,`updated_at`=? WHERE `id` = ?",
	)).
		WithArgs(int64(103), int64(52), int8(3), int8(2), "mbid-3", "Another Brick in the Wall", modelTestNow, modelTestNow, int64(3)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(
		t, SaveTrackAlbumTx(
			GetDB(),
			&TrackAlbum{
				ID:                     3,
				TrackID:                103,
				AlbumID:                52,
				TrackNumber:            3,
				DiscNumber:             2,
				MusicBrainzRecordingID: "mbid-3",
				Track:                  "Another Brick in the Wall",
				CreatedAt:              modelTestNow,
			},
		),
	)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCountTrackAlbumsByAlbumAndRecordingIDTx(t *testing.T) {
	_, mock := newModelTestDB(t)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(2)
	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT count(*) FROM `track_album` WHERE album_id = ? AND mb_recording_id = ?",
	)).
		WithArgs(int64(53), "mbid-53").
		WillReturnRows(rows)

	count, err := CountTrackAlbumsByAlbumAndRecordingIDTx(GetDB(), 53, "mbid-53")
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTrackAlbumTx(t *testing.T) {
	_, mock := newModelTestDB(t)

	mock.ExpectExec(regexp.QuoteMeta(
		"INSERT INTO `track_album` (`track_id`,`album_id`,`track_number`,`disc_number`,`mb_recording_id`,`track`) VALUES (?,?,?,?,?,?)",
	)).
		WithArgs(int64(104), int64(54), int8(4), int8(1), "mbid-54", "Mother").
		WillReturnResult(sqlmock.NewResult(10, 1))

	require.NoError(
		t, CreateTrackAlbumTx(
			GetDB(),
			&TrackAlbum{
				TrackID:                104,
				AlbumID:                54,
				TrackNumber:            4,
				DiscNumber:             1,
				MusicBrainzRecordingID: "mbid-54",
				Track:                  "Mother",
			},
		),
	)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteTrackAlbumLink(t *testing.T) {
	_, mock := newModelTestDB(t)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `track_album` WHERE track_id = ? AND album_id = ?")).
		WithArgs(int64(106), int64(56)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, DeleteTrackAlbumLink(ctx, 106, 56))
	require.NoError(t, mock.ExpectationsWereMet())
}
