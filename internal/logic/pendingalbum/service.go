package pendingalbum

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uploadedlobster.com/mbtypes"
	"go.uploadedlobster.com/musicbrainzws2"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/log"
	coremusicbrainz "github.com/vincentchyu/sonic-lens/core/musicbrainz"
	artworklogic "github.com/vincentchyu/sonic-lens/internal/logic/artwork"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

const (
	pendingAlbumMaintenanceModeMusicBrainz = "musicbrainz"
	pendingAlbumMaintenanceModeManual      = "manual"
)

// DeepMaintainPendingAlbumReport 返回工作项执行后的汇总结果。
type DeepMaintainPendingAlbumReport struct {
	Mode                  string `json:"mode"`
	ResolvedAlbumID       int64  `json:"resolved_album_id"`
	ReusedHeardTracks     int    `json:"reused_heard_tracks"`
	CreatedTracks         int    `json:"created_tracks"`
	TrackAlbumWrites      int    `json:"track_album_writes"`
	AppliedPlayRecords    int    `json:"applied_play_records"`
	AppliedFavoriteEvents int    `json:"applied_favorite_events"`
}

// ManualPendingAlbumInput 描述手动维护专辑时的统一输入。
type ManualPendingAlbumInput struct {
	ManualAlbum  ManualPendingAlbumAlbumInput   `json:"manual_album"`
	ManualTracks []ManualPendingAlbumTrackInput `json:"manual_tracks"`
}

// ManualPendingAlbumAlbumInput 描述手动维护时的专辑级元数据。
type ManualPendingAlbumAlbumInput struct {
	Name                string `json:"name"`
	AlbumArtist         string `json:"album_artist"`
	DisplayArtist       string `json:"display_artist"`
	ReleaseDate         string `json:"release_date"`
	OriginalReleaseDate string `json:"original_release_date"`
	Genre               string `json:"genre"`
	Country             string `json:"country"`
	Status              string `json:"status"`
	Packaging           string `json:"packaging"`
	Barcode             string `json:"barcode"`
	CoverArtURL         string `json:"cover_art_url"`
}

// ManualPendingAlbumTrackInput 描述手动维护时的曲目级元数据。
type ManualPendingAlbumTrackInput struct {
	DiscNumber     int8     `json:"disc_number"`
	TrackNumber    int8     `json:"track_number"`
	Title          string   `json:"title"`
	Artist         string   `json:"artist"`
	Duration       int64    `json:"duration"`
	Composer       string   `json:"composer"`
	MusicBrainzID  string   `json:"music_brainz_id"`
	EvidenceTitles []string `json:"evidence_titles"`
}

// Service 定义待归因专辑工作台能力。
type Service interface {
	GetPendingAlbumGroups(ctx context.Context, limit int) ([]*model.PendingAlbumGroup, error)
	CreateOrGetPendingAlbumWorkItem(ctx context.Context, identityKey string) (*model.PendingAlbumWorkItem, error)
	GetPendingAlbumWorkItemDetail(ctx context.Context, workItemID int64) (*model.PendingAlbumWorkItemDetail, error)
	RefreshPendingAlbumWorkItemContext(ctx context.Context, workItemID int64) (*model.PendingAlbumWorkItem, error)
	SearchPendingAlbumMBReleases(ctx context.Context, workItemID int64) ([]*model.ReleaseMB, error)
	LinkPendingAlbumMBRelease(ctx context.Context, workItemID, releaseMBID int64, mbid string) error
	DeepMaintainPendingAlbumWorkItem(ctx context.Context, workItemID int64) (*DeepMaintainPendingAlbumReport, error)
	ManualMaintainPendingAlbumWorkItem(
		ctx context.Context, workItemID int64, input ManualPendingAlbumInput,
	) (*DeepMaintainPendingAlbumReport, error)
	ListWorkItems(ctx context.Context, limit, offset int, keyword string, statusGroup string) ([]*model.PendingAlbumWorkItem, int64, error)
}

type serviceImpl struct{}

type pendingEvidence struct {
	Title            string
	TrackNumber      int8
	DiscNumber       int8
	ResolvedTrackID  int64
	PlayRecordIDs    []int64
	FavoriteEventIDs []int64
}

type pendingAlbumTrackDraft struct {
	Title          string
	TrackArtist    string
	AlbumArtist    string
	DiscNumber     int8
	TrackNumber    int8
	Duration       int64
	Composer       string
	MusicBrainzID  string
	EvidenceTitles []string
}

type pendingAlbumMaintenanceMaterial struct {
	Mode             string
	AlbumCandidate   *model.Album
	TrackDrafts      []pendingAlbumTrackDraft
	ReleaseMB        *model.ReleaseMB
	CoverEnsureInput artworklogic.EnsureAlbumCoverInput
}

