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
