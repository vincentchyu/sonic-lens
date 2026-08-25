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
	"github.com/vincentchyu/sonic-lens/internal/logic/musicbrainz"
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
	ReleaseMBID  int64                          `json:"release_mb_id"`
	MBID         string                         `json:"mbid"`
	ManualAlbum  ManualPendingAlbumAlbumInput   `json:"manual_album"`
	ManualTracks []ManualPendingAlbumTrackInput `json:"manual_tracks"`
}

// ManualPendingAlbumAlbumInput 描述手动维护时的专辑级元数据。
type ManualPendingAlbumAlbumInput struct {
	Name                string `json:"name"`
	AlbumSubtitle       string `json:"album_subtitle"`
	ReleaseType         string `json:"release_type"`
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
	Genre          string   `json:"genre"`
	Duration       int64    `json:"duration"`
	Composer       string   `json:"composer"`
	MusicBrainzID  string   `json:"music_brainz_id"`
	EvidenceTitles []string `json:"evidence_titles"`
}

// PendingAlbumAlbumPreview 描述在预审中针对专辑元数据的对比。
type PendingAlbumAlbumPreview struct {
	Name                string `json:"name"`
	AlbumSubtitle       string `json:"album_subtitle"`
	ReleaseType         string `json:"release_type"`
	EvidenceReleaseType string `json:"evidence_release_type"`
	MBReleaseType       string `json:"mb_release_type"`
	AlbumArtist         string `json:"album_artist"`
	DisplayArtist       string `json:"display_artist"`
	ReleaseDate         string `json:"release_date"`
	OriginalReleaseDate string `json:"original_release_date"`
	Genre               string `json:"genre"`
	EvidenceGenre       string `json:"evidence_genre"`
	Country             string `json:"country"`
	Status              string `json:"status"`
	Packaging           string `json:"packaging"`
	Barcode             string `json:"barcode"`
	CoverArtURL         string `json:"cover_art_url"`
}

// PendingAlbumTrackPreview 描述在预审中针对单首曲目的对比与建议。
type PendingAlbumTrackPreview struct {
	DiscNumber     int8     `json:"disc_number"`
	TrackNumber    int8     `json:"track_number"`
	Title          string   `json:"title"`
	Artist         string   `json:"artist"`
	Genre          string   `json:"genre"`
	EvidenceGenre  string   `json:"evidence_genre"`
	Duration       int64    `json:"duration"`
	Composer       string   `json:"composer"`
	MusicBrainzID  string   `json:"music_brainz_id"`
	EvidenceTitles []string `json:"evidence_titles"`
	MBTitle        string   `json:"mb_title"`
	EvidenceTitle  string   `json:"evidence_title"`
	HasDiff        bool     `json:"has_diff"`
}

// PendingAlbumMaintenancePreview 描述维护预审的整体对比快照。
type PendingAlbumMaintenancePreview struct {
	WorkItemID     int64                      `json:"work_item_id"`
	ReleaseMBID    int64                      `json:"release_mb_id"`
	MBID           string                     `json:"mbid"`
	AlbumPreview   PendingAlbumAlbumPreview   `json:"album_preview"`
	TrackPreviews  []PendingAlbumTrackPreview `json:"track_previews"`
	DiffTrackCount int                        `json:"diff_track_count"`
	SuggestedInput ManualPendingAlbumInput    `json:"suggested_input"`
}

// Service 定义待归因专辑工作台能力。
type Service interface {
	GetPendingAlbumGroups(ctx context.Context, limit int) ([]*model.PendingAlbumGroup, error)
	GetPendingAlbumGroupsWithOptions(
		ctx context.Context, opts model.PendingAlbumGroupQueryOptions,
	) ([]*model.PendingAlbumGroup, error)
	CreateOrGetPendingAlbumWorkItem(ctx context.Context, identityKey string) (*model.PendingAlbumWorkItem, error)
	GetPendingAlbumWorkItemDetail(ctx context.Context, workItemID int64) (*model.PendingAlbumWorkItemDetail, error)
	RefreshPendingAlbumWorkItemContext(ctx context.Context, workItemID int64) (*model.PendingAlbumWorkItem, error)
	SearchPendingAlbumMBReleases(ctx context.Context, workItemID int64) ([]*model.ReleaseMB, error)
	LinkPendingAlbumMBRelease(ctx context.Context, workItemID, releaseMBID int64, mbid string) error
	PreviewPendingAlbumMBMaintenance(
		ctx context.Context, workItemID, releaseMBID int64, mbid string, forceRefresh bool,
	) (*PendingAlbumMaintenancePreview, error)
	SavePendingAlbumStagingDraft(
		ctx context.Context, workItemID int64, draft *PendingAlbumMaintenancePreview,
	) error
	DeepMaintainPendingAlbumWorkItem(ctx context.Context, workItemID int64) (*DeepMaintainPendingAlbumReport, error)
	ManualMaintainPendingAlbumWorkItem(
		ctx context.Context, workItemID int64, input ManualPendingAlbumInput,
	) (*DeepMaintainPendingAlbumReport, error)
	ListWorkItems(
		ctx context.Context, limit, offset int, keyword string, statusGroup string,
	) ([]*model.PendingAlbumWorkItem, int64, error)
	PreviewAlbumMBMaintenance(
		ctx context.Context, albumID, releaseMBID int64, mbid string, forceRefresh bool,
	) (*PendingAlbumMaintenancePreview, error)
	ApplyAlbumMBMaintenance(
		ctx context.Context, albumID int64, input *ManualPendingAlbumInput,
	) error
	SaveAlbumStagingDraft(
		ctx context.Context, albumID int64, draft *PendingAlbumMaintenancePreview,
	) error
}