// NewService 创建待归因专辑服务。
func NewService() Service {
	return &serviceImpl{}
}

func (s *serviceImpl) GetPendingAlbumGroups(ctx context.Context, limit int) ([]*model.PendingAlbumGroup, error) {
	return model.GetPendingAlbumGroups(ctx, limit)
}

func (s *serviceImpl) CreateOrGetPendingAlbumWorkItem(
	ctx context.Context, identityKey string,
) (*model.PendingAlbumWorkItem, error) {
	return model.CreateOrGetPendingAlbumWorkItem(ctx, identityKey)
}

func (s *serviceImpl) GetPendingAlbumWorkItemDetail(
	ctx context.Context, workItemID int64,
) (*model.PendingAlbumWorkItemDetail, error) {
	return model.GetPendingAlbumWorkItemDetail(ctx, workItemID)
}

func (s *serviceImpl) RefreshPendingAlbumWorkItemContext(
	ctx context.Context, workItemID int64,
) (*model.PendingAlbumWorkItem, error) {
	log.Info(ctx, "刷新待归因专辑冻结上下文", zap.Int64("work_item_id", workItemID))
	item, err := model.RefreshPendingAlbumWorkItemContext(ctx, workItemID)
	if err != nil {
		log.Error(ctx, "刷新待归因专辑冻结上下文失败", zap.Int64("work_item_id", workItemID), zap.Error(err))
		return nil, err
	}
	log.Info(ctx, "待归因专辑冻结上下文刷新完成", zap.Int64("work_item_id", workItemID))
	return item, nil
}

func (s *serviceImpl) ListWorkItems(
	ctx context.Context, limit, offset int, keyword string, statusGroup string,
) ([]*model.PendingAlbumWorkItem, int64, error) {
	return model.ListPendingAlbumWorkItems(ctx, limit, offset, keyword, statusGroup)
}

func escapeLucene(in string) string {
	var out string
	for _, r := range in {
		switch r {
		case '\\', '+', '-', '&', '|', '!', '(', ')', '{', '}', '[', ']', '^', '"', '~', '*', '?', ':', '/':
			out += "\\" + string(r)
		default:
			out += string(r)
		}
	}
	return out
}

func pendingAlbumOwner(item *model.PendingAlbumWorkItem) string {
	if item == nil {
		return ""
	}
	if strings.TrimSpace(item.AlbumArtist) != "" {
		return strings.TrimSpace(item.AlbumArtist)
	}
	return strings.TrimSpace(item.Artist)
}

func normalizeTrackLookupKey(title string) string {
	title = strings.TrimSpace(common.UnityFixAll(title))
	if title == "" {
		return ""
	}
	return strings.ToLower(common.ConversionSimplifiedFx(title))
}

func (s *serviceImpl) SearchPendingAlbumMBReleases(ctx context.Context, workItemID int64) ([]*model.ReleaseMB, error) {
	log.Info(ctx, "搜索待归因专辑的 MB Releases", zap.Int64("work_item_id", workItemID))
	item, err := model.GetPendingAlbumWorkItemByID(ctx, workItemID)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(
		"release:\"%s\" AND artist:\"%s\"",
		escapeLucene(item.Album),
		escapeLucene(pendingAlbumOwner(item)),
	)
	log.Info(ctx, "搜索待归因专辑的 MB Releases", zap.String("query", query))
	searchRes, err := coremusicbrainz.SearchReleases(
		ctx,
		musicbrainzws2.SearchFilter{Query: query},
		musicbrainzws2.Paginator{Limit: 10},
	)
	if err != nil {
		log.Error(ctx, "搜索待归因专辑的 MB Releases 失败", zap.String("query", query), zap.Error(err))
		return nil, err
	}

	results := make([]*model.ReleaseMB, 0, len(searchRes.Releases))
	for idx, release := range searchRes.Releases {
		raw, _ := json.Marshal(release)
		results = append(
			results,
			&model.ReleaseMB{
				ID:       int64(idx + 1),
				MBID:     string(release.ID),
				AlbumID:  0,
				Name:     release.Title,
				JSONData: string(raw),
			},
		)
	}
	log.Info(ctx, "搜索待归因专辑的 MB Releases 完成", zap.Int64("work_item_id", workItemID), zap.Int("count", len(results)))
	return results, nil
}

func (s *serviceImpl) LinkPendingAlbumMBRelease(ctx context.Context, workItemID, releaseMBID int64, mbid string) error {
	return model.UpdatePendingAlbumWorkItemSelection(ctx, workItemID, releaseMBID, strings.TrimSpace(mbid))
}

