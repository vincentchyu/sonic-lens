package model

import (
	"context"
	"errors"
	"fmt"
	"slices"

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

	slices.SortStableFunc(candidates, compareDuplicateAlbumCandidate)

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
	if err := ReconcileAlbumPlayCountsTx(tx, canonical.ID); err != nil {
		return err
	}

	fields := buildAlbumMergeUpdates(&canonical.Album, &source.Album)
	if len(fields) > 0 {
		if err := UpdateAlbumFieldsTx(tx, canonical.ID, fields); err != nil {
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
		fields["genre"] = NormalizeGenre(nil, source.Genre)
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

	mergedSyncStatus := max(canonical.SyncStatus, source.SyncStatus)
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
		if err := upsertTrackAlbumTx(tx, &moved, false, true); err != nil {
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
		if !errors.Is(err, gorm.ErrRecordNotFound) {
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

// ============================================================
// CleanupReleaseTypeSuffixes：历史数据清洗工具
// ============================================================

// CleanupReleaseTypeSuffixesParams 描述发行类型后缀清洗任务参数。
type CleanupReleaseTypeSuffixesParams struct {
	Ctx    context.Context
	Limit  int
	DryRun bool
}

// ReleaseTypeSuffixCleanupItem 描述单条清洗结果。
type ReleaseTypeSuffixCleanupItem struct {
	AlbumID      int64
	OldName      string
	NewName      string
	ReleaseType  string
	MergedIntoID int64 // 非零表示该专辑已合并到目标专辑并被删除
}

// ReleaseTypeSuffixCleanupReport 汇总清洗结果。
type ReleaseTypeSuffixCleanupReport struct {
	Items   []ReleaseTypeSuffixCleanupItem
	Skipped []ReleaseTypeSuffixCleanupItem // 发生错误被跳过的条目
}

// CleanupReleaseTypeSuffixes 扫描专辑名中含 " - EP"/" - Single"/" - LP" 的历史记录，
// 剥离后缀并写入 release_type 列；同时检测是否已存在同名干净专辑，若存在则合并。
// 全部操作在同一事务内完成，DryRun=true 时只扫描不写入。
func CleanupReleaseTypeSuffixes(params CleanupReleaseTypeSuffixesParams) (*ReleaseTypeSuffixCleanupReport, error) {
	report := &ReleaseTypeSuffixCleanupReport{}

	// 先在事务外收集待处理专辑（避免在读时持有写锁）
	items, err := listAlbumsWithReleaseTypeSuffix(params.Ctx, params.Limit)
	if err != nil {
		return nil, fmt.Errorf("扫描含后缀专辑失败: %w", err)
	}

	for _, album := range items {
		cleanedName, rt := parseAlbumNameSuffix(album.Name)
		if rt == "" || cleanedName == album.Name {
			continue
		}

		item := ReleaseTypeSuffixCleanupItem{
			AlbumID:     album.ID,
			OldName:     album.Name,
			NewName:     cleanedName,
			ReleaseType: rt,
		}

		if params.DryRun {
			report.Items = append(report.Items, item)
			continue
		}

		mergedInto, err := applyReleaseTypeSuffixCleanup(params.Ctx, &album, cleanedName, rt)
		if err != nil {
			item.MergedIntoID = mergedInto
			report.Skipped = append(report.Skipped, item)
			continue
		}
		item.MergedIntoID = mergedInto
		report.Items = append(report.Items, item)
	}

	return report, nil
}

// listAlbumsWithReleaseTypeSuffix 扫描名称含 " - EP"/" - Single"/" - LP" 的专辑记录。
func listAlbumsWithReleaseTypeSuffix(ctx context.Context, limit int) ([]Album, error) {
	query := GetDB().WithContext(ctx).Model(&Album{}).
		Where(
			"name LIKE '% - EP' OR name LIKE '% - Single' OR name LIKE '% - LP'" +
				" OR name LIKE '% - ep' OR name LIKE '% - single' OR name LIKE '% - lp'",
		).
		Order("id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	var albums []Album
	if err := query.Find(&albums).Error; err != nil {
		return nil, err
	}
	return albums, nil
}

// parseAlbumNameSuffix 提取专辑名的 release type 后缀（包装 common 层函数，供 model 层使用）。
func parseAlbumNameSuffix(name string) (cleanName string, releaseType string) {
	// 此处通过 common 层函数，避免在 model 层重复正则逻辑
	// 直接使用已有的 normalizeAlbumReleaseTypeSuffix 逻辑：
	tmp := &Album{Name: name}
	normalizeAlbumReleaseTypeSuffix(tmp)
	return tmp.Name, tmp.ReleaseType
}

// applyReleaseTypeSuffixCleanup 将单个含后缀专辑的名称修正为干净主标题并写入 release_type。
// 若已存在同名同作者干净专辑，则把当前含后缀专辑的曲目/关联合并过去后删除。
// 返回合并目标的 albumID（未合并则为 0）。
func applyReleaseTypeSuffixCleanup(ctx context.Context, album *Album, cleanedName, rt string) (int64, error) {
	var mergedIntoID int64
	err := InTx(
		ctx, func(tx *gorm.DB) error {
			// 查找是否已存在名称干净、同作者的专辑（需精确匹配 subtitle 与 release_date）
			var existing Album
			lookupErr := tx.Where(
				"artist = ? AND name = ? AND COALESCE(name_subtitle, '') = ? AND COALESCE(release_date, '') = ?",
				album.Artist, cleanedName,
				normalizedAlbumSubtitle(album.NameSubtitle),
				album.ReleaseDate,
			).First(&existing).Error

			if lookupErr != nil && !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
				return fmt.Errorf("查找同名干净专辑失败: %w", lookupErr)
			}

			if errors.Is(lookupErr, gorm.ErrRecordNotFound) {
				// 无同名专辑 -> 直接更新当前记录的 name 和 release_type
				updates := map[string]interface{}{
					"name":         cleanedName,
					"release_type": rt,
				}
				if err := tx.Model(&Album{}).Where("id = ?", album.ID).Updates(updates).Error; err != nil {
					return fmt.Errorf("更新专辑名称失败 album_id=%d: %w", album.ID, err)
				}
				return nil
			}

			// 已存在同名干净专辑 -> 合并
			sourceCand := &duplicateAlbumCandidate{Album: *album}
			canonicalCand := &duplicateAlbumCandidate{Album: existing}
			if err := mergeAlbumFieldsAndRelationsTx(tx, canonicalCand, sourceCand); err != nil {
				return fmt.Errorf("合并专辑失败 from=%d to=%d: %w", album.ID, existing.ID, err)
			}
			// 确保目标专辑写入 release_type
			if existing.ReleaseType == "" {
				if err := tx.Model(&Album{}).Where("id = ?", existing.ID).
					Update("release_type", rt).Error; err != nil {
					return fmt.Errorf("更新目标专辑 release_type 失败: %w", err)
				}
			}
			mergedIntoID = existing.ID
			return nil
		},
	)
	return mergedIntoID, err
}
