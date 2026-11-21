package model

import (
	"context"
	"testing"
	"time"

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

func TestGetPlayTrendFromStatByDaysUsesHalfOpenWindow(t *testing.T) {
	_, mock := newModelTestDB(t)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startDateKey := today.AddDate(0, 0, -6).Format("2006-01-02")
	todayKey := today.Format("2006-01-02")
	nextDayKey := today.AddDate(0, 0, 1).Format("2006-01-02")

	dailyRows := sqlmock.NewRows([]string{"stat_date", "play_count"}).
		AddRow(todayKey, 8)
	hourlyRows := sqlmock.NewRows([]string{"stat_date", "hour", "play_count"}).
		AddRow(todayKey, 14, 5).
		AddRow(todayKey, 20, 3)

	mock.ExpectQuery(`SELECT DATE\(stat_date\) as stat_date, play_count FROM play_trend_daily_stat WHERE stat_date >= \? AND stat_date < \?`).
		WithArgs(startDateKey, nextDayKey).
		WillReturnRows(dailyRows)
	mock.ExpectQuery(`SELECT DATE\(stat_date\) as stat_date, hour, play_count FROM play_trend_hourly_stat WHERE stat_date >= \? AND stat_date < \?`).
		WithArgs(startDateKey, nextDayKey).
		WillReturnRows(hourlyRows)

	daily, hourly, err := GetPlayTrendFromStatByDays(t.Context(), 7)
	require.NoError(t, err)
	assert.Equal(t, 8, daily[todayKey])
	require.Contains(t, hourly, todayKey)
	assert.Equal(t, 5, hourly[todayKey].Hourly[14])
	assert.Equal(t, 3, hourly[todayKey].Hourly[20])

	require.NoError(t, mock.ExpectationsWereMet())
}