func buildPendingEvidence(detail *model.PendingAlbumWorkItemDetail) (
	map[string]pendingEvidence, map[string]pendingEvidence,
) {
	byPosition := make(map[string]pendingEvidence)
	byTitle := make(map[string]pendingEvidence)

	register := func(title string, trackNumber, discNumber int8, resolvedTrackID, playRecordID, favoriteEventID int64) {
		if trackNumber > 0 {
			if discNumber <= 0 {
				discNumber = 1
			}
			key := fmt.Sprintf("%d|%d", discNumber, trackNumber)
			current := byPosition[key]
			if current.Title == "" {
				current.Title = title
			}
			if current.TrackNumber == 0 {
				current.TrackNumber = trackNumber
			}
			if current.DiscNumber == 0 {
				current.DiscNumber = discNumber
			}
			if current.ResolvedTrackID == 0 && resolvedTrackID > 0 {
				current.ResolvedTrackID = resolvedTrackID
			}
			if playRecordID > 0 {
				current.PlayRecordIDs = append(current.PlayRecordIDs, playRecordID)
			}
			if favoriteEventID > 0 {
				current.FavoriteEventIDs = append(current.FavoriteEventIDs, favoriteEventID)
			}
			byPosition[key] = current
		}
		if key := normalizeTrackLookupKey(title); key != "" {
			current := byTitle[key]
			if current.Title == "" {
				current.Title = title
			}
			if current.TrackNumber == 0 {
				current.TrackNumber = trackNumber
			}
			if current.DiscNumber == 0 {
				current.DiscNumber = discNumber
			}
			if current.ResolvedTrackID == 0 && resolvedTrackID > 0 {
				current.ResolvedTrackID = resolvedTrackID
			}
			if playRecordID > 0 {
				current.PlayRecordIDs = append(current.PlayRecordIDs, playRecordID)
			}
			if favoriteEventID > 0 {
				current.FavoriteEventIDs = append(current.FavoriteEventIDs, favoriteEventID)
			}
			byTitle[key] = current
		}
	}

	for _, record := range detail.PlayRecords {
		register(record.Track, record.TrackNumber, record.DiscNumber, record.ResolvedTrackID, record.ID, 0)
	}
	for _, event := range detail.FavoriteEvents {
		register(event.Track, event.TrackNumber, event.DiscNumber, event.ResolvedTrackID, 0, event.ID)
	}
	return byPosition, byTitle
}

func chooseEvidenceForTrackDraft(
	draft pendingAlbumTrackDraft,
	byPosition map[string]pendingEvidence,
	byTitle map[string]pendingEvidence,
) (pendingEvidence, bool) {
	if draft.TrackNumber > 0 {
		discNumber := draft.DiscNumber
		if discNumber <= 0 {
			discNumber = 1
		}
		posKey := fmt.Sprintf("%d|%d", discNumber, draft.TrackNumber)
		if evidence, ok := byPosition[posKey]; ok {
			return evidence, true
		}
	}

	candidates := make([]string, 0, 1+len(draft.EvidenceTitles))
	candidates = append(candidates, draft.Title)
	candidates = append(candidates, draft.EvidenceTitles...)
	for _, title := range candidates {
		if evidence, ok := byTitle[normalizeTrackLookupKey(title)]; ok {
			return evidence, true
		}
	}
	return pendingEvidence{}, false
}

func buildAlbumDiscInfosFromTrackDrafts(trackDrafts []pendingAlbumTrackDraft) (int, string) {
	discInfosMap := make(map[int]int)
	totalDiscs := 0
	for _, draft := range trackDrafts {
		discNumber := int(draft.DiscNumber)
		if discNumber <= 0 {
			discNumber = 1
		}
		discInfosMap[discNumber]++
		if discNumber > totalDiscs {
			totalDiscs = discNumber
		}
	}
	if totalDiscs == 0 {
		totalDiscs = 1
		discInfosMap[1] = 0
	}
	raw, _ := json.Marshal(discInfosMap)
	return totalDiscs, string(raw)
}

func extractReleaseGenres(release musicbrainzws2.Release) string {
	if len(release.Genres) == 0 {
		return ""
	}
	genres := make([]string, 0, len(release.Genres))
	for _, genre := range release.Genres {
		if strings.TrimSpace(genre.Name) == "" {
			continue
		}
		genres = append(genres, genre.Name)
	}
	return strings.Join(genres, ",")
}

func (s *serviceImpl) DeepMaintainPendingAlbumWorkItem(
	ctx context.Context, workItemID int64,
) (*DeepMaintainPendingAlbumReport, error) {
	log.Info(ctx, "开始 MusicBrainz 深度维护待归因专辑工作项", zap.Int64("work_item_id", workItemID))
	detail, err := model.GetPendingAlbumWorkItemDetail(ctx, workItemID)
	if err != nil {
		log.Error(ctx, "获取待归因专辑工作项详情失败", zap.Int64("work_item_id", workItemID), zap.Error(err))
		return nil, err
	}

	material, err := s.resolveMusicBrainzMaterial(ctx, detail)
	if err != nil {
		log.Error(ctx, "解析 MusicBrainz 维护数据失败", zap.Int64("work_item_id", workItemID), zap.Error(err))
		return nil, err
	}
	return s.performPendingAlbumMaintenance(ctx, workItemID, detail, material)
}

