package model

import (
	"context"
	"time"
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
	TotalDiscs  int       `gorm:"column:total_discs;type:int;default:1" json:"total_discs"` // 总碟数
	DiscInfos   string    `gorm:"column:disc_infos;type:varchar(255)" json:"disc_infos"`    // 各碟信息(如 track counts)
	SyncStatus  int       `gorm:"column:sync_status;type:tinyint;default:0" json:"sync_status"` // 0:默认, 1:初选搜索完成, 2:初选关联完成, 3:精选维护完成
	CreatedAt   time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName sets the table name for the Album model
func (Album) TableName() string {
	return "album"
}

// GetOrCreateAlbum gets an album by artist and name, or creates it if it doesn't exist
func GetOrCreateAlbum(ctx context.Context, album *Album) error {
	return GetDB().WithContext(ctx).Where(
		"artist = ? AND name = ? AND release_date = ?", album.Artist, album.Name, album.ReleaseDate,
	).FirstOrCreate(album).Error
}

func GetAlbum(ctx context.Context, id int64) (*Album, error) {
	var album Album
	err := GetDB().WithContext(ctx).First(&album, id).Error
	return &album, err
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
