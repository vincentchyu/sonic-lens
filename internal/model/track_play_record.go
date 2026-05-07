package model

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"

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
	ID                   int64                          `gorm:"column:id;type:bigint;primaryKey;autoIncrement;index:idx_play_time_id,sort:desc,priority:2" json:"id"`
	Artist               string                         `gorm:"column:artist;type:varchar(180);not null;index:idx_track_play_records_artist;index:idx_track_play_records_identity_subtitle" json:"artist"`
	AlbumArtist          string                         `gorm:"column:album_artist;type:varchar(180)" json:"album_artist"`
	Track                string                         `gorm:"column:track;type:varchar(180);not null;index:idx_track_play_records_identity_subtitle" json:"track"`
	Album                string                         `gorm:"column:album;type:varchar(180);not null;index:idx_track_play_records_identity_subtitle" json:"album"`
	AlbumSubtitle        string                         `gorm:"column:album_subtitle;type:varchar(60);index:idx_track_play_records_identity_subtitle" json:"album_subtitle"`
	Genre                string                         `gorm:"column:genre;type:varchar(255);index:idx_track_play_records_genre" json:"genre"`
	AlbumID              int64                          `gorm:"column:album_id;type:bigint;default:0;index:idx_track_play_records_album_id" json:"album_id"`
	Duration             int64                          `gorm:"column:duration;type:bigint" json:"duration"`
	PlayTime             time.Time                      `gorm:"column:play_time;type:timestamp;not null;default:CURRENT_TIMESTAMP;index:idx_play_time_id,sort:desc,priority:1" json:"play_time"`
	Scrobbled            bool                           `gorm:"column:scrobbled;type:tinyint(1);not null;default:0;index:idx_track_play_records_scrobbled" json:"scrobbled"`
	MusicBrainzID        string                         `gorm:"column:music_brainz_id;type:varchar(255)" json:"music_brainz_id"`
	TrackNumber          int8                           `gorm:"column:track_number;type:tinyint;index:idx_track_play_records_identity_subtitle" json:"track_number"`
	DiscNumber           int8                           `gorm:"column:disc_number;type:tinyint;default:1;index:idx_track_play_records_identity_subtitle" json:"disc_number"`
	Source               string                         `gorm:"column:source;type:varchar(100);not null;index:idx_track_play_records_source" json:"source"`
	CoverArtPath         string                         `gorm:"column:cover_art_path;type:varchar(1024)" json:"cover_art_path"`                                                                               // 客户端可直接拼接或复用的封面路径
	TraceID              string                         `gorm:"column:trace_id;type:varchar(32);index:idx_track_play_records_trace_id" json:"trace_id"`                                                       // 关联当前播放链路的 TraceID，便于从播放流水反查观测链路
	RootSpanID           string                         `gorm:"column:root_span_id;type:varchar(16)" json:"root_span_id"`                                                                                     // 当前播放根 span 的 SpanID，便于从播放流水定位单首歌根节点
	TraceSampled         bool                           `gorm:"column:trace_sampled;type:tinyint(1);not null;default:0" json:"trace_sampled"`                                                                 // 记录该次播放链路是否命中采样，避免库里有 trace_id 但观测平台无样本时误判
	ResolvedTrackID      int64                          `gorm:"column:resolved_track_id;type:bigint;default:0;index:idx_track_play_records_resolved_track_id" json:"resolved_track_id"`                       // 本次播放最终归因到的 track.id，0 表示未归因
	ResolutionStatus     string                         `gorm:"column:resolution_status;type:varchar(32);not null;default:'pending';index:idx_track_play_records_resolution_status" json:"resolution_status"` // 归因状态：pending/resolved/unresolved/ambiguous
	ResolutionConfidence common.TrackMetadataConfidence `gorm:"column:resolution_confidence;type:tinyint;default:0" json:"resolution_confidence"`                                                             // 归因置信度，取值对应 common.TrackMetadataConfidence*
	LibraryApplied       bool                           `gorm:"column:library_applied;type:tinyint(1);not null;default:0;index:idx_track_play_records_library_applied" json:"library_applied"`                // 是否已将该播放应用到主资料库(track/album/track_album)
	CreatedAt            time.Time                      `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt            time.Time                      `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

type ReplayTrackPlayRecordsParams struct {
	Ctx            context.Context
	Limit          int
	Source         string
	RecordIDs      []int64
	PlayedFrom     time.Time
	PlayedTo       time.Time
	DryRun         bool
	OnlyUnapplied  bool
	OnlyUnresolved bool
}

