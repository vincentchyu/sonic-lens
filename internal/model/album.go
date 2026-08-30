package model

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/objectstorage"
	"github.com/vincentchyu/sonic-lens/core/telemetry"
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
	PlayCount         int64     `gorm:"column:play_count;type:bigint;default:0;index:idx_album_play_count" json:"play_count"` // 持久化播放次数
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
	album.Genre = NormalizeGenre(db, album.Genre)
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

func mergeAlbumFields(target *Album, source *Album) {
	if target == nil || source == nil {
		return
	}

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
		target.Genre = NormalizeGenre(nil, source.Genre)
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

// GetAlbumByArtistNameAndSubtitle 根据艺术家、专辑名与副标题查询专辑。
func GetAlbumByArtistNameAndSubtitle(ctx context.Context, artist, albumName, subtitle string) (*Album, error) {
	var album Album
	db := GetDB().WithContext(ctx).Where("artist = ? AND name = ?", artist, albumName)
	if subtitle == "" {
		db = db.Where("COALESCE(name_subtitle, '') = ''")
	} else {
		db = db.Where("name_subtitle = ?", subtitle)
	}
	err := db.First(&album).Error
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

// GetAlbumsByGenre 根据流派英文标准名检索关联专辑列表（支持专辑主表流派和曲目反向归因，结合听歌流水与曲目播放量实现热度贯通）
// buildExactGenreMatchClause 构建跨多标签流派列的精确词界匹配 WHERE 条件
func buildExactGenreMatchClause(column string, genre string) (string, []interface{}) {
	clean := strings.TrimSpace(genre)
	if clean == "" {
		return "1 = 0", nil
	}

	conds := []string{
		column + " = ?",
		column + " LIKE ?",
		column + " LIKE ?",
		column + " LIKE ?",
		column + " LIKE ?",
		column + " LIKE ?",
		column + " LIKE ?",
		column + " LIKE ?",
	}
	args := []interface{}{
		clean,
		clean + ",%",
		clean + "/%",
		clean + ";%",
		"%, " + clean,
		"%," + clean,
		"%/" + clean,
		"%;" + clean,
	}

	midDelims := []string{", ", ",", "/", ";"}
	for _, d1 := range midDelims {
		for _, d2 := range midDelims {
			conds = append(conds, column+" LIKE ?")
			args = append(args, "%"+d1+clean+d2+"%")
		}
	}

	clause := "(" + strings.Join(conds, " OR ") + ")"
	return clause, args
}

func GetAlbumsByGenre(ctx context.Context, genre string, limit, offset int, sortBy string) ([]*Album, error) {
	var albums []*Album
	cleanGenre := NormalizeGenreQueryToken(genre)
	if cleanGenre == "" {
		return albums, nil
	}

	clauseDirect, argsDirect := buildExactGenreMatchClause("genre", cleanGenre)
	clauseTrack, argsTrack := buildExactGenreMatchClause("t.genre", cleanGenre)

	subQueryDirect := GetDB().WithContext(ctx).Table("album").Select("id").Where(clauseDirect, argsDirect...)
	subQueryTrack := GetDB().WithContext(ctx).Table("album a").Select("a.id").
		Joins("INNER JOIN track_album ta ON ta.album_id = a.id").
		Joins("INNER JOIN track t ON t.id = ta.track_id").
		Where(clauseTrack, argsTrack...)

	db := GetDB().WithContext(ctx).Model(&Album{}).
		Where("id IN (?) OR id IN (?)", subQueryDirect, subQueryTrack)

	switch sortBy {
	case "release_date":
		db = db.Order("release_date DESC, id DESC")
	case "name":
		db = db.Order("name ASC, id DESC")
	default:
		db = db.Order("play_count DESC, id DESC")
	}

	if limit > 0 {
		db = db.Limit(limit).Offset(offset)
	}

	err := db.Find(&albums).Error
	if err != nil || len(albums) == 0 {
		return albums, err
	}

	// 实时补全专辑封面 (包含 ObjectStorage 转化、Track 反向回填与动态 API 兜底)
	PopulateAlbumsCoverArt(ctx, albums)

	return albums, nil
}

// PopulateAlbumsPlayCount 批量补全专辑对象的真实播放量 (整合曲目累计与 track_play_record 听歌流水)
func PopulateAlbumsPlayCount(ctx context.Context, albums []*Album) {
	if len(albums) == 0 {
		return
	}

	albumIDs := make([]int64, 0, len(albums))
	albumMap := make(map[int64]*Album)

	for _, a := range albums {
		if a == nil || a.ID == 0 {
			continue
		}
		albumIDs = append(albumIDs, a.ID)
		albumMap[a.ID] = a
	}

	// 1. 从 track 表通过 track_album 累加曲目播放量
	type trackStat struct {
		AlbumID   int64 `gorm:"column:album_id"`
		TrackPlay int64 `gorm:"column:track_play"`
	}
	var trackStats []trackStat
	_ = GetDB().WithContext(ctx).Table("track_album ta").
		Select("ta.album_id, SUM(t.play_count) as track_play").
		Joins("INNER JOIN track t ON t.id = ta.track_id").
		Where("ta.album_id IN (?)", albumIDs).
		Group("ta.album_id").
		Scan(&trackStats).Error

	for _, stat := range trackStats {
		if a, ok := albumMap[stat.AlbumID]; ok {
			if stat.TrackPlay > a.PlayCount {
				a.PlayCount = stat.TrackPlay
			}
		}
	}

	// 2. 从 track_play_records 听歌流水按 album_id 累加
	type recordStatByID struct {
		AlbumID int64 `gorm:"column:album_id"`
		Cnt     int64 `gorm:"column:cnt"`
	}
	var recordStatsByID []recordStatByID
	_ = GetDB().WithContext(ctx).Model(&TrackPlayRecord{}).
		Select("album_id, COUNT(*) as cnt").
		Where("album_id IN (?)", albumIDs).
		Group("album_id").
		Scan(&recordStatsByID).Error

	for _, stat := range recordStatsByID {
		if a, ok := albumMap[stat.AlbumID]; ok {
			if stat.Cnt > a.PlayCount {
				a.PlayCount = stat.Cnt
			}
		}
	}
}

// GetAlbumsByGenreCount 根据流派英文标准名计算关联专辑总数
func GetAlbumsByGenreCount(ctx context.Context, genre string) (int64, error) {
	cleanGenre := NormalizeGenreQueryToken(genre)
	if cleanGenre == "" {
		return 0, nil
	}

	clauseDirect, argsDirect := buildExactGenreMatchClause("genre", cleanGenre)
	clauseTrack, argsTrack := buildExactGenreMatchClause("t.genre", cleanGenre)

	subQueryDirect := GetDB().WithContext(ctx).Table("album").Select("id").Where(clauseDirect, argsDirect...)
	subQueryTrack := GetDB().WithContext(ctx).Table("album a").Select("a.id").
		Joins("INNER JOIN track_album ta ON ta.album_id = a.id").
		Joins("INNER JOIN track t ON t.id = ta.track_id").
		Where(clauseTrack, argsTrack...)

	var count int64
	err := GetDB().WithContext(ctx).Model(&Album{}).
		Where("id IN (?) OR id IN (?)", subQueryDirect, subQueryTrack).
		Count(&count).Error
	return count, err
}

// PopulateAlbumsCoverArt 批量对补齐专辑封面 URL (包含 ObjectStorage 转化、Track 反向回填与动态 API 兜底)
func PopulateAlbumsCoverArt(ctx context.Context, albums []*Album) {
	if len(albums) == 0 {
		return
	}

	provider := objectstorage.Get()
	missingCoverIDs := make([]int64, 0)
	missingCoverMap := make(map[int64]*Album)

	for _, a := range albums {
		if a == nil {
			continue
		}
		a.CoverArtURL = strings.TrimSpace(a.CoverArtURL)
		a.CoverArtObjectKey = strings.TrimSpace(a.CoverArtObjectKey)

		// 1. 如果已有 ObjectKey 但 CoverArtURL 为空，通过存储 Provider 转换
		if a.CoverArtURL == "" && a.CoverArtObjectKey != "" && provider != nil {
			a.CoverArtURL = provider.GetObjectCDNURL(a.CoverArtObjectKey)
		}

		// 2. 若仍为空，记录下来准备从 track 表反向回填
		if a.CoverArtURL == "" && a.ID > 0 {
			missingCoverIDs = append(missingCoverIDs, a.ID)
			missingCoverMap[a.ID] = a
		}
	}

	if len(missingCoverIDs) == 0 {
		return
	}

	// 从 track + track_album 联表查询该专辑下首个有效的 cover_art_url 或 cover_art_object_key
	type trackCoverStat struct {
		AlbumID           int64  `gorm:"column:album_id"`
		CoverArtURL       string `gorm:"column:cover_art_url"`
		CoverArtObjectKey string `gorm:"column:cover_art_object_key"`
	}
	var trackCovers []trackCoverStat
	_ = GetDB().WithContext(ctx).Table("track_album ta").
		Select("ta.album_id, MIN(NULLIF(t.cover_art_url, '')) as cover_art_url, MIN(NULLIF(t.cover_art_object_key, '')) as cover_art_object_key").
		Joins("INNER JOIN track t ON t.id = ta.track_id").
		Where("ta.album_id IN (?)", missingCoverIDs).
		Group("ta.album_id").
		Scan(&trackCovers).Error

	for _, tc := range trackCovers {
		a, ok := missingCoverMap[tc.AlbumID]
		if !ok || a == nil {
			continue
		}
		url := strings.TrimSpace(tc.CoverArtURL)
		objKey := strings.TrimSpace(tc.CoverArtObjectKey)

		if url == "" && objKey != "" && provider != nil {
			url = provider.GetObjectCDNURL(objKey)
		}

		if url != "" {
			a.CoverArtURL = url
			if a.CoverArtObjectKey == "" && objKey != "" {
				a.CoverArtObjectKey = objKey
			}
			// 异步安全携程持久化回写至数据库，一劳永逸
			albumID := a.ID
			coverURL := a.CoverArtURL
			coverObjKey := a.CoverArtObjectKey
			telemetry.GoOnlySafe(ctx, func(asyncCtx context.Context) {
				_ = GetDB().WithContext(asyncCtx).Model(&Album{}).Where("id = ?", albumID).Updates(map[string]interface{}{
					"cover_art_url":        coverURL,
					"cover_art_object_key": coverObjKey,
				}).Error
			})
		}
	}

	// 3. 兜底防护：对于依然为空的专辑，给其构造标准的 /api/albums/:id/cover 动态路由 URL，确保绝不会出现空封面
	for _, a := range missingCoverMap {
		if a != nil && a.CoverArtURL == "" && a.ID > 0 {
			a.CoverArtURL = fmt.Sprintf("/api/albums/%d/cover", a.ID)
		}
	}
}

// IncrementAlbumPlayCountTx 原子递增专辑播放量
func IncrementAlbumPlayCountTx(tx *gorm.DB, albumID int64) error {
	if tx == nil || albumID <= 0 {
		return nil
	}
	return tx.Model(&Album{}).Where("id = ?", albumID).
		Update("play_count", gorm.Expr("play_count + 1")).Error
}

// ReconcileAlbumPlayCountsTx 在事务中根据听歌流水和 track 播放数对全量或指定专辑的 play_count 进行纠偏校准
func ReconcileAlbumPlayCountsTx(tx *gorm.DB, targetAlbumIDs ...int64) error {
	if tx == nil {
		return errors.New("tx is nil")
	}

	type albumStat struct {
		AlbumID   int64 `gorm:"column:album_id"`
		PlayCount int64 `gorm:"column:play_count"`
	}

	var stats []albumStat

	// 1. 听歌流水按 album_id 统计
	query1 := tx.Model(&TrackPlayRecord{}).
		Select("album_id, COUNT(*) as play_count").
		Where("album_id > 0")
	if len(targetAlbumIDs) > 0 {
		query1 = query1.Where("album_id IN (?)", targetAlbumIDs)
	}
	query1 = query1.Group("album_id")

	if err := query1.Scan(&stats).Error; err != nil {
		return err
	}

	albumMaxPlayMap := make(map[int64]int64)
	for _, s := range stats {
		if s.AlbumID > 0 {
			albumMaxPlayMap[s.AlbumID] = s.PlayCount
		}
	}

	// 2. 从 track_album 累加曲目 play_count
	var trackStats []albumStat
	query2 := tx.Table("track_album ta").
		Select("ta.album_id, SUM(t.play_count) as play_count").
		Joins("INNER JOIN track t ON t.id = ta.track_id").
		Where("ta.album_id > 0")
	if len(targetAlbumIDs) > 0 {
		query2 = query2.Where("ta.album_id IN (?)", targetAlbumIDs)
	}
	query2 = query2.Group("ta.album_id")

	if err := query2.Scan(&trackStats).Error; err != nil {
		return err
	}

	for _, ts := range trackStats {
		if ts.AlbumID > 0 {
			if ts.PlayCount > albumMaxPlayMap[ts.AlbumID] {
				albumMaxPlayMap[ts.AlbumID] = ts.PlayCount
			}
		}
	}

	// 3. 更新 album.play_count 物理列
	for albumID, maxPlay := range albumMaxPlayMap {
		if err := tx.Model(&Album{}).Where("id = ?", albumID).Update("play_count", maxPlay).Error; err != nil {
			return err
		}
	}

	return nil
}

// ReconcileAlbumPlayCounts 对全量或指定专辑执行播放对账
func ReconcileAlbumPlayCounts(ctx context.Context, targetAlbumIDs ...int64) error {
	return InTx(ctx, func(tx *gorm.DB) error {
		return ReconcileAlbumPlayCountsTx(tx, targetAlbumIDs...)
	})
}
