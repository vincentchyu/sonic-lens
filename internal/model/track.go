package model

import (
	"context"
	"errors"
	"sort"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/log"
)

// Track represents a music track with play statistics and favorite status
/*type Track struct {
	ID              int64     `gorm:"primaryKey" json:"id"`
	Artist          string    `gorm:"index;uniqueIndex:uidx_artist_album_track" json:"artist"`
	AlbumArtist     string    `gorm:"index" json:"album_artist"` // 专辑艺术家
	Album           string    `gorm:"index;uniqueIndex:uidx_artist_album_track" json:"album"`
	Track           string    `gorm:"index;uniqueIndex:uidx_artist_album_track" json:"track"` // 歌曲名称
	TrackNumber     int8      `json:"track_number"`                                           // 曲目编号
	Duration        int64     `json:"duration"`                                               // 持续时间(秒)
	Genre           string    `gorm:"index" json:"genre"`                                     // 流派
	Composer        string    `json:"composer"`                                               // 作曲家
	ReleaseDate     string    `json:"release_date"`                                           // 发布日期
	MusicBrainzID   string    `gorm:"column:music_brainz_id;index" json:"musicbrainz_id"`     // MusicBrainz ID
	PlayCount       int       `json:"play_count"`                                             // 播放次数
	IsAppleMusicFav bool      `json:"is_apple_music_fav"`                                     // 是否Apple Music喜欢
	IsLastFmFav     bool      `gorm:"column:is_last_fm_fav" json:"is_lastfm_fav"`             // 是否Last.fm喜欢
	Source          string    `gorm:"index" json:"source"`                                    // 数据来源：Apple Music, Audirvana, Roon等
	BundleID        string    `json:"bundle_id"`                                              // 应用标识符 (用于media-control)
	UniqueID        string    `gorm:"index" json:"unique_id"`                                 // 唯一标识符 (用于media-control)
	Version         int       `gorm:"default:1" json:"version"`                               // 乐观锁版本号
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}*/
type Track struct {
	ID              int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	Artist          string    `gorm:"column:artist;type:varchar(255);not null;uniqueIndex:uidx_artist_album_track" json:"artist"`
	Album           string    `gorm:"column:album;type:varchar(255);not null;index:idx_track_album;uniqueIndex:uidx_artist_album_track" json:"album"`
	Track           string    `gorm:"column:track;type:varchar(255);not null;index:idx_track_track;uniqueIndex:uidx_artist_album_track" json:"track"`
	PlayCount       int       `gorm:"column:play_count;type:int;default:0" json:"play_count"`
	IsAppleMusicFav bool      `gorm:"column:is_apple_music_fav;type:tinyint(1);default:0" json:"is_apple_music_fav"`
	IsLastFmFav     bool      `gorm:"column:is_last_fm_fav;type:tinyint(1);default:0" json:"is_last_fm_fav"`
	Version         int       `gorm:"column:version;type:int;default:1" json:"version"`
	AlbumArtist     string    `gorm:"column:album_artist;type:varchar(255)" json:"album_artist"`
	TrackNumber     int8      `gorm:"column:track_number;type:tinyint" json:"track_number"`
	Duration        int64     `gorm:"column:duration;type:int" json:"duration"`
	Genre           string    `gorm:"column:genre;type:varchar(255);index:idx_track_genre" json:"genre"`
	Composer        string    `gorm:"column:composer;type:varchar(255)" json:"composer"`
	ReleaseDate     string    `gorm:"column:release_date;type:varchar(50)" json:"release_date"`
	MusicBrainzID   string    `gorm:"column:music_brainz_id;type:varchar(255)" json:"music_brainz_id"`
	Source          string    `gorm:"column:source;type:varchar(255);index:idx_track_source" json:"source"`
	BundleID        string    `gorm:"column:bundle_id;type:varchar(255)" json:"bundle_id"`
	UniqueID        string    `gorm:"column:unique_id;type:varchar(255);index:idx_track_unique_id" json:"unique_id"`
	CreatedAt       time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName sets the table name for the Track model
func (Track) TableName() string {
	return "track"
}

// TrackMetadata represents metadata for a music track
type TrackMetadata struct {
	AlbumArtist   string `json:"album_artist"`   // 专辑艺术家
	TrackNumber   int8   `json:"track_number"`   // 曲目编号
	Duration      int64  `json:"duration"`       // 持续时间(秒)
	Genre         string `json:"genre"`          // 流派
	Composer      string `json:"composer"`       // 作曲家
	ReleaseDate   string `json:"release_date"`   // 发布日期
	MusicBrainzID string `json:"musicbrainz_id"` // MusicBrainz ID
	Source        string `json:"source"`         // 数据来源：Apple Music, Audirvana, Roon等
	BundleID      string `json:"bundle_id"`      // 应用标识符 (用于media-control)
	UniqueID      string `json:"unique_id"`      // 唯一标识符 (用于media-control)
}

// IncrementTrackPlayCountParams represents parameters for IncrementTrackPlayCount function
type IncrementTrackPlayCountParams struct {
	Ctx           context.Context
	Artist        string
	Album         string
	Track         string
	TrackMetadata TrackMetadata
}

// SetFavoriteParams represents parameters for SetAppleMusicFavorite and SetLastFmFavorite functions
type SetFavoriteParams struct {
	Ctx           context.Context
	Artist        string
	Album         string
	Track         string
	IsFavorite    bool
	TrackMetadata TrackMetadata
}

// IncrementTrackPlayCount increments the play count for a track
func IncrementTrackPlayCount(params IncrementTrackPlayCountParams) error {
	// 验证艺术家、专辑和曲目信息
	if err := common.ValidateTrackInfo(params.Ctx, params.Artist, params.Album, params.Track); err != nil {
		return err
	}

	// 检查流派信息
	if params.TrackMetadata.Genre != "" {
		genreByName, err := GetGenreByName(params.Ctx, params.TrackMetadata.Genre)
		if err != nil {
			log.Warn(
				params.Ctx, "Error getting genre by name", zap.String("genre", params.TrackMetadata.Genre),
				zap.Error(err),
			)
			return err
		}
		if genreByName == nil {
			err := CreateGenre(
				params.Ctx, &Genre{
					Name:      params.TrackMetadata.Genre,
					NameZh:    "",
					Extra:     "",
					PlayCount: 0,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			)
			if err != nil {
				log.Warn(params.Ctx, "CreateGenre", zap.String("genre", params.TrackMetadata.Genre), zap.Error(err))
			}
		}
	}
	// 使用乐观锁机制更新播放次数
	for {
		var record Track
		err := GetDB().WithContext(params.Ctx).Where(
			"artist = ? AND album = ? AND track = ?", params.Artist, params.Album, params.Track,
		).First(&record).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Create new record
				record = Track{

					Artist:        params.Artist,
					AlbumArtist:   params.TrackMetadata.AlbumArtist,
					Album:         params.Album,
					Track:         params.Track,
					TrackNumber:   params.TrackMetadata.TrackNumber,
					Duration:      params.TrackMetadata.Duration,
					Genre:         params.TrackMetadata.Genre,
					Composer:      params.TrackMetadata.Composer,
					ReleaseDate:   params.TrackMetadata.ReleaseDate,
					MusicBrainzID: params.TrackMetadata.MusicBrainzID,
					Source:        params.TrackMetadata.Source,
					BundleID:      params.TrackMetadata.BundleID,
					UniqueID:      params.TrackMetadata.UniqueID,
					PlayCount:     1,
				}
				err = GetDB().WithContext(params.Ctx).Create(&record).Error
				if err != nil && !errors.Is(err, gorm.ErrDuplicatedKey) {
					return err
				}
				// 如果出现重复键错误，说明其他goroutine已经创建了记录，继续循环处理
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					continue
				}
				return nil
			}
			return err
		}

		// Update existing record with optimistic locking
		updatedRecord := Track{
			PlayCount: record.PlayCount + 1,
			Version:   record.Version + 1,
			// mata
			TrackNumber:   record.TrackNumber,
			Duration:      record.Duration,
			Genre:         record.Genre,
			Composer:      record.Composer,
			ReleaseDate:   record.ReleaseDate,
			MusicBrainzID: record.MusicBrainzID,
			Source:        record.Source,
			BundleID:      record.BundleID,
			UniqueID:      record.UniqueID,
		}

		// 更新meta信息
		UpdateTrackWithTrackMetadata(&updatedRecord, &params.TrackMetadata)

		result := GetDB().WithContext(params.Ctx).Where(
			"artist = ? AND album = ? AND track = ? AND version = ?",
			params.Artist, params.Album, params.Track, record.Version,
		).Updates(&updatedRecord)

		if result.Error != nil {
			return result.Error
		}

		// 如果更新成功，跳出循环
		if result.RowsAffected > 0 {
			break
		}
		// 如果更新失败（版本号不匹配），继续循环重试
	}

	return nil
}

