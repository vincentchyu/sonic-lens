package insight

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vincentchyu/sonic-lens/internal/model"
)

func TestPickBestTrackInsightWithScoresPrefersHigherScore(t *testing.T) {
	t.Parallel()

	older := &model.TrackInsight{
		ID:        1,
		CreatedAt: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
	}
	newer := &model.TrackInsight{
		ID:        2,
		CreatedAt: time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC),
	}

	best := pickBestTrackInsightWithScores(
		[]*model.TrackInsight{older, newer},
		map[int64]int{
			1: 5,
			2: 3,
		},
	)

	require.NotNil(t, best)
	require.Equal(t, int64(1), best.ID)
}

func TestPickBestTrackInsightWithScoresPrefersNewestOnTie(t *testing.T) {
	t.Parallel()

	older := &model.TrackInsight{
		ID:        11,
		CreatedAt: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
	}
	newer := &model.TrackInsight{
		ID:        12,
		CreatedAt: time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC),
	}

	best := pickBestTrackInsightWithScores(
		[]*model.TrackInsight{older, newer},
		map[int64]int{
			11: 4,
			12: 4,
		},
	)

	require.NotNil(t, best)
	require.Equal(t, int64(12), best.ID)
}

func TestTruncatePromptTextUsesRuneLength(t *testing.T) {
	t.Parallel()

	got := truncatePromptText("你好世界和平", 4)
	require.Equal(t, "你好世界...", got)
}
