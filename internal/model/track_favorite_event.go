package model

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/vincentchyu/sonic-lens/common"
	"gorm.io/gorm"
)

// TrackFavoriteEvent 对应收藏事件表，用于“先记事件，再归因回填”。
type TrackFavoriteEvent struct {
	ID                   int64                          `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	Source               string                         `gorm:"column:source;type:varchar(64);not null;index:idx_tfe_source" json:"source"`
	ProviderFavorite     bool                           `gorm:"column:provider_favorite;type:tinyint(1);not null;default:0" json:"provider_favorite"`
	Artist               string                         `gorm:"column:artist;type:varchar(255);not null;index:idx_tfe_identity_subtitle" json:"artist"`
	Album                string                         `gorm:"column:album;type:varchar(255);not null;index:idx_tfe_identity_subtitle" json:"album"`
	AlbumSubtitle        string                         `gorm:"column:album_subtitle;type:varchar(255);index:idx_tfe_identity_subtitle" json:"album_subtitle"`
	Track                string                         `gorm:"column:track;type:varchar(255);not null;index:idx_tfe_identity_subtitle" json:"track"`
	AlbumArtist          string                         `gorm:"column:album_artist;type:varchar(255)" json:"album_artist"`
	TrackNumber          int8                           `gorm:"column:track_number;type:tinyint;index:idx_tfe_identity_subtitle" json:"track_number"`
	DiscNumber           int8                           `gorm:"column:disc_number;type:tinyint;default:1;index:idx_tfe_identity_subtitle" json:"disc_number"`
	MusicBrainzID        string                         `gorm:"column:music_brainz_id;type:varchar(255)" json:"music_brainz_id"`
	Duration             int64                          `gorm:"column:duration;type:int" json:"duration"`
	BundleID             string                         `gorm:"column:bundle_id;type:varchar(255)" json:"bundle_id"`
	UniqueID             string                         `gorm:"column:unique_id;type:varchar(255)" json:"unique_id"`
	ResolvedTrackID      int64                          `gorm:"column:resolved_track_id;type:bigint;default:0;index:idx_tfe_resolved_track_id" json:"resolved_track_id"`
	ResolutionStatus     string                         `gorm:"column:resolution_status;type:varchar(32);not null;default:'pending';index:idx_tfe_resolution_status" json:"resolution_status"`
	ResolutionConfidence common.TrackMetadataConfidence `gorm:"column:resolution_confidence;type:tinyint;default:0" json:"resolution_confidence"`
	Applied              bool                           `gorm:"column:applied;type:tinyint(1);not null;default:0;index:idx_tfe_applied" json:"applied"`
	CreatedAt            time.Time                      `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt            time.Time                      `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

// TrackFavoritePendingSnapshot 描述同一首歌当前尚未归因完成的收藏意图。
type TrackFavoritePendingSnapshot struct {
	AppleMusicKnown    bool `json:"-"`
	AppleMusicFavorite bool `json:"-"`
	LastFmKnown        bool `json:"-"`
	LastFmFavorite     bool `json:"-"`
}

const (
	TrackFavoriteEventSourceAppleMusic = "Apple Music"
	TrackFavoriteEventSourceLastFm     = "Last.fm"

	TrackFavoriteEventResolutionPending    = "pending"
	TrackFavoriteEventResolutionResolved   = "resolved"
	TrackFavoriteEventResolutionUnresolved = "unresolved"
	TrackFavoriteEventResolutionAmbiguous  = "ambiguous"
)

// TableName sets the table name for the TrackFavoriteEvent model.
func (TrackFavoriteEvent) TableName() string {
	return "track_favorite_event"
}

// CreateTrackFavoriteEvent 创建收藏事件，作为后续归因和回填的上下文。
func CreateTrackFavoriteEvent(ctx context.Context, event *TrackFavoriteEvent) error {
	if event == nil {
		return errors.New("track favorite event is nil")
	}
	event.Artist = normalizeTrackStorageText(event.Artist)
	event.Album = normalizeTrackStorageText(event.Album)
	event.AlbumSubtitle = normalizeTrackStorageText(event.AlbumSubtitle)
	event.Track = normalizeTrackStorageText(event.Track)
	event.AlbumArtist = normalizeTrackStorageText(event.AlbumArtist)
	if event.ResolutionStatus == "" {
		event.ResolutionStatus = TrackFavoriteEventResolutionPending
	}
	return GetDB().WithContext(ctx).Create(event).Error
}

// GetPendingTrackFavoriteSnapshot 按曲目身份读取尚未归因完成的最新收藏意图。
func GetPendingTrackFavoriteSnapshot(ctx context.Context, identity TrackIdentity) (*TrackFavoritePendingSnapshot, error) {
	return getPendingTrackFavoriteSnapshotTx(GetDB().WithContext(ctx), identity)
}

func getPendingTrackFavoriteSnapshotTx(
	tx *gorm.DB, identity TrackIdentity,
) (*TrackFavoritePendingSnapshot, error) {
	if tx == nil {
		return nil, gorm.ErrInvalidDB
	}

	identity = normalizeTrackIdentity(identity)
	var events []*TrackFavoriteEvent
	query := tx.Model(&TrackFavoriteEvent{}).
		Where("applied = ?", false).
		Where("artist = ? AND album = ? AND track = ?", identity.Artist, identity.Album, identity.Track).
		Where("COALESCE(album_subtitle, '') = ?", identity.AlbumSubtitle).
		Where(
			"resolution_status IN ?", []string{
				TrackFavoriteEventResolutionPending,
				TrackFavoriteEventResolutionUnresolved,
				TrackFavoriteEventResolutionAmbiguous,
			},
		)

	if identity.TrackNumber > 0 {
		query = query.Where(
			"(track_number = 0 OR (track_number = ? AND (disc_number = 0 OR disc_number = ?)))",
			identity.TrackNumber,
			identity.DiscNumber,
		)
	}

	if err := query.Order("id DESC").Find(&events).Error; err != nil {
		return nil, err
	}

	snapshot := &TrackFavoritePendingSnapshot{}
	for _, event := range events {
		if event == nil {
			continue
		}
		switch event.Source {
		case TrackFavoriteEventSourceAppleMusic, "":
			if !snapshot.AppleMusicKnown {
				snapshot.AppleMusicKnown = true
				snapshot.AppleMusicFavorite = event.ProviderFavorite
			}
		case TrackFavoriteEventSourceLastFm:
			if !snapshot.LastFmKnown {
				snapshot.LastFmKnown = true
				snapshot.LastFmFavorite = event.ProviderFavorite
			}
		}
		if snapshot.AppleMusicKnown && snapshot.LastFmKnown {
			break
		}
	}

	return snapshot, nil
}

func getOpenTrackFavoriteEventTx(tx *gorm.DB, candidate *TrackFavoriteEvent) (*TrackFavoriteEvent, error) {
	if tx == nil || candidate == nil {
		return nil, gorm.ErrRecordNotFound
	}

	trackNumber, discNumber := normalizeTrackAlbumPosition(candidate.TrackNumber, candidate.DiscNumber)
	var event TrackFavoriteEvent
	err := tx.Where("source = ? AND provider_favorite = ?", candidate.Source, candidate.ProviderFavorite).
		Where("artist = ? AND album = ? AND track = ?", candidate.Artist, candidate.Album, candidate.Track).
		Where("COALESCE(album_subtitle, '') = ?", normalizeTrackStorageText(candidate.AlbumSubtitle)).
		Where("track_number = ? AND disc_number = ?", trackNumber, discNumber).
		Where("applied = ?", false).
		Where("resolution_status IN ?", []string{
			TrackFavoriteEventResolutionPending,
			TrackFavoriteEventResolutionUnresolved,
		}).
		Order("id DESC").
		First(&event).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func getOrCreateOpenTrackFavoriteEventTx(
	tx *gorm.DB, candidate *TrackFavoriteEvent,
) (*TrackFavoriteEvent, bool, error) {
	if candidate == nil {
		return nil, false, errors.New("track favorite event is nil")
	}
	candidate.TrackNumber, candidate.DiscNumber = normalizeTrackAlbumPosition(candidate.TrackNumber, candidate.DiscNumber)

	existing, err := getOpenTrackFavoriteEventTx(tx, candidate)
	if err == nil {
		return existing, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	if candidate.ResolutionStatus == "" {
		candidate.ResolutionStatus = TrackFavoriteEventResolutionPending
	}
	if err := tx.Create(candidate).Error; err != nil {
		return nil, false, err
	}
	return candidate, true, nil
}

func updateTrackFavoriteEventResolutionTx(
	tx *gorm.DB, eventID, resolvedTrackID int64, status string, confidence common.TrackMetadataConfidence, applied bool,
) error {
	fields := map[string]interface{}{
		"resolution_status":     status,
		"resolution_confidence": confidence,
		"applied":               applied,
	}
	if resolvedTrackID > 0 {
		fields["resolved_track_id"] = resolvedTrackID
	}
	return tx.Model(&TrackFavoriteEvent{}).Where("id = ?", eventID).Updates(fields).Error
}

// ResolvePendingTrackFavoriteEventsByTrackTx 将待归因收藏事件回填到已解析曲目上。
func ResolvePendingTrackFavoriteEventsByTrackTx(
	tx *gorm.DB, resolvedTrack *Track, confidence common.TrackMetadataConfidence,
) error {
	if tx == nil || resolvedTrack == nil || resolvedTrack.ID <= 0 {
		return nil
	}

	query := tx.Where("applied = ?", false).
		Where("artist = ? AND album = ? AND track = ?", resolvedTrack.Artist, resolvedTrack.Album, resolvedTrack.Track).
		Where("COALESCE(album_subtitle, '') = ?", normalizeTrackStorageText(resolvedTrack.AlbumSubtitle)).
		Where("resolution_status IN ?", []string{
			TrackFavoriteEventResolutionPending,
			TrackFavoriteEventResolutionUnresolved,
		}).
		Where(
			"(track_number = 0 OR (track_number = ? AND (disc_number = 0 OR disc_number = ?)))",
			resolvedTrack.TrackNumber, resolvedTrack.DiscNumber,
		)

	var events []TrackFavoriteEvent
	if err := query.Order("id ASC").Find(&events).Error; err != nil {
		return err
	}

	for i := range events {
		event := events[i]
		if err := applyTrackFavoriteBySourceTx(tx, resolvedTrack.ID, event.Source, event.ProviderFavorite); err != nil {
			return err
		}
		if err := updateTrackFavoriteEventResolutionTx(
			tx,
			event.ID,
			resolvedTrack.ID,
			TrackFavoriteEventResolutionResolved,
			confidence,
			true,
		); err != nil {
			return err
		}
	}

	return nil
}

// ApplyTrackFavoriteEventsByIDsResult 返回按指定事件列表回填收藏的结果。
type ApplyTrackFavoriteEventsByIDsResult struct {
	AppliedCount int `json:"applied_count"`
}

// ApplyTrackFavoriteEventsByIDs 按指定事件列表尝试回填收藏状态。
func ApplyTrackFavoriteEventsByIDs(ctx context.Context, eventIDs []int64) (*ApplyTrackFavoriteEventsByIDsResult, error) {
	return applyTrackFavoriteEventsByIDsTx(GetDB().WithContext(ctx), eventIDs)
}

func applyTrackFavoriteEventsByIDsTx(
	tx *gorm.DB, eventIDs []int64,
) (*ApplyTrackFavoriteEventsByIDsResult, error) {
	events, err := GetTrackFavoriteEventsByIDsTx(tx, eventIDs)
	if err != nil {
		return nil, err
	}

	result := &ApplyTrackFavoriteEventsByIDsResult{}
	for _, event := range events {
		if event == nil || event.Applied {
			continue
		}
		trackObj, resolveErr := findTrackByIdentityWithOptions(
			tx,
			TrackIdentity{
				Artist:        event.Artist,
				Album:         event.Album,
				AlbumSubtitle: event.AlbumSubtitle,
				Track:         event.Track,
				TrackNumber:   event.TrackNumber,
				DiscNumber:    event.DiscNumber,
			},
			trackIdentityResolveOptions{allowLooseNameFallback: strings.TrimSpace(event.MusicBrainzID) != ""},
		)
		if resolveErr != nil {
			if errors.Is(resolveErr, gorm.ErrRecordNotFound) {
				if err := updateTrackFavoriteEventResolutionTx(
					tx,
					event.ID,
					0,
					TrackFavoriteEventResolutionUnresolved,
					event.ResolutionConfidence,
					false,
				); err != nil {
					return nil, err
				}
				continue
			}
			return nil, resolveErr
		}
		if err := applyTrackFavoriteBySourceTx(tx, trackObj.ID, event.Source, event.ProviderFavorite); err != nil {
			return nil, err
		}
		if err := updateTrackFavoriteEventResolutionTx(
			tx,
			event.ID,
			trackObj.ID,
			TrackFavoriteEventResolutionResolved,
			common.TrackMetadataConfidenceHigh,
			true,
		); err != nil {
			return nil, err
		}
		result.AppliedCount++
	}

	return result, nil
}

// ApplyTrackFavoriteEventToResolvedTrackTx 将指定收藏事件显式绑定到目标曲目并应用收藏状态。
func ApplyTrackFavoriteEventToResolvedTrackTx(
	tx *gorm.DB, eventID, trackID int64, confidence common.TrackMetadataConfidence,
) (bool, error) {
	if tx == nil || eventID <= 0 || trackID <= 0 {
		return false, nil
	}

	var event TrackFavoriteEvent
	if err := tx.First(&event, eventID).Error; err != nil {
		return false, err
	}
	if !event.Applied {
		if err := applyTrackFavoriteBySourceTx(tx, trackID, event.Source, event.ProviderFavorite); err != nil {
			return false, err
		}
	}
	if err := updateTrackFavoriteEventResolutionTx(
		tx,
		event.ID,
		trackID,
		TrackFavoriteEventResolutionResolved,
		confidence,
		true,
	); err != nil {
		return false, err
	}
	return !event.Applied, nil
}