func (s *serviceImpl) ManualMaintainPendingAlbumWorkItem(
	ctx context.Context, workItemID int64, input ManualPendingAlbumInput,
) (*DeepMaintainPendingAlbumReport, error) {
	log.Info(ctx, "开始手动维护待归因专辑工作项", zap.Int64("work_item_id", workItemID))
	detail, err := model.GetPendingAlbumWorkItemDetail(ctx, workItemID)
	if err != nil {
		log.Error(ctx, "获取待归因专辑工作项详情失败", zap.Int64("work_item_id", workItemID), zap.Error(err))
		return nil, err
	}

	material, err := s.resolveManualMaterial(detail, input)
	if err != nil {
		log.Error(ctx, "解析手动维护数据失败", zap.Int64("work_item_id", workItemID), zap.Error(err))
		return nil, err
	}
	return s.performPendingAlbumMaintenance(ctx, workItemID, detail, material)
}

func (s *serviceImpl) resolveMusicBrainzMaterial(
	ctx context.Context, detail *model.PendingAlbumWorkItemDetail,
) (*pendingAlbumMaintenanceMaterial, error) {
	if detail == nil || detail.WorkItem == nil || strings.TrimSpace(detail.WorkItem.SelectedMBID) == "" {
		return nil, errors.New("work item has no selected mbid")
	}

	release, err := coremusicbrainz.LookupRelease(
		ctx,
		mbtypes.MBID(detail.WorkItem.SelectedMBID),
		musicbrainzws2.IncludesFilter{Includes: []string{"recordings", "media", "artist-credits", "genres", "release-groups"}},
	)
	if err != nil {
		return nil, err
	}

	totalDiscs, discInfos := buildAlbumDiscInfosFromRelease(release)
	releaseDate := release.Date.String()
	originalReleaseDate := extractPendingAlbumOriginalReleaseDate(release)
	genreStr := extractReleaseGenres(release)
	material := &pendingAlbumMaintenanceMaterial{
		Mode: pendingAlbumMaintenanceModeMusicBrainz,
		AlbumCandidate: &model.Album{
			Name:                detail.WorkItem.Album,
			NameSubtitle:        detail.WorkItem.AlbumSubtitle,
			Artist:              pendingAlbumOwner(detail.WorkItem),
			ReleaseDate:         releaseDate,
			OriginalReleaseDate: originalReleaseDate,
			Genre:               genreStr,
			Country:             string(release.CountryCode),
			Status:              release.Status,
			Packaging:           release.Packaging,
			Barcode:             string(release.Barcode),
			TotalDiscs:          totalDiscs,
			DiscInfos:           discInfos,
		},
		CoverEnsureInput: artworklogic.EnsureAlbumCoverInput{
			AlbumArtist: detail.WorkItem.AlbumArtist,
			Artist:      detail.WorkItem.Artist,
			Album:       detail.WorkItem.Album,
		},
	}

	releaseRaw, _ := json.Marshal(release)
	material.ReleaseMB = &model.ReleaseMB{
		MBID:     detail.WorkItem.SelectedMBID,
		AlbumID:  0,
		Name:     release.Title,
		JSONData: string(releaseRaw),
	}

	for _, medium := range release.Media {
		for _, releaseTrack := range medium.Tracks {
			material.TrackDrafts = append(
				material.TrackDrafts,
				pendingAlbumTrackDraft{
					Title:         common.UnityFixAll(coremusicbrainz.TrackTitleWithFeat(releaseTrack)),
					TrackArtist:   detail.WorkItem.Artist,
					AlbumArtist:   pendingAlbumOwner(detail.WorkItem),
					DiscNumber:    int8(medium.Position),
					TrackNumber:   int8(releaseTrack.Position),
					Duration:      int64(releaseTrack.Length.Seconds()),
					MusicBrainzID: string(releaseTrack.Recording.ID),
				},
			)
		}
	}

	return material, nil
}

func buildAlbumDiscInfosFromRelease(release musicbrainzws2.Release) (int, string) {
	totalDiscs := len(release.Media)
	discInfosMap := make(map[int]int, totalDiscs)
	for _, medium := range release.Media {
		discInfosMap[medium.Position] = medium.TrackCount
	}
	raw, _ := json.Marshal(discInfosMap)
	return totalDiscs, string(raw)
}

func extractPendingAlbumOriginalReleaseDate(release musicbrainzws2.Release) string {
	if release.ReleaseGroup == nil {
		return ""
	}
	return strings.TrimSpace(release.ReleaseGroup.FirstReleaseDate.String())
}