type ReplayTrackPlayRecordResult struct {
	ID              int64
	Artist          string
	Album           string
	Track           string
	Source          string
	BeforeStatus    string
	BeforeApplied   bool
	AfterStatus     string
	AfterApplied    bool
	ResolvedTrackID int64
}

type ReplayTrackPlayRecordsReport struct {
	Results []*ReplayTrackPlayRecordResult
}

const (
	// TrackPlayRecordResolutionPending 表示播放已入库但尚未完成归因。
	TrackPlayRecordResolutionPending = "pending"
	// TrackPlayRecordResolutionResolved 表示播放已稳定归因到某条曲目。
	TrackPlayRecordResolutionResolved = "resolved"
	// TrackPlayRecordResolutionUnresolved 表示本轮未能稳定归因，后续可重试。
	TrackPlayRecordResolutionUnresolved = "unresolved"
	// TrackPlayRecordResolutionAmbiguous 表示存在多个候选，无法安全落单。
	TrackPlayRecordResolutionAmbiguous = "ambiguous"
)

// TableName sets the table name for the TrackPlayRecord model
func (TrackPlayRecord) TableName() string {
	return "track_play_records"
}

// BuildTrackPlayRecordArtworkPath 将对象键或已知地址统一收敛成客户端可直接消费的封面路径。
func BuildTrackPlayRecordArtworkPath(coverArtURL, coverArtObjectKey string) string {
	if path := strings.TrimSpace(coverArtURL); path != "" {
		return path
	}
	if objectKey := strings.TrimSpace(coverArtObjectKey); objectKey != "" {
		return buildTrackPlayRecordArtworkPathFromObjectKey(objectKey)
	}
	return ""
}

func buildTrackPlayRecordArtworkPathFromObjectKey(objectKey string) string {
	prefix, ok := buildTrackPlayRecordObjectStorageCDNPrefix()
	if !ok {
		return ""
	}
	escapedKey := escapeTrackPlayRecordObjectKey(objectKey)
	if escapedKey == "" {
		return ""
	}
	return strings.TrimRight(prefix, "/") + "/" + escapedKey
}

func buildTrackPlayRecordObjectStorageCDNPrefix() (string, bool) {
	objectStorage := config.ConfigObj.ObjectStorage
	if !objectStorage.Enabled {
		return "", false
	}

	cdnURL := strings.TrimSpace(objectStorage.CDNURL)
	if cdnURL != "" {
		if strings.HasPrefix(cdnURL, "/") {
			return strings.TrimRight(cdnURL, "/"), true
		}
		if parsed, err := url.Parse(cdnURL); err == nil {
			if path := strings.TrimRight(parsed.Path, "/"); path != "" {
				return path, true
			}
		}
	}

	bucket := strings.TrimSpace(objectStorage.Bucket)
	if bucket == "" {
		return "", false
	}
	return "/" + strings.Trim(bucket, "/"), true
}

func escapeTrackPlayRecordObjectKey(objectKey string) string {
	parts := strings.Split(strings.TrimSpace(objectKey), "/")
	escaped := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		escaped = append(escaped, url.PathEscape(part))
	}
	return strings.Join(escaped, "/")
}

func InsertTrackPlayRecord(ctx context.Context, record *TrackPlayRecord) error {
	if record == nil {
		return errors.New("track play record is nil")
	}

	// 验证记录中的艺术家、专辑和曲目信息
	if err := common.ValidateTrackInfo(ctx, record.Artist, record.Album, record.Track); err != nil {
		return err
	}

	// 避免零时间被写入 MySQL 为 "0000-00-00 00:00:00"
	if record.PlayTime.IsZero() {
		record.PlayTime = time.Now()
	}

	// 自动填充 AlbumID
	if record.AlbumID == 0 {
		record.AlbumID = getAlbumIDByTrackInfo(
			ctx, record.Artist, record.Album, record.AlbumSubtitle, record.Track, record.TrackNumber, record.DiscNumber,
		)
	}
	if record.ResolutionStatus == "" {
		record.ResolutionStatus = TrackPlayRecordResolutionPending
	}

	return GetDB().WithContext(ctx).Create(record).Error
}

