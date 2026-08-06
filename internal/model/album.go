package model

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/common"
)

// Album represents a music album
type Album struct {
	ID                  int64                   `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	Name                string                  `gorm:"column:name;type:varchar(180);not null;uniqueIndex:uidx_album_artist_name_subtitle_release_date" json:"name"`
	NameSubtitle        string                  `gorm:"column:name_subtitle;type:varchar(60);uniqueIndex:uidx_album_artist_name_subtitle_release_date" json:"name_subtitle"`
	TitleMetadata       *AlbumTitleMetadataJSON `gorm:"column:title_metadata;type:longtext;->" json:"title_metadata"`
	Artist              string                  `gorm:"column:artist;type:varchar(180);not null;uniqueIndex:uidx_album_artist_name_subtitle_release_date" json:"artist"`
	ReleaseDate         string                  `gorm:"column:release_date;type:varchar(50);uniqueIndex:uidx_album_artist_name_subtitle_release_date" json:"release_date"`
	OriginalReleaseDate string                  `gorm:"column:original_release_date;type:varchar(50)" json:"original_release_date"`
	Genre               string                  `gorm:"column:genre;type:varchar(255)" json:"genre"`
	Country             string                  `gorm:"column:country;type:varchar(50)" json:"country"`
	Status              string                  `gorm:"column:status;type:varchar(50)" json:"status"`
	Packaging           string                  `gorm:"column:packaging;type:varchar(50)" json:"packaging"`
	Barcode             string                  `gorm:"column:barcode;type:varchar(255)" json:"barcode"`
	TotalDiscs          int                     `gorm:"column:total_discs;type:int;default:1" json:"total_discs"`              // 总碟数
	DiscInfos           string                  `gorm:"column:disc_infos;type:varchar(255)" json:"disc_infos"`                 // 各碟信息(如 track counts)
	SyncStatus          int                     `gorm:"column:sync_status;type:tinyint;not null;default:0" json:"sync_status"` // 0:默认, 1:初选搜索完成, 2:初选关联完成, 3:精选维护完成, 4:精选锁定完成
	// ReleaseType 承载专辑发行类型枚举（ep / single / lp / album 等），
	// 从 Apple Music " - EP" 等连字符后缀自动提取，也可由 MusicBrainz 同步写入。
	// 与 NameSubtitle（Deluxe/Remaster 等版本说明）语义完全独立，存储在独立列。
	ReleaseType       string    `gorm:"column:release_type;type:varchar(20)" json:"release_type"`
	CoverArtURL       string    `gorm:"column:cover_art_url;type:varchar(1024)" json:"cover_art_url"`
	CoverArtMime      string    `gorm:"column:cover_art_mime;type:varchar(128)" json:"cover_art_mime"`
	CoverArtObjectKey string    `gorm:"column:cover_art_object_key;type:varchar(512);index:idx_album_cover_art_object_key" json:"cover_art_object_key"`
	CreatedAt         time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

// AlbumCoverUpdate 用于更新专辑封面存储信息。
type AlbumCoverUpdate struct {
	CoverArtURL       string
	CoverArtMime      string
	CoverArtObjectKey string
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

func normalizedAlbumSubtitle(subtitle string) string {
	return strings.TrimSpace(subtitle)
}

// normalizeAlbumReleaseTypeSuffix 在专辑落库前检查并剥离 Apple Music 连字符发行类型后缀。
// 例如 "In The Sun - EP" -> Name="In The Sun", ReleaseType="ep"。
// 如果已有 ReleaseType 则不覆盖，保留显式设置的值。
func normalizeAlbumReleaseTypeSuffix(album *Album) {
	if album.Name == "" {
		return
	}
	title, rt := common.ParseAlbumTitleAndReleaseType(album.Name)
	if rt == "" {
		// 原标题不含连字符后缀，无需处理
		return
	}
	// 剥离后缀，写入干净主标题
	album.Name = title
	// ReleaseType 以已有值优先（精选维护可能已写入），未设置时才自动填充
	if album.ReleaseType == "" {
		album.ReleaseType = rt
	}
}

func getOrCreateAlbumTx(db *gorm.DB, album *Album) error {
	// 落库前统一剥离 Apple Music 发行类型连字符后缀，防止 "In The Sun - EP" 写入专辑名。
	normalizeAlbumReleaseTypeSuffix(album)
	subtitle := normalizedAlbumSubtitle(album.NameSubtitle)
	var exact Album
	err := db.Where(
		"artist = ? AND name = ? AND release_date = ? AND COALESCE(name_subtitle, '') = ?",
		album.Artist, album.Name, album.ReleaseDate, subtitle,
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
		err = db.Where(
			"artist = ? AND name = ? AND sync_status = ? AND COALESCE(name_subtitle, '') = ?",
			album.Artist,
			album.Name,
			3,
			subtitle,
		).
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
	fallbackQuery := db.Where(
		"artist = ? AND name = ? AND COALESCE(name_subtitle, '') = ?",
		album.Artist,
		album.Name,
		subtitle,
	)
	if strings.TrimSpace(album.ReleaseDate) != "" {
		fallbackQuery = fallbackQuery.Where("release_date = '' OR release_date IS NULL")
	}
	err = fallbackQuery.Order("CASE WHEN release_date = '' OR release_date IS NULL THEN 0 ELSE 1 END ASC, id ASC").
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
	if target.OriginalReleaseDate == "" && source.OriginalReleaseDate != "" {
		target.OriginalReleaseDate = source.OriginalReleaseDate
	}
	if target.NameSubtitle == "" && source.NameSubtitle != "" {
		target.NameSubtitle = source.NameSubtitle
	}
	if target.TitleMetadata == nil && source.TitleMetadata != nil {
		clone := *source.TitleMetadata
		target.TitleMetadata = &clone
	}
	if target.ReleaseType == "" && source.ReleaseType != "" {
		target.ReleaseType = source.ReleaseType
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
	if target.CoverArtURL == "" && source.CoverArtURL != "" {
		target.CoverArtURL = source.CoverArtURL
	}
	if target.CoverArtMime == "" && source.CoverArtMime != "" {
		target.CoverArtMime = source.CoverArtMime
	}
	if target.CoverArtObjectKey == "" && source.CoverArtObjectKey != "" {
		target.CoverArtObjectKey = source.CoverArtObjectKey
	}
}

func GetAlbum(ctx context.Context, id int64) (*Album, error) {
	var album Album
	err := GetDB().WithContext(ctx).First(&album, id).Error
	return &album, err
}

func GetAlbumTx(tx *gorm.DB, id int64) (*Album, error) {
	var album Album
	if err := tx.First(&album, id).Error; err != nil {
		return nil, err
	}
	return &album, nil
}

// UpdateAlbumSyncStatus 更新专辑同步状态，避免上层散落字段更新 SQL。
func UpdateAlbumSyncStatus(ctx context.Context, albumID int64, syncStatus int) error {
	return UpdateAlbumSyncStatusTx(GetDB().WithContext(ctx), albumID, syncStatus)
}

// UpdateAlbumSyncStatusTx 在事务内更新专辑同步状态。
func UpdateAlbumSyncStatusTx(tx *gorm.DB, albumID int64, syncStatus int) error {
	return updateAlbumByIDTx(
		tx,
		albumID,
		map[string]interface{}{
			"sync_status": syncStatus,
		},
		nil,
	)
}

// UpdateAlbumFields 更新专辑元数据字段集合。
func UpdateAlbumFields(ctx context.Context, albumID int64, fields map[string]interface{}) error {
	return UpdateAlbumFieldsTx(GetDB().WithContext(ctx), albumID, fields)
}

// UpdateAlbumFieldsTx 在事务内批量更新专辑元数据字段集合。
func UpdateAlbumFieldsTx(tx *gorm.DB, albumID int64, fields map[string]interface{}) error {
	return updateAlbumByIDTx(tx, albumID, fields, nil)
}

// UpsertAlbumCoverByID 根据专辑 ID 更新封面信息；当 object key 已一致时跳过写入。
func UpsertAlbumCoverByID(ctx context.Context, albumID int64, update AlbumCoverUpdate) error {
	return UpsertAlbumCoverByIDTx(GetDB().WithContext(ctx), albumID, update)
}

// UpsertAlbumCoverByIDTx 在事务内更新封面信息。
func UpsertAlbumCoverByIDTx(tx *gorm.DB, albumID int64, update AlbumCoverUpdate) error {
	if albumID <= 0 {
		return nil
	}

	updates := map[string]interface{}{}
	if v := strings.TrimSpace(update.CoverArtURL); v != "" {
		updates["cover_art_url"] = v
	}
	if v := strings.TrimSpace(update.CoverArtMime); v != "" {
		updates["cover_art_mime"] = v
	}
	if v := strings.TrimSpace(update.CoverArtObjectKey); v != "" {
		updates["cover_art_object_key"] = v
	}
	if len(updates) == 0 {
		return nil
	}

	return updateAlbumByIDTx(
		tx,
		albumID,
		updates,
		func(query *gorm.DB) *gorm.DB {
			if objectKey, ok := updates["cover_art_object_key"].(string); ok && objectKey != "" {
				return query.Where(
					"(cover_art_object_key IS NULL OR cover_art_object_key = '' OR cover_art_object_key <> ?)",
					objectKey,
				)
			}
			return query
		},
	)
}

// UpdateAlbumTitleMetadataByID 根据专辑 ID 更新标题元数据 JSON；内容未变化时跳过写入。
func UpdateAlbumTitleMetadataByID(ctx context.Context, albumID int64, metadata *common.AlbumTitleMetadata) error {
	return UpdateAlbumTitleMetadataByIDTx(GetDB().WithContext(ctx), albumID, metadata)
}

// UpdateAlbumTitleMetadataByIDTx 在事务内更新专辑标题元数据 JSON。
func UpdateAlbumTitleMetadataByIDTx(tx *gorm.DB, albumID int64, metadata *common.AlbumTitleMetadata) error {
	if tx == nil || albumID <= 0 || metadata == nil {
		return nil
	}

	metadataValue := AlbumTitleMetadataJSONFromCommon(metadata)
	if metadataValue == nil {
		return nil
	}

	result := tx.Session(&gorm.Session{SkipHooks: true}).Exec(
		"UPDATE `album` SET `title_metadata`=?,`updated_at`=? WHERE id = ? AND sync_status <> 4 AND ((title_metadata IS NULL OR title_metadata = '' OR title_metadata <> ?))",
		metadataValue,
		tx.NowFunc(),
		albumID,
		metadataValue,
	)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected <= 0 {
		return nil
	}
	return appendLibraryChangeTx(tx, LibraryEntityAlbum, albumID, LibraryOpUpsert)
}

func updateAlbumByIDTx(
	tx *gorm.DB, albumID int64, fields map[string]interface{}, decorate func(query *gorm.DB) *gorm.DB,
) error {
	if tx == nil || albumID <= 0 || len(fields) == 0 {
		return nil
	}

	updates := make(map[string]interface{}, len(fields)+1)
	for key, value := range fields {
		updates[key] = value
	}
	updates["updated_at"] = tx.NowFunc()

	query := tx.Session(&gorm.Session{SkipHooks: true}).Model(&Album{}).Where("id = ?", albumID)

	// 检查专辑锁定状态 (sync_status=4)
	var currentStatus int
	if err := tx.Model(&Album{}).Where("id = ?", albumID).Select("sync_status").Scan(&currentStatus).Error; err == nil {
		if currentStatus == 4 {
			// 只有当本次更新是尝试将状态改为 3 (解锁) 时，才允许继续
			newStatus, ok := fields["sync_status"].(int)
			if !ok || newStatus != 3 {
				return errors.New("专辑已锁定 (sync_status=4)")
			}
		}
	}

	if decorate != nil {
		query = decorate(query)
	}
	result := query.Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected <= 0 {
		return nil
	}
	return appendLibraryChangeTx(tx, LibraryEntityAlbum, albumID, LibraryOpUpsert)
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
