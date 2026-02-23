package model

import (
	"context"
	"time"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/config"
)

// 索引优化建议:
// 1. 对于按时间范围查询的场景，建议在 play_time 字段上创建索引
//    例如: GetRecentPlayRecordsByDays 函数会按 play_time 进行筛选和排序
// 2. 对于按艺术家和专辑查询的场景，建议创建复合索引 (artist, album)
//    这可以优化同时按艺术家和专辑筛选的查询性能
// 3. 对于按来源和同步状态查询的场景，建议创建复合索引 (source, scrobbled)
//    这可以优化按来源筛选未同步记录的查询性能
// 4. 对于按专辑艺术家和艺术家查询的场景，建议创建复合索引 (album_artist, artist)
//    这可以优化同时按专辑艺术家和艺术家筛选的查询性能
/*
     优化策略                                                                                                  │
 │                                                                                                              │
 │     1. 避免在查询字段上使用函数：                                                                            │
 │       最佳实践是重写查询条件，避免在 play_time 字段上使用函数。可以将查询改写为：                            │
 │     1    SELECT * FROM `track_play_records`                                                                  │
 │     2    WHERE `play_time` > '2025-08-20 23:59:59'                                                           │
 │     3    ORDER BY play_time DESC;                                                                            │
 │       这样可以直接利用 play_time 字段上的索引。                                                              │
 │                                                                                                              │
 │     2. 添加合适的索引：                                                                                      │
 │       为 play_time 字段添加索引可以显著提高查询性能：                                                        │
 │     1    CREATE INDEX idx_track_play_records_play_time ON track_play_records (play_time);                    │
 │                                                                                                              │
 │       如果经常需要按日期范围查询并按来源(source)过滤，可以考虑创建复合索引：                                 │
 │     1    CREATE INDEX idx_track_play_records_play_time_source ON track_play_records (play_time,              │
 │       source);                                                                                               │
 │                                                                                                              │
 │     3. 函数索引（MySQL 8.0+）：                                                                              │
 │       如果您使用的是 MySQL 8.0 或更高版本，可以创建函数索引，直接针对 DATE_FORMAT 函数的结果建立索引：       │
 │     1    CREATE INDEX idx_track_play_records_play_time_date ON track_play_records                            │
 │       ((DATE_FORMAT(play_time, '%Y-%m-%d')));                                                                │
 │       这样即使在查询中使用 DATE_FORMAT 函数，也能利用索引。                                                  │
 │                                                                                                              │
 │     4. 考虑分区表：                                                                                          │
 │       如果数据量非常大且主要按日期查询，可以考虑按日期分区表，但这需要更复杂的表结构变更。                   │
 │                                                                                                              │
 │    实施建议                                                                                                  │
 │                                                                                                              │
 │     1. 首先添加 play_time 字段的索引：                                                                       │
 │     1    CREATE INDEX idx_track_play_records_play_time ON track_play_records (play_time);                    │
 │                                                                                                              │
 │     2. 修改查询语句，避免在 play_time 字段上使用函数：                                                       │
 │     1    SELECT * FROM `track_play_records`                                                                  │
 │     2    WHERE `play_time` > '2025-08-20 23:59:59'                                                           │
 │     3    ORDER BY play_time DESC;                                                                            │
 │                                                                                                              │
 │     3. 如果使用 MySQL 8.0+，可以考虑添加函数索引以支持现有查询模式：                                         │
 │     1    CREATE INDEX idx_track_play_records_play_time_date ON track_play_records                            │
 │       ((DATE_FORMAT(play_time, '%Y-%m-%d')));                                                                │
 │                                                                                                              │
 │    通过这些优化，查询性能应该会得到显著提升，避免全表扫描的问题。
*/

// PlayTrendData represents data for play trend visualization
type PlayTrendData struct {
	Date  string `json:"date"`  // 日期
	Count int    `json:"count"` // 播放次数
	Size  int    `json:"size"`  // 气泡大小（可以和count相同，或者根据其他因素计算）
}

