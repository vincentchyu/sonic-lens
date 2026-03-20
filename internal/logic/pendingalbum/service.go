package pendingalbum

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go.uploadedlobster.com/mbtypes"
	"go.uploadedlobster.com/musicbrainzws2"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/common"
	coremusicbrainz "github.com/vincentchyu/sonic-lens/core/musicbrainz"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

// DeepMaintainPendingAlbumReport 返回工作项执行后的汇总结果。
type DeepMaintainPendingAlbumReport struct {
	ResolvedAlbumID       int64 `json:"resolved_album_id"`
	ReusedHeardTracks     int   `json:"reused_heard_tracks"`
	CreatedTracks         int   `json:"created_tracks"`
	TrackAlbumWrites      int   `json:"track_album_writes"`
	AppliedPlayRecords    int   `json:"applied_play_records"`
	AppliedFavoriteEvents int   `json:"applied_favorite_events"`
}

// Service 定义待归因专辑工作台能力。
type Service interface {
	GetPendingAlbumGroups(ctx context.Context, limit int) ([]*model.PendingAlbumGroup, error)
	CreateOrGetPendingAlbumWorkItem(ctx context.Context, identityKey string) (*model.PendingAlbumWorkItem, error)
	GetPendingAlbumWorkItemDetail(ctx context.Context, workItemID int64) (*model.PendingAlbumWorkItemDetail, error)
	SearchPendingAlbumMBReleases(ctx context.Context, workItemID int64) ([]*model.ReleaseMB, error)
	LinkPendingAlbumMBRelease(ctx context.Context, workItemID, releaseMBID int64, mbid string) error
	DeepMaintainPendingAlbumWorkItem(ctx context.Context, workItemID int64) (*DeepMaintainPendingAlbumReport, error)
	ListWorkItems(ctx context.Context, limit, offset int, keyword string, statusGroup string) ([]*model.PendingAlbumWorkItem, int64, error)
}

type serviceImpl struct{}

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
	item, err := model.GetPendingAlbumWorkItemByID(ctx, workItemID)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(
		"release:\"%s\" AND artist:\"%s\"",
		escapeLucene(item.Album),
		escapeLucene(pendingAlbumOwner(item)),
	)
	client := coremusicbrainz.GetClient()
	searchRes, err := client.SearchReleases(
		ctx,
		musicbrainzws2.SearchFilter{Query: query},
		musicbrainzws2.Paginator{Limit: 10},
	)
	if err != nil {
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
	return results, nil
}

func (s *serviceImpl) LinkPendingAlbumMBRelease(ctx context.Context, workItemID, releaseMBID int64, mbid string) error {
	return model.UpdatePendingAlbumWorkItemSelection(ctx, workItemID, releaseMBID, strings.TrimSpace(mbid))
}

type pendingEvidence struct {
	Title           string
	TrackNumber     int8
	DiscNumber      int8
	ResolvedTrackID int64
}

func buildPendingEvidence(detail *model.PendingAlbumWorkItemDetail) (
	map[string]pendingEvidence, map[string]pendingEvidence,
) {
	byPosition := make(map[string]pendingEvidence)
	byTitle := make(map[string]pendingEvidence)

	register := func(title string, trackNumber, discNumber int8, resolvedTrackID int64) {
		evidence := pendingEvidence{
			Title:           title,
			TrackNumber:     trackNumber,
			DiscNumber:      discNumber,
			ResolvedTrackID: resolvedTrackID,
		}
		if trackNumber > 0 {
			if discNumber <= 0 {
				discNumber = 1
			}
			key := fmt.Sprintf("%d|%d", discNumber, trackNumber)
			if current, ok := byPosition[key]; !ok || current.ResolvedTrackID == 0 {
				byPosition[key] = evidence
			}
		}
		if key := normalizeTrackLookupKey(title); key != "" {
			if current, ok := byTitle[key]; !ok || current.ResolvedTrackID == 0 {
				byTitle[key] = evidence
			}
		}
	}

	for _, record := range detail.PlayRecords {
		register(record.Track, record.TrackNumber, record.DiscNumber, record.ResolvedTrackID)
	}
	for _, event := range detail.FavoriteEvents {
		register(event.Track, event.TrackNumber, event.DiscNumber, event.ResolvedTrackID)
	}
	return byPosition, byTitle
}

