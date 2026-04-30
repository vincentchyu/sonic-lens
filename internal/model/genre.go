package model

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/core/log"
)

// Genre represents a music genre
type Genre struct {
	ID        int64     `gorm:"column:id;type:int;primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"column:name;type:varchar(255);not null;unique;index:idx_genre_name" json:"name"`
	NameZh    string    `gorm:"column:name_zh;type:varchar(255)" json:"name_zh"`
	Extra     string    `gorm:"column:extra;type:text" json:"extra"`
	PlayCount int64     `gorm:"column:play_count;type:bigint" json:"play_count"`
	CreatedAt time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

// TopGenre represents a top genre with play count
type TopGenre struct {
	TrackGenreName  string `json:"track_genre_name"`  // 流派英文名
	TrackGenreCount int64  `json:"track_genre_count"` // 流派播放次数
	GenreNameZh     string `json:"genre_name_zh"`     // 流派中文名
	GenreCount      int64  `json:"genre_count"`       // 流派总播放次数
	Rank            int    `json:"rank"`              // 排名
}

// TableName sets the table name for the Genre model
func (Genre) TableName() string {
	return "genre"
}

// CreateGenre creates a new genre
func CreateGenre(ctx context.Context, genre *Genre) error {
	log.Debug(ctx, "creating genre", zap.Any("genre", genre))
	return GetDB().WithContext(ctx).Create(genre).Error
}

// GetGenreByName retrieves a genre by name
func GetGenreByName(ctx context.Context, name string) (*Genre, error) {
	var genre Genre
	err := GetDB().WithContext(ctx).Where("name = ?", name).First(&genre).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &genre, nil
}

// GetGenreByID retrieves a genre by ID
func GetGenreByID(ctx context.Context, id uint) (*Genre, error) {
	var genre Genre
	err := GetDB().WithContext(ctx).Where("id = ?", id).First(&genre).Error
	if err != nil {
		return nil, err
	}
	return &genre, nil
}

// GetAllGenres retrieves all genres with pagination
func GetAllGenres(ctx context.Context, limit, offset int) ([]*Genre, error) {
	var genres []*Genre
	err := GetDB().WithContext(ctx).Order("play_count DESC").Limit(limit).Offset(offset).Find(&genres).Error
	if err != nil {
		return nil, err
	}
	return genres, nil
}

// UpdateGenre updates a genre
func UpdateGenre(ctx context.Context, genre *Genre) error {
	return GetDB().WithContext(ctx).Save(genre).Error
}

// DeleteGenre deletes a genre
func DeleteGenre(ctx context.Context, id uint) error {
	return GetDB().WithContext(ctx).Delete(&Genre{}, id).Error
}

// IncrementGenrePlayCount increments the play count for a genre, creating it if it doesn't exist
func IncrementGenrePlayCount(ctx context.Context, name string) error {
	return InTx(
		ctx, func(tx *gorm.DB) error {
			var genre Genre
			// 使用 FirstOrCreate 确保流派记录存在
			if err := tx.Where("name = ?", name).FirstOrCreate(&genre, Genre{Name: name}).Error; err != nil {
				return err
			}
			// 原子增加播放次数
			return tx.Model(&genre).Update("play_count", gorm.Expr("play_count + 1")).Error
		},
	)
}

// GetGenreCount returns the total number of genres
func GetGenreCount(ctx context.Context) (int64, error) {
	var count int64
	err := GetDB().WithContext(ctx).Model(&Genre{}).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetTopGenresByPlayCount returns the top genres by play count
func GetTopGenresByPlayCount(ctx context.Context, limit int) ([]*Genre, error) {
	var genres []*Genre
	err := GetDB().WithContext(ctx).Order("play_count DESC").Limit(limit).Find(&genres).Error
	if err != nil {
		return nil, err
	}
	return genres, nil
}

// GetTopGenresWithDetails returns the top genres with detailed information including track count
func GetTopGenresWithDetails(ctx context.Context, limit int) ([]*TopGenre, error) {
	if statRows, err := GetTopGenresWithDetailsFromStat(ctx, limit); err == nil && len(statRows) > 0 {
		return statRows, nil
	}

	var result []*TopGenre

	// 根据数据库类型使用不同的SQL语法
	// 现在优先从 genre 表获取播放统计，以包含未归因的数据
	err := GetDB().WithContext(ctx).Raw(
		`SELECT name AS track_genre_name, play_count AS genre_count, name_zh AS genre_name_zh, play_count AS track_genre_count
		 FROM genre 
		 WHERE name != ''
		 ORDER BY play_count DESC 
		 LIMIT ?`, limit,
	).Scan(&result).Error
	if err != nil {
		return nil, err
	}

	for index := range result {
		result[index].Rank = index + 1
	}

	return result, nil
}

/*// GetGenreCache returns the global genre cache instance
func GetGenreCache() *cache.GenreCache {
	return cache.GetGenreCache()
}
*/