type serviceImpl struct{}

type pendingEvidence struct {
	Title            string
	Genre            string
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

func (s *serviceImpl) GetPendingAlbumGroupsWithOptions(
	ctx context.Context, opts model.PendingAlbumGroupQueryOptions,
) ([]*model.PendingAlbumGroup, error) {
	return model.GetPendingAlbumGroupsWithOptions(ctx, opts)
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

func extractReleaseTypeFromRelease(release musicbrainzws2.Release) string {
	if release.ReleaseGroup != nil && strings.TrimSpace(release.ReleaseGroup.PrimaryType) != "" {
		switch strings.ToLower(strings.TrimSpace(release.ReleaseGroup.PrimaryType)) {
		case "ep":
			return "ep"
		case "single":
			return "single"
		case "album", "lp":
			return "album"
		case "broadcast", "other":
			return strings.ToLower(strings.TrimSpace(release.ReleaseGroup.PrimaryType))
		default:
			return strings.ToLower(strings.TrimSpace(release.ReleaseGroup.PrimaryType))
		}
	}
	_, parsedRT := common.ParseAlbumTitleAndReleaseType(release.Title)
	return strings.TrimSpace(parsedRT)
}

func extractPendingEvidenceReleaseType(detail *model.PendingAlbumWorkItemDetail) string {
	if detail == nil {
		return ""
	}
	for _, rec := range detail.PlayRecords {
		if rec != nil && strings.TrimSpace(rec.ReleaseType) != "" {
			return strings.TrimSpace(rec.ReleaseType)
		}
	}
	return ""
}

func (s *serviceImpl) SearchPendingAlbumMBReleases(ctx context.Context, workItemID int64) ([]*model.ReleaseMB, error) {
	log.Info(ctx, "搜索待归因专辑的 MB Releases", zap.Int64("work_item_id", workItemID))
	item, err := model.GetPendingAlbumWorkItemByID(ctx, workItemID)
	if err != nil {
		return nil, err
	}

	_, itemRT := common.ParseAlbumTitleAndReleaseType(item.Album)
	if itemRT == "" {
		if detail, err := model.GetPendingAlbumWorkItemDetail(ctx, workItemID); err == nil && detail != nil {
			itemRT = extractPendingEvidenceReleaseType(detail)
		}
	}

	// 复用 musicbrainz 模块的统一两阶段搜索（含连字符后缀识别与 primarytype 过滤）
	searchRes, err := musicbrainz.SearchMBReleases(ctx, item.Album, pendingAlbumOwner(item), itemRT)
	if err != nil {
		return nil, err
	}

	results := make([]*model.ReleaseMB, 0, len(searchRes.Releases)+1)
	foundSelected := false
	for idx, release := range searchRes.Releases {
		if string(release.ID) == item.SelectedMBID {
			foundSelected = true
		}
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

	if item.SelectedMBID != "" && !foundSelected {
		candidateTitle := item.Album
		if strings.TrimSpace(item.StagingDraftJSON) != "" {
			var draft PendingAlbumMaintenancePreview
			if unmarshalErr := json.Unmarshal(
				[]byte(item.StagingDraftJSON), &draft,
			); unmarshalErr == nil && draft.AlbumPreview.Name != "" {
				candidateTitle = draft.AlbumPreview.Name
			}
		}
		rel, lookupErr := coremusicbrainz.LookupRelease(
			ctx, mbtypes.MBID(item.SelectedMBID), musicbrainzws2.IncludesFilter{},
		)
		if lookupErr == nil && rel.Title != "" {
			candidateTitle = rel.Title
		}
		results = append(
			[]*model.ReleaseMB{
				{
					ID:       item.SelectedReleaseMBID,
					MBID:     item.SelectedMBID,
					AlbumID:  0,
					Name:     candidateTitle,
					JSONData: "{}",
				},
			}, results...,
		)
	}

	log.Info(
		ctx, "搜索待归因专辑的 MB Releases 完成", zap.Int64("work_item_id", workItemID), zap.Int("count", len(results)),
	)
	return results, nil
}

func (s *serviceImpl) LinkPendingAlbumMBRelease(ctx context.Context, workItemID, releaseMBID int64, mbid string) error {
	return model.UpdatePendingAlbumWorkItemSelection(ctx, workItemID, releaseMBID, strings.TrimSpace(mbid))
}

func (s *serviceImpl) PreviewPendingAlbumMBMaintenance(
	ctx context.Context, workItemID, releaseMBID int64, mbid string, forceRefresh bool,
) (*PendingAlbumMaintenancePreview, error) {
	log.Info(
		ctx, "生成/查看待归因专辑的 MB 维护草稿与预审 Preview", zap.Int64("work_item_id", workItemID),
		zap.String("mbid", mbid), zap.Bool("force_refresh", forceRefresh),
	)
	detail, err := model.GetPendingAlbumWorkItemDetail(ctx, workItemID)
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.WorkItem == nil {
		return nil, errors.New("work item detail not found")
	}

	targetMBID := strings.TrimSpace(mbid)
	if targetMBID == "" {
		targetMBID = strings.TrimSpace(detail.WorkItem.SelectedMBID)
	}

	if !forceRefresh && strings.TrimSpace(detail.WorkItem.StagingDraftJSON) != "" {
		// 只要目标 MBID 跟数据库当前已绑定的 MBID 相同，就可以复用草稿快照
		if targetMBID == detail.WorkItem.SelectedMBID && (releaseMBID <= 0 || releaseMBID == detail.WorkItem.SelectedReleaseMBID) {
			var cachedPreview PendingAlbumMaintenancePreview
			if unmarshalErr := json.Unmarshal(
				[]byte(detail.WorkItem.StagingDraftJSON), &cachedPreview,
			); unmarshalErr == nil {
				log.Info(ctx, "优先使用数据库已存的待归因草稿快照", zap.Int64("work_item_id", workItemID))
				return &cachedPreview, nil
			}
		}
	}

	if targetMBID == "" {
		return nil, errors.New("请先在 MusicBrainz 候选列表中点击选定一个版本")
	}

	if targetMBID != detail.WorkItem.SelectedMBID || (releaseMBID > 0 && releaseMBID != detail.WorkItem.SelectedReleaseMBID) {
		if updateErr := model.UpdatePendingAlbumWorkItemSelection(
			ctx, workItemID, releaseMBID, targetMBID,
		); updateErr != nil {
			log.Warn(ctx, "预审前更新 MBID 绑定失败", zap.Error(updateErr))
		}
		detail.WorkItem.SelectedMBID = targetMBID
		detail.WorkItem.SelectedReleaseMBID = releaseMBID
	}

	material, err := s.resolveMusicBrainzMaterial(ctx, detail)
	if err != nil {
		log.Error(ctx, "解析 MusicBrainz 维护数据失败", zap.Int64("work_item_id", workItemID), zap.Error(err))
		return nil, err
	}

	byPosition, byTitle := buildPendingEvidence(detail)

	evidenceGenre := ""
	for _, rec := range detail.PlayRecords {
		if rec != nil && strings.TrimSpace(rec.Genre) != "" {
			evidenceGenre = strings.TrimSpace(rec.Genre)
			break
		}
	}
	evidenceReleaseType := extractPendingEvidenceReleaseType(detail)
	mbReleaseType := ""
	if material.ReleaseMB != nil && material.ReleaseMB.JSONData != "" {
		var rel musicbrainzws2.Release
		if err := json.Unmarshal([]byte(material.ReleaseMB.JSONData), &rel); err == nil {
			mbReleaseType = extractReleaseTypeFromRelease(rel)
		}
	}

	albumPreview := PendingAlbumAlbumPreview{
		Name:                material.AlbumCandidate.Name,
		AlbumSubtitle:       material.AlbumCandidate.NameSubtitle,
		ReleaseType:         material.AlbumCandidate.ReleaseType,
		EvidenceReleaseType: evidenceReleaseType,
		MBReleaseType:       mbReleaseType,
		AlbumArtist:         material.AlbumCandidate.Artist,
		DisplayArtist:       material.AlbumCandidate.Artist,
		ReleaseDate:         material.AlbumCandidate.ReleaseDate,
		OriginalReleaseDate: material.AlbumCandidate.OriginalReleaseDate,
		Genre:               material.AlbumCandidate.Genre,
		EvidenceGenre:       evidenceGenre,
		Country:             material.AlbumCandidate.Country,
		Status:              material.AlbumCandidate.Status,
		Packaging:           material.AlbumCandidate.Packaging,
		Barcode:             material.AlbumCandidate.Barcode,
		CoverArtURL:         "",
	}

	suggestedInput := ManualPendingAlbumInput{
		ManualAlbum: ManualPendingAlbumAlbumInput{
			Name:                albumPreview.Name,
			AlbumSubtitle:       albumPreview.AlbumSubtitle,
			ReleaseType:         albumPreview.ReleaseType,
			AlbumArtist:         albumPreview.AlbumArtist,
			DisplayArtist:       albumPreview.DisplayArtist,
			ReleaseDate:         albumPreview.ReleaseDate,
			OriginalReleaseDate: albumPreview.OriginalReleaseDate,
			Genre:               albumPreview.Genre,
			Country:             albumPreview.Country,
			Status:              albumPreview.Status,
			Packaging:           albumPreview.Packaging,
			Barcode:             albumPreview.Barcode,
			CoverArtURL:         albumPreview.CoverArtURL,
		},
		ManualTracks: make([]ManualPendingAlbumTrackInput, 0, len(material.TrackDrafts)),
	}

	trackPreviews := make([]PendingAlbumTrackPreview, 0, len(material.TrackDrafts))
	diffCount := 0

	for _, draft := range material.TrackDrafts {
		evidence, hasEvidence := chooseEvidenceForTrackDraft(draft, byPosition, byTitle)
		evidenceTitle := ""
		evidenceTitles := make([]string, 0)
		if hasEvidence {
			evidenceTitle = evidence.Title
			if evidence.Title != "" {
				evidenceTitles = append(evidenceTitles, evidence.Title)
			}
		}

		mbTitle := draft.Title
		hasDiff := false
		if evidenceTitle != "" && evidenceTitle != mbTitle {
			hasDiff = true
			diffCount++
		}

		evidenceGenreTrack := ""
		if hasEvidence {
			evidenceGenreTrack = evidence.Genre
		}

		trackPrev := PendingAlbumTrackPreview{
			DiscNumber:     draft.DiscNumber,
			TrackNumber:    draft.TrackNumber,
			Title:          mbTitle,
			Artist:         draft.TrackArtist,
			Genre:          evidenceGenreTrack,
			EvidenceGenre:  evidenceGenreTrack,
			Duration:       draft.Duration,
			Composer:       draft.Composer,
			MusicBrainzID:  draft.MusicBrainzID,
			EvidenceTitles: evidenceTitles,
			MBTitle:        mbTitle,
			EvidenceTitle:  evidenceTitle,
			HasDiff:        hasDiff,
		}
		trackPreviews = append(trackPreviews, trackPrev)

		suggestedInput.ManualTracks = append(
			suggestedInput.ManualTracks, ManualPendingAlbumTrackInput{
				DiscNumber:     draft.DiscNumber,
				TrackNumber:    draft.TrackNumber,
				Title:          mbTitle,
				Artist:         draft.TrackArtist,
				Genre:          evidenceGenreTrack,
				Duration:       draft.Duration,
				Composer:       draft.Composer,
				MusicBrainzID:  draft.MusicBrainzID,
				EvidenceTitles: evidenceTitles,
			},
		)
	}

	preview := &PendingAlbumMaintenancePreview{
		WorkItemID:     workItemID,
		ReleaseMBID:    releaseMBID,
		MBID:           targetMBID,
		AlbumPreview:   albumPreview,
		TrackPreviews:  trackPreviews,
		DiffTrackCount: diffCount,
		SuggestedInput: suggestedInput,
	}

	draftRaw, _ := json.Marshal(preview)
	if err := model.SavePendingAlbumWorkItemStagingDraft(
		ctx, workItemID, releaseMBID, targetMBID, string(draftRaw),
	); err != nil {
		log.Error(ctx, "保存待归因专辑草稿快照失败", zap.Int64("work_item_id", workItemID), zap.Error(err))
		return nil, err
	}

	return preview, nil
}

func (s *serviceImpl) SavePendingAlbumStagingDraft(
	ctx context.Context, workItemID int64, draft *PendingAlbumMaintenancePreview,
) error {
	if draft == nil {
		return errors.New("draft payload cannot be nil")
	}
	draft.WorkItemID = workItemID
	draftRaw, err := json.Marshal(draft)
	if err != nil {
		return fmt.Errorf("serialize draft error: %w", err)
	}
	if saveErr := model.SavePendingAlbumWorkItemStagingDraft(
		ctx, workItemID, draft.ReleaseMBID, draft.MBID, string(draftRaw),
	); saveErr != nil {
		log.Error(ctx, "手动保存待归因专辑草稿失败", zap.Int64("work_item_id", workItemID), zap.Error(saveErr))
		return saveErr
	}
	log.Info(ctx, "已手动更新并保存待归因专辑草稿快照", zap.Int64("work_item_id", workItemID))
	return nil
}

func buildPendingEvidence(detail *model.PendingAlbumWorkItemDetail) (
	map[string]pendingEvidence, map[string]pendingEvidence,
) {
	byPosition := make(map[string]pendingEvidence)
	byTitle := make(map[string]pendingEvidence)

	register := func(
		title, genre string, trackNumber, discNumber int8, resolvedTrackID, playRecordID, favoriteEventID int64,
	) {
		if trackNumber > 0 {
			if discNumber <= 0 {
				discNumber = 1
			}
			key := fmt.Sprintf("%d|%d", discNumber, trackNumber)
			current := byPosition[key]
			if current.Title == "" {
				current.Title = title
			}
			if current.Genre == "" {
				current.Genre = genre
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
			if current.Genre == "" {
				current.Genre = genre
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
		register(
			record.Track, record.Genre, record.TrackNumber, record.DiscNumber, record.ResolvedTrackID, record.ID, 0,
		)
	}
	for _, event := range detail.FavoriteEvents {
		register(event.Track, "", event.TrackNumber, event.DiscNumber, event.ResolvedTrackID, 0, event.ID)
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
	for _, genre := range release.Genres {
		if strings.TrimSpace(genre.Name) == "" {
			continue
		}
		return model.NormalizeGenre(nil, genre.Name)
	}
	return ""
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

	material, err := s.resolveManualMaterial(ctx, detail, input)
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
		musicbrainzws2.IncludesFilter{
			Includes: []string{
				"recordings", "media", "artist-credits", "genres", "release-groups",
			},
		},
	)
	if err != nil {
		return nil, err
	}

	totalDiscs, discInfos := buildAlbumDiscInfosFromRelease(release)
	releaseDate := release.Date.String()
	originalReleaseDate := extractPendingAlbumOriginalReleaseDate(release)
	genreStr := extractReleaseGenres(release)
	cleanedAlbum, rt := common.ParseAlbumTitleAndReleaseType(detail.WorkItem.Album)
	mbReleaseType := extractReleaseTypeFromRelease(release)
	evidenceReleaseType := extractPendingEvidenceReleaseType(detail)

	finalReleaseType := rt
	if finalReleaseType == "" {
		finalReleaseType = mbReleaseType
	}
	if finalReleaseType == "" {
		finalReleaseType = evidenceReleaseType
	}
	if finalReleaseType == "" {
		_, mbTitleRT := common.ParseAlbumTitleAndReleaseType(release.Title)
		finalReleaseType = mbTitleRT
	}
	if finalReleaseType == "" {
		finalReleaseType = "album"
	}

	material := &pendingAlbumMaintenanceMaterial{
		Mode: pendingAlbumMaintenanceModeMusicBrainz,
		AlbumCandidate: &model.Album{
			Name:                cleanedAlbum,
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
			ReleaseType:         finalReleaseType,
		},
		CoverEnsureInput: artworklogic.EnsureAlbumCoverInput{
			AlbumArtist: detail.WorkItem.AlbumArtist,
			Artist:      detail.WorkItem.Artist,
			Album:       cleanedAlbum,
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
	ctx context.Context, detail *model.PendingAlbumWorkItemDetail, input ManualPendingAlbumInput,
) (*pendingAlbumMaintenanceMaterial, error) {
	if detail == nil || detail.WorkItem == nil {
		return nil, errors.New("work item detail is nil")
	}

	rawAlbumName := strings.TrimSpace(input.ManualAlbum.Name)
	if rawAlbumName == "" {
		return nil, errors.New("manual_album.name is required")
	}
	albumName, parsedRT := common.ParseAlbumTitleAndReleaseType(rawAlbumName)
	if albumName == "" {
		albumName = rawAlbumName
	}
	albumArtist := strings.TrimSpace(input.ManualAlbum.AlbumArtist)
	if albumArtist == "" {
		return nil, errors.New("manual_album.album_artist is required")
	}
	if len(input.ManualTracks) == 0 {
		return nil, errors.New("manual_tracks is required")
	}

	candidateSubtitle := strings.TrimSpace(input.ManualAlbum.AlbumSubtitle)
	if candidateSubtitle == "" && detail.WorkItem != nil {
		candidateSubtitle = strings.TrimSpace(detail.WorkItem.AlbumSubtitle)
	}
	if candidateSubtitle == "" {
		_, sub := common.ParseAlbumTitleAndSubtitle(rawAlbumName)
		candidateSubtitle = strings.TrimSpace(sub)
	}

	candidateReleaseType := strings.TrimSpace(input.ManualAlbum.ReleaseType)
	if candidateReleaseType == "" {
		candidateReleaseType = strings.TrimSpace(parsedRT)
	}
	if candidateReleaseType == "" && detail.WorkItem != nil {
		_, itemRT := common.ParseAlbumTitleAndReleaseType(detail.WorkItem.Album)
		candidateReleaseType = strings.TrimSpace(itemRT)
	}
	if candidateReleaseType == "" {
		candidateReleaseType = extractPendingEvidenceReleaseType(detail)
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
			discNumber = 1
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

	var releaseMB *model.ReleaseMB
	targetMBID := strings.TrimSpace(input.MBID)
	if targetMBID == "" && detail.WorkItem != nil {
		targetMBID = strings.TrimSpace(detail.WorkItem.SelectedMBID)
	}
	if targetMBID != "" {
		rel, lookupErr := coremusicbrainz.LookupRelease(
			ctx,
			mbtypes.MBID(targetMBID),
			musicbrainzws2.IncludesFilter{
				Includes: []string{
					"recordings", "media", "artist-credits", "genres", "release-groups",
				},
			},
		)
		if lookupErr == nil {
			if candidateReleaseType == "" {
				candidateReleaseType = extractReleaseTypeFromRelease(rel)
			}
			releaseRaw, _ := json.Marshal(rel)
			releaseMB = &model.ReleaseMB{
				MBID:     targetMBID,
				AlbumID:  0,
				Name:     rel.Title,
				JSONData: string(releaseRaw),
			}
		} else {
			log.Warn(
				ctx, "Manual maintenance lookup MB release failed, using fallback", zap.String("mbid", targetMBID),
				zap.Error(lookupErr),
			)
			if existingReleaseMB, getErr := model.GetReleaseMBByMBID(
				ctx, 0, targetMBID,
			); getErr == nil && existingReleaseMB.ID > 0 {
				releaseMB = existingReleaseMB
			} else {
				releaseMB = &model.ReleaseMB{
					MBID:     targetMBID,
					AlbumID:  0,
					Name:     albumName,
					JSONData: "{}",
				}
			}
		}
	}

	if candidateReleaseType == "" {
		candidateReleaseType = "album"
	}

	totalDiscs, discInfos := buildAlbumDiscInfosFromTrackDrafts(trackDrafts)
	return &pendingAlbumMaintenanceMaterial{
		Mode: pendingAlbumMaintenanceModeManual,
		AlbumCandidate: &model.Album{
			Name:                albumName,
			NameSubtitle:        candidateSubtitle,
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
			ReleaseType:         candidateReleaseType,
		},
		TrackDrafts: trackDrafts,
		ReleaseMB:   releaseMB,
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

	if coverErr := ensurePendingAlbumCover(
		ctx, detail, report.ResolvedAlbumID, material.CoverEnsureInput,
	); coverErr != nil {
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

	if report.ResolvedAlbumID > 0 {
		if reconcileErr := model.ReconcileAlbumPlayCounts(ctx, report.ResolvedAlbumID); reconcileErr != nil {
			log.Warn(
				ctx,
				"待归因专辑维护后播放数对账失败",
				zap.Int64("album_id", report.ResolvedAlbumID),
				zap.Error(reconcileErr),
			)
		}
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
			if nameSubtitle := strings.TrimSpace(material.AlbumCandidate.NameSubtitle); nameSubtitle != "" {
				fields["name_subtitle"] = nameSubtitle
			}
			if releaseType := strings.TrimSpace(material.AlbumCandidate.ReleaseType); releaseType != "" {
				fields["release_type"] = releaseType
			} else {
				fields["release_type"] = "album"
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

func (s *serviceImpl) PreviewAlbumMBMaintenance(
	ctx context.Context, albumID, releaseMBID int64, mbid string, forceRefresh bool,
) (*PendingAlbumMaintenancePreview, error) {
	log.Info(
		ctx, "生成/查看已生成专辑的 MB 维护草稿与预审 Preview", zap.Int64("album_id", albumID),
		zap.String("mbid", mbid), zap.Bool("force_refresh", forceRefresh),
	)
	albumDetail, err := model.GetAlbumWithTracks(ctx, albumID)
	if err != nil {
		return nil, fmt.Errorf("album not found: %w", err)
	}
	if albumDetail == nil || albumDetail.Album.ID <= 0 {
		return nil, errors.New("album not found")
	}

	targetMBID := strings.TrimSpace(mbid)
	if targetMBID == "" && albumDetail.ReleaseMB != nil {
		targetMBID = strings.TrimSpace(albumDetail.ReleaseMB.MBID)
	}

	if !forceRefresh {
		if workItem, _ := model.GetPendingAlbumWorkItemByResolvedAlbumID(
			ctx, albumID,
		); workItem != nil && strings.TrimSpace(workItem.StagingDraftJSON) != "" {
			if targetMBID == "" || targetMBID == workItem.SelectedMBID {
				var cachedPreview PendingAlbumMaintenancePreview
				if unmarshalErr := json.Unmarshal(
					[]byte(workItem.StagingDraftJSON), &cachedPreview,
				); unmarshalErr == nil {
					log.Info(ctx, "优先使用正式专辑已存的精选维护草稿快照", zap.Int64("album_id", albumID))
					return &cachedPreview, nil
				}
			}
		}
	}

	if targetMBID == "" {
		return nil, errors.New("请先在 MusicBrainz 候选列表中选定一个版本")
	}

	rel, lookupErr := coremusicbrainz.LookupRelease(
		ctx, mbtypes.MBID(targetMBID), musicbrainzws2.IncludesFilter{
			Includes: []string{"recordings", "media", "artist-credits", "genres", "release-groups"},
		},
	)
	if lookupErr != nil {
		return nil, fmt.Errorf("lookup musicbrainz release failed: %w", lookupErr)
	}

	releaseDate := strings.TrimSpace(rel.Date.String())
	if releaseDate == "" {
		releaseDate = albumDetail.Album.ReleaseDate
	}
	origReleaseDate := extractPendingAlbumOriginalReleaseDate(rel)
	if origReleaseDate == "" {
		origReleaseDate = albumDetail.Album.OriginalReleaseDate
	}
	mbGenre := ""
	if len(rel.Genres) > 0 {
		mbGenre = model.NormalizeGenre(nil, rel.Genres[0].Name)
	}
	albumGenre := mbGenre
	if albumGenre == "" {
		albumGenre = albumDetail.Album.Genre
	}

	cleanedMBTitle, parsedMBRT := common.ParseAlbumTitleAndReleaseType(rel.Title)
	if cleanedMBTitle == "" {
		cleanedMBTitle = rel.Title
	}
	mbReleaseType := extractReleaseTypeFromRelease(rel)
	evidenceReleaseType := albumDetail.Album.ReleaseType
	releaseType := albumDetail.Album.ReleaseType
	if releaseType == "" {
		releaseType = mbReleaseType
	}
	if releaseType == "" {
		releaseType = parsedMBRT
	}
	if releaseType == "" {
		releaseType = "album"
	}

	albumPreview := PendingAlbumAlbumPreview{
		Name:                cleanedMBTitle,
		AlbumSubtitle:       albumDetail.Album.NameSubtitle,
		ReleaseType:         releaseType,
		EvidenceReleaseType: evidenceReleaseType,
		MBReleaseType:       mbReleaseType,
		AlbumArtist:         albumDetail.Album.Artist,
		DisplayArtist:       albumDetail.Album.Artist,
		ReleaseDate:         releaseDate,
		OriginalReleaseDate: origReleaseDate,
		Genre:               albumGenre,
		EvidenceGenre:       albumDetail.Album.Genre,
		Country:             string(rel.CountryCode),
		Status:              rel.Status,
		Packaging:           rel.Packaging,
		Barcode:             string(rel.Barcode),
	}

	dbTrackMapByPosition := make(map[string]*model.Track)
	dbTrackMapByTitle := make(map[string]*model.Track)
	for _, ta := range albumDetail.TrackAlbums {
		key := fmt.Sprintf("%d|%d", ta.DiscNumber, ta.TrackNumber)
		var matchedTrack *model.Track
		for _, t := range albumDetail.Tracks {
			if t.ID == ta.TrackID {
				matchedTrack = t
				break
			}
		}
		if matchedTrack != nil {
			if key != "" {
				dbTrackMapByPosition[key] = matchedTrack
			}
			titleKey := normalizeTrackLookupKey(matchedTrack.Track)
			if titleKey != "" {
				dbTrackMapByTitle[titleKey] = matchedTrack
			}
		}
	}

	diffCount := 0
	var trackPreviews []PendingAlbumTrackPreview
	var suggestedManualTracks []ManualPendingAlbumTrackInput

	discNum := int8(1)
	trackNum := int8(1)
	for _, media := range rel.Media {
		if media.Position > 0 {
			discNum = int8(media.Position)
		}
		for _, track := range media.Tracks {
			if track.Position > 0 {
				trackNum = int8(track.Position)
			} else {
				trackNum++
			}
			mbTitle := coremusicbrainz.TrackTitleWithFeat(track)
			posKey := fmt.Sprintf("%d|%d", discNum, trackNum)
			titleKey := normalizeTrackLookupKey(mbTitle)

			existingTrack, ok := dbTrackMapByPosition[posKey]
			if !ok {
				existingTrack, ok = dbTrackMapByTitle[titleKey]
			}

			evidenceTitle := ""
			evidenceGenreTrack := ""
			if existingTrack != nil {
				evidenceTitle = existingTrack.Track
				evidenceGenreTrack = existingTrack.Genre
			}
			if evidenceGenreTrack == "" {
				evidenceGenreTrack = albumDetail.Album.Genre
			}

			hasDiff := evidenceTitle != "" && normalizeTrackLookupKey(evidenceTitle) != normalizeTrackLookupKey(mbTitle)
			if hasDiff {
				diffCount++
			}

			chosenTitle := mbTitle
			mbidStr := string(track.ID)
			if string(track.Recording.ID) != "" {
				mbidStr = string(track.Recording.ID)
			}

			trackPrev := PendingAlbumTrackPreview{
				DiscNumber:     discNum,
				TrackNumber:    trackNum,
				Title:          chosenTitle,
				Artist:         albumDetail.Album.Artist,
				Genre:          evidenceGenreTrack,
				EvidenceGenre:  evidenceGenreTrack,
				Duration:       int64(track.Length.Seconds()),
				MusicBrainzID:  mbidStr,
				EvidenceTitles: []string{evidenceTitle},
				MBTitle:        mbTitle,
				EvidenceTitle:  evidenceTitle,
				HasDiff:        hasDiff,
			}
			trackPreviews = append(trackPreviews, trackPrev)

			suggestedManualTracks = append(
				suggestedManualTracks, ManualPendingAlbumTrackInput{
					DiscNumber:     discNum,
					TrackNumber:    trackNum,
					Title:          chosenTitle,
					Artist:         albumDetail.Album.Artist,
					Genre:          evidenceGenreTrack,
					Duration:       int64(track.Length.Seconds()),
					MusicBrainzID:  mbidStr,
					EvidenceTitles: []string{evidenceTitle},
				},
			)
		}
	}

	return &PendingAlbumMaintenancePreview{
		WorkItemID:     0,
		ReleaseMBID:    releaseMBID,
		MBID:           targetMBID,
		AlbumPreview:   albumPreview,
		TrackPreviews:  trackPreviews,
		DiffTrackCount: diffCount,
		SuggestedInput: ManualPendingAlbumInput{
			ManualAlbum: ManualPendingAlbumAlbumInput{
				Name:                albumPreview.Name,
				AlbumSubtitle:       albumPreview.AlbumSubtitle,
				ReleaseType:         albumPreview.ReleaseType,
				AlbumArtist:         albumPreview.AlbumArtist,
				DisplayArtist:       albumPreview.DisplayArtist,
				ReleaseDate:         albumPreview.ReleaseDate,
				OriginalReleaseDate: albumPreview.OriginalReleaseDate,
				Genre:               albumPreview.Genre,
				Country:             albumPreview.Country,
				Status:              albumPreview.Status,
				Packaging:           albumPreview.Packaging,
				Barcode:             albumPreview.Barcode,
				CoverArtURL:         albumPreview.CoverArtURL,
			},
			ManualTracks: suggestedManualTracks,
		},
	}, nil
}

func (s *serviceImpl) ApplyAlbumMBMaintenance(
	ctx context.Context, albumID int64, input *ManualPendingAlbumInput,
) error {
	if albumID <= 0 {
		return errors.New("invalid album id")
	}
	if input == nil {
		return errors.New("input payload cannot be nil")
	}

	albumDetail, err := model.GetAlbumWithTracks(ctx, albumID)
	if err != nil || albumDetail == nil || albumDetail.Album.ID <= 0 {
		return errors.New("album not found")
	}

	return model.InTx(
		ctx, func(tx *gorm.DB) error {
			rawAlbumName := strings.TrimSpace(input.ManualAlbum.Name)
			cleanedAlbumName, parsedRT := common.ParseAlbumTitleAndReleaseType(rawAlbumName)
			if cleanedAlbumName == "" {
				cleanedAlbumName = rawAlbumName
			}
			candidateReleaseType := strings.TrimSpace(input.ManualAlbum.ReleaseType)
			if candidateReleaseType == "" {
				candidateReleaseType = strings.TrimSpace(parsedRT)
			}
			if candidateReleaseType == "" && albumDetail.Album.ReleaseType != "" {
				candidateReleaseType = albumDetail.Album.ReleaseType
			}

			candidateSubtitle := strings.TrimSpace(input.ManualAlbum.AlbumSubtitle)
			if candidateSubtitle == "" && albumDetail.Album.NameSubtitle != "" {
				candidateSubtitle = albumDetail.Album.NameSubtitle
			}

			targetMBID := strings.TrimSpace(input.MBID)
			if targetMBID == "" && albumDetail.ReleaseMB != nil {
				targetMBID = strings.TrimSpace(albumDetail.ReleaseMB.MBID)
			}
			if targetMBID != "" {
				rel, lookupErr := coremusicbrainz.LookupRelease(
					ctx,
					mbtypes.MBID(targetMBID),
					musicbrainzws2.IncludesFilter{
						Includes: []string{
							"recordings", "media", "artist-credits", "genres", "release-groups",
						},
					},
				)
				var releaseMB *model.ReleaseMB
				if lookupErr == nil {
					if candidateReleaseType == "" {
						candidateReleaseType = extractReleaseTypeFromRelease(rel)
					}
					releaseRaw, _ := json.Marshal(rel)
					releaseMB = &model.ReleaseMB{
						MBID:     targetMBID,
						AlbumID:  albumID,
						Name:     rel.Title,
						JSONData: string(releaseRaw),
					}
				} else {
					log.Warn(
						ctx, "Apply maintenance lookup MB release failed, using fallback",
						zap.String("mbid", targetMBID), zap.Error(lookupErr),
					)
					if existingReleaseMB, getErr := model.GetReleaseMBByMBID(
						ctx, albumID, targetMBID,
					); getErr == nil && existingReleaseMB.ID > 0 {
						releaseMB = existingReleaseMB
					} else {
						releaseMB = &model.ReleaseMB{
							MBID:     targetMBID,
							AlbumID:  albumID,
							Name:     cleanedAlbumName,
							JSONData: "{}",
						}
					}
				}

				if err := model.SaveReleaseMBTx(tx, releaseMB); err != nil {
					return fmt.Errorf("save release mb tx error: %w", err)
				}
				if err := model.LinkAlbumToMBIDTx(tx, albumID, releaseMB.ID, targetMBID); err != nil {
					return fmt.Errorf("link album to mbid tx error: %w", err)
				}
			}

			if candidateReleaseType == "" {
				candidateReleaseType = "album"
			}

			fields := map[string]interface{}{
				"name":                  cleanedAlbumName,
				"name_subtitle":         candidateSubtitle,
				"release_type":          candidateReleaseType,
				"artist":                strings.TrimSpace(input.ManualAlbum.AlbumArtist),
				"genre":                 strings.TrimSpace(input.ManualAlbum.Genre),
				"release_date":          strings.TrimSpace(input.ManualAlbum.ReleaseDate),
				"original_release_date": strings.TrimSpace(input.ManualAlbum.OriginalReleaseDate),
				"country":               strings.TrimSpace(input.ManualAlbum.Country),
				"sync_status":           3, // SyncStatusCuratedMaintenanceCompleted
			}
			if updateErr := model.UpdateAlbumFieldsTx(tx, albumID, fields); updateErr != nil {
				return fmt.Errorf("update album fields tx error: %w", updateErr)
			}

			for idx, manualTrack := range input.ManualTracks {
				discNum := manualTrack.DiscNumber
				if discNum <= 0 {
					discNum = 1
				}
				trackNum := manualTrack.TrackNumber
				if trackNum <= 0 {
					trackNum = int8(idx + 1)
				}
				title := strings.TrimSpace(common.UnityFixAll(manualTrack.Title))
				genre := strings.TrimSpace(manualTrack.Genre)
				if genre == "" {
					genre = strings.TrimSpace(input.ManualAlbum.Genre)
				}

				var targetTrackID int64
				for _, ta := range albumDetail.TrackAlbums {
					if ta.DiscNumber == discNum && ta.TrackNumber == trackNum {
						targetTrackID = ta.TrackID
						break
					}
				}
				if targetTrackID <= 0 {
					for _, t := range albumDetail.Tracks {
						if normalizeTrackLookupKey(t.Track) == normalizeTrackLookupKey(title) {
							targetTrackID = t.ID
							break
						}
					}
				}

				if targetTrackID > 0 {
					trackFields := map[string]interface{}{
						"track":           title,
						"artist":          strings.TrimSpace(manualTrack.Artist),
						"genre":           genre,
						"music_brainz_id": manualTrack.MusicBrainzID,
					}
					if manualTrack.Duration > 0 {
						trackFields["duration"] = manualTrack.Duration
					}
					if err := tx.Model(&model.Track{}).Where(
						"id = ?", targetTrackID,
					).Updates(trackFields).Error; err != nil {
						return fmt.Errorf("update track #%d error: %w", targetTrackID, err)
					}
				} else {
					newTrack := model.Track{
						Artist:        strings.TrimSpace(manualTrack.Artist),
						AlbumArtist:   strings.TrimSpace(input.ManualAlbum.AlbumArtist),
						Track:         title,
						Album:         strings.TrimSpace(input.ManualAlbum.Name),
						Genre:         genre,
						Duration:      manualTrack.Duration,
						MusicBrainzID: manualTrack.MusicBrainzID,
					}
					if err := tx.Create(&newTrack).Error; err != nil {
						return fmt.Errorf("create new track error: %w", err)
					}
					targetTrackID = newTrack.ID
				}

				var ta model.TrackAlbum
				if err := tx.Where(
					"album_id = ? AND track_id = ?", albumID, targetTrackID,
				).First(&ta).Error; err == nil {
					tx.Model(&ta).Updates(
						map[string]interface{}{
							"disc_number":     discNum,
							"track_number":    trackNum,
							"track":           title,
							"mb_recording_id": manualTrack.MusicBrainzID,
						},
					)
				} else {
					newTA := model.TrackAlbum{
						TrackID:                targetTrackID,
						AlbumID:                albumID,
						DiscNumber:             discNum,
						TrackNumber:            trackNum,
						Track:                  title,
						MusicBrainzRecordingID: manualTrack.MusicBrainzID,
					}
					_ = tx.Create(&newTA).Error
				}
			}

			if err := model.ReconcileAlbumPlayCountsTx(tx, albumID); err != nil {
				log.Warn(ctx, "精选维护正式专辑后播放数对账失败", zap.Int64("album_id", albumID), zap.Error(err))
			}

			log.Info(ctx, "精选维护正式专辑落库成功", zap.Int64("album_id", albumID))
			return nil
		},
	)
}

func (s *serviceImpl) SaveAlbumStagingDraft(
	ctx context.Context, albumID int64, draft *PendingAlbumMaintenancePreview,
) error {
	if albumID <= 0 {
		return errors.New("invalid album id")
	}
	if draft == nil {
		return errors.New("draft payload cannot be nil")
	}
	raw, err := json.Marshal(draft)
	if err != nil {
		return fmt.Errorf("marshal album draft json error: %w", err)
	}
	return model.SaveAlbumStagingDraftDB(ctx, albumID, draft.ReleaseMBID, draft.MBID, string(raw))
}
