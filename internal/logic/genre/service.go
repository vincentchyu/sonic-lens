package genre

import (
	"context"

	"github.com/vincentchyu/sonic-lens/internal/model"
)

// GenreService 定义流派相关服务接口
type GenreService interface {
	CreateGenre(ctx context.Context, genre *model.Genre) error
	GetGenreByName(ctx context.Context, name string) (*model.Genre, error)
	GetGenreByID(ctx context.Context, id int64) (*model.Genre, error)
	GetAllGenres(ctx context.Context, limit, offset int) ([]*model.Genre, error)
	GetAllGenresWithFilter(ctx context.Context, keyword, sortBy string, limit, offset int) ([]*model.Genre, int64, error)
	UpdateGenre(ctx context.Context, genre *model.Genre) error
	DeleteGenre(ctx context.Context, id int64) error
	IncrementGenrePlayCount(ctx context.Context, name string) error
	GetGenreCount(ctx context.Context) (int64, error)
	GetTopGenresByPlayCount(ctx context.Context, limit int) ([]*model.Genre, error)
	GetTopGenresWithDetails(ctx context.Context, limit int) ([]*model.TopGenre, error)
	GetAlbumsByGenre(ctx context.Context, genre string, limit, offset int, sortBy string) ([]*model.Album, error)
	GetAlbumsByGenreCount(ctx context.Context, genre string) (int64, error)
	ResolveGenreTest(ctx context.Context, rawTag string) (segment string, canonicalEng string, canonicalZh string, isMatched bool, normalized string)
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
func (s *GenreServiceImpl) GetGenreByID(ctx context.Context, id int64) (*model.Genre, error) {
	return model.GetGenreByID(ctx, id)
}

// GetAllGenres 获取所有流派（分页）
func (s *GenreServiceImpl) GetAllGenres(ctx context.Context, limit, offset int) ([]*model.Genre, error) {
	return model.GetAllGenres(ctx, limit, offset)
}

// GetAllGenresWithFilter 支持关键词过滤与排序的流派分页获取
func (s *GenreServiceImpl) GetAllGenresWithFilter(ctx context.Context, keyword, sortBy string, limit, offset int) ([]*model.Genre, int64, error) {
	return model.GetAllGenresWithFilter(ctx, keyword, sortBy, limit, offset)
}

// UpdateGenre 更新流派
func (s *GenreServiceImpl) UpdateGenre(ctx context.Context, genre *model.Genre) error {
	return model.UpdateGenre(ctx, genre)
}

// DeleteGenre 删除流派
func (s *GenreServiceImpl) DeleteGenre(ctx context.Context, id int64) error {
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

// GetAlbumsByGenre 根据英文流派标准名检索关联专辑
func (s *GenreServiceImpl) GetAlbumsByGenre(ctx context.Context, genre string, limit, offset int, sortBy string) ([]*model.Album, error) {
	return model.GetAlbumsByGenre(ctx, genre, limit, offset, sortBy)
}

// GetAlbumsByGenreCount 根据英文流派标准名计算关联专辑数
func (s *GenreServiceImpl) GetAlbumsByGenreCount(ctx context.Context, genre string) (int64, error) {
	return model.GetAlbumsByGenreCount(ctx, genre)
}

// ResolveGenreTest 对流派进行沙盒解析测试，返回分段提取、权威英文名、规范中文名、是否命中权威缓存与整体归一化结果
func (s *GenreServiceImpl) ResolveGenreTest(ctx context.Context, rawTag string) (segment string, canonicalEng string, canonicalZh string, isMatched bool, normalized string) {
	segment = model.ExtractPrimaryGenreTag(rawTag)
	if segment == "" {
		return "", "", "", false, ""
	}
	isMatched, canonicalEng, canonicalZh = model.ResolveStrictGenreIdentity(nil, segment)
	normalized = model.NormalizeGenre(nil, rawTag)
	return segment, canonicalEng, canonicalZh, isMatched, normalized
}