func resolveTrackForPlayRecord(
	tx *gorm.DB, artist, album, track string, metadata TrackMetadata,
) (*Track, string, common.TrackMetadataConfidence, error) {
	if historicalTrack, confidence, err := findLatestResolvedTrackByIdentityAndSourceTx(
		tx,
		artist,
		album,
		metadata.AlbumSubtitle,
		track,
		strings.TrimSpace(metadata.Source),
		metadata.TrackNumber,
		metadata.DiscNumber,
	); err == nil {
		return historicalTrack, TrackPlayRecordResolutionResolved, confidence, nil
	}

	identity, resolvedTrack, err := resolveTrackIdentityWithOptions(
		tx,
		artist,
		album,
		track,
		metadata,
		trackIdentityResolveOptions{
			allowLooseNameFallback: metadataAllowsLibraryMutation(metadata),
			allowUniqueIDHint:      metadataAllowsLibraryMutation(metadata),
		},
	)
	if err != nil {
		return nil, "", 0, err
	}
	if resolvedTrack != nil {
		confidence := metadataConfidence(metadata)
		if confidence < common.TrackMetadataConfidenceAuthoritative {
			authoritative, authErr := HasAuthoritativeTrackAlbumBindingTx(tx, resolvedTrack.ID)
			if authErr != nil {
				return nil, "", 0, authErr
			}
			if authoritative {
				confidence = common.TrackMetadataConfidenceAuthoritative
			}
		}
		return resolvedTrack, TrackPlayRecordResolutionResolved, confidence, nil
	}

	if identity.TrackNumber == 0 && identity.DiscNumber == 0 {
		return nil, TrackPlayRecordResolutionUnresolved, metadataConfidence(metadata), nil
	}
	return nil, TrackPlayRecordResolutionUnresolved, metadataConfidence(metadata), nil
}

// ResolveTrackPlayRecord 根据最新曲目解析结果回填播放记录归因状态。
func ResolveTrackPlayRecord(
	ctx context.Context, recordID int64, artist, album, track string, metadata TrackMetadata,
) error {
	if recordID <= 0 {
		return nil
	}

	return InTx(
		ctx, func(tx *gorm.DB) error {
			resolvedTrack, status, confidence, err := resolveTrackForPlayRecord(tx, artist, album, track, metadata)
			if err != nil {
				return err
			}

			fields := map[string]interface{}{
				"resolution_status":     status,
				"resolution_confidence": confidence,
			}
			if resolvedTrack != nil {
				if err := ResolvePendingTrackFavoriteEventsByTrackTx(tx, resolvedTrack, confidence); err != nil {
					return err
				}
				fields["resolved_track_id"] = resolvedTrack.ID
				if resolvedTrack.MusicBrainzID != "" {
					fields["music_brainz_id"] = resolvedTrack.MusicBrainzID
				}
				fields["album_id"] = getAlbumIDByTrackInfoTx(
					tx,
					resolvedTrack.Artist,
					resolvedTrack.Album,
					resolvedTrack.AlbumSubtitle,
					resolvedTrack.Track,
					resolvedTrack.TrackNumber,
					resolvedTrack.DiscNumber,
				)
			}
			return tx.Model(&TrackPlayRecord{}).Where("id = ?", recordID).Updates(fields).Error
		},
	)
}

func getTrackPlayRecordByIDTx(tx *gorm.DB, recordID int64) (*TrackPlayRecord, error) {
	var record TrackPlayRecord
	if err := tx.Where("id = ?", recordID).First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func findLatestResolvedTrackByIdentityAndSourceTx(
	tx *gorm.DB, artist, album, albumSubtitle, track, source string, trackNumber, discNumber int8,
) (*Track, common.TrackMetadataConfidence, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, 0, gorm.ErrRecordNotFound
	}

	query := tx.Model(&TrackPlayRecord{}).
		Where("artist = ? AND album = ? AND track = ?", artist, album, track).
		Where("COALESCE(album_subtitle, '') = ?", normalizeTrackStorageText(albumSubtitle)).
		Where("source = ?", source).
		Where("resolution_status = ? AND resolved_track_id > 0", TrackPlayRecordResolutionResolved).
		Where("library_applied = ?", true)

	trackNumber, discNumber = normalizeTrackAlbumPosition(trackNumber, discNumber)
	if trackNumber > 0 {
		query = query.Where("track_number = ? AND disc_number = ?", trackNumber, discNumber)
	}

	var row struct {
		ResolvedTrackID      int64 `gorm:"column:resolved_track_id"`
		ResolutionConfidence int8  `gorm:"column:resolution_confidence"`
	}
	if err := query.Order("play_time DESC, id DESC").Limit(1).Scan(&row).Error; err != nil {
		return nil, 0, err
	}
	if row.ResolvedTrackID <= 0 {
		return nil, 0, gorm.ErrRecordNotFound
	}

	trackObj, err := GetTrackByIDTx(tx, row.ResolvedTrackID)
	if err != nil {
		return nil, 0, err
	}
	return trackObj, common.TrackMetadataConfidence(row.ResolutionConfidence), nil
}

