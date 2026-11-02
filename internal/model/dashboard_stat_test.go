package model

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTopArtistsFromStatReturnsRawRows(t *testing.T) {
	_, mock := newModelTestDB(t)

	topArtistRows := sqlmock.NewRows(
		[]string{"period_days", "metric_type", "artist", "metric_value", "rank", "updated_at"},
	).
		AddRow(0, "plays", "Pink Floyd", 305, 1, modelTestNow).
		AddRow(0, "plays", "Radiohead", 266, 2, modelTestNow)
	mock.ExpectQuery("SELECT .* FROM `top_artist_stat`").
		WithArgs(0, "plays", 2).
		WillReturnRows(topArtistRows)

	result, err := GetTopArtistsFromStat(context.Background(), "plays", 2)
	require.NoError(t, err)
	require.Len(t, result, 2)

	assert.Equal(t, "Pink Floyd", result[0]["artist"])
	assert.EqualValues(t, 305, result[0]["play_count"])

	assert.Equal(t, "Radiohead", result[1]["artist"])
	assert.EqualValues(t, 266, result[1]["play_count"])

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTopTracksByPlayCountFromStatReturnsCoverArtURL(t *testing.T) {
	_, mock := newModelTestDB(t)

	rows := sqlmock.NewRows(
		[]string{"track_id", "track", "album", "artist", "play_count", "rank", "cover_art_url", "cover_art_mime", "cover_art_object_key"},
	).
		AddRow(0, "Here Comes the Sun (2019 Mix)", "Abbey Road", "The Beatles", 4, 1, "/album/v1/originals/42f3ba944229ed8629f32bc4375f3ed5c87dcc38", "image/jpeg", "").
		AddRow(123, "你的城市", "劳动之余", "声音玩具", 3, 2, "/album/v1/originals/aac8c6e69cf34225620859e37673c249139a53f7", "image/jpeg", "v1/originals/aac8c6e69cf34225620859e37673c249139a53f7")

	mock.ExpectQuery("SELECT track_id, track, album, artist, play_count, `rank`, cover_art_url, cover_art_mime, cover_art_object_key FROM `track_rank_stat` WHERE period_type = \\? AND TRIM\\(COALESCE\\(track, ''\\)\\) <> '' ORDER BY play_count DESC, `rank` ASC LIMIT \\?").
		WithArgs("month", 2).
		WillReturnRows(rows)

	result, err := GetTopTracksByPlayCountFromStat(context.Background(), 30, 2)
	require.NoError(t, err)
	require.Len(t, result, 2)

	assert.Equal(t, "Here Comes the Sun (2019 Mix)", result[0].Track)
	assert.Equal(t, "/album/v1/originals/42f3ba944229ed8629f32bc4375f3ed5c87dcc38", result[0].CoverArtURL)
	assert.Equal(t, "image/jpeg", result[0].CoverArtMime)
	assert.Equal(t, 1, result[0].Rank)

	assert.Equal(t, "你的城市", result[1].Track)
	assert.Equal(t, "/album/v1/originals/aac8c6e69cf34225620859e37673c249139a53f7", result[1].CoverArtURL)

	require.NoError(t, mock.ExpectationsWereMet())
}
