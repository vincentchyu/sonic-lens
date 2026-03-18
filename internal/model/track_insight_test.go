package model

import (
	"context"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetTrackInsightByID(t *testing.T) {
	_, mock := newModelTestDB(t)
	ctx := context.Background()

	rows := sqlmock.NewRows(
		[]string{
			"id", "track_id", "artist", "album", "track", "lyrics_translation", "analysis_summary",
			"analysis_by_section", "background_info", "era_context", "llm_provider", "lang_source",
			"lang_target", "metadata", "like_count", "dislike_count", "created_at", "updated_at",
			"last_used_at", "is_disabled",
		},
	).AddRow(
		17, 90, "Pink Floyd", "The Wall", "Comfortably Numb", "译文", "总结",
		[]byte(`{"verse":"..."}`), "背景", "时代", "openai:gpt-4.1", "en",
		"zh-CN", `{"source":"test"}`, 3, 0, modelTestNow, modelTestNow, modelTestNow, false,
	)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `track_insight` WHERE `track_insight`.`id` = ? ORDER BY `track_insight`.`id` LIMIT ?")).
		WithArgs(int64(17), 1).
		WillReturnRows(rows)

	insight, err := GetTrackInsightByID(ctx, 17)
	require.NoError(t, err)
	require.Equal(t, int64(17), insight.ID)
	require.Equal(t, "Comfortably Numb", insight.Track)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTrackInsightByIDPropagatesNotFound(t *testing.T) {
	_, mock := newModelTestDB(t)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `track_insight` WHERE `track_insight`.`id` = ? ORDER BY `track_insight`.`id` LIMIT ?")).
		WithArgs(int64(18), 1).
		WillReturnRows(sqlmock.NewRows(
			[]string{
				"id", "track_id", "artist", "album", "track", "lyrics_translation", "analysis_summary",
				"analysis_by_section", "background_info", "era_context", "llm_provider", "lang_source",
				"lang_target", "metadata", "like_count", "dislike_count", "created_at", "updated_at",
				"last_used_at", "is_disabled",
			},
		))

	_, err := GetTrackInsightByID(ctx, 18)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}
