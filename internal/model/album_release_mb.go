package model

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// AlbumReleaseMB represents the confirmation link between an album and a MusicBrainz release
type AlbumReleaseMB struct {
	ID          int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	AlbumID     int64     `gorm:"column:album_id;type:bigint;not null;uniqueIndex:uidx_album_release" json:"album_id"`
	ReleaseMBID int64     `gorm:"column:release_mb_id;type:bigint;not null;uniqueIndex:uidx_album_release" json:"release_mb_id"`
	MBID        string    `gorm:"column:mbid;type:varchar(255);not null" json:"mbid"`
	Confirmed   bool      `gorm:"column:confirmed;type:tinyint(1);default:1" json:"confirmed"`
	CreatedAt   time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
}

// TableName sets the table name for the AlbumReleaseMB model
func (AlbumReleaseMB) TableName() string {
	return "album_release_mb"
}

// LinkAlbumToMBID 在关联精选结果后写入确认记录并同步专辑状态。
func LinkAlbumToMBID(ctx context.Context, albumID int64, releaseMBID int64, mbid string) error {
	return InTx(
		ctx, func(tx *gorm.DB) error {
			return LinkAlbumToMBIDTx(tx, albumID, releaseMBID, mbid)
		},
	)
}

// LinkAlbumToMBIDTx 在事务内写入专辑与 MusicBrainz 发行版的确认关系。
func LinkAlbumToMBIDTx(tx *gorm.DB, albumID int64, releaseMBID int64, mbid string) error {
	link := &AlbumReleaseMB{
		AlbumID:     albumID,
		ReleaseMBID: releaseMBID,
		MBID:        mbid,
		Confirmed:   true,
	}
	err := tx.Where("album_id = ? AND release_mb_id = ?", albumID, releaseMBID).FirstOrCreate(link).Error
	if err == nil {
		// 关联成功后，将专辑状态更新为 2 (初选关联完成)
		err = UpdateAlbumSyncStatusTx(tx, albumID, 2)
	}
	return err
}

// DeleteAlbumReleaseMBByAlbumID 删除专辑对应的已确认发行版关联。
func DeleteAlbumReleaseMBByAlbumID(ctx context.Context, albumID int64) error {
	return DeleteAlbumReleaseMBByAlbumIDTx(GetDB().WithContext(ctx), albumID)
}

// DeleteAlbumReleaseMBByAlbumIDTx 在事务内删除专辑对应的已确认发行版关联。
func DeleteAlbumReleaseMBByAlbumIDTx(tx *gorm.DB, albumID int64) error {
	return tx.Where("album_id = ?", albumID).Delete(&AlbumReleaseMB{}).Error
}

// GetAlbumReleaseMBByAlbumID 获取专辑当前确认的 MusicBrainz 发行版关联。
func GetAlbumReleaseMBByAlbumID(ctx context.Context, albumID int64) (*AlbumReleaseMB, error) {
	var results AlbumReleaseMB
	err := GetDB().WithContext(ctx).Where("album_id = ?", albumID).First(&results).Error
	return &results, err
}