// SetAppleMusicFavorite updates the Apple Music favorite status for a track
func SetAppleMusicFavorite(params SetFavoriteParams) error {
	// 验证艺术家、专辑和曲目信息
	if err := common.ValidateTrackInfo(params.Ctx, params.Artist, params.Album, params.Track); err != nil {
		return err
	}

	// 使用乐观锁机制更新喜欢状态
	for {
		var record Track
		err := GetDB().WithContext(params.Ctx).Where(
			"artist = ? AND album = ? AND track = ?", params.Artist, params.Album, params.Track,
		).First(&record).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Create new record
				record = Track{
					Artist:          params.Artist,
					AlbumArtist:     params.TrackMetadata.AlbumArtist,
					Album:           params.Album,
					Track:           params.Track,
					TrackNumber:     params.TrackMetadata.TrackNumber,
					Duration:        params.TrackMetadata.Duration,
					Genre:           params.TrackMetadata.Genre,
					Composer:        params.TrackMetadata.Composer,
					ReleaseDate:     params.TrackMetadata.ReleaseDate,
					MusicBrainzID:   params.TrackMetadata.MusicBrainzID,
					Source:          params.TrackMetadata.Source,
					BundleID:        params.TrackMetadata.BundleID,
					UniqueID:        params.TrackMetadata.UniqueID,
					PlayCount:       0,
					IsAppleMusicFav: params.IsFavorite,
				}
				err = GetDB().WithContext(params.Ctx).Create(&record).Error
				if err != nil && !errors.Is(err, gorm.ErrDuplicatedKey) {
					return err
				}
				// 如果出现重复键错误，说明其他goroutine已经创建了记录，继续循环处理
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					continue
				}
				return nil
			}
			return err
		}

		// Update existing record with optimistic locking
		updatedRecord := Track{
			IsAppleMusicFav: params.IsFavorite,
			Version:         record.Version + 1,
		}
		UpdateTrackWithTrackMetadata(&record, &params.TrackMetadata)

		result := GetDB().WithContext(params.Ctx).Where(
			"artist = ? AND album = ? AND track = ? AND version = ?",
			params.Artist, params.Album, params.Track, record.Version,
		).Updates(&updatedRecord)

		if result.Error != nil {
			return result.Error
		}

		// 如果更新成功，跳出循环
		if result.RowsAffected > 0 {
			break
		}
		// 如果更新失败（版本号不匹配），继续循环重试
	}

	return nil
}

