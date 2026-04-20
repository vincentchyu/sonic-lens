package model

import (
	"context"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestCreateAlbumInsightFeedback(t *testing.T) {
	_, mock := newModelTestDB(t)
	ctx := context.Background()

	feedback := &AlbumInsightFeedback{
		InsightID:      88,
		Score:          -1,
		Comment:        "补充专辑结构层次",
		ReasonCodes:    StringArray{"结构混乱"},
		SectionKey:     "summary",
		SourcePlatform: "iphone",
		CreatedAt:      modelTestNow,
	}

	mock.ExpectExec(
		regexp.QuoteMeta(
			"INSERT INTO `album_insight_feedbacks` (`insight_id`,`score`,`comment`,`reason_codes`,`section_key`,`source_platform`,`created_at`) VALUES (?,?,?,?,?,?,?)",
		),
	).
		WithArgs(int64(88), -1, "补充专辑结构层次", "[\"结构混乱\"]", "summary", "iphone", modelTestNow).
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, CreateAlbumInsightFeedback(ctx, feedback))
	require.Equal(t, int64(1), feedback.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAlbumInsightFeedbacks(t *testing.T) {
	_, mock := newModelTestDB(t)
	ctx := context.Background()

	rows := sqlmock.NewRows(
		[]string{"id", "insight_id", "score", "comment", "reason_codes", "section_key", "source_platform", "created_at"},
	).AddRow(12, 88, -1, "建议补充章节脉络", "[\"缺少关键信息\"]", "summary", "mac", modelTestNow)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `album_insight_feedbacks` WHERE insight_id = ? ORDER BY created_at DESC, id DESC")).
		WithArgs(int64(88)).
		WillReturnRows(rows)

	feedbacks, err := GetAlbumInsightFeedbacks(ctx, 88)
	require.NoError(t, err)
	require.Len(t, feedbacks, 1)
	require.Equal(t, int64(12), feedbacks[0].ID)
	require.Equal(t, "建议补充章节脉络", feedbacks[0].Comment)
	require.Equal(t, []string{"缺少关键信息"}, []string(feedbacks[0].ReasonCodes))
	require.NoError(t, mock.ExpectationsWereMet())
}
