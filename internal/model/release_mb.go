package model

import (
	"context"
	"time"
)

// ReleaseMB stores the raw JSON data from MusicBrainz for a release
type ReleaseMB struct {
	ID        int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	MBID      string    `gorm:"column:mbid;type:varchar(255);not null;index:idx_release_mbid" json:"mbid"`
	AlbumID   int64     `gorm:"column:album_id;type:bigint;not null;index:idx_release_mbid_album" json:"album_id"`
	Name      string    `gorm:"column:name;type:varchar(255)" json:"name"`
	JSONData  string    `gorm:"column:json_data;type:longtext" json:"json_data"`
	CreatedAt time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName sets the table name for the ReleaseMB model
func (ReleaseMB) TableName() string {
	return "release_mb"
}

func SaveReleaseMB(ctx context.Context, r *ReleaseMB) error {
	var existing ReleaseMB
	err := GetDB().WithContext(ctx).Where("mbid = ? AND album_id = ?", r.MBID, r.AlbumID).First(&existing).Error
	if err == nil {
		// 存在则更新
		return GetDB().WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
			"name":      r.Name,
			"json_data": r.JSONData,
		}).Error
	}

	// 不存在则创建
	return GetDB().WithContext(ctx).Create(r).Error
}

func GetReleaseMBByMBID(ctx context.Context, albumID int64, mbid string) (*ReleaseMB, error) {
	var r ReleaseMB
	err := GetDB().WithContext(ctx).Where("album_id = ? AND mbid = ?", albumID, mbid).First(&r).Error
	return &r, err
}

func GetReleasesByAlbumID(ctx context.Context, albumID int64) ([]*ReleaseMB, error) {
	var results []*ReleaseMB
	err := GetDB().WithContext(ctx).Where("album_id = ?", albumID).Find(&results).Error
	return results, err
}