// HourlyPlayTrendData represents hourly play trend data for a specific date
type HourlyPlayTrendData struct {
	Date   string      `json:"date"`   // 日期
	Total  int         `json:"total"`  // 当日总播放次数
	Hourly map[int]int `json:"hourly"` // 按小时统计的播放次数，key为小时(0-23)，value为播放次数
}

// TrackPlayRecord 对应 track_play_records 表
type TrackPlayRecord struct {
	ID            int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	Artist        string    `gorm:"column:artist;type:varchar(255);not null;index:idx_track_play_records_artist" json:"artist"`
	AlbumArtist   string    `gorm:"column:album_artist;type:varchar(255)" json:"album_artist"`
	Track         string    `gorm:"column:track;type:varchar(255);not null" json:"track"`
	Album         string    `gorm:"column:album;type:varchar(255);not null" json:"album"`
	Duration      int64     `gorm:"column:duration;type:int" json:"duration"`
	PlayTime      time.Time `gorm:"column:play_time;type:timestamp;not null;default:CURRENT_TIMESTAMP" json:"play_time"`
	Scrobbled     bool      `gorm:"column:scrobbled;type:tinyint(1);not null;default:0;index:idx_track_play_records_scrobbled" json:"scrobbled"`
	MusicBrainzID string    `gorm:"column:music_brainz_id;type:varchar(255)" json:"music_brainz_id"`
	TrackNumber   int8      `gorm:"column:track_number;type:tinyint" json:"track_number"`
	Source        string    `gorm:"column:source;type:varchar(100);not null;index:idx_track_play_records_source" json:"source"`
	CreatedAt     time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName sets the table name for the TrackPlayRecord model
func (TrackPlayRecord) TableName() string {
	return "track_play_records"
}

func InsertTrackPlayRecord(ctx context.Context, record *TrackPlayRecord) error {
	// 验证记录中的艺术家、专辑和曲目信息
	if err := common.ValidateTrackInfo(ctx, record.Artist, record.Album, record.Track); err != nil {
		return err
	}

	return GetDB().WithContext(ctx).Create(record).Error
}

func UpdateScrobbledStatus(ctx context.Context, id int64, scrobbled bool) error {
	return GetDB().WithContext(ctx).Model(&TrackPlayRecord{}).Where("id = ?", id).Update("scrobbled", scrobbled).Error
}

func GetUnscrobbledRecords(ctx context.Context, limit int) ([]*TrackPlayRecord, error) {
	var trackPlayRecords []*TrackPlayRecord
	err := GetDB().WithContext(ctx).Where(
		"scrobbled = ?", false,
	).Order("play_time ASC").Limit(limit).Find(&trackPlayRecords).Error
	if err != nil {
		return nil, err
	}
	return trackPlayRecords, nil
}

// GetRecentPlayRecords 获取最近播放的记录
func GetRecentPlayRecords(ctx context.Context, limit int) ([]*TrackPlayRecord, error) {
	var records []*TrackPlayRecord
	err := GetDB().WithContext(ctx).Order("play_time DESC").Limit(limit).Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}

// GetRecentPlayRecordsByDays 获取指定天数内的播放记录
func GetRecentPlayRecordsByDays(ctx context.Context, days int) (map[string][]*TrackPlayRecord, error) {
	var records []*TrackPlayRecord
	// 计算从现在开始往前推指定天数的时间
	startTime := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	// 根据数据库类型使用不同的日期函数
	var err error
	if config.ConfigObj.Database.Type == string(common.DatabaseTypeMySQL) {
		err = GetDB().WithContext(ctx).Where(
			"DATE_FORMAT(`play_time`, '%Y-%m-%d') > ?", startTime,
		).Order("play_time DESC").Find(&records).Error
	} else {
		err = GetDB().WithContext(ctx).Where(
			"strftime('%Y-%m-%d',`play_time`) > ?", startTime,
		).Order("play_time DESC").Find(&records).Error
	}

	if err != nil {
		return nil, err
	}
	result := make(map[string][]*TrackPlayRecord, len(records))
	for _, data := range records {
		format := data.PlayTime.Format("2006-01-02")
		if _, ok := result[format]; !ok {
			result[format] = make([]*TrackPlayRecord, 0)
		}
		result[format] = append(result[format], data)
	}
	return result, nil
}

// GetUnscrobbledRecordsWithPagination 分页获取未同步到Last.fm的播放记录
func GetUnscrobbledRecordsWithPagination(ctx context.Context, limit, offset int) ([]*TrackPlayRecord, error) {
	var trackPlayRecords []*TrackPlayRecord
	err := GetDB().WithContext(ctx).Where(
		"scrobbled = ?", false,
	).Order("play_time ASC").Limit(limit).Offset(offset).Find(&trackPlayRecords).Error
	if err != nil {
		return nil, err
	}
	return trackPlayRecords, nil
}

// GetUnscrobbledRecordsCount 获取未同步到Last.fm的播放记录总数
func GetUnscrobbledRecordsCount(ctx context.Context) (int64, error) {
	var count int64
	err := GetDB().WithContext(ctx).Model(&TrackPlayRecord{}).Where("scrobbled = ?", false).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// BatchUpdateScrobbledStatus 批量更新播放记录的同步状态
func BatchUpdateScrobbledStatus(ctx context.Context, ids []int64, scrobbled bool) error {
	return GetDB().WithContext(ctx).Model(&TrackPlayRecord{}).Where("id IN ?", ids).Update("scrobbled", scrobbled).Error
}

// GetUnscrobbledRecordsByIds 通过ID列表获取未同步的播放记录
func GetUnscrobbledRecordsByIds(ctx context.Context, ids []int64) ([]*TrackPlayRecord, error) {
	// 获取指定ID的未同步记录
	var records []*TrackPlayRecord
	err := GetDB().WithContext(ctx).Where("id IN ? AND scrobbled = ?", ids, false).Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

// GetPlayCountsBySource 获取按来源统计的播放次数
func GetPlayCountsBySource(ctx context.Context) (map[string]int64, error) {
	var result []map[string]interface{}
	err := GetDB().WithContext(ctx).Model(&TrackPlayRecord{}).
		Select("source, COUNT(*) as count").
		Group("source").
		Find(&result).Error
	if err != nil {
		return nil, err
	}

	// 转换为map[string]int64
	sourceCounts := make(map[string]int64)
	for _, item := range result {
		if source, ok := item["source"].(string); ok {
			if count, ok := item["count"].(int64); ok {
				sourceCounts[source] = count
			} else if countFloat, ok := item["count"].(float64); ok {
				sourceCounts[source] = int64(countFloat)
			}
		}
	}

	return sourceCounts, nil
}

// GetTopAlbumsByPlayCount 获取按播放次数统计的热门专辑
type TopAlbum struct {
	Album     string `json:"album"`
	Artist    string `json:"artist"`
	PlayCount int    `json:"play_count"`
}

// GetTopAlbumsByPlayCount 获取按播放次数统计的热门专辑
func GetTopAlbumsByPlayCount(ctx context.Context, days int, limit int) ([]*TopAlbum, error) {
	var result []*TopAlbum

	// 计算时间范围
	var startTime time.Time
	if days > 0 {
		startTime = time.Now().AddDate(0, 0, -days)
	}

	// 构建查询
	query := GetDB().WithContext(ctx).Model(&TrackPlayRecord{})

	// 如果指定了时间范围，则添加时间条件
	if days > 0 {
		query = query.Where("DATE_FORMAT(`play_time`, '%Y-%m-%d') > ?", startTime.Format("2006-01-02"))
	}

	err := query.Select("album, MIN(artist) as artist, COUNT(album) as play_count").
		Group("album").
		Order("play_count DESC").
		Limit(limit).
		Find(&result).Error

	if err != nil {
		return nil, err
	}
	return result, nil
}
