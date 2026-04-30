package genre

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/vincentchyu/sonic-lens/internal/model"
)

// MockGenreService is a mock implementation of GenreService for testing
type MockGenreService struct {
	mock.Mock
}

func (m *MockGenreService) CreateGenre(ctx context.Context, genre *model.Genre) error {
	args := m.Called(ctx, genre)
	return args.Error(0)
}

func (m *MockGenreService) GetGenreByName(ctx context.Context, name string) (*model.Genre, error) {
	args := m.Called(ctx, name)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.(*model.Genre), args.Error(1)
}

func (m *MockGenreService) GetGenreByID(ctx context.Context, id uint) (*model.Genre, error) {
	args := m.Called(ctx, id)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.(*model.Genre), args.Error(1)
}

func (m *MockGenreService) GetAllGenres(ctx context.Context, limit, offset int) ([]*model.Genre, error) {
	args := m.Called(ctx, limit, offset)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.([]*model.Genre), args.Error(1)
}

func (m *MockGenreService) UpdateGenre(ctx context.Context, genre *model.Genre) error {
	args := m.Called(ctx, genre)
	return args.Error(0)
}

func (m *MockGenreService) DeleteGenre(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockGenreService) IncrementGenrePlayCount(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockGenreService) GetTopGenresWithDetails(ctx context.Context, limit int) ([]*model.TopGenre, error) {
	args := m.Called(ctx, limit)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.([]*model.TopGenre), args.Error(1)
}

func (m *MockGenreService) GetGenreCount(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockGenreService) GetTopGenresByPlayCount(ctx context.Context, limit int) ([]*model.Genre, error) {
	args := m.Called(ctx, limit)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.([]*model.Genre), args.Error(1)
}