func (s *serviceImpl) resolveManualMaterial(
	detail *model.PendingAlbumWorkItemDetail, input ManualPendingAlbumInput,
) (*pendingAlbumMaintenanceMaterial, error) {
	if detail == nil || detail.WorkItem == nil {
		return nil, errors.New("work item detail is nil")
	}

	albumName := strings.TrimSpace(input.ManualAlbum.Name)
	if albumName == "" {
		return nil, errors.New("manual_album.name is required")
	}
	albumArtist := strings.TrimSpace(input.ManualAlbum.AlbumArtist)
	if albumArtist == "" {
		return nil, errors.New("manual_album.album_artist is required")
	}
	if len(input.ManualTracks) == 0 {
		return nil, errors.New("manual_tracks is required")
	}

	trackDrafts := make([]pendingAlbumTrackDraft, 0, len(input.ManualTracks))
	positionSet := make(map[string]struct{}, len(input.ManualTracks))
	for idx, track := range input.ManualTracks {
		title := strings.TrimSpace(track.Title)
		if title == "" {
			return nil, fmt.Errorf("manual_tracks[%d].title is required", idx)
		}
		discNumber := track.DiscNumber
		if discNumber <= 0 {
			return nil, fmt.Errorf("manual_tracks[%d].disc_number must be greater than 0", idx)
		}
		trackNumber := track.TrackNumber
		if trackNumber <= 0 {
			return nil, fmt.Errorf("manual_tracks[%d].track_number must be greater than 0", idx)
		}
		posKey := fmt.Sprintf("%d|%d", discNumber, trackNumber)
		if _, ok := positionSet[posKey]; ok {
			return nil, fmt.Errorf("manual_tracks has duplicated disc/track position: %s", posKey)
		}
		positionSet[posKey] = struct{}{}

		evidenceTitles := make([]string, 0, len(track.EvidenceTitles))
		for _, evidenceTitle := range track.EvidenceTitles {
			trimmed := strings.TrimSpace(evidenceTitle)
			if trimmed == "" {
				continue
			}
			evidenceTitles = append(evidenceTitles, trimmed)
		}
		trackArtist := strings.TrimSpace(track.Artist)
		if trackArtist == "" {
			trackArtist = strings.TrimSpace(input.ManualAlbum.DisplayArtist)
		}
		if trackArtist == "" {
			trackArtist = strings.TrimSpace(detail.WorkItem.Artist)
		}
		if trackArtist == "" {
			trackArtist = albumArtist
		}

		trackDrafts = append(
			trackDrafts,
			pendingAlbumTrackDraft{
				Title:          title,
				TrackArtist:    trackArtist,
				AlbumArtist:    albumArtist,
				DiscNumber:     discNumber,
				TrackNumber:    trackNumber,
				Duration:       track.Duration,
				Composer:       strings.TrimSpace(track.Composer),
				MusicBrainzID:  strings.TrimSpace(track.MusicBrainzID),
				EvidenceTitles: evidenceTitles,
			},
		)
	}

	totalDiscs, discInfos := buildAlbumDiscInfosFromTrackDrafts(trackDrafts)
	return &pendingAlbumMaintenanceMaterial{
		Mode: pendingAlbumMaintenanceModeManual,
		AlbumCandidate: &model.Album{
			Name:                albumName,
			Artist:              albumArtist,
			ReleaseDate:         strings.TrimSpace(input.ManualAlbum.ReleaseDate),
			OriginalReleaseDate: strings.TrimSpace(input.ManualAlbum.OriginalReleaseDate),
			Genre:               strings.TrimSpace(input.ManualAlbum.Genre),
			Country:             strings.TrimSpace(input.ManualAlbum.Country),
			Status:              strings.TrimSpace(input.ManualAlbum.Status),
			Packaging:           strings.TrimSpace(input.ManualAlbum.Packaging),
			Barcode:             strings.TrimSpace(input.ManualAlbum.Barcode),
			TotalDiscs:          totalDiscs,
			DiscInfos:           discInfos,
		},
		TrackDrafts: trackDrafts,
		CoverEnsureInput: artworklogic.EnsureAlbumCoverInput{
			AlbumArtist:  albumArtist,
			Artist:       strings.TrimSpace(input.ManualAlbum.DisplayArtist),
			Album:        albumName,
			CoverArtURL:  strings.TrimSpace(input.ManualAlbum.CoverArtURL),
			CoverArtMime: "",
		},
	}, nil
}

