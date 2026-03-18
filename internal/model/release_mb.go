package model

import (
	"context"
	"time"

	"gorm.io/gorm"
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
	return SaveReleaseMBTx(GetDB().WithContext(ctx), r)
}

// SaveReleaseMBTx 在事务内写入或更新 release_mb 缓存。
func SaveReleaseMBTx(tx *gorm.DB, r *ReleaseMB) error {
	var existing ReleaseMB
	err := tx.Where("mbid = ? AND album_id = ?", r.MBID, r.AlbumID).First(&existing).Error
	if err == nil {
		return tx.Model(&existing).Updates(map[string]interface{}{
			"name":      r.Name,
			"json_data": r.JSONData,
		}).Error
	}

	return tx.Create(r).Error
}

func GetReleaseMBByMBID(ctx context.Context, albumID int64, mbid string) (*ReleaseMB, error) {
	var r ReleaseMB
	err := GetDB().WithContext(ctx).Where("album_id = ? AND mbid = ?", albumID, mbid).First(&r).Error
	return &r, err
}

// UpdateReleaseMBJSONDataTx 在事务内刷新候选发行版缓存 JSON。
func UpdateReleaseMBJSONDataTx(tx *gorm.DB, albumID int64, mbid string, jsonData string) (bool, error) {
	var release ReleaseMB
	err := tx.Where("album_id = ? AND mbid = ?", albumID, mbid).First(&release).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}

	release.JSONData = jsonData
	if err := tx.Save(&release).Error; err != nil {
		return false, err
	}
	return true, nil
}

func GetReleasesByAlbumID(ctx context.Context, albumID int64) ([]*ReleaseMB, error) {
	var results []*ReleaseMB
	err := GetDB().WithContext(ctx).Where("album_id = ?", albumID).Find(&results).Error
	return results, err
}
