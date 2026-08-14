package genre

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

func TestMockGenreService_GetTopGenresWithDetails(t *testing.T) {
	mockService := new(MockGenreService)
	ctx := context.Background()

	// 模拟返回 50 个热门流派
	expectedGenres := make([]*model.TopGenre, 0, 50)
	for i := 1; i <= 50; i++ {
		expectedGenres = append(expectedGenres, &model.TopGenre{
			TrackGenreName:  fmt.Sprintf("Genre-%d", i),
			TrackGenreCount: int64(1000 - i*10),
			GenreNameZh:     fmt.Sprintf("流派-%d", i),
			GenreCount:      int64(1000 - i*10),
			Rank:            i,
		})
	}

	mockService.On("GetTopGenresWithDetails", ctx, 50).Return(expectedGenres, nil)

	genres, err := mockService.GetTopGenresWithDetails(ctx, 50)
	assert.NoError(t, err)
	assert.Len(t, genres, 50)
	assert.Equal(t, "Genre-1", genres[0].TrackGenreName)
	assert.Equal(t, 1, genres[0].Rank)
	assert.Equal(t, "Genre-50", genres[49].TrackGenreName)
	assert.Equal(t, 50, genres[49].Rank)

	mockService.AssertExpectations(t)
}