func (s *serviceImpl) performPendingAlbumMaintenance(
	ctx context.Context,
	workItemID int64,
	detail *model.PendingAlbumWorkItemDetail,
	material *pendingAlbumMaintenanceMaterial,
) (*DeepMaintainPendingAlbumReport, error) {
	if detail == nil || detail.WorkItem == nil {
		return nil, errors.New("work item detail is nil")
	}
	if material == nil || material.AlbumCandidate == nil {
		return nil, errors.New("maintenance material is nil")
	}

	report := &DeepMaintainPendingAlbumReport{Mode: material.Mode}
	if err := model.UpdatePendingAlbumWorkItemProgress(
		ctx,
		workItemID,
		model.PendingAlbumWorkItemStatusDeepMaintaning,
		detail.WorkItem.ResolvedAlbumID,
		"",
	); err != nil {
		log.Error(ctx, "更新待归因专辑工作项进度失败", zap.Int64("work_item_id", workItemID), zap.Error(err))
		return nil, err
	}

	err := model.InTx(
		ctx,
		func(tx *gorm.DB) error {
			return applyPendingAlbumStructureTx(tx, detail, material, report)
		},
	)
	if err != nil {
		_ = model.UpdatePendingAlbumWorkItemProgress(
			ctx, workItemID, model.PendingAlbumWorkItemStatusFailed, report.ResolvedAlbumID, err.Error(),
		)
		log.Error(
			ctx,
			"待归因专辑结构维护事务执行失败",
			zap.String("mode", material.Mode),
			zap.Int64("work_item_id", workItemID),
			zap.Int64("album_id", report.ResolvedAlbumID),
			zap.Error(err),
		)
		return nil, err
	}

	if coverErr := ensurePendingAlbumCover(ctx, detail, report.ResolvedAlbumID, material.CoverEnsureInput); coverErr != nil {
		log.Warn(
			ctx,
			"待归因专辑维护补齐封面失败",
			zap.String("mode", material.Mode),
			zap.Int64("work_item_id", workItemID),
			zap.Int64("album_id", report.ResolvedAlbumID),
			zap.Error(coverErr),
		)
	}

	if err := model.UpdatePendingAlbumWorkItemProgress(
		ctx,
		workItemID,
		model.PendingAlbumWorkItemStatusApplying,
		report.ResolvedAlbumID,
		"",
	); err != nil {
		log.Error(ctx, "更新待归因专辑工作项为应用阶段失败", zap.Int64("work_item_id", workItemID), zap.Error(err))
		return nil, err
	}

	if err := applyPendingAlbumFrozenContext(ctx, detail, report, workItemID); err != nil {
		return nil, err
	}

	if err := model.UpdatePendingAlbumWorkItemProgress(
		ctx,
		workItemID,
		model.PendingAlbumWorkItemStatusCompleted,
		report.ResolvedAlbumID,
		"",
	); err != nil {
		log.Error(ctx, "更新待归因专辑工作项为完成失败", zap.Int64("work_item_id", workItemID), zap.Error(err))
		return nil, err
	}

	log.Info(
		ctx,
		"待归因专辑维护完成",
		zap.String("mode", material.Mode),
		zap.Int64("work_item_id", workItemID),
		zap.Int64("album_id", report.ResolvedAlbumID),
		zap.Int("reused_heard_tracks", report.ReusedHeardTracks),
		zap.Int("created_tracks", report.CreatedTracks),
		zap.Int("track_album_writes", report.TrackAlbumWrites),
		zap.Int("applied_play_records", report.AppliedPlayRecords),
		zap.Int("applied_favorite_events", report.AppliedFavoriteEvents),
	)
	return report, nil
}