// SetLastFmFavorite updates the Last.fm favorite status for a track
func SetLastFmFavorite(params SetFavoriteParams) error {
	// 验证艺术家、专辑和曲目信息
	if err := common.ValidateTrackInfo(params.Ctx, params.Artist, params.Album, params.Track); err != nil {
		return err
	}

	// 使用乐观锁机制更新喜欢状态
	for {
		var record Track
		err := GetDB().WithContext(params.Ctx).Where(
			"artist = ? AND album = ? AND track = ?", params.Artist, params.Album, params.Track,
		).First(&record).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Create new record
				record = Track{
					Artist:        params.Artist,
					AlbumArtist:   params.TrackMetadata.AlbumArtist,
					Album:         params.Album,
					Track:         params.Track,
					TrackNumber:   params.TrackMetadata.TrackNumber,
					Duration:      params.TrackMetadata.Duration,
					Genre:         params.TrackMetadata.Genre,
					Composer:      params.TrackMetadata.Composer,
					ReleaseDate:   params.TrackMetadata.ReleaseDate,
					MusicBrainzID: params.TrackMetadata.MusicBrainzID,
					Source:        params.TrackMetadata.Source,
					BundleID:      params.TrackMetadata.BundleID,
					UniqueID:      params.TrackMetadata.UniqueID,
					PlayCount:     0,
					IsLastFmFav:   params.IsFavorite,
				}
				err = GetDB().WithContext(params.Ctx).Create(&record).Error
				if err != nil && !errors.Is(err, gorm.ErrDuplicatedKey) {
					return err
				}
				// 如果出现重复键错误，说明其他goroutine已经创建了记录，继续循环处理
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					continue
				}
				return nil
			}
			return err
		}

		// Update existing record with optimistic locking
		updatedRecord := Track{
			IsLastFmFav: params.IsFavorite,
			Version:     record.Version + 1,
		}
		UpdateTrackWithTrackMetadata(&record, &params.TrackMetadata)

		result := GetDB().WithContext(params.Ctx).Where(
			"artist = ? AND album = ? AND track = ? AND version = ?",
			params.Artist, params.Album, params.Track, record.Version,
		).Updates(&updatedRecord)

		if result.Error != nil {
			return result.Error
		}

		// 如果更新成功，跳出循环
		if result.RowsAffected > 0 {
			break
		}
		// 如果更新失败（版本号不匹配），继续循环重试
	}

	return nil
}

