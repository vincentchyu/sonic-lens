package model

import (
	"context"
	"errors"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestInTxRollsBackOnError(t *testing.T) {
	_, mock := newModelTestDB(t)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE album SET sync_status = 1 WHERE id = 9")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectRollback()

	err := InTx(
		ctx, func(tx *gorm.DB) error {
			if err := tx.Exec("UPDATE album SET sync_status = 1 WHERE id = 9").Error; err != nil {
				return err
			}
			return errors.New("rollback")
		},
	)
	require.EqualError(t, err, "rollback")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMusicBrainzDAOHelpersWorkInsideTransaction(t *testing.T) {
	_, mock := newModelTestDB(t)
	ctx := context.Background()

	releaseRows := sqlmock.NewRows(
		[]string{"id", "mbid", "album_id", "name", "json_data", "created_at", "updated_at"},
	).AddRow(8, "mbid-1", 7, "The Wall", `{"old":true}`, modelTestNow, modelTestNow)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `album_release_mb` WHERE album_id = ?")).
		WithArgs(int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `track_album` SET `mb_recording_id`=?,`updated_at`=? WHERE album_id = ?")).
		WithArgs("", modelTestNow, int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT * FROM `release_mb` WHERE album_id = ? AND mbid = ? ORDER BY `release_mb`.`id` LIMIT ?",
	)).
		WithArgs(int64(7), "mbid-1", 1).
		WillReturnRows(releaseRows)
	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE `release_mb` SET `mbid`=?,`album_id`=?,`name`=?,`json_data`=?,`created_at`=?,`updated_at`=? WHERE `id` = ?",
	)).
		WithArgs("mbid-1", int64(7), "The Wall", `{"fresh":true}`, modelTestNow, modelTestNow, int64(8)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(
		t, InTx(
			ctx, func(tx *gorm.DB) error {
				if err := DeleteAlbumReleaseMBByAlbumIDTx(tx, 7); err != nil {
					return err
				}
				if err := ClearTrackAlbumMBRecordingIDByAlbumIDTx(tx, 7); err != nil {
					return err
				}
				updated, err := UpdateReleaseMBJSONDataTx(tx, 7, "mbid-1", `{"fresh":true}`)
				if err != nil {
					return err
				}
				require.True(t, updated)
				return nil
			},
		),
	)
	require.NoError(t, mock.ExpectationsWereMet())
}
