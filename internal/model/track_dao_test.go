package model

import (
	"context"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/config"
)

func trackRowsColumns() []string {
	return []string{
		"id", "artist", "album", "album_subtitle", "track", "play_count", "is_apple_music_fav", "is_last_fm_fav", "version",
		"album_artist", "track_number", "disc_number", "duration", "genre", "composer", "release_date",
		"music_brainz_id", "source", "bundle_id", "unique_id", "created_at", "updated_at",
	}
}

func TestGetTrackByIDTx(t *testing.T) {
	_, mock := newModelTestDB(t)

	rows := sqlmock.NewRows(
		[]string{
			"id", "artist", "album", "album_subtitle", "track", "play_count", "is_apple_music_fav", "is_last_fm_fav", "version",
			"album_artist", "track_number", "disc_number", "duration", "genre", "composer", "release_date",
			"music_brainz_id", "source", "bundle_id", "unique_id", "created_at", "updated_at",
		},
	).AddRow(
		88, "Pink Floyd", "The Wall", "", "Comfortably Numb", 10, false, false, 1,
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
				"id", "artist", "album", "album_subtitle", "track", "play_count", "is_apple_music_fav", "is_last_fm_fav", "version",
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

func TestUpdateTrackCuratedMetadataTxNormalizesChineseText(t *testing.T) {
	_, mock := newModelTestDB(t)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `track` WHERE `track`.`id` = ? ORDER BY `track`.`id` LIMIT ?")).
		WithArgs(int64(92), 1).
		WillReturnRows(sqlmock.NewRows(trackRowsColumns()).AddRow(
			92, "周杰倫", "范特西", "豪华版", "髮如雪", 12, false, false, 3,
			"周杰倫", 3, 1, 250, "中國流行樂", "周杰倫", "2001-09-14",
			"mbid-track-curated", "MusicBrainz", "bundle-1", "uid-1", modelTestNow, modelTestNow,
		))
	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE `track` SET `album`=?,`album_artist`=?,`album_subtitle`=?,`artist`=?,`bundle_id`=?,`composer`=?,`disc_number`=?,`duration`=?,`genre`=?,`music_brainz_id`=?,`release_date`=?,`source`=?,`track`=?,`track_number`=?,`unique_id`=?,`version`=?,`updated_at`=? WHERE id = ? AND version = ?",
	)).
		WithArgs(
			"范特西",
			"周杰伦",
			"豪华版",
			"周杰伦",
			"bundle-1",
			"周杰伦",
			int8(1),
			int64(250),
			"中国流行乐",
			"mbid-track-curated",
			"2001-09-14",
			"MusicBrainz",
			"发如雪",
			int8(3),
			"uid-1",
			4,
			modelTestNow,
			int64(92),
			3,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, UpdateTrackCuratedMetadataTx(
		GetDB(),
		92,
		&TrackIdentity{
			Artist:        "周杰倫",
			Album:         "范特西",
			AlbumSubtitle: "豪华版",
			Track:         "髮如雪",
			TrackNumber:   3,
			DiscNumber:    1,
		},
		&TrackMetadata{
			AlbumArtist:   "周杰倫",
			AlbumSubtitle: "豪华版",
			TrackNumber:   3,
			DiscNumber:    1,
			Duration:      250,
			Genre:         "中國流行樂",
			Composer:      "周杰倫",
			ReleaseDate:   "2001-09-14",
			MusicBrainzID: "mbid-track-curated",
			Source:        "MusicBrainz",
			BundleID:      "bundle-1",
			UniqueID:      "uid-1",
		},
	))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrCreateTrackByIdentityTxNormalizesCreatedTrackText(t *testing.T) {
	_, mock := newModelTestDB(t)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `track` WHERE artist = ? AND album = ? AND COALESCE(album_subtitle, '') = ? AND track = ? AND track_number = ? AND disc_number = ? ORDER BY `track`.`id` LIMIT ?")).
		WithArgs("周杰伦", "范特西", "豪华版", "发如雪", int8(3), int8(1), 1).
		WillReturnRows(sqlmock.NewRows(trackRowsColumns()))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `track` WHERE artist = ? AND album = ? AND COALESCE(album_subtitle, '') = ? AND track = ? AND track_number = ? AND disc_number = ? ORDER BY `track`.`id` LIMIT ?")).
		WithArgs("周杰伦", "范特西", "豪华版", "发如雪", int8(3), int8(1), 1).
		WillReturnRows(sqlmock.NewRows(trackRowsColumns()))
	mock.ExpectExec(regexp.QuoteMeta(
		"INSERT INTO `track` (`artist`,`album`,`album_subtitle`,`track`,`play_count`,`is_apple_music_fav`,`is_last_fm_fav`,`version`,`album_artist`,`track_number`,`disc_number`,`duration`,`genre`,`composer`,`release_date`,`music_brainz_id`,`source`,`bundle_id`,`unique_id`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
	)).
		WithArgs(
			"周杰伦",
			"范特西",
			"豪华版",
			"发如雪",
			0,
			false,
			false,
			1,
			"周杰伦",
			int8(3),
			int8(1),
			int64(250),
			"中国流行乐",
			"周杰伦",
			"2001-09-14",
			"mbid-track-create",
			"MusicBrainz",
			"bundle-2",
			"uid-2",
		).
		WillReturnResult(sqlmock.NewResult(93, 1))
	mock.ExpectExec(regexp.QuoteMeta(
		"INSERT INTO `library_change_log` (`entity_type`,`entity_id`,`operation`) VALUES (?,?,?)",
	)).
		WithArgs(LibraryEntityTrack, int64(93), LibraryOpUpsert).
		WillReturnResult(sqlmock.NewResult(22, 1))

	track, err := GetOrCreateTrackByIdentityTx(
		GetDB(),
		&Track{
			Artist:        "周杰倫",
			Album:         "范特西",
			AlbumSubtitle: "豪华版",
			Track:         "髮如雪",
			TrackNumber:   3,
			DiscNumber:    1,
			AlbumArtist:   "周杰倫",
			Duration:      250,
			Genre:         "中國流行樂",
			Composer:      "周杰倫",
			ReleaseDate:   "2001-09-14",
			MusicBrainzID: "mbid-track-create",
			Source:        "MusicBrainz",
			BundleID:      "bundle-2",
			UniqueID:      "uid-2",
		},
	)
	require.NoError(t, err)
	require.Equal(t, "周杰伦", track.Artist)
	require.Equal(t, "范特西", track.Album)
	require.Equal(t, "豪华版", track.AlbumSubtitle)
	require.Equal(t, "发如雪", track.Track)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateTrackWithTrackMetadataNormalizesChineseText(t *testing.T) {
	track := &Track{
		Artist: "周杰倫",
		Album:  "范特西",
		Track:  "髮如雪",
	}

	UpdateTrackWithTrackMetadata(track, &TrackMetadata{
		AlbumArtist:   "周杰倫",
		AlbumSubtitle: " 豪華版 ",
		Genre:         "中國流行樂",
		Composer:      "周杰倫",
		Source:        " MusicBrainz ",
	})

	require.Equal(t, "周杰伦", track.AlbumArtist)
	require.Equal(t, "豪华版", track.AlbumSubtitle)
	require.Equal(t, "中国流行乐", track.Genre)
	require.Equal(t, "周杰伦", track.Composer)
	require.Equal(t, "MusicBrainz", track.Source)
}

func TestGetOrCreateTrackByIdentityTxKeepsTrackArtistAndAlbumArtistSeparate(t *testing.T) {
	_, mock := newModelTestDB(t)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `track` WHERE artist = ? AND album = ? AND COALESCE(album_subtitle, '') = ? AND track = ? AND track_number = ? AND disc_number = ? ORDER BY `track`.`id` LIMIT ?")).
		WithArgs("Track Artist", "Compilation Album", "", "Featured Song", int8(1), int8(1), 1).
		WillReturnRows(sqlmock.NewRows(trackRowsColumns()))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `track` WHERE artist = ? AND album = ? AND COALESCE(album_subtitle, '') = ? AND track_number = ? AND disc_number = ? LIMIT ?")).
		WithArgs("Track Artist", "Compilation Album", "", int8(1), int8(1), 2).
		WillReturnRows(sqlmock.NewRows(trackRowsColumns()))
	mock.ExpectExec(regexp.QuoteMeta(
		"INSERT INTO `track` (`artist`,`album`,`album_subtitle`,`track`,`play_count`,`is_apple_music_fav`,`is_last_fm_fav`,`version`,`album_artist`,`track_number`,`disc_number`,`duration`,`genre`,`composer`,`release_date`,`music_brainz_id`,`source`,`bundle_id`,`unique_id`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
	)).
		WithArgs(
			"Track Artist",
			"Compilation Album",
			"",
			"Featured Song",
			0,
			false,
			false,
			1,
			"Various Artists",
			int8(1),
			int8(1),
			int64(210),
			"",
			"",
			"2024-01-01",
			"",
			"Apple Music",
			"",
			"",
		).
		WillReturnResult(sqlmock.NewResult(301, 1))
	mock.ExpectExec(regexp.QuoteMeta(
		"INSERT INTO `library_change_log` (`entity_type`,`entity_id`,`operation`) VALUES (?,?,?)",
	)).
		WithArgs(LibraryEntityTrack, int64(301), LibraryOpUpsert).
		WillReturnResult(sqlmock.NewResult(23, 1))

	track, err := GetOrCreateTrackByIdentityTx(
		GetDB(),
		&Track{
			Artist:        "Track Artist",
			Album:         "Compilation Album",
			Track:         "Featured Song",
			TrackNumber:   1,
			DiscNumber:    1,
			AlbumArtist:   "Various Artists",
			Duration:      210,
			ReleaseDate:   "2024-01-01",
			Source:        "Apple Music",
			MusicBrainzID: "",
		},
	)
	require.NoError(t, err)
	require.Equal(t, "Track Artist", track.Artist)
	require.Equal(t, "Various Artists", track.AlbumArtist)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrCreatePlaybackAlbumTxPrefersAlbumArtist(t *testing.T) {
	_, mock := newModelTestDB(t)

	metadata := TrackMetadata{
		AlbumArtist: "Various Artists",
		ReleaseDate: "2024-01-01",
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT * FROM `album` WHERE artist = ? AND name = ? ORDER BY sync_status DESC, id ASC,`album`.`id` LIMIT ?"),
	).
		WithArgs("Various Artists", "Compilation Album", 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "artist", "release_date", "original_release_date",
		}))
	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT * FROM `album` WHERE artist = ? AND name = ? AND release_date = ? AND COALESCE(name_subtitle, '') = ? ORDER BY `album`.`id` LIMIT ?"),
	).
		WithArgs("Various Artists", "Compilation Album", "2024-01-01", "", 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "artist", "release_date", "original_release_date",
		}))
	mock.ExpectQuery(
		regexp.QuoteMeta(
			"SELECT * FROM `album` WHERE (artist = ? AND name = ? AND COALESCE(name_subtitle, '') = ?) AND (release_date = '' OR release_date IS NULL) ORDER BY CASE WHEN release_date = '' OR release_date IS NULL THEN 0 ELSE 1 END ASC, id ASC,`album`.`id` LIMIT ?",
		),
	).
		WithArgs("Various Artists", "Compilation Album", "", 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "artist", "release_date", "original_release_date",
		}))
	mock.ExpectExec(
		regexp.QuoteMeta(
			"INSERT INTO `album` (`name`,`name_subtitle`,`artist`,`release_date`,`original_release_date`,`genre`,`country`,`status`,`packaging`,`barcode`,`total_discs`,`disc_infos`,`sync_status`,`cover_art_url`,`cover_art_mime`,`cover_art_object_key`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
		),
	).
		WithArgs(
			"Compilation Album", "", "Various Artists", "2024-01-01", "", "", "", "", "", "", 1, "", 0, "", "", "",
		).
		WillReturnResult(sqlmock.NewResult(401, 1))
	mock.ExpectExec(
		regexp.QuoteMeta("INSERT INTO `library_change_log` (`entity_type`,`entity_id`,`operation`) VALUES (?,?,?)"),
	).
		WithArgs("album", int64(401), "upsert").
		WillReturnResult(sqlmock.NewResult(24, 1))

	album, err := getOrCreatePlaybackAlbumTx(GetDB(), "Track Artist", "Compilation Album", metadata)
	require.NoError(t, err)
	require.Equal(t, int64(401), album.ID)
	require.Equal(t, "Various Artists", album.Artist)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPreferredPlaybackAlbumArtistFallsBackToTrackArtist(t *testing.T) {
	require.Equal(t, "Track Artist", preferredPlaybackAlbumArtist("Track Artist", ""))
	require.Equal(t, "Various Artists", preferredPlaybackAlbumArtist("Track Artist", "Various Artists"))
}