func applyPendingAlbumStructureTx(
	tx *gorm.DB,
	detail *model.PendingAlbumWorkItemDetail,
	material *pendingAlbumMaintenanceMaterial,
	report *DeepMaintainPendingAlbumReport,
) error {
	resolvedAlbum, err := model.ResolveCanonicalAlbumForPendingContextTx(tx, material.AlbumCandidate)
	if err != nil {
		return err
	}
	report.ResolvedAlbumID = resolvedAlbum.ID

	if material.ReleaseMB != nil {
		material.ReleaseMB.AlbumID = resolvedAlbum.ID
		if err := model.SaveReleaseMBTx(tx, material.ReleaseMB); err != nil {
			return err
		}
		if err := model.LinkAlbumToMBIDTx(
			tx,
			resolvedAlbum.ID,
			material.ReleaseMB.ID,
			material.ReleaseMB.MBID,
		); err != nil {
			return err
		}
	}

	byPosition, byTitle := buildPendingEvidence(detail)
	for _, draft := range material.TrackDrafts {
		evidence, hasEvidence := chooseEvidenceForTrackDraft(draft, byPosition, byTitle)
		trackObj, reusedTrack, err := resolvePendingTrackTx(tx, resolvedAlbum, draft, evidence, hasEvidence)
		if err != nil {
			return err
		}
		if reusedTrack {
			report.ReusedHeardTracks++
		} else {
			report.CreatedTracks++
		}

		if err := model.UpdateTrackCuratedMetadataTx(
			tx,
			trackObj.ID,
			&model.TrackIdentity{
				Artist:        draft.TrackArtist,
				Album:         resolvedAlbum.Name,
				AlbumSubtitle: resolvedAlbum.NameSubtitle,
				Track:         draft.Title,
				TrackNumber:   draft.TrackNumber,
				DiscNumber:    draft.DiscNumber,
			},
			&model.TrackMetadata{
				AlbumArtist:   draft.AlbumArtist,
				AlbumSubtitle: resolvedAlbum.NameSubtitle,
				TrackNumber:   draft.TrackNumber,
				DiscNumber:    draft.DiscNumber,
				Duration:      draft.Duration,
				Composer:      draft.Composer,
				Genre:         material.AlbumCandidate.Genre,
				ReleaseDate:   material.AlbumCandidate.ReleaseDate,
				MusicBrainzID: draft.MusicBrainzID,
				Source:        "PendingAlbumWorkItem",
			},
		); err != nil {
			return err
		}
		if err := model.UpsertTrackAlbumTx(
			tx,
			&model.TrackAlbum{
				TrackID:                trackObj.ID,
				AlbumID:                resolvedAlbum.ID,
				TrackNumber:            draft.TrackNumber,
				DiscNumber:             draft.DiscNumber,
				Track:                  draft.Title,
				MusicBrainzRecordingID: draft.MusicBrainzID,
			},
			true,
		); err != nil {
			return err
		}
		report.TrackAlbumWrites++
		appliedPlayCount, appliedFavoriteCount, err := bindPendingEvidenceToTrackTx(
			tx,
			trackObj,
			evidence,
			hasEvidence,
		)
		if err != nil {
			return err
		}
		report.AppliedPlayRecords += appliedPlayCount
		report.AppliedFavoriteEvents += appliedFavoriteCount
	}

	return model.UpdateAlbumFieldsTx(
		tx,
		resolvedAlbum.ID,
		func() map[string]interface{} {
			fields := map[string]interface{}{
				"sync_status": 3,
			}
			if releaseDate := strings.TrimSpace(material.AlbumCandidate.ReleaseDate); releaseDate != "" {
				fields["release_date"] = releaseDate
			}
			if originalReleaseDate := strings.TrimSpace(material.AlbumCandidate.OriginalReleaseDate); originalReleaseDate != "" {
				fields["original_release_date"] = originalReleaseDate
			}
			if genre := strings.TrimSpace(material.AlbumCandidate.Genre); genre != "" {
				fields["genre"] = genre
			}
			if country := strings.TrimSpace(material.AlbumCandidate.Country); country != "" {
				fields["country"] = country
			}
			if status := strings.TrimSpace(material.AlbumCandidate.Status); status != "" {
				fields["status"] = status
			}
			if packaging := strings.TrimSpace(material.AlbumCandidate.Packaging); packaging != "" {
				fields["packaging"] = packaging
			}
			if barcode := strings.TrimSpace(material.AlbumCandidate.Barcode); barcode != "" {
				fields["barcode"] = barcode
			}
			if material.AlbumCandidate.TotalDiscs > 0 {
				fields["total_discs"] = material.AlbumCandidate.TotalDiscs
			}
			if discInfos := strings.TrimSpace(material.AlbumCandidate.DiscInfos); discInfos != "" {
				fields["disc_infos"] = discInfos
			}
			return fields
		}(),
	)
}

func bindPendingEvidenceToTrackTx(
	tx *gorm.DB,
	trackObj *model.Track,
	evidence pendingEvidence,
	hasEvidence bool,
) (int, int, error) {
	if tx == nil || trackObj == nil || trackObj.ID <= 0 || !hasEvidence {
		return 0, 0, nil
	}

	appliedPlayCount := 0
	for _, recordID := range evidence.PlayRecordIDs {
		applied, err := model.ApplyTrackPlayRecordToResolvedTrackTx(
			tx,
			recordID,
			trackObj.ID,
			common.TrackMetadataConfidenceAuthoritative,
		)
		if err != nil {
			return 0, 0, err
		}
		if applied {
			appliedPlayCount++
		}
	}

	appliedFavoriteCount := 0
	for _, eventID := range evidence.FavoriteEventIDs {
		applied, err := model.ApplyTrackFavoriteEventToResolvedTrackTx(
			tx,
			eventID,
			trackObj.ID,
			common.TrackMetadataConfidenceAuthoritative,
		)
		if err != nil {
			return 0, 0, err
		}
		if applied {
			appliedFavoriteCount++
		}
	}

	return appliedPlayCount, appliedFavoriteCount, nil
}

