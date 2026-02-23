package model

import (
	"context"
	"time"
)

// TrackLyrics 存储歌曲歌词数据
type TrackLyrics struct {
	ID             int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	TrackID        int64     `gorm:"column:track_id;type:bigint;index" json:"track_id"`
	Artist         string    `gorm:"column:artist;type:varchar(255);uniqueIndex:idx_lyrics_artist_album_track" json:"artist"`
	Album          string    `gorm:"column:album;type:varchar(255);uniqueIndex:idx_lyrics_artist_album_track" json:"album"`
	Track          string    `gorm:"column:track;type:varchar(255);uniqueIndex:idx_lyrics_artist_album_track" json:"track"`
	LyricsOriginal string    `gorm:"column:lyrics_original;type:text" json:"lyrics_original"`
	LyricsSource   string    `gorm:"column:lyrics_source;type:varchar(64)" json:"lyrics_source"`
	LangCode       string    `gorm:"column:lang_code;type:varchar(16)" json:"lang_code"`
	Synced         bool      `gorm:"column:synced;type:tinyint(1);default:0" json:"synced"`
	CreatedAt      time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName 自定义表名
func (TrackLyrics) TableName() string {
	return "track_lyrics"
}

// GetTrackLyrics 查询歌词
func GetTrackLyrics(ctx context.Context, artist, album, track string) (*TrackLyrics, error) {
	var lyrics TrackLyrics
	err := GetDB().WithContext(ctx).
		Where("artist = ? AND album = ? AND track = ?", artist, album, track).
		First(&lyrics).Error
	if err != nil {
		return nil, err
	}
	return &lyrics, nil
}

// CreateTrackLyrics 创建歌词记录
func CreateTrackLyrics(ctx context.Context, lyrics *TrackLyrics) error {
	return GetDB().WithContext(ctx).Create(lyrics).Error
}

// UpdateTrackLyrics 更新歌词
func UpdateTrackLyrics(ctx context.Context, lyrics *TrackLyrics) error {
	return GetDB().WithContext(ctx).Save(lyrics).Error
}

// GetOrCreateTrackLyrics 获取或创建歌词记录(用于并发安全的获取)
func GetOrCreateTrackLyrics(ctx context.Context, lyrics *TrackLyrics) (*TrackLyrics, error) {
	existing, err := GetTrackLyrics(ctx, lyrics.Artist, lyrics.Album, lyrics.Track)
	if err == nil {
		return existing, nil
	}

	// 如果不存在,创建新记录
	if err := CreateTrackLyrics(ctx, lyrics); err != nil {
		// 可能是并发插入导致的唯一索引冲突,再次查询
		existing, err = GetTrackLyrics(ctx, lyrics.Artist, lyrics.Album, lyrics.Track)
		if err != nil {
			return nil, err
		}
		return existing, nil
	}

	return lyrics, nil
}
