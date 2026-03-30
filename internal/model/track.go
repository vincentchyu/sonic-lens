package model

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/log"
)

// Track represents a music track with play statistics and favorite status
/*type Track struct {
	ID              int64     `gorm:"primaryKey" json:"id"`
	Artist          string    `gorm:"index;uniqueIndex:uidx_artist_album_track" json:"artist"`
	AlbumArtist     string    `gorm:"index" json:"album_artist"` // 专辑艺术家
	Album           string    `gorm:"index;uniqueIndex:uidx_artist_album_track" json:"album"`
	Track           string    `gorm:"index;uniqueIndex:uidx_artist_album_track" json:"track"` // 歌曲名称
	TrackNumber     int8      `json:"track_number"`                                           // 曲目编号
	Duration        int64     `json:"duration"`                                               // 持续时间(秒)
	Genre           string    `gorm:"index" json:"genre"`                                     // 流派
	Composer        string    `json:"composer"`                                               // 作曲家
	ReleaseDate     string    `json:"release_date"`                                           // 发布日期
	MusicBrainzID   string    `gorm:"column:music_brainz_id;index" json:"musicbrainz_id"`     // MusicBrainz ID
	PlayCount       int       `json:"play_count"`                                             // 播放次数
	IsAppleMusicFav bool      `json:"is_apple_music_fav"`                                     // 是否Apple Music喜欢
	IsLastFmFav     bool      `gorm:"column:is_last_fm_fav" json:"is_lastfm_fav"`             // 是否Last.fm喜欢
	Source          string    `gorm:"index" json:"source"`                                    // 数据来源：Apple Music, Audirvana, Roon等
	BundleID        string    `json:"bundle_id"`                                              // 应用标识符 (用于media-control)
	UniqueID        string    `gorm:"index" json:"unique_id"`                                 // 唯一标识符 (用于media-control)
	Version         int       `gorm:"default:1" json:"version"`                               // 乐观锁版本号
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}*/
type Track struct {
	ID              int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	Artist          string    `gorm:"column:artist;type:varchar(255);not null;uniqueIndex:uidx_t_aatdntn" json:"artist"`
	Album           string    `gorm:"column:album;type:varchar(255);not null;index:idx_track_album;uniqueIndex:uidx_t_aatdntn" json:"album"`
	Track           string    `gorm:"column:track;type:varchar(255);not null;index:idx_track_track;uniqueIndex:uidx_t_aatdntn" json:"track"`
	PlayCount       int       `gorm:"column:play_count;type:int;default:0" json:"play_count"`
	IsAppleMusicFav bool      `gorm:"column:is_apple_music_fav;type:tinyint(1);default:0" json:"is_apple_music_fav"`
	IsLastFmFav     bool      `gorm:"column:is_last_fm_fav;type:tinyint(1);default:0" json:"is_last_fm_fav"`
	Version         int       `gorm:"column:version;type:int;default:1" json:"version"`
	AlbumArtist     string    `gorm:"column:album_artist;type:varchar(255)" json:"album_artist"`
	TrackNumber     int8      `gorm:"column:track_number;type:tinyint;uniqueIndex:uidx_t_aatdntn" json:"track_number"`
	DiscNumber      int8      `gorm:"column:disc_number;type:tinyint;default:1;uniqueIndex:uidx_t_aatdntn" json:"disc_number"` // 碟号
	Duration        int64     `gorm:"column:duration;type:int" json:"duration"`
	Genre           string    `gorm:"column:genre;type:varchar(255);index:idx_track_genre" json:"genre"`
	Composer        string    `gorm:"column:composer;type:varchar(255)" json:"composer"`
	ReleaseDate     string    `gorm:"column:release_date;type:varchar(50)" json:"release_date"`
	MusicBrainzID   string    `gorm:"column:music_brainz_id;type:varchar(255)" json:"music_brainz_id"`
	Source          string    `gorm:"column:source;type:varchar(255);index:idx_track_source" json:"source"`
	BundleID        string    `gorm:"column:bundle_id;type:varchar(255)" json:"bundle_id"`
	UniqueID        string    `gorm:"column:unique_id;type:varchar(255);index:idx_track_unique_id" json:"unique_id"`
	CreatedAt       time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName sets the table name for the Track model
func (Track) TableName() string {
	return "track"
}

func (t *Track) AfterCreate(tx *gorm.DB) error {
	return appendLibraryChangeTx(tx, LibraryEntityTrack, t.ID, LibraryOpUpsert)
}

func (t *Track) AfterUpdate(tx *gorm.DB) error {
	return appendLibraryChangeTx(tx, LibraryEntityTrack, t.ID, LibraryOpUpsert)
}

func (t *Track) AfterDelete(tx *gorm.DB) error {
	return appendLibraryChangeTx(tx, LibraryEntityTrack, t.ID, LibraryOpDelete)
}

// TrackMetadata represents metadata for a music track
type TrackMetadata struct {
	AlbumArtist       string                         `json:"album_artist"`         // 专辑艺术家
	TrackNumber       int8                           `json:"track_number"`         // 曲目编号
	Duration          int64                          `json:"duration"`             // 持续时间(秒)
	Genre             string                         `json:"genre"`                // 流派
	Composer          string                         `json:"composer"`             // 作曲家
	ReleaseDate       string                         `json:"release_date"`         // 发布日期
	MusicBrainzID     string                         `json:"musicbrainz_id"`       // MusicBrainz ID
	Source            string                         `json:"source"`               // 数据来源：Apple Music, Audirvana, Roon等
	BundleID          string                         `json:"bundle_id"`            // 应用标识符 (用于media-control)
	UniqueID          string                         `json:"unique_id"`            // 唯一标识符 (用于media-control)
	DiscNumber        int8                           `json:"disc_number"`          // 盘编号
	PlayerType        string                         `json:"player_type"`          // 播放器类型
	Confidence        common.TrackMetadataConfidence `json:"confidence"`           // 元数据置信度
	ReleaseYear       int                            `json:"release_year"`         // 仅有年份时的弱提示
	CoverArtURL       string                         `json:"cover_art_url"`        // 播放期封面访问地址
	CoverArtMime      string                         `json:"cover_art_mime"`       // 封面 MIME
	CoverArtObjectKey string                         `json:"cover_art_object_key"` // 封面对象存储键
}

// IncrementTrackPlayCountParams represents parameters for IncrementTrackPlayCount function
type IncrementTrackPlayCountParams struct {
	Ctx           context.Context
	Artist        string
	Album         string
	Track         string
	TrackMetadata TrackMetadata
}

// SetFavoriteParams represents parameters for SetAppleMusicFavorite and SetLastFmFavorite functions
type SetFavoriteParams struct {
	Ctx           context.Context
	Artist        string
	Album         string
	Track         string
	IsFavorite    bool
	TrackMetadata TrackMetadata
}

// TrackIdentity 表示曲目的稳定身份键
type TrackIdentity struct {
	Artist      string
	Album       string
	Track       string
	TrackNumber int8
	DiscNumber  int8
}

func normalizeTrackIdentity(identity TrackIdentity) TrackIdentity {
	if identity.DiscNumber == 0 && identity.TrackNumber > 0 {
		identity.DiscNumber = 1
	}
	return identity
}

func mergeTrackIdentityWithMetadata(identity TrackIdentity, metadata TrackMetadata) TrackIdentity {
	identity = normalizeTrackIdentity(identity)

	if metadata.TrackNumber > 0 {
		identity.TrackNumber = metadata.TrackNumber
		if metadata.DiscNumber > 0 {
			identity.DiscNumber = metadata.DiscNumber
		} else if identity.DiscNumber == 0 {
			identity.DiscNumber = 1
		}
	}
	if metadata.DiscNumber > 0 {
		identity.DiscNumber = metadata.DiscNumber
	}

	return normalizeTrackIdentity(identity)
}

type trackIdentityResolveOptions struct {
	allowLooseNameFallback bool
	allowUniqueIDHint      bool
}

func findTrackByIdentity(tx *gorm.DB, identity TrackIdentity) (*Track, error) {
	return findTrackByIdentityWithOptions(
		tx, identity, trackIdentityResolveOptions{allowLooseNameFallback: true},
	)
}

func findTrackByIdentityWithOptions(
	tx *gorm.DB, identity TrackIdentity, options trackIdentityResolveOptions,
) (*Track, error) {
	identity = normalizeTrackIdentity(identity)
	var record Track

	if identity.TrackNumber > 0 || identity.DiscNumber > 0 {
		err := tx.Where(
			"artist = ? AND album = ? AND track = ? AND track_number = ? AND disc_number = ?",
			identity.Artist, identity.Album, identity.Track, identity.TrackNumber, identity.DiscNumber,
		).First(&record).Error
		if err == nil {
			return &record, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	if !options.allowLooseNameFallback {
		return nil, gorm.ErrRecordNotFound
	}

	rows, err := findTracksByArtistAlbumTrack(tx, identity.Artist, identity.Album, identity.Track, 2)
	if err != nil {
		return nil, err
	}
	if len(rows) == 1 {
		record = rows[0]
		return &record, nil
	}
	if len(rows) > 1 {
		return nil, gorm.ErrRecordNotFound
	}

	return nil, gorm.ErrRecordNotFound
}

func findTracksByArtistAlbumTrack(tx *gorm.DB, artist, album, track string, limit int) ([]Track, error) {
	var records []Track
	err := tx.Where(
		"artist = ? AND album = ? AND track = ?",
		artist, album, track,
	).Order("disc_number ASC, track_number ASC, id ASC").Limit(limit).Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}

func findUniqueTrackByDuration(
	tx *gorm.DB, artist, album, track string, duration int64,
) (*Track, error) {
	// todo 存在问题 duration只是播放歌曲的持续时间
	var records []Track
	err := tx.Where(
		"artist = ? AND album = ? AND track = ? AND duration BETWEEN ? AND ?",
		artist, album, track, duration-2, duration+2,
	).Order("disc_number ASC, track_number ASC, id ASC").Limit(2).Find(&records).Error
	if err != nil {
		return nil, err
	}
	if len(records) != 1 {
		return nil, gorm.ErrRecordNotFound
	}
	return &records[0], nil
}

func findSafeTrackByUniqueHint(
	tx *gorm.DB, artist, album, track string, metadata TrackMetadata,
) (*Track, error) {
	if metadata.UniqueID == "" {
		return nil, gorm.ErrRecordNotFound
	}

	query := tx.Where("unique_id = ?", metadata.UniqueID)
	if metadata.BundleID != "" {
		query = query.Where("bundle_id = ?", metadata.BundleID)
	}
	if metadata.Source != "" {
		query = query.Where("source = ?", metadata.Source)
	}

	var records []Track
	if err := query.Order("id ASC").Limit(2).Find(&records).Error; err != nil {
		return nil, err
	}
	if len(records) != 1 {
		return nil, gorm.ErrRecordNotFound
	}
	if records[0].Artist != artist || records[0].Album != album || records[0].Track != track {
		return nil, gorm.ErrRecordNotFound
	}
	return &records[0], nil
}

func resolveTrackIdentity(
	tx *gorm.DB, artist, album, track string, metadata TrackMetadata,
) (TrackIdentity, *Track, error) {
	return resolveTrackIdentityWithOptions(
		tx,
		artist,
		album,
		track,
		metadata,
		trackIdentityResolveOptions{
			allowLooseNameFallback: metadataConfidence(metadata) >= common.TrackMetadataConfidenceHigh,
			allowUniqueIDHint:      metadataConfidence(metadata) >= common.TrackMetadataConfidenceHigh,
		},
	)
}

func resolveTrackIdentityWithOptions(
	tx *gorm.DB, artist, album, track string, metadata TrackMetadata, options trackIdentityResolveOptions,
) (TrackIdentity, *Track, error) {
	identity := normalizeTrackIdentity(
		TrackIdentity{
			Artist:      artist,
			Album:       album,
			Track:       track,
			TrackNumber: metadata.TrackNumber,
			DiscNumber:  metadata.DiscNumber,
		},
	)

	if options.allowUniqueIDHint {
		byUniqueID, err := findSafeTrackByUniqueHint(tx, artist, album, track, metadata)
		if err == nil {
			resolvedIdentity := mergeTrackIdentityWithMetadata(
				TrackIdentity{
					Artist:      byUniqueID.Artist,
					Album:       byUniqueID.Album,
					Track:       byUniqueID.Track,
					TrackNumber: byUniqueID.TrackNumber,
					DiscNumber:  byUniqueID.DiscNumber,
				}, metadata,
			)
			return resolvedIdentity, byUniqueID, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return identity, nil, err
		}
	}

	if metadata.MusicBrainzID != "" {
		var byMusicBrainzID Track
		err := tx.Where(
			"music_brainz_id = ? AND artist = ? AND album = ? AND track = ?",
			metadata.MusicBrainzID, artist, album, track,
		).First(&byMusicBrainzID).Error
		if err == nil {
			resolvedIdentity := mergeTrackIdentityWithMetadata(
				TrackIdentity{
					Artist:      byMusicBrainzID.Artist,
					Album:       byMusicBrainzID.Album,
					Track:       byMusicBrainzID.Track,
					TrackNumber: byMusicBrainzID.TrackNumber,
					DiscNumber:  byMusicBrainzID.DiscNumber,
				}, metadata,
			)
			return resolvedIdentity, &byMusicBrainzID, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return identity, nil, err
		}
	}

	if metadata.Duration > 0 {
		byDuration, err := findUniqueTrackByDuration(tx, artist, album, track, metadata.Duration)
		if err == nil {
			resolvedIdentity := mergeTrackIdentityWithMetadata(
				TrackIdentity{
					Artist:      byDuration.Artist,
					Album:       byDuration.Album,
					Track:       byDuration.Track,
					TrackNumber: byDuration.TrackNumber,
					DiscNumber:  byDuration.DiscNumber,
				}, metadata,
			)
			return resolvedIdentity, byDuration, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return identity, nil, err
		}
	}

	record, err := findTrackByIdentityWithOptions(tx, identity, options)
	if err == nil {
		resolvedIdentity := mergeTrackIdentityWithMetadata(
			TrackIdentity{
				Artist:      record.Artist,
				Album:       record.Album,
				Track:       record.Track,
				TrackNumber: record.TrackNumber,
				DiscNumber:  record.DiscNumber,
			}, metadata,
		)
		return resolvedIdentity, record, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return identity, nil, err
	}

	return identity, nil, nil
}

func metadataConfidence(metadata TrackMetadata) common.TrackMetadataConfidence {
	if metadata.Confidence <= 0 {
		return common.TrackMetadataConfidenceHigh
	}
	return metadata.Confidence
}

func metadataAllowsLibraryMutation(metadata TrackMetadata) bool {
	return metadataConfidence(metadata) >= common.TrackMetadataConfidenceHigh
}

func metadataAllowsAlbumCreation(metadata TrackMetadata) bool {
	return metadataAllowsLibraryMutation(metadata)
}

func metadataAllowsTrackAlbumMutation(metadata TrackMetadata) bool {
	return metadataAllowsLibraryMutation(metadata) && metadata.TrackNumber > 0
}

func ensureGenreExistsTx(ctx context.Context, tx *gorm.DB, genreName string) error {
	if genreName == "" {
		return nil
	}

	var genre Genre
	err := tx.Where("name = ?", genreName).First(&genre).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	genre = Genre{
		Name:      genreName,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := tx.Create(&genre).Error; err != nil {
		log.Warn(ctx, "CreateGenre failed", zap.String("genre", genreName), zap.Error(err))
	}
	return nil
}

func getOrCreatePlaybackAlbumTx(tx *gorm.DB, artist, albumName string, metadata TrackMetadata) (*Album, error) {
	var existing Album
	err := tx.Where("artist = ? AND name = ?", artist, albumName).
		Order("sync_status DESC, id ASC").
		First(&existing).Error
	if err == nil && existing.SyncStatus == 3 {
		// 深度维护完成后的专辑元数据冻结，播放链路不再改写专辑基础字段。
		return &existing, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	album := &Album{
		Name:        albumName,
		Artist:      artist,
		ReleaseDate: metadata.ReleaseDate,
		Genre:       metadata.Genre,
	}
	if err := getOrCreateAlbumTx(tx, album); err != nil {
		return nil, err
	}
	return album, nil
}

func hydrateTrackMetadataFromAlbumPlaceholderTx(
	tx *gorm.DB, albumID int64, trackName string, metadata *TrackMetadata,
) error {
	if metadata == nil {
		return nil
	}

	placeholder, err := FindTrackAlbumPlaceholderTx(
		tx, TrackAlbumPlaceholderLookup{
			AlbumID:     albumID,
			Track:       trackName,
			TrackNumber: metadata.TrackNumber,
			DiscNumber:  metadata.DiscNumber,
		},
	)
	if err == nil {
		if metadata.MusicBrainzID == "" {
			metadata.MusicBrainzID = placeholder.MusicBrainzRecordingID
		}
		if metadata.TrackNumber == 0 {
			metadata.TrackNumber = placeholder.TrackNumber
		}
		if metadata.DiscNumber == 0 {
			metadata.DiscNumber = placeholder.DiscNumber
		}
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	return err
}

func upsertTrackPlayCountTx(
	tx *gorm.DB, artist, albumName, trackName string, metadata *TrackMetadata, existingTrack *Track,
) (*Track, error) {
	if metadata == nil {
		return nil, errors.New("track metadata is nil")
	}

	var track Track
	var err error
	for i := 0; i < 3; i++ {
		if existingTrack != nil {
			track = *existingTrack
			err = nil
			existingTrack = nil
		} else {
			err = tx.Where(
				"artist = ? AND album = ? AND track = ? AND track_number = ? AND disc_number = ?",
				artist, albumName, trackName, metadata.TrackNumber, metadata.DiscNumber,
			).First(&track).Error
		}
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				track = Track{
					Artist:        artist,
					AlbumArtist:   metadata.AlbumArtist,
					Album:         albumName,
					Track:         trackName,
					TrackNumber:   metadata.TrackNumber,
					Duration:      metadata.Duration,
					Genre:         metadata.Genre,
					Composer:      metadata.Composer,
					ReleaseDate:   metadata.ReleaseDate,
					MusicBrainzID: metadata.MusicBrainzID,
					Source:        metadata.Source,
					BundleID:      metadata.BundleID,
					UniqueID:      metadata.UniqueID,
					DiscNumber:    metadata.DiscNumber,
					PlayCount:     1,
					Version:       1,
				}
				if err := tx.Create(&track).Error; err != nil {
					if errors.Is(err, gorm.ErrDuplicatedKey) {
						continue
					}
					return nil, err
				}
				return &track, nil
			}
			return nil, err
		}

		updatedTrack := track
		updatedTrack.PlayCount = track.PlayCount + 1
		UpdateTrackWithTrackMetadata(&updatedTrack, metadata)
		updatedTrack.Version = track.Version + 1

		result := tx.Model(&Track{}).Where(
			"id = ? AND version = ?", track.ID, track.Version,
		).Updates(&updatedTrack)
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected > 0 {
			track = updatedTrack
			return &track, nil
		}
	}

	return nil, errors.New("track optimistic lock retries exhausted")
}

func upsertTrackAlbumLinkTx(tx *gorm.DB, albumID int64, track *Track) error {
	if track == nil {
		return nil
	}

	var ta TrackAlbum
	foundPlaceholder := false
	placeholder, err := FindTrackAlbumPlaceholderTx(
		tx, TrackAlbumPlaceholderLookup{
			AlbumID:     albumID,
			Track:       track.Track,
			TrackNumber: track.TrackNumber,
			DiscNumber:  track.DiscNumber,
		},
	)
	if err == nil {
		ta = *placeholder
		ta.TrackID = track.ID
		if track.TrackNumber > 0 {
			ta.TrackNumber = track.TrackNumber
		}
		if track.DiscNumber > 0 && ta.DiscNumber <= 0 {
			ta.DiscNumber = track.DiscNumber
		}
		if ta.Track == "" {
			ta.Track = track.Track
		}
		if ta.MusicBrainzRecordingID == "" {
			ta.MusicBrainzRecordingID = track.MusicBrainzID
		}
		if err := upsertTrackAlbumTx(tx, &ta, false); err != nil {
			return err
		}
		foundPlaceholder = true
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if foundPlaceholder {
		return nil
	}

	ta = TrackAlbum{
		TrackID:                track.ID,
		AlbumID:                albumID,
		Track:                  track.Track,
		TrackNumber:            track.TrackNumber,
		DiscNumber:             track.DiscNumber,
		MusicBrainzRecordingID: track.MusicBrainzID,
	}
	return upsertTrackAlbumTx(tx, &ta, true)
}

func incrementExistingTrackPlayCountTx(tx *gorm.DB, trackID int64) error {
	if trackID <= 0 {
		return nil
	}

	for i := 0; i < 3; i++ {
		current, err := GetTrackByIDTx(tx, trackID)
		if err != nil {
			return err
		}

		result := tx.Model(&Track{}).Where(
			"id = ? AND version = ?", current.ID, current.Version,
		).Updates(
			map[string]interface{}{
				"play_count": current.PlayCount + 1,
				"version":    current.Version + 1,
			},
		)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			return nil
		}
	}

	return errors.New("track optimistic lock retries exhausted")
}

func applyPlayToLibraryTx(tx *gorm.DB, params IncrementTrackPlayCountParams) error {
	// 流派更新
	if err := ensureGenreExistsTx(params.Ctx, tx, params.TrackMetadata.Genre); err != nil {
		return err
	}
	//
	album, err := getOrCreatePlaybackAlbumTx(tx, params.Artist, params.Album, params.TrackMetadata)
	if err != nil {
		return err
	}

	if err := hydrateTrackMetadataFromAlbumPlaceholderTx(
		tx, album.ID, params.Track, &params.TrackMetadata,
	); err != nil {
		return err
	}

	identity, existingTrack, err := resolveTrackIdentity(
		tx, params.Artist, params.Album, params.Track, params.TrackMetadata,
	)
	if err != nil {
		return err
	}
	params.TrackMetadata.TrackNumber = identity.TrackNumber
	params.TrackMetadata.DiscNumber = identity.DiscNumber

	track, err := upsertTrackPlayCountTx(
		tx, params.Artist, params.Album, params.Track, &params.TrackMetadata, existingTrack,
	)
	if err != nil {
		return err
	}

	if !metadataAllowsTrackAlbumMutation(params.TrackMetadata) {
		return nil
	}

	return upsertTrackAlbumLinkTx(tx, album.ID, track)
}

func applyTrackPlayMutationTx(tx *gorm.DB, params IncrementTrackPlayCountParams) (bool, error) {
	// common.TrackMetadataConfidenceHigh 不会来这里
	if !metadataAllowsLibraryMutation(params.TrackMetadata) {
		if historicalTrack, _, err := findLatestResolvedTrackByIdentityAndSourceTx(
			tx,
			params.Artist,
			params.Album,
			params.Track,
			params.TrackMetadata.Source,
			params.TrackMetadata.TrackNumber,
			params.TrackMetadata.DiscNumber,
		); err == nil {
			if err := incrementExistingTrackPlayCountTx(tx, historicalTrack.ID); err != nil {
				return false, err
			}
			return true, nil
		}

		identity, existingTrack, err := resolveTrackIdentityWithOptions(
			tx,
			params.Artist,
			params.Album,
			params.Track,
			params.TrackMetadata,
			trackIdentityResolveOptions{
				allowLooseNameFallback: false,
				allowUniqueIDHint:      false,
			},
		)
		if err != nil {
			return false, err
		}
		if existingTrack == nil {
			return false, nil
		}

		params.TrackMetadata.TrackNumber = identity.TrackNumber
		params.TrackMetadata.DiscNumber = identity.DiscNumber
		if err := incrementExistingTrackPlayCountTx(tx, existingTrack.ID); err != nil {
			return false, err
		}
		return true, nil
	}
	// todo 这里是被认证过的
	if err := applyPlayToLibraryTx(tx, params); err != nil {
		return false, err
	}
	return true, nil
}

// IncrementTrackPlayCount increments the play count for a track and ensures associated entities exist
func IncrementTrackPlayCount(params IncrementTrackPlayCountParams) error {
	// 验证艺术家、专辑和曲目信息
	if err := common.ValidateTrackInfo(params.Ctx, params.Artist, params.Album, params.Track); err != nil {
		return err
	}

	return GetDB().WithContext(params.Ctx).Transaction(
		func(tx *gorm.DB) error {
			_, err := applyTrackPlayMutationTx(tx, params)
			return err
		},
	)
}

func incrementTrackPlayCountResolvedOnly(params IncrementTrackPlayCountParams) error {
	return GetDB().WithContext(params.Ctx).Transaction(
		func(tx *gorm.DB) error {
			identity, existingTrack, err := resolveTrackIdentityWithOptions(
				tx,
				params.Artist,
				params.Album,
				params.Track,
				params.TrackMetadata,
				trackIdentityResolveOptions{
					allowLooseNameFallback: false,
					allowUniqueIDHint:      false,
				},
			)
			if err != nil {
				return err
			}
			if existingTrack == nil {
				return nil
			}

			params.TrackMetadata.TrackNumber = identity.TrackNumber
			params.TrackMetadata.DiscNumber = identity.DiscNumber

			return incrementExistingTrackPlayCountTx(tx, existingTrack.ID)
		},
	)
}

func setAppleMusicFavoriteByTrackIDTx(tx *gorm.DB, trackID int64, isFavorite bool) error {
	if trackID <= 0 {
		return gorm.ErrRecordNotFound
	}

	for i := 0; i < 3; i++ {
		record, err := GetTrackByIDTx(tx, trackID)
		if err != nil {
			return err
		}

		updatedRecord := *record
		updatedRecord.IsAppleMusicFav = isFavorite
		updatedRecord.Version = record.Version + 1

		result := tx.Where("id = ? AND version = ?", record.ID, record.Version).Updates(&updatedRecord)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			return nil
		}
	}

	return errors.New("track optimistic lock retries exhausted")
}

func setLastFmFavoriteByTrackIDTx(tx *gorm.DB, trackID int64, isFavorite bool) error {
	if trackID <= 0 {
		return gorm.ErrRecordNotFound
	}

	for i := 0; i < 3; i++ {
		record, err := GetTrackByIDTx(tx, trackID)
		if err != nil {
			return err
		}

		updatedRecord := *record
		updatedRecord.IsLastFmFav = isFavorite
		updatedRecord.Version = record.Version + 1

		result := tx.Where("id = ? AND version = ?", record.ID, record.Version).Updates(&updatedRecord)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			return nil
		}
	}

	return errors.New("track optimistic lock retries exhausted")
}

func findTrackForFavoriteWriteTx(
	tx *gorm.DB, params SetFavoriteParams,
) (*Track, TrackIdentity, error) {
	identity, existingTrack, err := resolveTrackIdentityWithOptions(
		tx,
		params.Artist,
		params.Album,
		params.Track,
		params.TrackMetadata,
		trackIdentityResolveOptions{
			allowLooseNameFallback: false,
			allowUniqueIDHint:      false,
		},
	)
	if err != nil {
		return nil, identity, err
	}
	if existingTrack != nil {
		return existingTrack, identity, nil
	}

	if trackID, _, err := FindLatestResolvedTrackIDByIdentityTx(
		tx,
		params.Artist,
		params.Album,
		params.Track,
		params.TrackMetadata.TrackNumber,
		params.TrackMetadata.DiscNumber,
	); err == nil && trackID > 0 {
		trackObj, getErr := GetTrackByIDTx(tx, trackID)
		return trackObj, identity, getErr
	}

	return nil, identity, nil
}

func applyTrackFavoriteBySourceTx(tx *gorm.DB, trackID int64, source string, isFavorite bool) error {
	switch source {
	case TrackFavoriteEventSourceLastFm:
		return setLastFmFavoriteByTrackIDTx(tx, trackID, isFavorite)
	case TrackFavoriteEventSourceAppleMusic, "":
		return setAppleMusicFavoriteByTrackIDTx(tx, trackID, isFavorite)
	default:
		return setAppleMusicFavoriteByTrackIDTx(tx, trackID, isFavorite)
	}
}

// SetAppleMusicFavorite updates the Apple Music favorite status for a track
func SetAppleMusicFavorite(params SetFavoriteParams) error {
	// 验证艺术家、专辑和曲目信息
	if err := common.ValidateTrackInfo(params.Ctx, params.Artist, params.Album, params.Track); err != nil {
		return err
	}

	return InTx(
		params.Ctx, func(tx *gorm.DB) error {
			existingTrack, identity, err := findTrackForFavoriteWriteTx(tx, params)
			if err != nil {
				return err
			}
			params.TrackMetadata.TrackNumber = identity.TrackNumber
			params.TrackMetadata.DiscNumber = identity.DiscNumber
			if existingTrack != nil && existingTrack.IsAppleMusicFav == params.IsFavorite {
				return nil
			}

			eventCandidate := &TrackFavoriteEvent{
				Source:           TrackFavoriteEventSourceAppleMusic,
				ProviderFavorite: params.IsFavorite,
				Artist:           params.Artist,
				Album:            params.Album,
				Track:            params.Track,
				AlbumArtist:      params.TrackMetadata.AlbumArtist,
				TrackNumber:      params.TrackMetadata.TrackNumber,
				DiscNumber:       params.TrackMetadata.DiscNumber,
				MusicBrainzID:    params.TrackMetadata.MusicBrainzID,
				Duration:         params.TrackMetadata.Duration,
				BundleID:         params.TrackMetadata.BundleID,
				UniqueID:         params.TrackMetadata.UniqueID,
				ResolutionStatus: TrackFavoriteEventResolutionPending,
			}
			event, _, err := getOrCreateOpenTrackFavoriteEventTx(tx, eventCandidate)
			if err != nil {
				return err
			}

			if trackID, confidence, err := FindLatestResolvedTrackIDByIdentityTx(
				tx, params.Artist, params.Album, params.Track, params.TrackMetadata.TrackNumber,
				params.TrackMetadata.DiscNumber,
			); err == nil {
				if err := setAppleMusicFavoriteByTrackIDTx(tx, trackID, params.IsFavorite); err != nil {
					return err
				}
				return updateTrackFavoriteEventResolutionTx(
					tx,
					event.ID,
					trackID,
					TrackFavoriteEventResolutionResolved,
					common.TrackMetadataConfidence(confidence),
					true,
				)
			}

			if existingTrack == nil {
				return updateTrackFavoriteEventResolutionTx(
					tx,
					event.ID,
					0,
					TrackFavoriteEventResolutionUnresolved,
					metadataConfidence(params.TrackMetadata),
					false,
				)
			}

			if err := setAppleMusicFavoriteByTrackIDTx(tx, existingTrack.ID, params.IsFavorite); err != nil {
				return err
			}
			return updateTrackFavoriteEventResolutionTx(
				tx,
				event.ID,
				existingTrack.ID,
				TrackFavoriteEventResolutionResolved,
				metadataConfidence(params.TrackMetadata),
				true,
			)
		},
	)
}

// SetLastFmFavorite updates the Last.fm favorite status for a track
func SetLastFmFavorite(params SetFavoriteParams) error {
	// 验证艺术家、专辑和曲目信息
	if err := common.ValidateTrackInfo(params.Ctx, params.Artist, params.Album, params.Track); err != nil {
		return err
	}

	return InTx(
		params.Ctx, func(tx *gorm.DB) error {
			existingTrack, identity, err := findTrackForFavoriteWriteTx(tx, params)
			if err != nil {
				return err
			}
			params.TrackMetadata.TrackNumber = identity.TrackNumber
			params.TrackMetadata.DiscNumber = identity.DiscNumber
			if existingTrack != nil && existingTrack.IsLastFmFav == params.IsFavorite {
				return nil
			}

			eventCandidate := &TrackFavoriteEvent{
				Source:           TrackFavoriteEventSourceLastFm,
				ProviderFavorite: params.IsFavorite,
				Artist:           params.Artist,
				Album:            params.Album,
				Track:            params.Track,
				AlbumArtist:      params.TrackMetadata.AlbumArtist,
				TrackNumber:      params.TrackMetadata.TrackNumber,
				DiscNumber:       params.TrackMetadata.DiscNumber,
				MusicBrainzID:    params.TrackMetadata.MusicBrainzID,
				Duration:         params.TrackMetadata.Duration,
				BundleID:         params.TrackMetadata.BundleID,
				UniqueID:         params.TrackMetadata.UniqueID,
				ResolutionStatus: TrackFavoriteEventResolutionPending,
			}
			event, _, err := getOrCreateOpenTrackFavoriteEventTx(tx, eventCandidate)
			if err != nil {
				return err
			}

			if trackID, confidence, err := FindLatestResolvedTrackIDByIdentityTx(
				tx, params.Artist, params.Album, params.Track, params.TrackMetadata.TrackNumber,
				params.TrackMetadata.DiscNumber,
			); err == nil {
				if err := setLastFmFavoriteByTrackIDTx(tx, trackID, params.IsFavorite); err != nil {
					return err
				}
				return updateTrackFavoriteEventResolutionTx(
					tx,
					event.ID,
					trackID,
					TrackFavoriteEventResolutionResolved,
					common.TrackMetadataConfidence(confidence),
					true,
				)
			}

			if existingTrack == nil {
				return updateTrackFavoriteEventResolutionTx(
					tx,
					event.ID,
					0,
					TrackFavoriteEventResolutionUnresolved,
					metadataConfidence(params.TrackMetadata),
					false,
				)
			}

			if err := setLastFmFavoriteByTrackIDTx(tx, existingTrack.ID, params.IsFavorite); err != nil {
				return err
			}
			return updateTrackFavoriteEventResolutionTx(
				tx,
				event.ID,
				existingTrack.ID,
				TrackFavoriteEventResolutionResolved,
				metadataConfidence(params.TrackMetadata),
				true,
			)
		},
	)
}

// GetTracks retrieves track play counts with pagination and optional keyword search
func GetTracks(ctx context.Context, limit, offset int, keyword string) ([]*Track, error) {
	if statRows, err := GetTrackPlayCountsFromStat(
		ctx, "all", limit, offset, keyword,
	); err == nil && len(statRows) > 0 {
		return statRows, nil
	}

	var records []*Track
	db := GetDB().WithContext(ctx)
	if keyword != "" {
		db = db.Where("MATCH(track, artist) AGAINST(? IN BOOLEAN MODE)", keyword)
	}
	err := db.Order("play_count DESC").Limit(limit).Offset(offset).Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}

// GetTrackCounts returns the total number of tracks
func GetTrackCounts(ctx context.Context) (int64, error) {
	stat, err := GetDashboardOverviewFromStat(ctx)
	if err == nil && stat != nil {
		return stat.TotalTracks, nil
	}
	var count int64
	err = GetDB().WithContext(ctx).Model(&Track{}).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetTrack retrieves a specific track's play count
func GetTrack(ctx context.Context, artist, album, track string) (*Track, error) {
	return GetTrackByIdentity(ctx, artist, album, track, 0, 0)
}

// GetTrackByIdentity 按五元组优先查询曲目，缺少编号时回退到旧三元组
func GetTrackByIdentity(ctx context.Context, artist, album, track string, trackNumber, discNumber int8) (
	*Track, error,
) {
	record, err := GetTrackByIdentityTx(
		GetDB().WithContext(ctx),
		artist,
		album,
		track,
		trackNumber,
		discNumber,
	)
	if err != nil {
		return nil, err
	}
	return record, nil
}

// GetTrackByIdentityTx 在事务内按五元组优先查询曲目，缺少编号时回退到旧三元组。
func GetTrackByIdentityTx(
	tx *gorm.DB, artist, album, track string, trackNumber, discNumber int8,
) (*Track, error) {
	return findTrackByIdentity(
		tx,
		TrackIdentity{
			Artist:      artist,
			Album:       album,
			Track:       track,
			TrackNumber: trackNumber,
			DiscNumber:  discNumber,
		},
	)
}

// GetTrackByIDTx 在事务内按主键获取曲目，供上层编排多个 DAO 时复用。
func GetTrackByIDTx(tx *gorm.DB, trackID int64) (*Track, error) {
	var track Track
	if err := tx.First(&track, trackID).Error; err != nil {
		return nil, err
	}
	return &track, nil
}

// GetTrackByMusicBrainzIDTx 在事务内按 MusicBrainz Recording ID 获取曲目。
func GetTrackByMusicBrainzIDTx(tx *gorm.DB, musicBrainzID string) (*Track, error) {
	if musicBrainzID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var track Track
	if err := tx.Where("music_brainz_id = ?", musicBrainzID).First(&track).Error; err != nil {
		return nil, err
	}
	return &track, nil
}

// GetOrCreateTrackByIdentityTx 在事务内按五元组获取或创建曲目，创建时默认播放次数为 0。
func GetOrCreateTrackByIdentityTx(tx *gorm.DB, candidate *Track) (*Track, error) {
	if candidate == nil {
		return nil, errors.New("candidate track is nil")
	}

	identity := TrackIdentity{
		Artist:      candidate.Artist,
		Album:       candidate.Album,
		Track:       candidate.Track,
		TrackNumber: candidate.TrackNumber,
		DiscNumber:  candidate.DiscNumber,
	}

	existing, err := findTrackByIdentityWithOptions(
		tx,
		identity,
		trackIdentityResolveOptions{allowLooseNameFallback: false},
	)
	if err == nil {
		updated := *existing
		UpdateTrackWithTrackMetadata(
			&updated,
			&TrackMetadata{
				AlbumArtist:   candidate.AlbumArtist,
				TrackNumber:   candidate.TrackNumber,
				DiscNumber:    candidate.DiscNumber,
				Duration:      candidate.Duration,
				Genre:         candidate.Genre,
				Composer:      candidate.Composer,
				ReleaseDate:   candidate.ReleaseDate,
				MusicBrainzID: candidate.MusicBrainzID,
				Source:        candidate.Source,
				BundleID:      candidate.BundleID,
				UniqueID:      candidate.UniqueID,
			},
		)
		result := tx.Model(&Track{}).Where("id = ?", existing.ID).Updates(
			map[string]interface{}{
				"album_artist":    updated.AlbumArtist,
				"duration":        updated.Duration,
				"genre":           updated.Genre,
				"composer":        updated.Composer,
				"release_date":    updated.ReleaseDate,
				"music_brainz_id": updated.MusicBrainzID,
				"source":          updated.Source,
				"bundle_id":       updated.BundleID,
				"unique_id":       updated.UniqueID,
				"disc_number":     updated.DiscNumber,
				"track_number":    updated.TrackNumber,
			},
		)
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected > 0 {
			if err := appendLibraryChangeTx(tx, LibraryEntityTrack, existing.ID, LibraryOpUpsert); err != nil {
				return nil, err
			}
		}
		return GetTrackByIDTx(tx, existing.ID)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	newTrack := *candidate
	newTrack.PlayCount = 0
	if newTrack.Version <= 0 {
		newTrack.Version = 1
	}
	if err := tx.Create(&newTrack).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return GetOrCreateTrackByIdentityTx(tx, candidate)
		}
		return nil, err
	}
	return &newTrack, nil
}

// UpdateTrackMusicBrainzPositionTx 在事务内同步曲目的 MusicBrainz 标识和物理位置。
func UpdateTrackMusicBrainzPositionTx(
	tx *gorm.DB, trackID int64, musicBrainzID string, discNumber, trackNumber int8,
) error {
	result := tx.Model(&Track{}).Where("id = ?", trackID).Updates(
		map[string]interface{}{
			"music_brainz_id": musicBrainzID,
			"disc_number":     discNumber,
			"track_number":    trackNumber,
		},
	)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected <= 0 {
		return nil
	}
	return appendLibraryChangeTx(tx, LibraryEntityTrack, trackID, LibraryOpUpsert)
}

// UpdateTrackMusicBrainzMetadataTx 在事务内同步曲目的 MB 标识、位置和时长。
func UpdateTrackMusicBrainzMetadataTx(
	tx *gorm.DB, trackID int64, musicBrainzID string, discNumber, trackNumber int8, duration int64,
) error {
	fields := map[string]interface{}{
		"music_brainz_id": musicBrainzID,
		"disc_number":     discNumber,
		"track_number":    trackNumber,
	}
	if duration > 0 {
		fields["duration"] = duration
	}
	result := tx.Model(&Track{}).Where("id = ?", trackID).Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected <= 0 {
		return nil
	}
	return appendLibraryChangeTx(tx, LibraryEntityTrack, trackID, LibraryOpUpsert)
}

// UpdateTrackCuratedMetadataTx 在事务内以精选维护优先的规则同步曲目身份和元数据。
func UpdateTrackCuratedMetadataTx(
	tx *gorm.DB, trackID int64, identity *TrackIdentity, metadata *TrackMetadata,
) error {
	if tx == nil || trackID <= 0 {
		return nil
	}

	for i := 0; i < 3; i++ {
		current, err := GetTrackByIDTx(tx, trackID)
		if err != nil {
			return err
		}

		updated := *current
		if identity != nil {
			updated.Artist = identity.Artist
			updated.Album = identity.Album
			updated.Track = identity.Track
			if identity.TrackNumber > 0 {
				updated.TrackNumber = identity.TrackNumber
			}
			if identity.DiscNumber > 0 {
				updated.DiscNumber = identity.DiscNumber
			}
		}
		if metadata != nil {
			if metadata.TrackNumber > 0 {
				updated.TrackNumber = metadata.TrackNumber
			}
			if metadata.DiscNumber > 0 {
				updated.DiscNumber = metadata.DiscNumber
			}
			if metadata.Duration > 0 {
				updated.Duration = metadata.Duration
			}
			if metadata.AlbumArtist != "" {
				updated.AlbumArtist = metadata.AlbumArtist
			}
			if metadata.Genre != "" {
				updated.Genre = metadata.Genre
			}
			if metadata.Composer != "" {
				updated.Composer = metadata.Composer
			}
			if metadata.ReleaseDate != "" {
				updated.ReleaseDate = metadata.ReleaseDate
			}
			if metadata.MusicBrainzID != "" {
				updated.MusicBrainzID = metadata.MusicBrainzID
			}
			if metadata.Source != "" {
				updated.Source = metadata.Source
			}
			if metadata.BundleID != "" {
				updated.BundleID = metadata.BundleID
			}
			if metadata.UniqueID != "" {
				updated.UniqueID = metadata.UniqueID
			}
		}
		updated.Version = current.Version + 1

		result := tx.Where("id = ? AND version = ?", current.ID, current.Version).Updates(&updated)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			return nil
		}
	}

	return errors.New("track optimistic lock retries exhausted")
}

// GetAllTrackPlayCounts retrieves all track play counts
func GetAllTrackPlayCounts(ctx context.Context) ([]*Track, error) {
	var allTracks []*Track
	pageSize := 100
	offset := 0

	for {
		var tracks []*Track
		err := GetDB().WithContext(ctx).Order("play_count DESC").Limit(pageSize).Offset(offset).Find(&tracks).Error
		if err != nil {
			return nil, err
		}

		allTracks = append(allTracks, tracks...)

		// 如果返回的记录数少于pageSize，说明已经获取完所有记录
		if len(tracks) < pageSize {
			break
		}

		offset += pageSize
	}

	return allTracks, nil
}

// GetTracksByArtist retrieves all tracks by a specific artist
func GetTracksByArtist(ctx context.Context, artist string) ([]*Track, error) {
	var tracks []*Track
	err := GetDB().WithContext(ctx).Where("artist = ?", artist).Find(&tracks).Error
	if err != nil {
		return nil, err
	}
	return tracks, nil
}

// GetTotalPlayCount returns the total play count across all tracks
func GetTotalPlayCount(ctx context.Context) (int64, error) {
	stat, err := GetDashboardOverviewFromStat(ctx)
	if err == nil && stat != nil {
		return stat.TotalPlays, nil
	}
	var total int64
	err = GetDB().WithContext(ctx).Model(&Track{}).Select("SUM(play_count)").Scan(&total).Error
	if err != nil {
		return 0, err
	}
	return total, nil
}

// GetArtistCounts returns the total number of unique artists
func GetArtistCounts(ctx context.Context) (int64, error) {
	stat, err := GetDashboardOverviewFromStat(ctx)
	if err == nil && stat != nil {
		return stat.TotalArtist, nil
	}
	var count int64
	err = GetDB().WithContext(ctx).Model(&Track{}).Distinct("artist").Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetAlbumCounts returns the total number of unique albums
func GetAlbumCounts(ctx context.Context) (int64, error) {
	stat, err := GetDashboardOverviewFromStat(ctx)
	if err == nil && stat != nil {
		return stat.TotalAlbums, nil
	}
	var count int64
	err = GetDB().WithContext(ctx).Model(&Track{}).Distinct("album").Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetTopArtistsByPlayCount returns the top artists by play count
func GetTopArtistsByPlayCount(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	statResult, err := GetTopArtistsFromStat(ctx, "plays", limit)
	if err == nil && len(statResult) > 0 {
		return statResult, nil
	}
	var result []map[string]interface{}
	err = GetDB().WithContext(ctx).Model(&Track{}).
		Select("artist, SUM(play_count) as play_count").
		Group("artist").
		Order("SUM(play_count) DESC").
		Limit(limit).
		Find(&result).Error
	if err != nil {
		return nil, err
	}
	for index := range result {
		result[index]["rank"] = index + 1
	}
	return result, nil
}

// GetTopArtistsByTrackCount returns the top artists by track count
func GetTopArtistsByTrackCount(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	statResult, err := GetTopArtistsFromStat(ctx, "tracks", limit)
	if err == nil && len(statResult) > 0 {
		return statResult, nil
	}
	var result []map[string]interface{}
	err = GetDB().WithContext(ctx).Model(&Track{}).
		Select("artist, COUNT(*) as track_count").
		Group("artist").
		Order("COUNT(*) DESC").
		Limit(limit).
		Find(&result).Error
	if err != nil {
		return nil, err
	}
	for index := range result {
		result[index]["rank"] = index + 1
	}
	return result, nil
}

// GetTracksByPeriod retrieves track play counts for a specific period with optional keyword search
func GetTracksByPeriod(ctx context.Context, limit int, offset int, period string, keyword string) ([]*Track, error) {
	if statRows, err := GetTrackPlayCountsFromStat(
		ctx, period, limit, offset, keyword,
	); err == nil && len(statRows) > 0 {
		return statRows, nil
	}

	// 计算时间范围
	var startTime time.Time
	switch period {
	case "week":
		startTime = time.Now().AddDate(0, 0, -7)
	case "month":
		startTime = time.Now().AddDate(0, -1, 0)
	default:
		// 默认返回所有时间的数据
		return GetTracks(ctx, limit, offset, keyword)
	}

	type aggRow struct {
		Artist      string
		Album       string
		Track       string
		TrackNumber int8
		DiscNumber  int8
		PlayCount   int64
	}
	var rows []aggRow
	db := GetDB().WithContext(ctx).Model(&TrackPlayRecord{}).Where("play_time >= ?", startTime)
	if keyword != "" {
		if config.ConfigObj.Database.Type == string(common.DatabaseTypeMySQL) {
			db = db.Where("MATCH(track, artist, album) AGAINST(? IN BOOLEAN MODE)", keyword)
		} else {
			kw := "%" + keyword + "%"
			db = db.Where("track LIKE ? OR artist LIKE ? OR album LIKE ?", kw, kw, kw)
		}
	}
	err := db.Select("artist, album, track, track_number, disc_number, COUNT(*) as play_count").
		Group("artist, album, track, track_number, disc_number").
		Order("play_count DESC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]*Track, 0, len(rows))
	for _, row := range rows {
		result = append(
			result,
			&Track{
				Artist:      row.Artist,
				Album:       row.Album,
				Track:       row.Track,
				TrackNumber: row.TrackNumber,
				DiscNumber:  row.DiscNumber,
				PlayCount:   int(row.PlayCount),
			},
		)
	}
	return result, nil
}

// GetAppleMusicFavorite retrieves the Apple Music favorite status for a track
func GetAppleMusicFavorite(ctx context.Context, artist, album, track string) (bool, error) {
	record, err := GetTrack(ctx, artist, album, track)
	if err != nil {
		return false, err
	}
	return record.IsAppleMusicFav, nil
}

// GetAppleMusicFavoriteByIdentity 获取指定身份曲目的 Apple Music 收藏状态
func GetAppleMusicFavoriteByIdentity(
	ctx context.Context, artist, album, track string, trackNumber, discNumber int8,
) (bool, error) {
	record, err := GetTrackByIdentity(ctx, artist, album, track, trackNumber, discNumber)
	if err != nil {
		return false, err
	}
	return record.IsAppleMusicFav, nil
}

// GetLastFmFavorite retrieves the Last.fm favorite status for a track
func GetLastFmFavorite(ctx context.Context, artist, album, track string) (bool, error) {
	record, err := GetTrack(ctx, artist, album, track)
	if err != nil {
		return false, err
	}
	return record.IsLastFmFav, nil
}

// GetLastFmFavoriteByIdentity 获取指定身份曲目的 Last.fm 收藏状态
func GetLastFmFavoriteByIdentity(
	ctx context.Context, artist, album, track string, trackNumber, discNumber int8,
) (bool, error) {
	record, err := GetTrackByIdentity(ctx, artist, album, track, trackNumber, discNumber)
	if err != nil {
		return false, err
	}
	return record.IsLastFmFav, nil
}

func UpdateTrackWithTrackMetadata(track *Track, newTrack *TrackMetadata) {
	if track == nil || newTrack == nil {
		return
	}

	// Update fields that might be missing from exiftool but available in media control
	if track.Duration == 0 && newTrack.Duration > 0 {
		track.Duration = newTrack.Duration
	}

	if track.AlbumArtist == "" && newTrack.AlbumArtist != "" {
		track.AlbumArtist = newTrack.AlbumArtist
	}

	if track.TrackNumber == 0 && newTrack.TrackNumber > 0 {
		track.TrackNumber = newTrack.TrackNumber
	}

	if track.DiscNumber == 0 && newTrack.DiscNumber > 0 {
		track.DiscNumber = newTrack.DiscNumber
	}

	if track.MusicBrainzID == "" && newTrack.MusicBrainzID != "" {
		track.MusicBrainzID = newTrack.MusicBrainzID
	}

	if track.Genre == "" && newTrack.Genre != "" {
		track.Genre = newTrack.Genre
	}

	if track.ReleaseDate == "" && newTrack.ReleaseDate != "" {
		track.ReleaseDate = newTrack.ReleaseDate
	}

	if track.Composer == "" && newTrack.Composer != "" {
		track.Composer = newTrack.Composer
	}

	if track.BundleID == "" && newTrack.BundleID != "" {
		track.BundleID = newTrack.BundleID
	}

	if track.UniqueID == "" && newTrack.UniqueID != "" {
		track.UniqueID = newTrack.UniqueID
	}

	// Update source if not set
	if track.Source == "" && newTrack.Source != "" {
		track.Source = newTrack.Source
	}
}

// GetTracksOrderedByAlbum retrieves tracks ordered by album name, disc number and track number
func GetTracksOrderedByAlbum(ctx context.Context, limit, offset int, keyword string) ([]*Track, error) {
	var tracks []*Track
	db := GetDB().WithContext(ctx)
	if keyword != "" {
		kw := "%" + keyword + "%"
		db = db.Where("track LIKE ? OR artist LIKE ? OR album LIKE ?", kw, kw, kw)
	}
	err := db.Order("album ASC, disc_number ASC, track_number ASC").Limit(limit).Offset(offset).Find(&tracks).Error
	return tracks, err
}

func GetTracksOrderedByAlbumCount(ctx context.Context, keyword string) (int64, error) {
	var count int64
	db := GetDB().WithContext(ctx).Model(&Track{})
	if keyword != "" {
		kw := "%" + keyword + "%"
		db = db.Where("track LIKE ? OR artist LIKE ? OR album LIKE ?", kw, kw, kw)
	}
	err := db.Count(&count).Error
	return count, err
}
