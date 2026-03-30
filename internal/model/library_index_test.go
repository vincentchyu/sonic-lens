package model

import (
	"context"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestGetAlbumIndexRowsIncludesCoverFields(t *testing.T) {
	_, mock := newModelTestDB(t)

	mock.ExpectQuery(
		regexp.QuoteMeta(
			"SELECT a.id, a.name, a.artist, a.release_date,\n" +
				"a.cover_art_url, a.cover_art_mime, a.cover_art_object_key,\n" +
				"EXISTS(\n" +
				"    SELECT 1\n" +
				"    FROM album_insight AS ai\n" +
				"    WHERE ai.is_disabled = false\n" +
				"      AND (ai.album_id = a.id OR ((ai.album_id = 0 OR ai.album_id IS NULL) AND ai.artist = a.artist AND ai.album = a.name))\n" +
				") AS has_insight,\n" +
				"COALESCE(SUM(t.play_count), 0) AS play_count,\n" +
				"a.created_at, MAX(COALESCE(t.updated_at, a.updated_at, a.created_at)) AS updated_at FROM album AS a LEFT JOIN track AS t ON t.album = a.name AND t.artist = a.artist GROUP BY a.id, a.name, a.artist, a.release_date, a.cover_art_url, a.cover_art_mime, a.cover_art_object_key, a.created_at ORDER BY a.id ASC",
		),
	).
		WillReturnRows(
			sqlmock.NewRows(
				[]string{
					"id", "name", "artist", "release_date",
					"cover_art_url", "cover_art_mime", "cover_art_object_key",
					"has_insight", "play_count", "created_at", "updated_at",
				},
			).AddRow(
				int64(1001), "Abbey Road", "The Beatles", "1969-09-26",
				"/api/artwork/v1/originals/beatles-abbey-road.jpg", "image/jpeg", "v1/originals/beatles-abbey-road.jpg",
				true, 37, modelTestNow, modelTestNow,
			),
		)

	rows, err := GetAlbumIndexRows(context.Background(), time.Time{})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "/api/artwork/v1/originals/beatles-abbey-road.jpg", rows[0].CoverArtURL)
	require.Equal(t, "image/jpeg", rows[0].CoverArtMime)
	require.Equal(t, "v1/originals/beatles-abbey-road.jpg", rows[0].CoverArtObjectKey)
	require.True(t, rows[0].HasInsight)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAlbumIndexRowsByIDsIncludesCoverFields(t *testing.T) {
	_, mock := newModelTestDB(t)

	mock.ExpectQuery(
		regexp.QuoteMeta(
			"SELECT a.id, a.name, a.artist, a.release_date,\n" +
				"a.cover_art_url, a.cover_art_mime, a.cover_art_object_key,\n" +
				"EXISTS(\n" +
				"    SELECT 1\n" +
				"    FROM album_insight AS ai\n" +
				"    WHERE ai.is_disabled = false\n" +
				"      AND (ai.album_id = a.id OR ((ai.album_id = 0 OR ai.album_id IS NULL) AND ai.artist = a.artist AND ai.album = a.name))\n" +
				") AS has_insight,\n" +
				"COALESCE(SUM(t.play_count), 0) AS play_count,\n" +
				"a.created_at, MAX(COALESCE(t.updated_at, a.updated_at, a.created_at)) AS updated_at FROM album AS a LEFT JOIN track AS t ON t.album = a.name AND t.artist = a.artist WHERE a.id IN (?) GROUP BY a.id, a.name, a.artist, a.release_date, a.cover_art_url, a.cover_art_mime, a.cover_art_object_key, a.created_at ORDER BY a.id ASC",
		),
	).
		WithArgs(int64(77)).
		WillReturnRows(
			sqlmock.NewRows(
				[]string{
					"id", "name", "artist", "release_date",
					"cover_art_url", "cover_art_mime", "cover_art_object_key",
					"has_insight", "play_count", "created_at", "updated_at",
				},
			).AddRow(
				int64(77), "In Rainbows", "Radiohead", "2007-10-10",
				"/api/artwork/v1/originals/radiohead-in-rainbows.webp", "image/webp", "v1/originals/radiohead-in-rainbows.webp",
				false, 54, modelTestNow, modelTestNow,
			),
		)

	rows, err := GetAlbumIndexRowsByIDs(context.Background(), []int64{77})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "/api/artwork/v1/originals/radiohead-in-rainbows.webp", rows[0].CoverArtURL)
	require.Equal(t, "image/webp", rows[0].CoverArtMime)
	require.Equal(t, "v1/originals/radiohead-in-rainbows.webp", rows[0].CoverArtObjectKey)
	require.False(t, rows[0].HasInsight)
	require.NoError(t, mock.ExpectationsWereMet())
}
