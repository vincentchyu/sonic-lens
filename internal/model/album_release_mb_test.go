package model

import (
	"context"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestDeleteAlbumReleaseMBByAlbumIDTx(t *testing.T) {
	_, mock := newModelTestDB(t)

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `album_release_mb` WHERE album_id = ?")).
		WithArgs(int64(21)).
		WillReturnResult(sqlmock.NewResult(0, 2))

	require.NoError(t, DeleteAlbumReleaseMBByAlbumIDTx(GetDB(), 21))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteAlbumReleaseMBByAlbumIDUsesContextDB(t *testing.T) {
	_, mock := newModelTestDB(t)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `album_release_mb` WHERE album_id = ?")).
		WithArgs(int64(22)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, DeleteAlbumReleaseMBByAlbumID(ctx, 22))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLinkAlbumToMBIDTx(t *testing.T) {
	_, mock := newModelTestDB(t)

	rows := sqlmock.NewRows([]string{"id", "album_id", "release_mb_id", "mbid", "confirmed", "created_at"})

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT * FROM `album_release_mb` WHERE album_id = ? AND release_mb_id = ? ORDER BY `album_release_mb`.`id` LIMIT ?",
	)).
		WithArgs(int64(30), int64(40), 1).
		WillReturnRows(rows)
	mock.ExpectExec(regexp.QuoteMeta(
		"INSERT INTO `album_release_mb` (`album_id`,`release_mb_id`,`mbid`,`confirmed`) VALUES (?,?,?,?)",
	)).
		WithArgs(int64(30), int64(40), "mbid-30", true).
		WillReturnResult(sqlmock.NewResult(5, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `album` SET `sync_status`=?,`updated_at`=? WHERE id = ?")).
		WithArgs(2, modelTestNow, int64(30)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, LinkAlbumToMBIDTx(GetDB(), 30, 40, "mbid-30"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLinkAlbumToMBIDUsesContextDB(t *testing.T) {
	_, mock := newModelTestDB(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "album_id", "release_mb_id", "mbid", "confirmed", "created_at"})

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT * FROM `album_release_mb` WHERE album_id = ? AND release_mb_id = ? ORDER BY `album_release_mb`.`id` LIMIT ?",
	)).
		WithArgs(int64(31), int64(41), 1).
		WillReturnRows(rows)
	mock.ExpectExec(regexp.QuoteMeta(
		"INSERT INTO `album_release_mb` (`album_id`,`release_mb_id`,`mbid`,`confirmed`) VALUES (?,?,?,?)",
	)).
		WithArgs(int64(31), int64(41), "mbid-31", true).
		WillReturnResult(sqlmock.NewResult(7, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `album` SET `sync_status`=?,`updated_at`=? WHERE id = ?")).
		WithArgs(2, modelTestNow, int64(31)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, LinkAlbumToMBID(ctx, 31, 41, "mbid-31"))
	require.NoError(t, mock.ExpectationsWereMet())
}