// FindLatestResolvedTrackIDByIdentityTx 优先复用播放归因结果，降低收藏写入的误归因风险。
func FindLatestResolvedTrackIDByIdentityTx(
	tx *gorm.DB, artist, album, albumSubtitle, track string, trackNumber, discNumber int8,
) (int64, int8, error) {
	query := tx.Model(&TrackPlayRecord{}).
		Where("artist = ? AND album = ? AND track = ?", artist, album, track).
		Where("COALESCE(album_subtitle, '') = ?", normalizeTrackStorageText(albumSubtitle)).
		Where("resolution_status = ? AND resolved_track_id > 0", TrackPlayRecordResolutionResolved)

	trackNumber, discNumber = normalizeTrackAlbumPosition(trackNumber, discNumber)
	if trackNumber > 0 {
		query = query.Where("track_number = ? AND disc_number = ?", trackNumber, discNumber)
	}

	var row struct {
		ResolvedTrackID      int64 `gorm:"column:resolved_track_id"`
		ResolutionConfidence int8  `gorm:"column:resolution_confidence"`
	}
	err := query.Order("play_time DESC, id DESC").Limit(1).Scan(&row).Error
	if err != nil {
		return 0, 0, err
	}
	if row.ResolvedTrackID <= 0 {
		return 0, 0, gorm.ErrRecordNotFound
	}
	return row.ResolvedTrackID, row.ResolutionConfidence, nil
}

func buildIncrementTrackPlayCountParamsFromRecord(
	ctx context.Context, record *TrackPlayRecord, metadata TrackMetadata,
) IncrementTrackPlayCountParams {
	if metadata.AlbumSubtitle == "" && record != nil {
		metadata.AlbumSubtitle = record.AlbumSubtitle
	}
	return IncrementTrackPlayCountParams{
		Ctx:           ctx,
		Artist:        record.Artist,
		Album:         record.Album,
		Track:         record.Track,
		TrackMetadata: metadata,
	}
}

func buildTrackPlayRecordResolvedFields(
	tx *gorm.DB,
	record *TrackPlayRecord,
	resolvedTrack *Track,
	albumID int64,
) map[string]interface{} {
	/*
		INSERT INTO multimedia.track_play_records (id, artist, album_artist, track, album, duration, play_time, scrobbled, music_brainz_id, track_number, disc_number, source, cover_art_path, trace_id, root_span_id, trace_sampled, resolved_track_id, resolution_status, resolution_confidence, library_applied, created_at, updated_at, album_id) VALUES (6695, '万能青年旅店', '万能青年旅店', '十万嬉皮', '万能青年旅店', 284, '2026-04-03 12:34:05', 1, '24308649-8842-41e4-b515-c503d0d2932a', 7, 1, 'Apple Music', '/album/v1/originals/d82d06648b8f5f17547ef07d4f286081d53818e7', 'a6f19f00299498144d48c1364f79684f', '3d81ea6868795264', 1, 2795, 'resolved', 4, 1, '2026-04-03 12:36:41', '2026-04-03 12:36:42', 47);

	*/

	fields := map[string]interface{}{}
	albumCoverArtPath, _ := getAlbumCoverArtPathByIDTx(tx, albumID)
	if record != nil {
		if coverArtPath := normalizeTrackPlayRecordCoverArtPath(
			record.CoverArtPath, albumCoverArtPath,
		); coverArtPath != "" {
			fields["cover_art_path"] = coverArtPath
		}
	}
	if _, ok := fields["cover_art_path"]; !ok && albumCoverArtPath != "" {
		fields["cover_art_path"] = albumCoverArtPath
	}
	if resolvedTrack == nil {
		return fields
	}

	fields["resolved_track_id"] = resolvedTrack.ID
	if resolvedTrack.MusicBrainzID != "" {
		fields["music_brainz_id"] = resolvedTrack.MusicBrainzID
	}
	if record != nil {
		if record.TrackNumber <= 0 && resolvedTrack.TrackNumber > 0 {
			fields["track_number"] = resolvedTrack.TrackNumber
		}
		if record.DiscNumber <= 0 && resolvedTrack.DiscNumber > 0 {
			fields["disc_number"] = resolvedTrack.DiscNumber
		}
	}
	fields["album_id"] = albumID
	if albumID > 0 {
		if albumObj, err := GetAlbumTx(tx, albumID); err == nil && albumObj != nil {
			fields["album_subtitle"] = albumObj.NameSubtitle
			if record != nil && strings.TrimSpace(record.Genre) == "" && albumObj.Genre != "" {
				fields["genre"] = albumObj.Genre
			}
		}
	}

	if record != nil && strings.TrimSpace(record.Genre) == "" && resolvedTrack.Genre != "" {
		fields["genre"] = resolvedTrack.Genre
	}

	if _, ok := fields["cover_art_path"]; !ok && albumCoverArtPath != "" {
		fields["cover_art_path"] = albumCoverArtPath
	}
	return fields
}

