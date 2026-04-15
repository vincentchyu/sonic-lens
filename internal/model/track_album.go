package model

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// TrackAlbum represents the relationship between a track and an album
type TrackAlbum struct {
	ID                     int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	TrackID                int64     `gorm:"column:track_id;type:bigint;not null;index:idx_ta_track_album" json:"track_id"`
	AlbumID                int64     `gorm:"column:album_id;type:bigint;not null;index:idx_ta_track_album;index:idx_ta_album_id;index:idx_ta_album_disc_track" json:"album_id"`
	TrackNumber            int8      `gorm:"column:track_number;type:tinyint;index:idx_ta_album_disc_track" json:"track_number"`
	DiscNumber             int8      `gorm:"column:disc_number;type:tinyint;default:1;index:idx_ta_album_disc_track" json:"disc_number"` // 碟号
	MusicBrainzRecordingID string    `gorm:"column:mb_recording_id;type:varchar(255)" json:"mb_recording_id"`                            // MusicBrainz Recording ID 冗余
	Track                  string    `gorm:"column:track;type:varchar(255)" json:"track"`                                                // track 冗余
	CreatedAt              time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt              time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName sets the table name for the TrackAlbum model
func (TrackAlbum) TableName() string {
	return "track_album"
}

// TrackAlbumPlaceholderLookup 描述占位符查找条件。
type TrackAlbumPlaceholderLookup struct {
	AlbumID     int64
	Track       string
	TrackNumber int8
	DiscNumber  int8
}

func normalizeTrackAlbumPosition(trackNumber, discNumber int8) (int8, int8) {
	if discNumber == 0 && trackNumber > 0 {
		discNumber = 1
	}
	return trackNumber, discNumber
}

// FindTrackAlbumByPositionTx 根据专辑内的物理位置查找关联记录。
func FindTrackAlbumByPositionTx(tx *gorm.DB, albumID int64, trackNumber, discNumber int8) (*TrackAlbum, error) {
	trackNumber, discNumber = normalizeTrackAlbumPosition(trackNumber, discNumber)
	if trackNumber <= 0 {
		return nil, gorm.ErrRecordNotFound
	}

	var result TrackAlbum
	err := tx.Where(
		"album_id = ? AND track_number = ? AND disc_number = ?",
		albumID, trackNumber, discNumber,
	).Order("track_id DESC, id ASC").First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// FindTrackAlbumPlaceholderTx 优先按物理位置匹配占位符，不足时再按曲名兜底。
func FindTrackAlbumPlaceholderTx(tx *gorm.DB, lookup TrackAlbumPlaceholderLookup) (*TrackAlbum, error) {
	lookup.TrackNumber, lookup.DiscNumber = normalizeTrackAlbumPosition(lookup.TrackNumber, lookup.DiscNumber)

	base := tx.Where("album_id = ? AND track_id = 0", lookup.AlbumID)
	if lookup.TrackNumber > 0 {
		var positional TrackAlbum
		err := base.Where(
			"track_number = ? AND disc_number = ?", lookup.TrackNumber, lookup.DiscNumber,
		).Order("id ASC").First(&positional).Error
		if err == nil {
			return &positional, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	if lookup.Track != "" {
		var named TrackAlbum
		err := base.Where("track = ?", lookup.Track).
			Order("disc_number ASC, track_number ASC, id ASC").
			First(&named).Error
		if err == nil {
			return &named, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	return nil, gorm.ErrRecordNotFound
}

func mergeTrackAlbumFields(target, source *TrackAlbum) {
	source.TrackNumber, source.DiscNumber = normalizeTrackAlbumPosition(source.TrackNumber, source.DiscNumber)
	target.TrackNumber, target.DiscNumber = normalizeTrackAlbumPosition(target.TrackNumber, target.DiscNumber)

	if source.TrackID > 0 {
		target.TrackID = source.TrackID
	}
	if source.Track != "" {
		target.Track = source.Track
	}
	if source.TrackNumber > 0 {
		target.TrackNumber = source.TrackNumber
	}
	if source.DiscNumber > 0 {
		target.DiscNumber = source.DiscNumber
	}
	if source.MusicBrainzRecordingID != "" {
		target.MusicBrainzRecordingID = source.MusicBrainzRecordingID
	}
}

func isAlbumTrackLayoutLockedTx(tx *gorm.DB, albumID int64) (bool, error) {
	type albumSyncRow struct {
		SyncStatus int `gorm:"column:sync_status"`
	}
	var row albumSyncRow
	err := tx.Table("album").Select("sync_status").Where("id = ?", albumID).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return row.SyncStatus == 3, nil
}

func upsertTrackAlbumTx(tx *gorm.DB, ta *TrackAlbum, isCreate bool) error {
	ta.TrackNumber, ta.DiscNumber = normalizeTrackAlbumPosition(ta.TrackNumber, ta.DiscNumber)
	layoutLocked, err := isAlbumTrackLayoutLockedTx(tx, ta.AlbumID)
	if err != nil {
		return err
	}

	if ta.TrackID > 0 {
		var exact TrackAlbum
		err = tx.Where("track_id = ? AND album_id = ?", ta.TrackID, ta.AlbumID).First(&exact).Error
		if err == nil {
			if layoutLocked {
				// 深度维护后的专辑结构冻结，播放链路不应再改写曲目物理位置或 MB 绑定。
				return nil
			}
			originalTrackNumber := exact.TrackNumber
			originalDiscNumber := exact.DiscNumber
			mergeTrackAlbumFields(&exact, ta)
			// 已存在记录本来就在目标物理位置上时，允许先保存自身。
			// 这类情况通常意味着历史脏数据导致同一位置存在重复真实记录，
			// 需要等后续错位记录在同一事务中被修正后再自然收敛，不能在这里提前报冲突中断修复。
			if originalTrackNumber == exact.TrackNumber && originalDiscNumber == exact.DiscNumber {
				return tx.Save(&exact).Error
			}
			if exact.TrackNumber > 0 {
				positional, posErr := FindTrackAlbumByPositionTx(tx, exact.AlbumID, exact.TrackNumber, exact.DiscNumber)
				if posErr == nil && positional.ID != exact.ID && positional.TrackID == 0 {
					mergeTrackAlbumFields(&exact, positional)
					if err := tx.Delete(&TrackAlbum{}, positional.ID).Error; err != nil {
						return err
					}
				} else if posErr == nil && positional.ID != exact.ID && positional.TrackID != exact.TrackID {
					return fmt.Errorf(
						"track_album position conflict: album_id=%d disc=%d track=%d existing_track_id=%d target_track_id=%d",
						exact.AlbumID, exact.DiscNumber, exact.TrackNumber, positional.TrackID, exact.TrackID,
					)
				} else if posErr != nil && !errors.Is(posErr, gorm.ErrRecordNotFound) {
					return posErr
				}
			}
			return tx.Save(&exact).Error
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}

	if ta.TrackNumber > 0 && !layoutLocked {
		positional, posErr := FindTrackAlbumByPositionTx(tx, ta.AlbumID, ta.TrackNumber, ta.DiscNumber)
		if posErr == nil {
			if positional.TrackID != 0 && positional.TrackID != ta.TrackID {
				return fmt.Errorf(
					"track_album position conflict: album_id=%d disc=%d track=%d existing_track_id=%d target_track_id=%d",
					ta.AlbumID, ta.DiscNumber, ta.TrackNumber, positional.TrackID, ta.TrackID,
				)
			}
			mergeTrackAlbumFields(positional, ta)
			return tx.Save(positional).Error
		}
		if !errors.Is(posErr, gorm.ErrRecordNotFound) {
			return posErr
		}
	}
	//
	if !isCreate && layoutLocked {
		return nil
	}
	return tx.Create(ta).Error
}

// UpsertTrackAlbumTx 在事务内按专辑物理位置与 TrackID 统一维护关联关系。
func UpsertTrackAlbumTx(tx *gorm.DB, ta *TrackAlbum, isCreate bool) error {
	return upsertTrackAlbumTx(tx, ta, isCreate)
}

func GetOrCreateTrackAlbum(ctx context.Context, ta *TrackAlbum) error {
	return upsertTrackAlbumTx(GetDB().WithContext(ctx), ta, false)
}

// GetTrackAlbumByTrackID 获取曲目对应的首条专辑映射，供上层查询 album_id 使用。
func GetTrackAlbumByTrackID(ctx context.Context, trackID int64) (*TrackAlbum, error) {
	var result TrackAlbum
	err := GetDB().WithContext(ctx).
		Where("track_id = ?", trackID).
		Order("disc_number ASC, track_number ASC, id ASC").
		First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func GetTrackAlbumsByAlbum(ctx context.Context, albumID int64) ([]*TrackAlbum, error) {
	return GetTrackAlbumsByAlbumTx(GetDB().WithContext(ctx), albumID)
}

// GetTrackAlbumsByAlbumTx 在事务内获取专辑关联的全部曲目映射。
func GetTrackAlbumsByAlbumTx(tx *gorm.DB, albumID int64) ([]*TrackAlbum, error) {
	var results []*TrackAlbum
	err := tx.Where("album_id = ?", albumID).
		Order("disc_number ASC, track_number ASC, id ASC").
		Find(&results).Error
	return results, err
}

// GetTrackAlbumByTrackAndAlbumIdentityTx 按曲目和专辑身份获取对应绑定，避免跨版本命中首条关联。
func GetTrackAlbumByTrackAndAlbumIdentityTx(
	tx *gorm.DB,
	trackID int64,
	artist string,
	album string,
	albumSubtitle string,
	trackNumber int8,
	discNumber int8,
) (*TrackAlbum, error) {
	if trackID <= 0 {
		return nil, gorm.ErrRecordNotFound
	}

	trackNumber, discNumber = normalizeTrackAlbumPosition(trackNumber, discNumber)

	query := tx.Table("track_album AS ta").
		Select("ta.*").
		Joins("JOIN album AS a ON a.id = ta.album_id").
		Where("ta.track_id = ?", trackID).
		Where("a.artist = ? AND a.name = ? AND COALESCE(a.name_subtitle, '') = ?", artist, album, albumSubtitle)
	if trackNumber > 0 {
		query = query.Where("ta.track_number = ? AND ta.disc_number = ?", trackNumber, discNumber)
	}

	var result TrackAlbum
	if err := query.Order("ta.id ASC").First(&result).Error; err == nil {
		return &result, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 兼容历史链路：若当前 track 只有唯一一条绑定，允许回退；
	// 但一旦存在多专辑绑定，就必须要求命中明确的专辑身份，避免串版。
	var candidates []TrackAlbum
	fallback := tx.Where("track_id = ?", trackID)
	if trackNumber > 0 {
		fallback = fallback.Where("track_number = ? AND disc_number = ?", trackNumber, discNumber)
	}
	if err := fallback.Order("id ASC").Limit(2).Find(&candidates).Error; err != nil {
		return nil, err
	}
	if len(candidates) == 1 {
		return &candidates[0], nil
	}
	return nil, gorm.ErrRecordNotFound
}

// ClearTrackAlbumMBRecordingIDByAlbumID 清空专辑下全部 TrackAlbum 的 MusicBrainz 录音 ID。
func ClearTrackAlbumMBRecordingIDByAlbumID(ctx context.Context, albumID int64) error {
	return ClearTrackAlbumMBRecordingIDByAlbumIDTx(GetDB().WithContext(ctx), albumID)
}

// ClearTrackAlbumMBRecordingIDByAlbumIDTx 在事务内清空专辑下全部 TrackAlbum 的 MusicBrainz 录音 ID。
func ClearTrackAlbumMBRecordingIDByAlbumIDTx(tx *gorm.DB, albumID int64) error {
	return tx.Model(&TrackAlbum{}).Where("album_id = ?", albumID).Update("mb_recording_id", "").Error
}

// SaveTrackAlbumTx 在事务内保存 TrackAlbum 记录。
func SaveTrackAlbumTx(tx *gorm.DB, ta *TrackAlbum) error {
	return tx.Save(ta).Error
}

// CountTrackAlbumsByAlbumAndRecordingIDTx 在事务内统计指定录音 ID 是否已存在关联。
func CountTrackAlbumsByAlbumAndRecordingIDTx(tx *gorm.DB, albumID int64, recordingID string) (int64, error) {
	var count int64
	err := tx.Model(&TrackAlbum{}).Where(
		"album_id = ? AND mb_recording_id = ?", albumID, recordingID,
	).Count(&count).Error
	return count, err
}

// GetTrackAlbumByAlbumAndRecordingIDTx 在事务内按专辑和录音 ID 获取关联。
func GetTrackAlbumByAlbumAndRecordingIDTx(tx *gorm.DB, albumID int64, recordingID string) (*TrackAlbum, error) {
	var ta TrackAlbum
	err := tx.Where("album_id = ? AND mb_recording_id = ?", albumID, recordingID).
		Order("track_id DESC, id ASC").
		First(&ta).Error
	if err != nil {
		return nil, err
	}
	return &ta, nil
}

// HasAuthoritativeTrackAlbumBindingTx 判断曲目是否存在可视为权威归因的专辑绑定。
func HasAuthoritativeTrackAlbumBindingTx(tx *gorm.DB, trackID int64) (bool, error) {
	type row struct {
		Count int64 `gorm:"column:cnt"`
	}
	var result row
	err := tx.Table("track_album ta").
		Select("COUNT(1) AS cnt").
		Joins("JOIN album a ON a.id = ta.album_id").
		Where("ta.track_id = ?", trackID).
		Where("a.sync_status = ?", 3).
		Where(
			"(ta.mb_recording_id IS NOT NULL AND ta.mb_recording_id <> '') OR (ta.track_number > 0 AND ta.disc_number > 0)",
		).
		Scan(&result).Error
	if err != nil {
		return false, err
	}
	return result.Count > 0, nil
}

// CreateTrackAlbumTx 在事务内创建新的 TrackAlbum 记录。
func CreateTrackAlbumTx(tx *gorm.DB, ta *TrackAlbum) error {
	return tx.Create(ta).Error
}

// DeleteTrackAlbumLink 删除指定曲目与专辑之间的关联，供人工修复入口复用。
func DeleteTrackAlbumLink(ctx context.Context, trackID, albumID int64) error {
	return GetDB().WithContext(ctx).
		Where("track_id = ? AND album_id = ?", trackID, albumID).
		Delete(&TrackAlbum{}).Error
}