func TestBuildTrackPlayRecordArtworkPath(t *testing.T) {
	prev := config.ConfigObj.ObjectStorage
	config.ConfigObj.ObjectStorage = config.ObjectStorageConfig{
		Enabled: true,
		CDNURL:  "/album",
		Bucket:  "album",
	}
	defer func() {
		config.ConfigObj.ObjectStorage = prev
	}()

	require.Equal(t, "/album/v1/originals/abc123", BuildTrackPlayRecordArtworkPath("", "v1/originals/abc123"))
	require.Equal(t, "https://cdn.example.com/cover.jpg", BuildTrackPlayRecordArtworkPath("https://cdn.example.com/cover.jpg", "v1/originals/abc123"))
}

func TestGetRecentPlayRecordsReturnsCoverArtPath(t *testing.T) {
	_, mock := newModelTestDB(t)

	rows := sqlmock.NewRows(
		[]string{"id", "artist", "album", "track", "play_time", "cover_art_path"},
	).AddRow(
		42, "Radiohead", "In Rainbows", "Nude", modelTestNow, "/album/v1/originals/in-rainbows.webp",
	)

	mock.ExpectQuery(regexp.MustCompile(
		"SELECT .*cover_art_path.*FROM track_play_records AS tpr.*ORDER BY play_time DESC, id DESC LIMIT \\?",
	).String()).
		WithArgs(1).
		WillReturnRows(rows)

	records, err := GetRecentPlayRecords(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, "/album/v1/originals/in-rainbows.webp", records[0].CoverArtPath)
	require.NoError(t, mock.ExpectationsWereMet())
}
