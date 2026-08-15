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

func TestMockGenreService_GetAllGenresWithFilter(t *testing.T) {
	mockService := new(MockGenreService)
	ctx := context.Background()

	expectedGenres := []*model.Genre{
		{ID: 1, Name: "Rock", NameZh: "摇滚", PlayCount: 500},
		{ID: 2, Name: "Pop", NameZh: "流行", PlayCount: 300},
	}

	mockService.On("GetAllGenresWithFilter", ctx, "Rock", "play_count DESC", 10, 0).Return(expectedGenres, int64(2), nil)

	genres, total, err := mockService.GetAllGenresWithFilter(ctx, "Rock", "play_count DESC", 10, 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, genres, 2)
	assert.Equal(t, "Rock", genres[0].Name)

	mockService.AssertExpectations(t)
}

func TestMockGenreService_ResolveGenreTest(t *testing.T) {
	mockService := new(MockGenreService)
	ctx := context.Background()

	mockService.On("ResolveGenreTest", ctx, "Pop / Rock").Return("Pop", "Pop", "流行", true, "Pop")

	segment, eng, zh, matched, norm := mockService.ResolveGenreTest(ctx, "Pop / Rock")
	assert.Equal(t, "Pop", segment)
	assert.Equal(t, "Pop", eng)
	assert.Equal(t, "流行", zh)
	assert.True(t, matched)
	assert.Equal(t, "Pop", norm)

	mockService.AssertExpectations(t)
}

