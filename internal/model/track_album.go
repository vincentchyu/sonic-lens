package model

import (
	"context"
	"time"
)

// TrackAlbum represents the relationship between a track and an album
type TrackAlbum struct {
	ID                     int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	TrackID                int64     `gorm:"column:track_id;type:bigint;not null;index:idx_ta_track_album" json:"track_id"`
	AlbumID                int64     `gorm:"column:album_id;type:bigint;not null;index:idx_ta_track_album;index:idx_ta_album_id" json:"album_id"`
	TrackNumber            int8      `gorm:"column:track_number;type:tinyint" json:"track_number"`
	DiscNumber             int8      `gorm:"column:disc_number;type:tinyint;default:1" json:"disc_number"`    // 碟号
	MusicBrainzRecordingID string    `gorm:"column:mb_recording_id;type:varchar(255)" json:"mb_recording_id"` // MusicBrainz Recording ID 冗余
	Track                  string    `gorm:"column:track;type:varchar(255)" json:"track"`                     // track 冗余
	CreatedAt              time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt              time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName sets the table name for the TrackAlbum model
func (TrackAlbum) TableName() string {
	return "track_album"
}

func GetOrCreateTrackAlbum(ctx context.Context, ta *TrackAlbum) error {
	return GetDB().WithContext(ctx).Where(
		"track_id = ? AND album_id = ?", ta.TrackID, ta.AlbumID,
	).FirstOrCreate(ta).Error
}

func GetTrackAlbumsByAlbum(ctx context.Context, albumID int64) ([]*TrackAlbum, error) {
	var results []*TrackAlbum
	err := GetDB().WithContext(ctx).Where("album_id = ?", albumID).Find(&results).Error
	return results, err
}