func normalizeTrackPlayRecordCoverArtPath(recordPath, albumPath string) string {
	recordPath = strings.TrimSpace(recordPath)
	albumPath = strings.TrimSpace(albumPath)
	if recordPath == "" {
		return albumPath
	}
	if strings.HasPrefix(recordPath, "/api/artwork/") {
		return albumPath
	}
	return recordPath
}

func getAlbumCoverArtPathByIDTx(tx *gorm.DB, albumID int64) (string, error) {
	if tx == nil || albumID <= 0 {
		return "", nil
	}

	var album Album
	if err := tx.First(&album, albumID).Error; err != nil {
		return "", err
	}
	return BuildTrackPlayRecordArtworkPath(album.CoverArtURL, album.CoverArtObjectKey), nil
}

func recentPlayRecordCoverArtPathExpr(tableAlias string) string {
	albumByIDExpr := buildAlbumCoverArtPathExpr("aa")
	albumByArtistExpr := buildAlbumCoverArtPathExpr("ab")

	return "COALESCE(NULLIF(" + tableAlias + ".cover_art_path, ''), " + albumByIDExpr + ", " + albumByArtistExpr + ") AS cover_art_path"
}

func buildAlbumCoverArtPathExpr(alias string) string {
	objectStoragePrefix, hasObjectStoragePrefix := buildTrackPlayRecordObjectStorageCDNPrefix()
	switch config.ConfigObj.Database.Type {
	case string(common.DatabaseTypeMySQL):
		if hasObjectStoragePrefix {
			return "COALESCE(NULLIF(" + alias + ".cover_art_url, ''), CASE WHEN " + alias + ".cover_art_object_key IS NOT NULL AND " + alias + ".cover_art_object_key <> '' THEN CONCAT(" + sqlStringLiteral(objectStoragePrefix) + ", '/', " + alias + ".cover_art_object_key) ELSE NULL END)"
		}
		return "NULLIF(" + alias + ".cover_art_url, '')"
	default:
		if hasObjectStoragePrefix {
			return "COALESCE(NULLIF(" + alias + ".cover_art_url, ''), CASE WHEN " + alias + ".cover_art_object_key IS NOT NULL AND " + alias + ".cover_art_object_key <> '' THEN " + sqlStringLiteral(
				strings.TrimRight(
					objectStoragePrefix, "/",
				)+"/",
			) + " || " + alias + ".cover_art_object_key ELSE NULL END)"
		}
		return "NULLIF(" + alias + ".cover_art_url, '')"
	}
}

func sqlStringLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func recentPlayRecordsQuery(ctx context.Context) *gorm.DB {
	db := GetDB().WithContext(ctx)
	return db.Table("track_play_records AS tpr").
		Select("tpr.*, " + recentPlayRecordCoverArtPathExpr("tpr")).
		Joins(
			"LEFT JOIN album AS aa ON aa.id = tpr.album_id AND tpr.album_id > 0",
		).
		Joins(
			"LEFT JOIN (" +
				"SELECT artist, name, COALESCE(name_subtitle, '') AS name_subtitle, MIN(id) AS id FROM album GROUP BY artist, name, COALESCE(name_subtitle, '')" +
				") AS ac ON ac.artist = tpr.artist AND ac.name = tpr.album AND COALESCE(ac.name_subtitle, '') = COALESCE(tpr.album_subtitle, '')",
		).
		Joins("LEFT JOIN album AS ab ON ab.id = ac.id")
}