func resolvePendingTrackTx(
	tx *gorm.DB,
	resolvedAlbum *model.Album,
	draft pendingAlbumTrackDraft,
	evidence pendingEvidence,
	hasEvidence bool,
) (*model.Track, bool, error) {
	var (
		trackObj *model.Track
		err      error
	)

	trackObj, err = model.GetTrackByIdentityTx(
		tx,
		draft.TrackArtist,
		resolvedAlbum.Name,
		resolvedAlbum.NameSubtitle,
		draft.Title,
		draft.TrackNumber,
		draft.DiscNumber,
	)
	if err == nil {
		return trackObj, true, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	if strings.TrimSpace(draft.MusicBrainzID) != "" {
		trackObj, err = model.GetTrackByMusicBrainzIdentityTx(
			tx,
			draft.MusicBrainzID,
			draft.TrackArtist,
			resolvedAlbum.Name,
			resolvedAlbum.NameSubtitle,
			draft.Title,
			draft.TrackNumber,
			draft.DiscNumber,
		)
		if err == nil {
			return trackObj, true, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, err
		}
	}

	if hasEvidence && evidence.ResolvedTrackID > 0 {
		trackObj, err = model.GetTrackByIDTx(tx, evidence.ResolvedTrackID)
		if err == nil {
			return trackObj, true, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, err
		}
	}
	trackObj, err = model.GetOrCreateTrackByIdentityTx(
		tx,
		&model.Track{
			Artist:        draft.TrackArtist,
			AlbumArtist:   draft.AlbumArtist,
			Album:         resolvedAlbum.Name,
			AlbumSubtitle: resolvedAlbum.NameSubtitle,
			Track:         draft.Title,
			TrackNumber:   draft.TrackNumber,
			DiscNumber:    draft.DiscNumber,
			Duration:      draft.Duration,
			Genre:         resolvedAlbum.Genre,
			Composer:      draft.Composer,
			ReleaseDate:   resolvedAlbum.ReleaseDate,
			MusicBrainzID: draft.MusicBrainzID,
			Source:        "PendingAlbumWorkItem",
		},
	)
	if err != nil {
		return nil, false, err
	}
	return trackObj, false, nil
}

func ensurePendingAlbumCover(
	ctx context.Context,
	detail *model.PendingAlbumWorkItemDetail,
	resolvedAlbumID int64,
	input artworklogic.EnsureAlbumCoverInput,
) error {
	input.AlbumID = resolvedAlbumID
	if strings.TrimSpace(input.AlbumArtist) == "" && detail != nil && detail.WorkItem != nil {
		input.AlbumArtist = detail.WorkItem.AlbumArtist
	}
	if strings.TrimSpace(input.Artist) == "" && detail != nil && detail.WorkItem != nil {
		input.Artist = detail.WorkItem.Artist
	}
	if strings.TrimSpace(input.Album) == "" && detail != nil && detail.WorkItem != nil {
		input.Album = detail.WorkItem.Album
	}
	return artworklogic.EnsureAlbumCover(ctx, input)
}

func applyPendingAlbumFrozenContext(
	ctx context.Context,
	detail *model.PendingAlbumWorkItemDetail,
	report *DeepMaintainPendingAlbumReport,
	workItemID int64,
) error {
	playRecordIDs := make([]int64, 0, len(detail.PlayRecords))
	for _, record := range detail.PlayRecords {
		playRecordIDs = append(playRecordIDs, record.ID)
	}
	favoriteEventIDs := make([]int64, 0, len(detail.FavoriteEvents))
	for _, event := range detail.FavoriteEvents {
		favoriteEventIDs = append(favoriteEventIDs, event.ID)
	}

	replayReport, err := model.ReplayTrackPlayRecords(
		model.ReplayTrackPlayRecordsParams{
			Ctx:       ctx,
			RecordIDs: playRecordIDs,
			DryRun:    false,
		},
	)
	if err != nil {
		_ = model.UpdatePendingAlbumWorkItemProgress(
			ctx, workItemID, model.PendingAlbumWorkItemStatusFailed, report.ResolvedAlbumID, err.Error(),
		)
		log.Error(ctx, "回放播放记录失败", zap.Int64("work_item_id", workItemID), zap.Error(err))
		return err
	}
	report.AppliedPlayRecords += len(replayReport.Results)

	favResult, err := model.ApplyTrackFavoriteEventsByIDs(ctx, favoriteEventIDs)
	if err != nil {
		_ = model.UpdatePendingAlbumWorkItemProgress(
			ctx, workItemID, model.PendingAlbumWorkItemStatusFailed, report.ResolvedAlbumID, err.Error(),
		)
		log.Error(ctx, "应用收藏事件失败", zap.Int64("work_item_id", workItemID), zap.Error(err))
		return err
	}
	report.AppliedFavoriteEvents += favResult.AppliedCount
	return nil
}
