package model

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/config"
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
}

// TableName sets the table name for the Genre model
func (Genre) TableName() string {
	return "genre"
}

// CreateGenre creates a new genre
func CreateGenre(ctx context.Context, genre *Genre) error {
	log.Debug(ctx, "creating genre", zap.Any("genre", genre))
	err := GetDB().WithContext(ctx).Create(genre).Error
	if err == nil {
		// Trigger cache refresh when a new genre is created
		go func() {
			// In a real implementation, we would update the cache with the new genre data
			// For now, we'll just trigger a full refresh
			// todo
			// _ = GetGenreCache().RefreshFromDB(ctx)
		}()
	}
	return err
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
	err := GetDB().WithContext(ctx).Save(genre).Error
	if err == nil {
		// Trigger cache refresh when a genre is updated
		go func() {
			// In a real implementation, we would update the cache with the modified genre data
			// For now, we'll just trigger a full refresh
			// todo
			// _ = GetGenreCache().RefreshFromDB(ctx)
		}()
	}
	return err
}

// DeleteGenre deletes a genre
func DeleteGenre(ctx context.Context, id uint) error {
	err := GetDB().WithContext(ctx).Delete(&Genre{}, id).Error
	if err == nil {
		// Trigger cache refresh when a genre is deleted
		go func() {
			// In a real implementation, we would remove the deleted genre from the cache
			// For now, we'll just trigger a full refresh
			// _ = GetGenreCache().RefreshFromDB(ctx)
		}()
	}
	return err
}

// IncrementGenrePlayCount increments the play count for a genre
func IncrementGenrePlayCount(ctx context.Context, name string) error {
	return GetDB().WithContext(ctx).Model(&Genre{}).Where("name = ?", name).Update(
		"play_count", gorm.Expr("play_count + 1"),
	).Error
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
	var result []*TopGenre

	// 根据数据库类型使用不同的SQL语法
	if config.ConfigObj.Database.Type == string(common.DatabaseTypeMySQL) {
		err := GetDB().WithContext(ctx).Raw(
			`select tg.track_genre_name, tg.track_genre_count, g.name_zh as genre_name_zh, g.play_count as genre_count
			from (select genre as track_genre_name, sum(play_count) as track_genre_count
				  from track
				  -- where genre != ''
				  group by genre
				  order by track_genre_count desc limit 20) as tg
				 inner join genre as g on tg.track_genre_name = g.name where tg.track_genre_name!='' limit ?`, limit,
		).Scan(&result).Error
		if err != nil {
			return nil, err
		}
	} else {
		err := GetDB().WithContext(ctx).Raw(
			`
			select tg.track_genre_name, tg.track_genre_count, g.name_zh as genre_name_zh, g.play_count as genre_count
			from (select genre as track_genre_name, sum(play_count) as track_genre_count
				  from track
				  where genre != ''
				  group by genre
				  order by track_genre_count desc limit ?) as tg
				 left join genre as g on tg.track_genre_name = g.name`, limit,
		).Scan(&result).Error
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

/*// GetGenreCache returns the global genre cache instance
func GetGenreCache() *cache.GenreCache {
	return cache.GetGenreCache()
}
*/
