package model

import (
	"context"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/common"
)

func TestGetOrCreateAlbumTxPrefersCuratedAlbumWhenReleaseDateMissing(t *testing.T) {
	_, mock := newModelTestDB(t)

	incoming := &Album{
		Name:   "Kind of Blue",
		Artist: "Miles Davis",
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT * FROM `album` WHERE artist = ? AND name = ? AND release_date = ? AND COALESCE(name_subtitle, '') = ? ORDER BY `album`.`id` LIMIT ?"),
	).
		WithArgs("Miles Davis", "Kind of Blue", "", "", 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "artist", "release_date", "original_release_date", "sync_status",
		}))

	mock.ExpectQuery(
		regexp.QuoteMeta(
			"SELECT * FROM `album` WHERE artist = ? AND name = ? AND sync_status = ? AND COALESCE(name_subtitle, '') = ? ORDER BY CASE WHEN release_date = '' OR release_date IS NULL THEN 1 ELSE 0 END ASC, id ASC,`album`.`id` LIMIT ?",
		),
	).
		WithArgs("Miles Davis", "Kind of Blue", 3, "", 1).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id", "name", "artist", "release_date", "original_release_date", "sync_status", "created_at", "updated_at",
			}).AddRow(21, "Kind of Blue", "Miles Davis", "1959-08-17", "1959-08-17", 3, modelTestNow, modelTestNow),
		)

	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE `album` SET `name`=?,`name_subtitle`=?,`artist`=?,`release_date`=?,`original_release_date`=?,`genre`=?,`country`=?,`status`=?,`packaging`=?,`barcode`=?,`total_discs`=?,`disc_infos`=?,`sync_status`=?,`cover_art_url`=?,`cover_art_mime`=?,`cover_art_object_key`=?,`created_at`=?,`updated_at`=? WHERE `id` = ?",
	)).
		WithArgs(
			"Kind of Blue", "", "Miles Davis", "1959-08-17", "1959-08-17", "", "", "", "", "", 0, "", 3, "", "", "",
			modelTestNow, modelTestNow, int64(21),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(
		regexp.QuoteMeta("INSERT INTO `library_change_log` (`entity_type`,`entity_id`,`operation`) VALUES (?,?,?)"),
	).
		WithArgs("album", int64(21), "upsert").
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, getOrCreateAlbumTx(GetDB(), incoming))
	require.Equal(t, int64(21), incoming.ID)
	require.Equal(t, "1959-08-17", incoming.ReleaseDate)
	require.Equal(t, "1959-08-17", incoming.OriginalReleaseDate)
	require.Equal(t, 3, incoming.SyncStatus)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrCreateAlbumTxCreatesNewAlbumWhenIncomingReleaseDateDiffers(t *testing.T) {
	_, mock := newModelTestDB(t)

	incoming := &Album{
		Name:                "The Dark Side of the Moon",
		Artist:              "Pink Floyd",
		ReleaseDate:         "1973-03-24",
		OriginalReleaseDate: "1973-03-01",
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT * FROM `album` WHERE artist = ? AND name = ? AND release_date = ? AND COALESCE(name_subtitle, '') = ? ORDER BY `album`.`id` LIMIT ?"),
	).
		WithArgs("Pink Floyd", "The Dark Side of the Moon", "1973-03-24", "", 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "artist", "release_date", "original_release_date",
		}))

	mock.ExpectQuery(
		regexp.QuoteMeta(
			"SELECT * FROM `album` WHERE (artist = ? AND name = ? AND COALESCE(name_subtitle, '') = ?) AND (release_date = '' OR release_date IS NULL) ORDER BY CASE WHEN release_date = '' OR release_date IS NULL THEN 0 ELSE 1 END ASC, id ASC,`album`.`id` LIMIT ?",
		),
	).
		WithArgs("Pink Floyd", "The Dark Side of the Moon", "", 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "artist", "release_date", "original_release_date",
		}))

	mock.ExpectExec(
		regexp.QuoteMeta(
			"INSERT INTO `album` (`name`,`name_subtitle`,`artist`,`release_date`,`original_release_date`,`genre`,`country`,`status`,`packaging`,`barcode`,`total_discs`,`disc_infos`,`sync_status`,`cover_art_url`,`cover_art_mime`,`cover_art_object_key`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
		),
	).
		WithArgs(
			"The Dark Side of the Moon", "", "Pink Floyd", "1973-03-24", "1973-03-01", "", "", "", "", "", 1, "", 0, "", "", "",
		).
		WillReturnResult(sqlmock.NewResult(178, 1))
	mock.ExpectExec(
		regexp.QuoteMeta("INSERT INTO `library_change_log` (`entity_type`,`entity_id`,`operation`) VALUES (?,?,?)"),
	).
		WithArgs("album", int64(178), "upsert").
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, getOrCreateAlbumTx(GetDB(), incoming))
	require.Equal(t, int64(178), incoming.ID)
	require.Equal(t, "1973-03-24", incoming.ReleaseDate)
	require.Equal(t, "1973-03-01", incoming.OriginalReleaseDate)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateAlbumSyncStatusTx(t *testing.T) {
	_, mock := newModelTestDB(t)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE `album` SET `sync_status`=?,`updated_at`=? WHERE id = ?")).
		WithArgs(2, modelTestNow, int64(11)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(
		regexp.QuoteMeta("INSERT INTO `library_change_log` (`entity_type`,`entity_id`,`operation`) VALUES (?,?,?)"),
	).
		WithArgs("album", int64(11), "upsert").
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, UpdateAlbumSyncStatusTx(GetDB(), 11, 2))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateAlbumFieldsTx(t *testing.T) {
	_, mock := newModelTestDB(t)

	mock.ExpectExec(
		regexp.QuoteMeta("UPDATE `album` SET `disc_infos`=?,`genre`=?,`updated_at`=? WHERE id = ?"),
	).
		WithArgs(`{"1":13}`, "Progressive Rock", modelTestNow, int64(12)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(
		regexp.QuoteMeta("INSERT INTO `library_change_log` (`entity_type`,`entity_id`,`operation`) VALUES (?,?,?)"),
	).
		WithArgs("album", int64(12), "upsert").
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(
		t, UpdateAlbumFieldsTx(
			GetDB(),
			12,
			map[string]interface{}{
				"disc_infos": `{"1":13}`,
				"genre":      "Progressive Rock",
			},
		),
	)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateAlbumSyncStatusUsesContextDB(t *testing.T) {
	_, mock := newModelTestDB(t)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE `album` SET `sync_status`=?,`updated_at`=? WHERE id = ?")).
		WithArgs(1, modelTestNow, int64(13)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(
		regexp.QuoteMeta("INSERT INTO `library_change_log` (`entity_type`,`entity_id`,`operation`) VALUES (?,?,?)"),
	).
		WithArgs("album", int64(13), "upsert").
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, UpdateAlbumSyncStatus(ctx, 13, 1))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateAlbumFieldsUsesContextDB(t *testing.T) {
	_, mock := newModelTestDB(t)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE `album` SET `total_discs`=?,`updated_at`=? WHERE id = ?")).
		WithArgs(2, modelTestNow, int64(14)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(
		regexp.QuoteMeta("INSERT INTO `library_change_log` (`entity_type`,`entity_id`,`operation`) VALUES (?,?,?)"),
	).
		WithArgs("album", int64(14), "upsert").
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, UpdateAlbumFields(ctx, 14, map[string]interface{}{"total_discs": 2}))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateAlbumTitleMetadataByIDTx(t *testing.T) {
	_, mock := newModelTestDB(t)

	mock.ExpectExec(
		regexp.QuoteMeta(
			"UPDATE `album` SET `title_metadata`=?,`updated_at`=? WHERE id = ? AND ((title_metadata IS NULL OR title_metadata = '' OR title_metadata <> ?))",
		),
	).
		WithArgs(
			`{"source_display_title":"Kid A (Deluxe Edition)","official_title":"Kid A","title_versions":[{"text":"Deluxe Edition","type":"edition","bracketed":false,"parenthesized":true}],"normalized_display_title":"Kid A (Deluxe Edition)"}`,
			modelTestNow,
			int64(15),
			`{"source_display_title":"Kid A (Deluxe Edition)","official_title":"Kid A","title_versions":[{"text":"Deluxe Edition","type":"edition","bracketed":false,"parenthesized":true}],"normalized_display_title":"Kid A (Deluxe Edition)"}`,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(
		regexp.QuoteMeta("INSERT INTO `library_change_log` (`entity_type`,`entity_id`,`operation`) VALUES (?,?,?)"),
	).
		WithArgs("album", int64(15), "upsert").
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(
		t,
		UpdateAlbumTitleMetadataByIDTx(
			GetDB(),
			15,
			&common.AlbumTitleMetadata{
				SourceDisplayTitle: "Kid A (Deluxe Edition)",
				OfficialTitle:      "Kid A",
				TitleVersions: []common.AlbumTitleVersion{
					{
						Text:          "Deluxe Edition",
						Type:          common.AlbumTitleVersionTypeEdition,
						Parenthesized: true,
					},
				},
				NormalizedDisplayTitle: "Kid A (Deluxe Edition)",
			},
		),
	)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateAlbumSyncStatusTxPropagatesExecError(t *testing.T) {
	_, mock := newModelTestDB(t)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE `album` SET `sync_status`=?,`updated_at`=? WHERE id = ?")).
		WithArgs(9, modelTestNow, int64(15)).
		WillReturnError(gorm.ErrInvalidDB)

	err := UpdateAlbumSyncStatusTx(GetDB(), 15, 9)
	require.ErrorIs(t, err, gorm.ErrInvalidDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertAlbumCoverByIDTx(t *testing.T) {
	_, mock := newModelTestDB(t)

	mock.ExpectExec(
		regexp.QuoteMeta(
			"UPDATE `album` SET `cover_art_mime`=?,`cover_art_object_key`=?,`cover_art_url`=?,`updated_at`=? WHERE id = ? AND ((cover_art_object_key IS NULL OR cover_art_object_key = '' OR cover_art_object_key <> ?))",
		),
	).
		WithArgs("image/jpeg", "v1/originals/abc", "http://127.0.0.1:9000/album/v1/originals/abc", modelTestNow, int64(18), "v1/originals/abc").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(
		regexp.QuoteMeta("INSERT INTO `library_change_log` (`entity_type`,`entity_id`,`operation`) VALUES (?,?,?)"),
	).
		WithArgs("album", int64(18), "upsert").
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(
		t,
		UpsertAlbumCoverByIDTx(
			GetDB(),
			18,
			AlbumCoverUpdate{
				CoverArtURL:       "http://127.0.0.1:9000/album/v1/originals/abc",
				CoverArtMime:      "image/jpeg",
				CoverArtObjectKey: "v1/originals/abc",
			},
		),
	)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertAlbumCoverByIDTxSkipsChangeLogWhenNoRowsAffected(t *testing.T) {
	_, mock := newModelTestDB(t)

	mock.ExpectExec(
		regexp.QuoteMeta(
			"UPDATE `album` SET `cover_art_mime`=?,`cover_art_object_key`=?,`cover_art_url`=?,`updated_at`=? WHERE id = ? AND ((cover_art_object_key IS NULL OR cover_art_object_key = '' OR cover_art_object_key <> ?))",
		),
	).
		WithArgs("image/jpeg", "v1/originals/existing", "http://127.0.0.1:9000/album/v1/originals/existing", modelTestNow, int64(19), "v1/originals/existing").
		WillReturnResult(sqlmock.NewResult(0, 0))

	require.NoError(
		t,
		UpsertAlbumCoverByIDTx(
			GetDB(),
			19,
			AlbumCoverUpdate{
				CoverArtURL:       "http://127.0.0.1:9000/album/v1/originals/existing",
				CoverArtMime:      "image/jpeg",
				CoverArtObjectKey: "v1/originals/existing",
			},
		),
	)
	require.NoError(t, mock.ExpectationsWereMet())
}
