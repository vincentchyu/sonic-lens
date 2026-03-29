package model

import (
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetTrackByIDTx(t *testing.T) {
	_, mock := newModelTestDB(t)

	rows := sqlmock.NewRows(
		[]string{
			"id", "artist", "album", "track", "play_count", "is_apple_music_fav", "is_last_fm_fav", "version",
			"album_artist", "track_number", "disc_number", "duration", "genre", "composer", "release_date",
			"music_brainz_id", "source", "bundle_id", "unique_id", "created_at", "updated_at",
		},
	).AddRow(
		88, "Pink Floyd", "The Wall", "Comfortably Numb", 10, false, false, 1,
		"Pink Floyd", 6, 2, 384, "Rock", "", "1979-11-30",
		"mbid-88", "Roon", "", "", modelTestNow, modelTestNow,
	)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `track` WHERE `track`.`id` = ? ORDER BY `track`.`id` LIMIT ?")).
		WithArgs(int64(88), 1).
		WillReturnRows(rows)

	track, err := GetTrackByIDTx(GetDB(), 88)
	require.NoError(t, err)
	require.Equal(t, int64(88), track.ID)
	require.Equal(t, "Comfortably Numb", track.Track)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTrackByIDTxPropagatesNotFound(t *testing.T) {
	_, mock := newModelTestDB(t)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `track` WHERE `track`.`id` = ? ORDER BY `track`.`id` LIMIT ?")).
		WithArgs(int64(89), 1).
		WillReturnRows(sqlmock.NewRows(
			[]string{
				"id", "artist", "album", "track", "play_count", "is_apple_music_fav", "is_last_fm_fav", "version",
				"album_artist", "track_number", "disc_number", "duration", "genre", "composer", "release_date",
				"music_brainz_id", "source", "bundle_id", "unique_id", "created_at", "updated_at",
			},
		))

	_, err := GetTrackByIDTx(GetDB(), 89)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateTrackMusicBrainzPositionTx(t *testing.T) {
	_, mock := newModelTestDB(t)

	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE `track` SET `disc_number`=?,`music_brainz_id`=?,`track_number`=?,`updated_at`=? WHERE id = ?",
	)).
		WithArgs(int8(2), "mbid-track", int8(6), modelTestNow, int64(90)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(
		"INSERT INTO `library_change_log` (`entity_type`,`entity_id`,`operation`) VALUES (?,?,?)",
	)).
		WithArgs("track", int64(90), "upsert").
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, UpdateTrackMusicBrainzPositionTx(GetDB(), 90, "mbid-track", 2, 6))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateTrackMusicBrainzMetadataTx(t *testing.T) {
	_, mock := newModelTestDB(t)

	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE `track` SET `disc_number`=?,`duration`=?,`music_brainz_id`=?,`track_number`=?,`updated_at`=? WHERE id = ?",
	)).
		WithArgs(int8(1), int64(253), "mbid-track-meta", int8(7), modelTestNow, int64(91)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(
		"INSERT INTO `library_change_log` (`entity_type`,`entity_id`,`operation`) VALUES (?,?,?)",
	)).
		WithArgs("track", int64(91), "upsert").
		WillReturnResult(sqlmock.NewResult(2, 1))

	require.NoError(t, UpdateTrackMusicBrainzMetadataTx(GetDB(), 91, "mbid-track-meta", 1, 7, 253))
	require.NoError(t, mock.ExpectationsWereMet())
}
