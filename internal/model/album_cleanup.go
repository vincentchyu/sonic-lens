package model

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"gorm.io/gorm"
)

// CleanupDuplicateAlbumsParams 描述重复专辑清洗任务的筛选与执行模式。
type CleanupDuplicateAlbumsParams struct {
	Ctx             context.Context
	Artist          string
	Name            string
	Limit           int
	DryRun          bool
	ContinueOnError bool
}

// DuplicateAlbumCleanupGroup 描述单个重复专辑组的清洗结果。
type DuplicateAlbumCleanupGroup struct {
	Artist           string
	Name             string
	CanonicalAlbumID int64
	MergedAlbumIDs   []int64
}

// DuplicateAlbumCleanupReport 汇总重复专辑清洗结果。
type DuplicateAlbumCleanupReport struct {
	Groups  []DuplicateAlbumCleanupGroup
	Skipped []DuplicateAlbumSkippedGroup
}

// DuplicateAlbumSkippedGroup 描述被保守跳过的重复专辑组。
type DuplicateAlbumSkippedGroup struct {
	Artist string
	Name   string
	Reason string
}

type duplicateAlbumGroupRow struct {
	Artist string
	Name   string
}

type duplicateAlbumCandidate struct {
	Album
	TrackCount       int64
	ReleaseLinkCount int64
}

