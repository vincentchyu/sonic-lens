package model

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/common"
)

const (
	PendingAlbumWorkItemStatusOpen           = "open"
	PendingAlbumWorkItemStatusMBSelected     = "mb_selected"
	PendingAlbumWorkItemStatusDeepMaintaning = "deep_maintaining"
	PendingAlbumWorkItemStatusApplying       = "applying"
	PendingAlbumWorkItemStatusCompleted      = "completed"
	PendingAlbumWorkItemStatusFailed         = "failed"
)

var pendingAlbumOpenStatuses = []string{
	PendingAlbumWorkItemStatusOpen,
	PendingAlbumWorkItemStatusMBSelected,
	PendingAlbumWorkItemStatusDeepMaintaning,
	PendingAlbumWorkItemStatusApplying,
}

// PendingAlbumWorkItem 记录一次人工待归因专辑维护的冻结上下文。
type PendingAlbumWorkItem struct {
	ID                    int64      `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	Artist                string     `gorm:"column:artist;type:varchar(255);not null;index:idx_pawi_identity_key" json:"artist"`
	Album                 string     `gorm:"column:album;type:varchar(255);not null;index:idx_pawi_identity_key" json:"album"`
	AlbumArtist           string     `gorm:"column:album_artist;type:varchar(255)" json:"album_artist"`
	NormalizedIdentityKey string     `gorm:"column:normalized_identity_key;type:varchar(255);not null;index:idx_pawi_identity_key" json:"normalized_identity_key"`
	PlayRecordIDsJSON     string     `gorm:"column:play_record_ids_json;type:longtext" json:"play_record_ids_json"`
	FavoriteEventIDsJSON  string     `gorm:"column:favorite_event_ids_json;type:longtext" json:"favorite_event_ids_json"`
	SelectedReleaseMBID   int64      `gorm:"column:selected_release_mb_id;type:bigint;default:0" json:"selected_release_mb_id"`
	SelectedMBID          string     `gorm:"column:selected_mbid;type:varchar(255)" json:"selected_mbid"`
	Status                string     `gorm:"column:status;type:varchar(64);not null;default:'open';index:idx_pawi_status" json:"status"`
	ResolvedAlbumID       int64      `gorm:"column:resolved_album_id;type:bigint;default:0" json:"resolved_album_id"`
	LastError             string     `gorm:"column:last_error;type:text" json:"last_error"`
	CompletedAt           *time.Time `gorm:"column:completed_at;type:timestamp" json:"completed_at"`
	CreatedAt             time.Time  `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt             time.Time  `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

func (PendingAlbumWorkItem) TableName() string {
	return "pending_album_work_item"
}

// PendingAlbumGroup 描述待归因专辑列表中的聚合结果。
type PendingAlbumGroup struct {
	IdentityKey        string    `json:"identity_key"`
	Artist             string    `json:"artist"`
	Album              string    `json:"album"`
	AlbumArtist        string    `json:"album_artist"`
	PlayRecordCount    int       `json:"play_record_count"`
	FavoriteEventCount int       `json:"favorite_event_count"`
	Sources            []string  `json:"sources"`
	TrackNames         []string  `json:"track_names"`
	PlayRecordIDs      []int64   `json:"play_record_ids"`
	FavoriteEventIDs   []int64   `json:"favorite_event_ids"`
	EarliestPlayTime   time.Time `json:"earliest_play_time"`
	LatestPlayTime     time.Time `json:"latest_play_time"`
	OpenWorkItemID     int64     `json:"open_work_item_id"`
	OpenWorkItemStatus string    `json:"open_work_item_status"`
}

// PendingAlbumContextTrack 描述同一待归因专辑下的上下文曲目摘要。
type PendingAlbumContextTrack struct {
	Track            string   `json:"track"`
	PlayRecordCount  int      `json:"play_record_count"`
	FavoriteCount    int      `json:"favorite_count"`
	Sources          []string `json:"sources"`
	PlayRecordIDs    []int64  `json:"play_record_ids"`
	FavoriteEventIDs []int64  `json:"favorite_event_ids"`
}

// PendingAlbumWorkItemDetail 返回工作项详情和冻结的上下文数据。
type PendingAlbumWorkItemDetail struct {
	WorkItem       *PendingAlbumWorkItem       `json:"work_item"`
	PlayRecords    []*TrackPlayRecord          `json:"play_records"`
	FavoriteEvents []*TrackFavoriteEvent       `json:"favorite_events"`
	ContextTracks  []*PendingAlbumContextTrack `json:"context_tracks"`
}

func normalizePendingAlbumIdentity(artist, albumArtist, album string) string {
	owner := strings.TrimSpace(albumArtist)
	if owner == "" {
		owner = strings.TrimSpace(artist)
	}
	owner = strings.ToLower(common.ConversionSimplifiedFx(common.UnityFixAll(owner)))
	album = strings.ToLower(common.ConversionSimplifiedFx(common.UnityFixAll(strings.TrimSpace(album))))
	return owner + "||" + album
}

func encodeInt64Slice(values []int64) string {
	if len(values) == 0 {
		return "[]"
	}
	raw, _ := json.Marshal(values)
	return string(raw)
}

func decodeInt64Slice(raw string) ([]int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var values []int64
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, err
	}
	return values, nil
}

func listPendingAlbumPlayRecords(ctx context.Context) ([]*TrackPlayRecord, error) {
	var records []*TrackPlayRecord
	err := GetDB().WithContext(ctx).
		Where("library_applied = ?", false).
		Order("play_time DESC, id DESC").
		Find(&records).Error
	return records, err
}

func listPendingAlbumFavoriteEvents(ctx context.Context) ([]*TrackFavoriteEvent, error) {
	var events []*TrackFavoriteEvent
	err := GetDB().WithContext(ctx).
		Where("applied = ?", false).
		Where(
			"resolution_status IN ?", []string{
				TrackFavoriteEventResolutionPending,
				TrackFavoriteEventResolutionUnresolved,
				TrackFavoriteEventResolutionAmbiguous,
			},
	).
		Order("created_at DESC, id DESC").
		Find(&events).Error
	return events, err
}

func getOpenPendingAlbumWorkItemsByKeys(ctx context.Context, keys []string) (map[string]*PendingAlbumWorkItem, error) {
	if len(keys) == 0 {
		return map[string]*PendingAlbumWorkItem{}, nil
	}
	var rows []*PendingAlbumWorkItem
	if err := GetDB().WithContext(ctx).
		Where("normalized_identity_key IN ?", keys).
		Where("status IN ?", pendingAlbumOpenStatuses).
		Order("id DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	results := make(map[string]*PendingAlbumWorkItem, len(rows))
	for _, row := range rows {
		if _, ok := results[row.NormalizedIdentityKey]; ok {
			continue
		}
		results[row.NormalizedIdentityKey] = row
	}
	return results, nil
}

// GetPendingAlbumGroups 获取按专辑维度聚合的待归因上下文列表。
func GetPendingAlbumGroups(ctx context.Context, limit int) ([]*PendingAlbumGroup, error) {
	playRecords, err := listPendingAlbumPlayRecords(ctx)
	if err != nil {
		return nil, err
	}
	favoriteEvents, err := listPendingAlbumFavoriteEvents(ctx)
	if err != nil {
		return nil, err
	}

	groupMap := make(map[string]*PendingAlbumGroup)
	trackNameSets := make(map[string]map[string]struct{})
	sourceSets := make(map[string]map[string]struct{})

	getGroup := func(key, artist, albumArtist, album string) *PendingAlbumGroup {
		group, ok := groupMap[key]
		if ok {
			return group
		}
		group = &PendingAlbumGroup{
			IdentityKey: key,
			Artist:      artist,
			AlbumArtist: albumArtist,
			Album:       album,
		}
		groupMap[key] = group
		trackNameSets[key] = make(map[string]struct{})
		sourceSets[key] = make(map[string]struct{})
		return group
	}

	for _, record := range playRecords {
		key := normalizePendingAlbumIdentity(record.Artist, record.AlbumArtist, record.Album)
		group := getGroup(key, record.Artist, record.AlbumArtist, record.Album)
		group.PlayRecordCount++
		group.PlayRecordIDs = append(group.PlayRecordIDs, record.ID)
		if group.LatestPlayTime.IsZero() || record.PlayTime.After(group.LatestPlayTime) {
			group.LatestPlayTime = record.PlayTime
		}
		if group.EarliestPlayTime.IsZero() || record.PlayTime.Before(group.EarliestPlayTime) {
			group.EarliestPlayTime = record.PlayTime
		}
		if record.Source != "" {
			sourceSets[key][record.Source] = struct{}{}
		}
		if title := strings.TrimSpace(record.Track); title != "" {
			trackNameSets[key][title] = struct{}{}
		}
	}

	for _, event := range favoriteEvents {
		key := normalizePendingAlbumIdentity(event.Artist, event.AlbumArtist, event.Album)
		group := getGroup(key, event.Artist, event.AlbumArtist, event.Album)
		group.FavoriteEventCount++
		group.FavoriteEventIDs = append(group.FavoriteEventIDs, event.ID)
		if event.Source != "" {
			sourceSets[key][event.Source] = struct{}{}
		}
		if title := strings.TrimSpace(event.Track); title != "" {
			trackNameSets[key][title] = struct{}{}
		}
	}

	keys := make([]string, 0, len(groupMap))
	for key := range groupMap {
		keys = append(keys, key)
	}
	openItems, err := getOpenPendingAlbumWorkItemsByKeys(ctx, keys)
	if err != nil {
		return nil, err
	}

	results := make([]*PendingAlbumGroup, 0, len(groupMap))
	for _, key := range keys {
		group := groupMap[key]
		if len(group.PlayRecordIDs) == 0 && len(group.FavoriteEventIDs) == 0 {
			continue
		}
		for source := range sourceSets[key] {
			group.Sources = append(group.Sources, source)
		}
		slices.Sort(group.Sources)
		for title := range trackNameSets[key] {
			group.TrackNames = append(group.TrackNames, title)
		}
		slices.Sort(group.TrackNames)
		if item := openItems[key]; item != nil {
			group.OpenWorkItemID = item.ID
			group.OpenWorkItemStatus = item.Status
		}
		results = append(results, group)
	}

	/*slices.SortFunc(
		results, func(a, b *PendingAlbumGroup) int {
			switch {
			case a.LatestPlayTime.After(b.LatestPlayTime):
				return -1
			case a.LatestPlayTime.Before(b.LatestPlayTime):
				return 1
			default:
				return strings.Compare(a.IdentityKey, b.IdentityKey)
			}
		},
	)*/

	slices.SortFunc(
		results, func(a, b *PendingAlbumGroup) int {
			if len(a.PlayRecordIDs) != len(b.PlayRecordIDs) {
				if len(a.PlayRecordIDs) > len(b.PlayRecordIDs) {
					return -1
				}
				return 1
			}
			switch {
			case a.LatestPlayTime.After(b.LatestPlayTime):
				return -1
			case a.LatestPlayTime.Before(b.LatestPlayTime):
				return 1
			default:
				return strings.Compare(a.IdentityKey, b.IdentityKey)
			}
		},
	)

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// CreateOrGetPendingAlbumWorkItem 根据 identity key 冻结当前上下文并返回工作项。
func CreateOrGetPendingAlbumWorkItem(ctx context.Context, identityKey string) (*PendingAlbumWorkItem, error) {
	groups, err := GetPendingAlbumGroups(ctx, 0)
	if err != nil {
		return nil, err
	}

	var selected *PendingAlbumGroup
	for _, group := range groups {
		if group.IdentityKey == identityKey {
			selected = group
			break
		}
	}
	if selected == nil {
		return nil, gorm.ErrRecordNotFound
	}
	if selected.OpenWorkItemID > 0 {
		return GetPendingAlbumWorkItemByID(ctx, selected.OpenWorkItemID)
	}

	item := &PendingAlbumWorkItem{
		Artist:                selected.Artist,
		Album:                 selected.Album,
		AlbumArtist:           selected.AlbumArtist,
		NormalizedIdentityKey: selected.IdentityKey,
		PlayRecordIDsJSON:     encodeInt64Slice(selected.PlayRecordIDs),
		FavoriteEventIDsJSON:  encodeInt64Slice(selected.FavoriteEventIDs),
		Status:                PendingAlbumWorkItemStatusOpen,
	}
	if err := GetDB().WithContext(ctx).Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

// GetPendingAlbumWorkItemByID 获取单个工作项。
func GetPendingAlbumWorkItemByID(ctx context.Context, workItemID int64) (*PendingAlbumWorkItem, error) {
	var item PendingAlbumWorkItem
	err := GetDB().WithContext(ctx).First(&item, workItemID).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdatePendingAlbumWorkItemSelection 更新工作项绑定的 MB 候选。
func UpdatePendingAlbumWorkItemSelection(ctx context.Context, workItemID, releaseMBID int64, mbid string) error {
	status := PendingAlbumWorkItemStatusOpen
	if strings.TrimSpace(mbid) != "" {
		status = PendingAlbumWorkItemStatusMBSelected
	}
	return GetDB().WithContext(ctx).
		Model(&PendingAlbumWorkItem{}).
		Where("id = ?", workItemID).
		Updates(
			map[string]interface{}{
				"selected_release_mb_id": releaseMBID,
				"selected_mbid":          mbid,
				"status":                 status,
				"last_error":             "",
			},
	).Error
}

// UpdatePendingAlbumWorkItemProgress 更新工作项过程状态。
func UpdatePendingAlbumWorkItemProgress(
	ctx context.Context, workItemID int64, status string, resolvedAlbumID int64, lastError string,
) error {
	fields := map[string]interface{}{
		"status":     status,
		"last_error": lastError,
	}
	if resolvedAlbumID > 0 {
		fields["resolved_album_id"] = resolvedAlbumID
	}
	if status == PendingAlbumWorkItemStatusCompleted {
		now := time.Now()
		fields["completed_at"] = &now
	}
	return GetDB().WithContext(ctx).Model(&PendingAlbumWorkItem{}).Where("id = ?", workItemID).Updates(fields).Error
}

// GetTrackPlayRecordsByIDs 获取一组播放流水。
func GetTrackPlayRecordsByIDs(ctx context.Context, ids []int64) ([]*TrackPlayRecord, error) {
	return GetTrackPlayRecordsByIDsTx(GetDB().WithContext(ctx), ids)
}

// GetTrackPlayRecordsByIDsTx 在事务内获取一组播放流水。
func GetTrackPlayRecordsByIDsTx(tx *gorm.DB, ids []int64) ([]*TrackPlayRecord, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var records []*TrackPlayRecord
	err := tx.Where("id IN ?", ids).Order("play_time ASC, id ASC").Find(&records).Error
	return records, err
}

// GetTrackFavoriteEventsByIDs 获取一组收藏事件。
func GetTrackFavoriteEventsByIDs(ctx context.Context, ids []int64) ([]*TrackFavoriteEvent, error) {
	return GetTrackFavoriteEventsByIDsTx(GetDB().WithContext(ctx), ids)
}

// GetTrackFavoriteEventsByIDsTx 在事务内获取一组收藏事件。
func GetTrackFavoriteEventsByIDsTx(tx *gorm.DB, ids []int64) ([]*TrackFavoriteEvent, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var events []*TrackFavoriteEvent
	err := tx.Where("id IN ?", ids).Order("created_at ASC, id ASC").Find(&events).Error
	return events, err
}

// GetPendingAlbumWorkItemDetail 返回工作项详情和冻结上下文。
func GetPendingAlbumWorkItemDetail(ctx context.Context, workItemID int64) (*PendingAlbumWorkItemDetail, error) {
	item, err := GetPendingAlbumWorkItemByID(ctx, workItemID)
	if err != nil {
		return nil, err
	}
	playRecordIDs, err := decodeInt64Slice(item.PlayRecordIDsJSON)
	if err != nil {
		return nil, err
	}
	favoriteEventIDs, err := decodeInt64Slice(item.FavoriteEventIDsJSON)
	if err != nil {
		return nil, err
	}

	playRecords, err := GetTrackPlayRecordsByIDs(ctx, playRecordIDs)
	if err != nil {
		return nil, err
	}
	favoriteEvents, err := GetTrackFavoriteEventsByIDs(ctx, favoriteEventIDs)
	if err != nil {
		return nil, err
	}

	trackMap := make(map[string]*PendingAlbumContextTrack)
	for _, record := range playRecords {
		key := strings.TrimSpace(record.Track)
		entry, ok := trackMap[key]
		if !ok {
			entry = &PendingAlbumContextTrack{Track: record.Track}
			trackMap[key] = entry
		}
		entry.PlayRecordCount++
		entry.PlayRecordIDs = append(entry.PlayRecordIDs, record.ID)
		if record.Source != "" && !slices.Contains(entry.Sources, record.Source) {
			entry.Sources = append(entry.Sources, record.Source)
		}
	}
	for _, event := range favoriteEvents {
		key := strings.TrimSpace(event.Track)
		entry, ok := trackMap[key]
		if !ok {
			entry = &PendingAlbumContextTrack{Track: event.Track}
			trackMap[key] = entry
		}
		entry.FavoriteCount++
		entry.FavoriteEventIDs = append(entry.FavoriteEventIDs, event.ID)
		if event.Source != "" && !slices.Contains(entry.Sources, event.Source) {
			entry.Sources = append(entry.Sources, event.Source)
		}
	}

	contextTracks := make([]*PendingAlbumContextTrack, 0, len(trackMap))
	for _, entry := range trackMap {
		slices.Sort(entry.Sources)
		contextTracks = append(contextTracks, entry)
	}
	slices.SortFunc(
		contextTracks, func(a, b *PendingAlbumContextTrack) int {
			if a.PlayRecordCount == b.PlayRecordCount {
				return strings.Compare(a.Track, b.Track)
			}
			if a.PlayRecordCount > b.PlayRecordCount {
				return -1
			}
			return 1
		},
	)

	return &PendingAlbumWorkItemDetail{
		WorkItem:       item,
		PlayRecords:    playRecords,
		FavoriteEvents: favoriteEvents,
		ContextTracks:  contextTracks,
	}, nil
}

// ResolveCanonicalAlbumForPendingContextTx 在事务内为待归因专辑选择唯一目标专辑。
func ResolveCanonicalAlbumForPendingContextTx(tx *gorm.DB, candidate *Album) (*Album, error) {
	if tx == nil || candidate == nil {
		return nil, errors.New("album candidate is nil")
	}

	var curated Album
	err := tx.Where("artist = ? AND name = ? AND sync_status = ?", candidate.Artist, candidate.Name, 3).
		Order("CASE WHEN release_date = '' OR release_date IS NULL THEN 1 ELSE 0 END ASC, id ASC").
		First(&curated).Error
	if err == nil {
		mergeAlbumFields(&curated, candidate)
		if saveErr := tx.Save(&curated).Error; saveErr != nil {
			return nil, saveErr
		}
		return &curated, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	working := *candidate
	if err := getOrCreateAlbumTx(tx, &working); err != nil {
		return nil, err
	}
	return &working, nil
}