// GetTracks retrieves track play counts with pagination
func GetTracks(ctx context.Context, limit, offset int) ([]*Track, error) {
	var records []*Track
	err := GetDB().WithContext(ctx).Order("play_count DESC").Limit(limit).Offset(offset).Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}

// GetTrackCounts returns the total number of tracks
func GetTrackCounts(ctx context.Context) (int64, error) {
	var count int64
	err := GetDB().WithContext(ctx).Model(&Track{}).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetTrack retrieves a specific track's play count
func GetTrack(ctx context.Context, artist, album, track string) (*Track, error) {
	var record Track
	err := GetDB().WithContext(ctx).Where(
		"artist = ? AND album = ? AND track = ?", artist, album, track,
	).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetAllTrackPlayCounts retrieves all track play counts
func GetAllTrackPlayCounts(ctx context.Context) ([]*Track, error) {
	var allTracks []*Track
	pageSize := 100
	offset := 0

	for {
		var tracks []*Track
		err := GetDB().WithContext(ctx).Order("play_count DESC").Limit(pageSize).Offset(offset).Find(&tracks).Error
		if err != nil {
			return nil, err
		}

		allTracks = append(allTracks, tracks...)

		// 如果返回的记录数少于pageSize，说明已经获取完所有记录
		if len(tracks) < pageSize {
			break
		}

		offset += pageSize
	}

	return allTracks, nil
}

// GetTracksByArtist retrieves all tracks by a specific artist
func GetTracksByArtist(ctx context.Context, artist string) ([]*Track, error) {
	var tracks []*Track
	err := GetDB().WithContext(ctx).Where("artist LIKE ?", "%"+artist+"%").Find(&tracks).Error
	if err != nil {
		return nil, err
	}
	return tracks, nil
}

// GetTotalPlayCount returns the total play count across all tracks
func GetTotalPlayCount(ctx context.Context) (int64, error) {
	var total int64
	err := GetDB().WithContext(ctx).Model(&Track{}).Select("SUM(play_count)").Scan(&total).Error
	if err != nil {
		return 0, err
	}
	return total, nil
}

// GetArtistCounts returns the total number of unique artists
func GetArtistCounts(ctx context.Context) (int64, error) {
	var count int64
	err := GetDB().WithContext(ctx).Model(&Track{}).Distinct("artist").Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetAlbumCounts returns the total number of unique albums
func GetAlbumCounts(ctx context.Context) (int64, error) {
	var count int64
	err := GetDB().WithContext(ctx).Model(&Track{}).Distinct("album").Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetTopArtistsByPlayCount returns the top artists by play count
func GetTopArtistsByPlayCount(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	err := GetDB().WithContext(ctx).Model(&Track{}).
		Select("artist, SUM(play_count) as play_count").
		Group("artist").
		Order("SUM(play_count) DESC").
		Limit(limit).
		Find(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetTopArtistsByTrackCount returns the top artists by track count
func GetTopArtistsByTrackCount(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	err := GetDB().WithContext(ctx).Model(&Track{}).
		Select("artist, COUNT(*) as track_count").
		Group("artist").
		Order("COUNT(*) DESC").
		Limit(limit).
		Find(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetTracksByPeriod retrieves track play counts for a specific period
func GetTracksByPeriod(ctx context.Context, limit int, offset int, period string) ([]*Track, error) {
	// 计算时间范围
	var startTime time.Time
	switch period {
	case "week":
		startTime = time.Now().AddDate(0, 0, -7)
	case "month":
		startTime = time.Now().AddDate(0, -1, 0)
	default:
		// 默认返回所有时间的数据
		return GetTracks(ctx, limit, offset)
	}

	// 先获取指定时间范围内的播放记录
	var playRecords []*TrackPlayRecord
	err := GetDB().WithContext(ctx).Where(
		"play_time >= ?", startTime,
	).Order("").Find(&playRecords).Error
	if err != nil {
		return nil, err
	}

	// 统计每个曲目的播放次数
	trackCountMap := make(map[string]*Track)
	for _, record := range playRecords {
		key := record.Artist + "|" + record.Album + "|" + record.Track
		if trackCount, exists := trackCountMap[key]; exists {
			trackCount.PlayCount++
		} else {
			trackCountMap[key] = &Track{
				Artist:    record.Artist,
				Album:     record.Album,
				Track:     record.Track,
				PlayCount: 1,
			}
		}
	}

	// 转换为切片并排序
	var trackCounts []*Track
	for _, trackCount := range trackCountMap {
		trackCounts = append(trackCounts, trackCount)
	}

	// 按播放次数排序
	sort.Slice(
		trackCounts, func(i, j int) bool {
			return trackCounts[i].PlayCount > trackCounts[j].PlayCount
		},
	)

	// 应用分页
	start := offset
	end := offset + limit
	if start >= len(trackCounts) {
		return []*Track{}, nil
	}
	if end > len(trackCounts) {
		end = len(trackCounts)
	}

	return trackCounts[start:end], nil
}

// GetAppleMusicFavorite retrieves the Apple Music favorite status for a track
func GetAppleMusicFavorite(ctx context.Context, artist, album, track string) (bool, error) {
	record, err := GetTrack(ctx, artist, album, track)
	if err != nil {
		return false, err
	}
	return record.IsAppleMusicFav, nil
}

// GetLastFmFavorite retrieves the Last.fm favorite status for a track
func GetLastFmFavorite(ctx context.Context, artist, album, track string) (bool, error) {
	record, err := GetTrack(ctx, artist, album, track)
	if err != nil {
		return false, err
	}
	return record.IsLastFmFav, nil
}

func UpdateTrackWithTrackMetadata(track *Track, newTrack *TrackMetadata) {
	if track == nil || newTrack == nil {
		return
	}

	// Update fields that might be missing from exiftool but available in media control
	if track.Duration == 0 && newTrack.Duration > 0 {
		track.Duration = newTrack.Duration
	}

	if track.AlbumArtist == "" && newTrack.AlbumArtist != "" {
		track.AlbumArtist = newTrack.AlbumArtist
	}

	if track.TrackNumber == 0 && newTrack.TrackNumber > 0 {
		track.TrackNumber = newTrack.TrackNumber
	}

	if track.MusicBrainzID == "" && newTrack.MusicBrainzID != "" {
		track.MusicBrainzID = newTrack.MusicBrainzID
	}

	if track.Genre == "" && newTrack.Genre != "" {
		track.Genre = newTrack.Genre
	}

	if track.ReleaseDate == "" && newTrack.ReleaseDate != "" {
		track.ReleaseDate = newTrack.ReleaseDate
	}

	if track.Composer == "" && newTrack.Composer != "" {
		track.Composer = newTrack.Composer
	}

	if track.BundleID == "" && newTrack.BundleID != "" {
		track.BundleID = newTrack.BundleID
	}

	if track.UniqueID == "" && newTrack.UniqueID != "" {
		track.UniqueID = newTrack.UniqueID
	}

	// Update source if not set
	if track.Source == "" && newTrack.Source != "" {
		track.Source = newTrack.Source
	}
}