// ProcessTrackPlayRecord 统一处理播放流水的资料库写入和归因回填。
func ProcessTrackPlayRecord(ctx context.Context, recordID int64, metadata TrackMetadata) error {
	if recordID <= 0 {
		return nil
	}

	return InTx(
		ctx, func(tx *gorm.DB) error {
			record, err := getTrackPlayRecordByIDTx(tx, recordID)
			if err != nil {
				return err
			}
			// 检查流水的情况 有没有条件 对track 更新？

			applied, err := applyTrackPlayMutationTx(
				tx, buildIncrementTrackPlayCountParamsFromRecord(ctx, record, metadata),
			)
			if err != nil {
				return err
			}
			// 更新PlayRecord
			resolvedTrack, status, confidence, err := resolveTrackForPlayRecord(
				tx, record.Artist, record.Album, record.Track, metadata,
			)
			if err != nil {
				return err
			}

			fields := map[string]interface{}{
				"resolution_status":     status,
				"resolution_confidence": confidence,
			}
			if resolvedTrack != nil {
				// 只是闭环了喜欢的数据 回填了播放数据的
				if err := ResolvePendingTrackFavoriteEventsByTrackTx(tx, resolvedTrack, confidence); err != nil {
					return err
				}
				albumID := getAlbumIDByTrackInfoTx(
					tx,
					resolvedTrack.Artist,
					resolvedTrack.Album,
					resolvedTrack.AlbumSubtitle,
					resolvedTrack.Track,
					resolvedTrack.TrackNumber,
					resolvedTrack.DiscNumber,
				)
				for key, value := range buildTrackPlayRecordResolvedFields(tx, record, resolvedTrack, albumID) {
					fields[key] = value
				}
				if applied && albumID > 0 {
					fields["library_applied"] = true
				}
			}
			return tx.Model(&TrackPlayRecord{}).Where("id = ?", recordID).Updates(fields).Error
		},
	)
}

func inferReplayTrackMetadata(record *TrackPlayRecord) TrackMetadata {
	metadata := TrackMetadata{
		AlbumArtist:   record.AlbumArtist,
		AlbumSubtitle: record.AlbumSubtitle,
		TrackNumber:   record.TrackNumber,
		DiscNumber:    record.DiscNumber,
		Duration:      record.Duration,
		MusicBrainzID: record.MusicBrainzID,
		Source:        record.Source,
		PlayerType:    record.Source,
		Genre:         record.Genre,
		Confidence:    common.TrackMetadataConfidenceLow,
	}

	switch strings.TrimSpace(record.Source) {
	case string(common.PlayerAudirvana):
		metadata.Confidence = common.TrackMetadataConfidenceHigh
	case string(common.PlayerAppleMusic):
		if record.TrackNumber > 0 && record.Duration > 0 {
			metadata.Confidence = common.TrackMetadataConfidenceMedium
		}
	case string(common.PlayerRoon):
		if record.TrackNumber > 0 && record.Duration > 0 {
			metadata.Confidence = common.TrackMetadataConfidenceMedium
		}
	default:
		if record.TrackNumber > 0 && record.Duration > 0 {
			metadata.Confidence = common.TrackMetadataConfidenceMedium
		}
	}

	if record.MusicBrainzID != "" {
		metadata.Confidence = common.TrackMetadataConfidenceHigh
	}

	return metadata
}

// GetReplayableTrackPlayRecords 获取待补归因或待应用到资料库的播放流水。
func GetReplayableTrackPlayRecords(
	ctx context.Context,
	limit int,
	source string,
	recordIDs []int64,
	playedFrom, playedTo time.Time,
	onlyUnapplied, onlyUnresolved bool,
) ([]*TrackPlayRecord, error) {
	var records []*TrackPlayRecord
	db := GetDB().WithContext(ctx).Model(&TrackPlayRecord{})

	if source != "" {
		db = db.Where("source = ?", source)
	}
	if len(recordIDs) > 0 {
		db = db.Where("id IN ?", recordIDs)
	}
	if !playedFrom.IsZero() {
		db = db.Where("play_time >= ?", playedFrom)
	}
	if !playedTo.IsZero() {
		db = db.Where("play_time <= ?", playedTo)
	}

	switch {
	case onlyUnapplied && onlyUnresolved:
		db = db.Where(
			"library_applied = ? OR resolution_status IN ?", false, []string{
				TrackPlayRecordResolutionPending,
				TrackPlayRecordResolutionUnresolved,
			},
		)
	case onlyUnapplied:
		db = db.Where("library_applied = ?", false)
	case onlyUnresolved:
		db = db.Where(
			"resolution_status IN ?", []string{
				TrackPlayRecordResolutionPending,
				TrackPlayRecordResolutionUnresolved,
			},
		)
	default:
		// 默认只处理尚未应用到资料库的新流程记录，避免已封板的历史流水继续进入 replay 队列。
		db = db.Where("library_applied = ?", false)
	}

	if limit > 0 {
		db = db.Limit(limit)
	}

	err := db.Order("play_time ASC, id ASC").Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}

