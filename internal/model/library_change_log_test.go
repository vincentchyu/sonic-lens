package model

import (
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestAppendLibraryChangeTxInsertsChangeLog(t *testing.T) {
	_, mock := newModelTestDB(t)

	mock.ExpectExec(
		regexp.QuoteMeta(
			"INSERT INTO `library_change_log` (`entity_type`,`entity_id`,`operation`) VALUES (?,?,?)",
		),
	).
		WithArgs(LibraryEntityTrack, int64(3022), LibraryOpUpsert).
		WillReturnResult(sqlmock.NewResult(18, 1))

	require.NoError(t, appendLibraryChangeTx(GetDB(), LibraryEntityTrack, 3022, LibraryOpUpsert))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAppendLibraryChangeTxSkipsInvalidEntityID(t *testing.T) {
	_, mock := newModelTestDB(t)

	require.NoError(t, appendLibraryChangeTx(GetDB(), LibraryEntityAlbum, 0, LibraryOpUpsert))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAlbumHooksAppendLibraryChangeLog(t *testing.T) {
	_, mock := newModelTestDB(t)

	mock.ExpectExec(
		regexp.QuoteMeta(
			"INSERT INTO `library_change_log` (`entity_type`,`entity_id`,`operation`) VALUES (?,?,?)",
		),
	).
		WithArgs(LibraryEntityAlbum, int64(4097), LibraryOpUpsert).
		WillReturnResult(sqlmock.NewResult(19, 1))

	album := &Album{ID: 4097}
	require.NoError(t, album.AfterUpdate(GetDB()))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTrackDeleteHookAppendsLibraryChangeLog(t *testing.T) {
	_, mock := newModelTestDB(t)

	mock.ExpectExec(
		regexp.QuoteMeta(
			"INSERT INTO `library_change_log` (`entity_type`,`entity_id`,`operation`) VALUES (?,?,?)",
		),
	).
		WithArgs(LibraryEntityTrack, int64(3022), LibraryOpDelete).
		WillReturnResult(sqlmock.NewResult(20, 1))

	track := &Track{ID: 3022}
	require.NoError(t, track.AfterDelete(GetDB()))
	require.NoError(t, mock.ExpectationsWereMet())
}
