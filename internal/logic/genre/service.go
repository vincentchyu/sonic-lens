package genre

import (
	"context"

	"github.com/vincentchyu/sonic-lens/internal/model"
)

// GenreService 定义流派相关服务接口
type GenreService interface {
	CreateGenre(ctx context.Context, genre *model.Genre) error
	GetGenreByName(ctx context.Context, name string) (*model.Genre, error)
	GetGenreByID(ctx context.Context, id uint) (*model.Genre, error)
	GetAllGenres(ctx context.Context, limit, offset int) ([]*model.Genre, error)
	UpdateGenre(ctx context.Context, genre *model.Genre) error
	DeleteGenre(ctx context.Context, id uint) error
	IncrementGenrePlayCount(ctx context.Context, name string) error
	GetGenreCount(ctx context.Context) (int64, error)
	GetTopGenresByPlayCount(ctx context.Context, limit int) ([]*model.Genre, error)
	GetTopGenresWithDetails(ctx context.Context, limit int) ([]*model.TopGenre, error)
}

// GenreServiceImpl 实现GenreService接口
type GenreServiceImpl struct{}

// NewGenreService 创建GenreService实例
func NewGenreService() GenreService {
	return &GenreServiceImpl{}
}

// CreateGenre 创建新的流派
func (s *GenreServiceImpl) CreateGenre(ctx context.Context, genre *model.Genre) error {
	return model.CreateGenre(ctx, genre)
}

// GetGenreByName 根据名称获取流派
func (s *GenreServiceImpl) GetGenreByName(ctx context.Context, name string) (*model.Genre, error) {
	return model.GetGenreByName(ctx, name)
}

// GetGenreByID 根据ID获取流派
func (s *GenreServiceImpl) GetGenreByID(ctx context.Context, id uint) (*model.Genre, error) {
	return model.GetGenreByID(ctx, id)
}

// GetAllGenres 获取所有流派（分页）
func (s *GenreServiceImpl) GetAllGenres(ctx context.Context, limit, offset int) ([]*model.Genre, error) {
	return model.GetAllGenres(ctx, limit, offset)
}

// UpdateGenre 更新流派
func (s *GenreServiceImpl) UpdateGenre(ctx context.Context, genre *model.Genre) error {
	return model.UpdateGenre(ctx, genre)
}

// DeleteGenre 删除流派
func (s *GenreServiceImpl) DeleteGenre(ctx context.Context, id uint) error {
	return model.DeleteGenre(ctx, id)
}

// IncrementGenrePlayCount 增加流派播放次数
func (s *GenreServiceImpl) IncrementGenrePlayCount(ctx context.Context, name string) error {
	return model.IncrementGenrePlayCount(ctx, name)
}

// GetGenreCount 获取流派总数
func (s *GenreServiceImpl) GetGenreCount(ctx context.Context) (int64, error) {
	return model.GetGenreCount(ctx)
}

// GetTopGenresByPlayCount 获取按播放次数排序的流派
func (s *GenreServiceImpl) GetTopGenresByPlayCount(ctx context.Context, limit int) ([]*model.Genre, error) {
	return model.GetTopGenresByPlayCount(ctx, limit)
}

// GetTopGenresWithDetails 获取热门流派的详细信息
func (s *GenreServiceImpl) GetTopGenresWithDetails(ctx context.Context, limit int) ([]*model.TopGenre, error) {
	return model.GetTopGenresWithDetails(ctx, limit)
}