// CleanupDuplicateAlbums 统一清洗同名同作者的重复专辑。
func CleanupDuplicateAlbums(params CleanupDuplicateAlbumsParams) (*DuplicateAlbumCleanupReport, error) {
	report := &DuplicateAlbumCleanupReport{}

	err := InTx(
		params.Ctx, func(tx *gorm.DB) error {
			groups, err := listDuplicateAlbumGroupsTx(tx, params)
			if err != nil {
				return err
			}

			for _, group := range groups {
				result, err := cleanupDuplicateAlbumGroupTx(tx, group.Artist, group.Name, params.DryRun)
				if err != nil {
					if params.ContinueOnError {
						report.Skipped = append(
							report.Skipped, DuplicateAlbumSkippedGroup{
								Artist: group.Artist,
								Name:   group.Name,
								Reason: err.Error(),
							},
						)
						continue
					}
					return err
				}
				if result == nil {
					continue
				}
				report.Groups = append(report.Groups, *result)
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return report, nil
}

func listDuplicateAlbumGroupsTx(tx *gorm.DB, params CleanupDuplicateAlbumsParams) ([]duplicateAlbumGroupRow, error) {
	query := tx.Model(&Album{}).
		Select("artist, name").
		Group("artist, name").
		Having("COUNT(*) > 1").
		Order("MIN(id) ASC")

	if params.Artist != "" {
		query = query.Where("artist = ?", params.Artist)
	}
	if params.Name != "" {
		query = query.Where("name = ?", params.Name)
	}
	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}

	var groups []duplicateAlbumGroupRow
	if err := query.Scan(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func cleanupDuplicateAlbumGroupTx(
	tx *gorm.DB, artist, name string, dryRun bool,
) (*DuplicateAlbumCleanupGroup, error) {
	candidates, err := loadDuplicateAlbumCandidatesTx(tx, artist, name)
	if err != nil {
		return nil, err
	}
	if len(candidates) <= 1 {
		return nil, nil
	}

	sort.SliceStable(
		candidates, func(i, j int) bool {
			return compareDuplicateAlbumCandidate(candidates[i], candidates[j]) < 0
		},
	)

	canonical := candidates[0]
	mergedIDs := make([]int64, 0, len(candidates)-1)
	for _, candidate := range candidates[1:] {
		mergedIDs = append(mergedIDs, candidate.ID)
	}

	if dryRun {
		return &DuplicateAlbumCleanupGroup{
			Artist:           artist,
			Name:             name,
			CanonicalAlbumID: canonical.ID,
			MergedAlbumIDs:   mergedIDs,
		}, nil
	}

	for _, source := range candidates[1:] {
		if err := mergeAlbumFieldsAndRelationsTx(tx, &canonical, &source); err != nil {
			return nil, err
		}
	}

	return &DuplicateAlbumCleanupGroup{
		Artist:           artist,
		Name:             name,
		CanonicalAlbumID: canonical.ID,
		MergedAlbumIDs:   mergedIDs,
	}, nil
}

func loadDuplicateAlbumCandidatesTx(tx *gorm.DB, artist, name string) ([]duplicateAlbumCandidate, error) {
	var albums []Album
	if err := tx.Where("artist = ? AND name = ?", artist, name).Order("id ASC").Find(&albums).Error; err != nil {
		return nil, err
	}

	candidates := make([]duplicateAlbumCandidate, 0, len(albums))
	for _, album := range albums {
		candidate := duplicateAlbumCandidate{Album: album}

		if err := tx.Model(&TrackAlbum{}).
			Distinct("track_id").
			Where("album_id = ?", album.ID).
			Count(&candidate.TrackCount).Error; err != nil {
			return nil, err
		}
		if err := tx.Model(&AlbumReleaseMB{}).
			Distinct("release_mb_id").
			Where("album_id = ?", album.ID).
			Count(&candidate.ReleaseLinkCount).Error; err != nil {
			return nil, err
		}

		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

func compareDuplicateAlbumCandidate(left, right duplicateAlbumCandidate) int {
	if left.SyncStatus != right.SyncStatus {
		return compareIntDesc(left.SyncStatus, right.SyncStatus)
	}
	if left.ReleaseLinkCount != right.ReleaseLinkCount {
		return compareInt64Desc(left.ReleaseLinkCount, right.ReleaseLinkCount)
	}
	if left.TrackCount != right.TrackCount {
		return compareInt64Desc(left.TrackCount, right.TrackCount)
	}
	leftHasDate := left.ReleaseDate != ""
	rightHasDate := right.ReleaseDate != ""
	if leftHasDate != rightHasDate {
		if leftHasDate {
			return -1
		}
		return 1
	}
	if left.ID < right.ID {
		return -1
	}
	if left.ID > right.ID {
		return 1
	}
	return 0
}

func compareIntDesc(left, right int) int {
	switch {
	case left > right:
		return -1
	case left < right:
		return 1
	default:
		return 0
	}
}

func compareInt64Desc(left, right int64) int {
	switch {
	case left > right:
		return -1
	case left < right:
		return 1
	default:
		return 0
	}
}

func mergeAlbumFieldsAndRelationsTx(
	tx *gorm.DB, canonical, source *duplicateAlbumCandidate,
) error {
	if canonical == nil || source == nil {
		return nil
	}
	if canonical.ID == source.ID {
		return nil
	}

	if err := moveTrackAlbumsToCanonicalTx(tx, canonical.ID, source.ID); err != nil {
		return err
	}
	if err := moveAlbumReleaseLinksToCanonicalTx(tx, canonical.ID, source.ID); err != nil {
		return err
	}
	if err := tx.Delete(&Album{}, source.ID).Error; err != nil {
		return err
	}

	fields := buildAlbumMergeUpdates(&canonical.Album, &source.Album)
	if len(fields) > 0 {
		if err := tx.Model(&Album{}).Where("id = ?", canonical.ID).Updates(fields).Error; err != nil {
			return err
		}
	}
	applyAlbumMergeUpdates(&canonical.Album, fields)

	canonical.TrackCount += source.TrackCount
	canonical.ReleaseLinkCount += source.ReleaseLinkCount
	return nil
}

func buildAlbumMergeUpdates(canonical, source *Album) map[string]interface{} {
	if canonical == nil || source == nil {
		return nil
	}

	fields := make(map[string]interface{})
	if canonical.ReleaseDate == "" && source.ReleaseDate != "" {
		fields["release_date"] = source.ReleaseDate
	}
	if canonical.Genre == "" && source.Genre != "" {
		fields["genre"] = source.Genre
	}
	if canonical.Country == "" && source.Country != "" {
		fields["country"] = source.Country
	}
	if canonical.Status == "" && source.Status != "" {
		fields["status"] = source.Status
	}
	if canonical.Packaging == "" && source.Packaging != "" {
		fields["packaging"] = source.Packaging
	}
	if canonical.Barcode == "" && source.Barcode != "" {
		fields["barcode"] = source.Barcode
	}
	if canonical.TotalDiscs == 0 && source.TotalDiscs > 0 {
		fields["total_discs"] = source.TotalDiscs
	}
	if canonical.DiscInfos == "" && source.DiscInfos != "" {
		fields["disc_infos"] = source.DiscInfos
	}

	mergedSyncStatus := maxInt(canonical.SyncStatus, source.SyncStatus)
	if mergedSyncStatus != canonical.SyncStatus {
		fields["sync_status"] = mergedSyncStatus
	}

	return fields
}

func applyAlbumMergeUpdates(album *Album, fields map[string]interface{}) {
	if album == nil {
		return
	}

	if value, ok := fields["release_date"].(string); ok {
		album.ReleaseDate = value
	}
	if value, ok := fields["genre"].(string); ok {
		album.Genre = value
	}
	if value, ok := fields["country"].(string); ok {
		album.Country = value
	}
	if value, ok := fields["status"].(string); ok {
		album.Status = value
	}
	if value, ok := fields["packaging"].(string); ok {
		album.Packaging = value
	}
	if value, ok := fields["barcode"].(string); ok {
		album.Barcode = value
	}
	if value, ok := fields["total_discs"].(int); ok {
		album.TotalDiscs = value
	}
	if value, ok := fields["disc_infos"].(string); ok {
		album.DiscInfos = value
	}
	if value, ok := fields["sync_status"].(int); ok {
		album.SyncStatus = value
	}
}

// todo delete
func moveTrackAlbumsToCanonicalTx(tx *gorm.DB, canonicalAlbumID, sourceAlbumID int64) error {
	var rows []TrackAlbum
	if err := tx.Where("album_id = ?", sourceAlbumID).Order("id ASC").Find(&rows).Error; err != nil {
		return err
	}

	for _, row := range rows {
		moved := row
		moved.ID = 0
		moved.AlbumID = canonicalAlbumID
		if err := upsertTrackAlbumTx(tx, &moved, false); err != nil {
			return fmt.Errorf(
				"merge track_album from album %d to %d failed: %w", sourceAlbumID, canonicalAlbumID, err,
			)
		}
		if err := tx.Delete(&TrackAlbum{}, row.ID).Error; err != nil {
			return err
		}
	}
	return nil
}

func moveAlbumReleaseLinksToCanonicalTx(tx *gorm.DB, canonicalAlbumID, sourceAlbumID int64) error {
	var rows []AlbumReleaseMB
	if err := tx.Where("album_id = ?", sourceAlbumID).Order("id ASC").Find(&rows).Error; err != nil {
		return err
	}

	for _, row := range rows {
		var existing AlbumReleaseMB
		err := tx.Where("album_id = ? AND release_mb_id = ?", canonicalAlbumID, row.ReleaseMBID).
			First(&existing).Error
		if err == nil {
			continue
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		link := row
		link.ID = 0
		link.AlbumID = canonicalAlbumID
		if err := tx.Create(&link).Error; err != nil {
			return err
		}
	}

	return tx.Where("album_id = ?", sourceAlbumID).Delete(&AlbumReleaseMB{}).Error
}

func maxInt(left, right int) int {
	if left >= right {
		return left
	}
	return right
}