// ReplayTrackPlayRecords 批量重放播放流水，用于后台补归因和资料库补写。
func ReplayTrackPlayRecords(params ReplayTrackPlayRecordsParams) (*ReplayTrackPlayRecordsReport, error) {
	ctx := params.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	// 获取待补归因或待应用到资料库的播放流水。
	records, err := GetReplayableTrackPlayRecords(
		ctx,
		params.Limit,
		params.Source,
		params.RecordIDs,
		params.PlayedFrom,
		params.PlayedTo,
		params.OnlyUnapplied,
		params.OnlyUnresolved,
	)
	if err != nil {
		return nil, err
	}

	report := &ReplayTrackPlayRecordsReport{
		Results: make([]*ReplayTrackPlayRecordResult, 0, len(records)),
	}

	for _, record := range records {
		result := &ReplayTrackPlayRecordResult{
			ID:            record.ID,
			Artist:        record.Artist,
			Album:         record.Album,
			Track:         record.Track,
			Source:        record.Source,
			BeforeStatus:  record.ResolutionStatus,
			BeforeApplied: record.LibraryApplied,
		}

		if !params.DryRun {
			// 统一处理播放流水的资料库写入和归因回填
			if err := ProcessTrackPlayRecord(ctx, record.ID, inferReplayTrackMetadata(record)); err != nil {
				return nil, err
			}

			stored, err := GetTrackPlayRecordByID(ctx, record.ID)
			if err != nil {
				return nil, err
			}
			result.AfterStatus = stored.ResolutionStatus
			result.AfterApplied = stored.LibraryApplied
			result.ResolvedTrackID = stored.ResolvedTrackID
		}

		report.Results = append(report.Results, result)
	}

	return report, nil
}

// ApplyTrackPlayRecordToResolvedTrackTx 将指定播放流水显式绑定到目标曲目，并同步资料库应用状态。
func ApplyTrackPlayRecordToResolvedTrackTx(
	tx *gorm.DB, recordID, trackID int64, confidence common.TrackMetadataConfidence,
) (bool, error) {
	if tx == nil || recordID <= 0 || trackID <= 0 {
		return false, nil
	}

	record, err := getTrackPlayRecordByIDTx(tx, recordID)
	if err != nil {
		return false, err
	}
	trackObj, err := GetTrackByIDTx(tx, trackID)
	if err != nil {
		return false, err
	}

	albumID := getAlbumIDByTrackInfoTx(
		tx,
		trackObj.Artist,
		trackObj.Album,
		trackObj.AlbumSubtitle,
		trackObj.Track,
		trackObj.TrackNumber,
		trackObj.DiscNumber,
	)
	if albumID <= 0 {
		return false, gorm.ErrRecordNotFound
	}

	appliedNow := false
	if !record.LibraryApplied {
		if err := incrementExistingTrackPlayCountTx(tx, trackID); err != nil {
			return false, err
		}
		appliedNow = true
	}

	fields := map[string]interface{}{
		"resolution_status":     TrackPlayRecordResolutionResolved,
		"resolution_confidence": confidence,
		"library_applied":       true,
	}
	for key, value := range buildTrackPlayRecordResolvedFields(tx, record, trackObj, albumID) {
		fields[key] = value
	}
	if err := tx.Model(&TrackPlayRecord{}).Where("id = ?", recordID).Updates(fields).Error; err != nil {
		return false, err
	}
	return appliedNow, nil
}

// GetTrackPlayRecordByID 根据主键获取播放流水。
func GetTrackPlayRecordByID(ctx context.Context, recordID int64) (*TrackPlayRecord, error) {
	return getTrackPlayRecordByIDTx(GetDB().WithContext(ctx), recordID)
}

// getAlbumIDByTrackInfo 通过 Track -> TrackAlbum 关联获取 AlbumID
func getAlbumIDByTrackInfo(
	ctx context.Context, artist, album, albumSubtitle, track string, trackNumber, discNumber int8,
) int64 {
	return getAlbumIDByTrackInfoTx(
		GetDB().WithContext(ctx), artist, album, albumSubtitle, track, trackNumber, discNumber,
	)
}

