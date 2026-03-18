package model

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// Album represents a music album
type Album struct {
	ID          int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"column:name;type:varchar(255);not null;uniqueIndex:uidx_album_artist_name_release_date" json:"name"`
	Artist      string    `gorm:"column:artist;type:varchar(255);not null;uniqueIndex:uidx_album_artist_name_release_date" json:"artist"`
	ReleaseDate string    `gorm:"column:release_date;type:varchar(50);uniqueIndex:uidx_album_artist_name_release_date" json:"release_date"`
	Genre       string    `gorm:"column:genre;type:varchar(255)" json:"genre"`
	Country     string    `gorm:"column:country;type:varchar(50)" json:"country"`
	Status      string    `gorm:"column:status;type:varchar(50)" json:"status"`
	Packaging   string    `gorm:"column:packaging;type:varchar(50)" json:"packaging"`
	Barcode     string    `gorm:"column:barcode;type:varchar(255)" json:"barcode"`
	TotalDiscs  int       `gorm:"column:total_discs;type:int;default:1" json:"total_discs"`     // 总碟数
	DiscInfos   string    `gorm:"column:disc_infos;type:varchar(255)" json:"disc_infos"`        // 各碟信息(如 track counts)
	SyncStatus  int       `gorm:"column:sync_status;type:tinyint;default:0" json:"sync_status"` // 0:默认, 1:初选搜索完成, 2:初选关联完成, 3:精选维护完成
	CreatedAt   time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName sets the table name for the Album model
func (Album) TableName() string {
	return "album"
}

func (a *Album) AfterCreate(tx *gorm.DB) error {
	return appendLibraryChangeTx(tx, LibraryEntityAlbum, a.ID, LibraryOpUpsert)
}

func (a *Album) AfterUpdate(tx *gorm.DB) error {
	return appendLibraryChangeTx(tx, LibraryEntityAlbum, a.ID, LibraryOpUpsert)
}

func (a *Album) AfterDelete(tx *gorm.DB) error {
	return appendLibraryChangeTx(tx, LibraryEntityAlbum, a.ID, LibraryOpDelete)
}

// GetOrCreateAlbum gets an album by artist and name, or creates it if it doesn't exist
func GetOrCreateAlbum(ctx context.Context, album *Album) error {
	return getOrCreateAlbumTx(GetDB().WithContext(ctx), album)
}

func getOrCreateAlbumTx(db *gorm.DB, album *Album) error {
	var exact Album
	err := db.Where(
		"artist = ? AND name = ? AND release_date = ?", album.Artist, album.Name, album.ReleaseDate,
	).First(&exact).Error
	if err == nil {
		mergeAlbumFields(&exact, album)
		if err := db.Save(&exact).Error; err != nil {
			return err
		}
		*album = exact
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// 当外部客户端缺少发行日期时，优先复用已深度维护确认过的专辑，
	// 避免继续命中空发行日期占位记录，丢失精选维护沉淀的元数据。
	if album.ReleaseDate == "" {
		var curated Album
		err = db.Where("artist = ? AND name = ? AND sync_status = ?", album.Artist, album.Name, 3).
			Order("CASE WHEN release_date = '' OR release_date IS NULL THEN 1 ELSE 0 END ASC, id ASC").
			First(&curated).Error
		if err == nil {
			mergeAlbumFields(&curated, album)
			if err := db.Save(&curated).Error; err != nil {
				return err
			}
			*album = curated
			return nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}

	var fallback Album
	err = db.Where("artist = ? AND name = ?", album.Artist, album.Name).
		Order("CASE WHEN release_date = '' OR release_date IS NULL THEN 0 ELSE 1 END ASC, id ASC").
		First(&fallback).Error
	if err == nil {
		mergeAlbumFields(&fallback, album)
		if err := db.Save(&fallback).Error; err != nil {
			return err
		}
		*album = fallback
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return db.Create(album).Error
}

func mergeAlbumFields(target, source *Album) {
	if target.ReleaseDate == "" && source.ReleaseDate != "" {
		target.ReleaseDate = source.ReleaseDate
	}
	if target.Genre == "" && source.Genre != "" {
		target.Genre = source.Genre
	}
	if target.Country == "" && source.Country != "" {
		target.Country = source.Country
	}
	if target.Status == "" && source.Status != "" {
		target.Status = source.Status
	}
	if target.Packaging == "" && source.Packaging != "" {
		target.Packaging = source.Packaging
	}
	if target.Barcode == "" && source.Barcode != "" {
		target.Barcode = source.Barcode
	}
	if target.TotalDiscs == 0 && source.TotalDiscs > 0 {
		target.TotalDiscs = source.TotalDiscs
	}
	if target.DiscInfos == "" && source.DiscInfos != "" {
		target.DiscInfos = source.DiscInfos
	}
	if target.SyncStatus == 0 && source.SyncStatus > 0 {
		target.SyncStatus = source.SyncStatus
	}
}

func GetAlbum(ctx context.Context, id int64) (*Album, error) {
	var album Album
	err := GetDB().WithContext(ctx).First(&album, id).Error
	return &album, err
}

// UpdateAlbumSyncStatus 更新专辑同步状态，避免上层散落字段更新 SQL。
func UpdateAlbumSyncStatus(ctx context.Context, albumID int64, syncStatus int) error {
	return UpdateAlbumSyncStatusTx(GetDB().WithContext(ctx), albumID, syncStatus)
}

// UpdateAlbumSyncStatusTx 在事务内更新专辑同步状态。
func UpdateAlbumSyncStatusTx(tx *gorm.DB, albumID int64, syncStatus int) error {
	return tx.Model(&Album{}).Where("id = ?", albumID).Update("sync_status", syncStatus).Error
}

// UpdateAlbumFields 更新专辑元数据字段集合。
func UpdateAlbumFields(ctx context.Context, albumID int64, fields map[string]interface{}) error {
	return UpdateAlbumFieldsTx(GetDB().WithContext(ctx), albumID, fields)
}

// UpdateAlbumFieldsTx 在事务内批量更新专辑元数据字段集合。
func UpdateAlbumFieldsTx(tx *gorm.DB, albumID int64, fields map[string]interface{}) error {
	return tx.Model(&Album{}).Where("id = ?", albumID).Updates(fields).Error
}

func GetAlbumByArtistAndName(ctx context.Context, artist, albumName string) (*Album, error) {
	var album Album
	err := GetDB().WithContext(ctx).Where("artist = ? AND name = ?", artist, albumName).First(&album).Error
	return &album, err
}

// AlbumDetail 包含专辑及其关联的所有曲目信息，以及确认关联的 MusicBrainz 记录
type AlbumDetail struct {
	Album
	Tracks      []*Track        `json:"tracks"`
	TrackAlbums []*TrackAlbum   `json:"track_album"`
	ReleaseMB   *AlbumReleaseMB `json:"release_mb"`
}

// GetAlbumWithTracks 根据专辑 ID 获取专辑及其所有曲目
func GetAlbumWithTracks(ctx context.Context, albumID int64) (*AlbumDetail, error) {
	var album Album
	if err := GetDB().WithContext(ctx).First(&album, albumID).Error; err != nil {
		return nil, err
	}

	// 加载 MusicBrainz 关联
	mbLink, _ := GetAlbumReleaseMBByAlbumID(ctx, albumID)

	var tracks []*Track
	// 按照用户要求的关联逻辑查询曲目
	err := GetDB().WithContext(ctx).Table("track t").
		Select("t.*").
		Joins("left join track_album ta ON t.id = ta.track_id").
		Where("ta.album_id = ?", albumID).
		Order("ta.disc_number ASC, ta.track_number ASC").
		Find(&tracks).Error

	if err != nil {
		return nil, err
	}
	// 加载曲目关联详情 (TrackAlbum 冗余数据)
	trackAlbums, _ := GetTrackAlbumsByAlbum(ctx, albumID)

	return &AlbumDetail{
		Album:       album,
		Tracks:      tracks,
		TrackAlbums: trackAlbums,
		ReleaseMB:   mbLink,
	}, nil
}

// GetAlbums retrieves albums with pagination and optional keyword search
func GetAlbums(ctx context.Context, limit, offset int, keyword string) ([]*Album, error) {
	var albums []*Album
	db := GetDB().WithContext(ctx)
	if keyword != "" {
		kw := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR artist LIKE ?", kw, kw)
	}
	err := db.Order("name ASC").Limit(limit).Offset(offset).Find(&albums).Error
	return albums, err
}

func GetAlbumsCount(ctx context.Context, keyword string) (int64, error) {
	var count int64
	db := GetDB().WithContext(ctx).Model(&Album{})
	if keyword != "" {
		kw := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR artist LIKE ?", kw, kw)
	}
	err := db.Count(&count).Error
	return count, err
}