func chooseEvidenceForMBTrack(
	mbTitle string,
	trackNumber, discNumber int8,
	byPosition map[string]pendingEvidence,
	byTitle map[string]pendingEvidence,
) (pendingEvidence, bool) {
	posKey := fmt.Sprintf("%d|%d", discNumber, trackNumber)
	if evidence, ok := byPosition[posKey]; ok {
		return evidence, true
	}
	if evidence, ok := byTitle[normalizeTrackLookupKey(mbTitle)]; ok {
		return evidence, true
	}
	return pendingEvidence{}, false
}

func buildAlbumDiscInfos(release musicbrainzws2.Release) (int, string) {
	totalDiscs := len(release.Media)
	discInfosMap := make(map[int]int, totalDiscs)
	for _, medium := range release.Media {
		discInfosMap[medium.Position] = medium.TrackCount
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
	detail, err := model.GetPendingAlbumWorkItemDetail(ctx, workItemID)
	if err != nil {
		return nil, err
	}
	if detail.WorkItem == nil || strings.TrimSpace(detail.WorkItem.SelectedMBID) == "" {
		return nil, errors.New("work item has no selected mbid")
	}

	if err := model.UpdatePendingAlbumWorkItemProgress(
		ctx,
		workItemID,
		model.PendingAlbumWorkItemStatusDeepMaintaning,
		detail.WorkItem.ResolvedAlbumID,
		"",
	); err != nil {
		return nil, err
	}

	client := coremusicbrainz.GetClient()
	release, err := client.LookupRelease(
		ctx,
		mbtypes.MBID(detail.WorkItem.SelectedMBID),
		musicbrainzws2.IncludesFilter{Includes: []string{"recordings", "media", "artist-credits", "genres"}},
	)
	if err != nil {
		_ = model.UpdatePendingAlbumWorkItemProgress(
			ctx, workItemID, model.PendingAlbumWorkItemStatusFailed, 0, err.Error(),
		)
		return nil, err
	}

	byPosition, byTitle := buildPendingEvidence(detail)
	report := &DeepMaintainPendingAlbumReport{}

	err = model.InTx(
		ctx,
		func(tx *gorm.DB) error {
			totalDiscs, discInfos := buildAlbumDiscInfos(release)
			releaseDate := release.Date.String()
			genreStr := extractReleaseGenres(release)
			albumCandidate := &model.Album{
				Name:        detail.WorkItem.Album,
				Artist:      pendingAlbumOwner(detail.WorkItem),
				ReleaseDate: releaseDate,
				Genre:       genreStr,
				Country:     string(release.CountryCode),
				Status:      release.Status,
				Packaging:   release.Packaging,
				Barcode:     string(release.Barcode),
				TotalDiscs:  totalDiscs,
				DiscInfos:   discInfos,
			}
			resolvedAlbum, resolveErr := model.ResolveCanonicalAlbumForPendingContextTx(tx, albumCandidate)
			if resolveErr != nil {
				return resolveErr
			}
			report.ResolvedAlbumID = resolvedAlbum.ID

			releaseRaw, _ := json.Marshal(release)
			releaseRow := &model.ReleaseMB{
				MBID:     detail.WorkItem.SelectedMBID,
				AlbumID:  resolvedAlbum.ID,
				Name:     release.Title,
				JSONData: string(releaseRaw),
			}
			if err := model.SaveReleaseMBTx(tx, releaseRow); err != nil {
				return err
			}
			if err := model.LinkAlbumToMBIDTx(
				tx, resolvedAlbum.ID, releaseRow.ID, detail.WorkItem.SelectedMBID,
			); err != nil {
				return err
			}

			for _, medium := range release.Media {
				for _, releaseTrack := range medium.Tracks {
					recordingID := string(releaseTrack.Recording.ID)
					discNumber := int8(medium.Position)
					trackNumber := int8(releaseTrack.Position)
					mbTitle := common.UnityFixAll(coremusicbrainz.TrackTitleWithFeat(releaseTrack))

					evidence, hasEvidence := chooseEvidenceForMBTrack(
						mbTitle, trackNumber, discNumber, byPosition, byTitle,
					)
					var trackObj *model.Track
					reusedTrack := false

					if hasEvidence && evidence.ResolvedTrackID > 0 {
						trackObj, err = model.GetTrackByIDTx(tx, evidence.ResolvedTrackID)
						if err == nil {
							reusedTrack = true
						} else if !errors.Is(err, gorm.ErrRecordNotFound) {
							return err
						}
					}
					if trackObj == nil && recordingID != "" {
						trackObj, err = model.GetTrackByMusicBrainzIDTx(tx, recordingID)
						if err == nil {
							reusedTrack = true
						} else if !errors.Is(err, gorm.ErrRecordNotFound) {
							return err
						}
					}

					duration := int64(releaseTrack.Length.Seconds())
					titleForTrack := mbTitle
					if hasEvidence && strings.TrimSpace(evidence.Title) != "" {
						titleForTrack = evidence.Title
					}
					if trackObj == nil {
						trackObj, err = model.GetOrCreateTrackByIdentityTx(
							tx,
							&model.Track{
								Artist:        detail.WorkItem.Artist,
								AlbumArtist:   pendingAlbumOwner(detail.WorkItem),
								Album:         resolvedAlbum.Name,
								Track:         titleForTrack,
								TrackNumber:   trackNumber,
								DiscNumber:    discNumber,
								Duration:      duration,
								Genre:         genreStr,
								ReleaseDate:   releaseDate,
								MusicBrainzID: recordingID,
								Source:        "PendingAlbumWorkItem",
							},
						)
						if err != nil {
							return err
						}
						if reusedTrack {
							report.ReusedHeardTracks++
						} else {
							report.CreatedTracks++
						}
					} else {
						report.ReusedHeardTracks++
					}

					if err := model.UpdateTrackMusicBrainzMetadataTx(
						tx,
						trackObj.ID,
						recordingID,
						discNumber,
						trackNumber,
						duration,
					); err != nil {
						return err
					}
					if err := model.UpsertTrackAlbumTx(
						tx,
						&model.TrackAlbum{
							TrackID:                trackObj.ID,
							AlbumID:                resolvedAlbum.ID,
							TrackNumber:            trackNumber,
							DiscNumber:             discNumber,
							Track:                  mbTitle,
							MusicBrainzRecordingID: recordingID,
						},
						true,
					); err != nil {
						return err
					}
					report.TrackAlbumWrites++
				}
			}

			return model.UpdateAlbumFieldsTx(
				tx,
				resolvedAlbum.ID,
				map[string]interface{}{
					"sync_status":  3,
					"release_date": releaseDate,
					"genre":        genreStr,
					"country":      string(release.CountryCode),
					"status":       release.Status,
					"packaging":    release.Packaging,
					"barcode":      string(release.Barcode),
					"total_discs":  totalDiscs,
					"disc_infos":   discInfos,
				},
			)
		},
	)
	if err != nil {
		_ = model.UpdatePendingAlbumWorkItemProgress(
			ctx, workItemID, model.PendingAlbumWorkItemStatusFailed, report.ResolvedAlbumID, err.Error(),
		)
		return nil, err
	}

	if err := model.UpdatePendingAlbumWorkItemProgress(
		ctx,
		workItemID,
		model.PendingAlbumWorkItemStatusApplying,
		report.ResolvedAlbumID,
		"",
	); err != nil {
		return nil, err
	}

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
		return nil, err
	}
	report.AppliedPlayRecords = len(replayReport.Results)

	favResult, err := model.ApplyTrackFavoriteEventsByIDs(ctx, favoriteEventIDs)
	if err != nil {
		_ = model.UpdatePendingAlbumWorkItemProgress(
			ctx, workItemID, model.PendingAlbumWorkItemStatusFailed, report.ResolvedAlbumID, err.Error(),
		)
		return nil, err
	}
	report.AppliedFavoriteEvents = favResult.AppliedCount

	if err := model.UpdatePendingAlbumWorkItemProgress(
		ctx,
		workItemID,
		model.PendingAlbumWorkItemStatusCompleted,
		report.ResolvedAlbumID,
		"",
	); err != nil {
		return nil, err
	}
	return report, nil
}