func getAlbumIDByTrackInfoTx(
	tx *gorm.DB, artist, album, albumSubtitle, track string, trackNumber, discNumber int8,
) int64 {
	trackObj, err := findTrackByIdentityWithOptions(
		tx,
		TrackIdentity{
			Artist:        artist,
			Album:         album,
			AlbumSubtitle: albumSubtitle,
			Track:         track,
			TrackNumber:   trackNumber,
			DiscNumber:    discNumber,
		},
		trackIdentityResolveOptions{allowLooseNameFallback: false},
	)
	if err != nil {
		return 0
	}

	trackAlbum, err := GetTrackAlbumByTrackAndAlbumIdentityTx(
		tx,
		trackObj.ID,
		artist,
		album,
		albumSubtitle,
		trackNumber,
		discNumber,
	)
	if err != nil {
		return 0
	}
	return trackAlbum.AlbumID
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
	err := recentPlayRecordsQuery(ctx).Order("play_time DESC, id DESC").Limit(limit).Find(&records).Error
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
		err = recentPlayRecordsQuery(ctx).Where(
			"DATE_FORMAT(`tpr`.`play_time`, '%Y-%m-%d') > ?", startTime,
		).Order("play_time DESC, id DESC").Find(&records).Error
	} else {
		err = recentPlayRecordsQuery(ctx).Where(
			"strftime('%Y-%m-%d',`tpr`.`play_time`) > ?", startTime,
		).Order("play_time DESC, id DESC").Find(&records).Error
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
	sourceCounts, err := GetPlayCountsBySourceFromStat(ctx)
	if err == nil && len(sourceCounts) > 0 {
		return sourceCounts, nil
	}

	var result []map[string]interface{}
	err = GetDB().WithContext(ctx).Model(&TrackPlayRecord{}).
		Select("source, COUNT(*) as count").
		Group("source").
		Find(&result).Error
	if err != nil {
		return nil, err
	}

	// 转换为map[string]int64
	sourceCounts = make(map[string]int64)
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

type TopAlbum struct {
	AlbumID           int64  `json:"album_id"`
	Album             string `json:"album"`
	AlbumSubtitle     string `json:"album_subtitle"`
	Artist            string `json:"artist"`
	PlayCount         int    `json:"play_count"`
	CoverArtURL       string `json:"cover_art_url"`
	CoverArtMime      string `json:"cover_art_mime"`
	CoverArtObjectKey string `json:"cover_art_object_key"`
}

type TopTrack struct {
	TrackID           int64  `json:"track_id"`
	Track             string `json:"track"`
	Album             string `json:"album"`
	Artist            string `json:"artist"`
	PlayCount         int    `json:"play_count"`
	Rank              int    `json:"rank"`
	CoverArtURL       string `json:"cover_art_url"`
	CoverArtMime      string `json:"cover_art_mime"`
	CoverArtObjectKey string `json:"cover_art_object_key"`
}

// GetTopAlbumsByPlayCount 获取按播放次数统计的热门专辑
func GetTopAlbumsByPlayCount(ctx context.Context, days int, limit int) ([]*TopAlbum, error) {
	if statRows, err := GetTopAlbumsByPlayCountFromStat(ctx, days, limit); err == nil && len(statRows) > 0 {
		return statRows, nil
	}

	type topAlbumRow struct {
		Album         string
		AlbumSubtitle string
		Artist        string
		PlayCount     int
	}
	var rows []topAlbumRow

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

	err := query.Select("album, MIN(album_subtitle) as album_subtitle, MIN(artist) as artist, COUNT(album) as play_count").
		Group("album").
		Order("play_count DESC").
		Limit(limit).
		Find(&rows).Error

	if err != nil {
		return nil, err
	}

	result := make([]*TopAlbum, 0, len(rows))
	for _, row := range rows {
		var albumObj Album
		albumID := int64(0)
		if err := GetDB().WithContext(ctx).Where(
			"name = ? AND artist = ?", row.Album, row.Artist,
		).First(&albumObj).Error; err == nil {
			albumID = albumObj.ID
		}

		result = append(
			result, &TopAlbum{
				AlbumID:           albumID,
				Album:             row.Album,
				AlbumSubtitle:     row.AlbumSubtitle,
				Artist:            row.Artist,
				PlayCount:         row.PlayCount,
				CoverArtURL:       albumObj.CoverArtURL,
				CoverArtMime:      albumObj.CoverArtMime,
				CoverArtObjectKey: albumObj.CoverArtObjectKey,
			},
		)
	}

	return result, nil
}

// GetTopTracksByPlayCount 获取指定时间窗口内的热门曲目，并携带可复用的专辑封面信息。
func GetTopTracksByPlayCount(ctx context.Context, days int, limit int) ([]*TopTrack, error) {
	return GetTopTracksByPlayCountFromStat(ctx, days, limit)
}
