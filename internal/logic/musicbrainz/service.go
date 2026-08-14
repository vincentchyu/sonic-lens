package musicbrainz

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
	"github.com/vincentchyu/sonic-lens/core/musicbrainz"
	artworklogic "github.com/vincentchyu/sonic-lens/internal/logic/artwork"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

type mbTrackInfo struct {
	musicbrainzws2.Track
	DiscNumber int8
}

// Service 定义 API 层使用的 MusicBrainz 业务入口，统一收口到 logic 层。
type Service interface {
	// SearchAndCacheReleases 搜索并缓存候选发行版。
	SearchAndCacheReleases(ctx context.Context, albumID int64) error
	// GetReleasesByAlbumID 获取某专辑对应的缓存候选结果。
	GetReleasesByAlbumID(ctx context.Context, albumID int64) ([]*model.ReleaseMB, error)
	// LinkAlbumToMBID 确认专辑与选定发行版的关联。
	LinkAlbumToMBID(ctx context.Context, albumID int64, releaseMBID int64, mbid string) error
	// DeepingMaintenance 执行精选维护与轨道修正。
	DeepingMaintenance(ctx context.Context, albumID int64) error
}

type serviceImpl struct{}

// NewService 创建 MusicBrainz 逻辑服务，供 API 层统一调用。
func NewService() Service {
	return &serviceImpl{}
}

// SearchAndCacheReleases 搜索并缓存候选发行版。
func (s *serviceImpl) SearchAndCacheReleases(ctx context.Context, albumID int64) error {
	return searchAndCacheReleases(ctx, albumID)
}

// GetReleasesByAlbumID 获取某专辑对应的缓存候选结果。
func (s *serviceImpl) GetReleasesByAlbumID(ctx context.Context, albumID int64) ([]*model.ReleaseMB, error) {
	return model.GetReleasesByAlbumID(ctx, albumID)
}

// LinkAlbumToMBID 确认专辑与选定发行版的关联。
func (s *serviceImpl) LinkAlbumToMBID(ctx context.Context, albumID int64, releaseMBID int64, mbid string) error {
	return model.LinkAlbumToMBID(ctx, albumID, releaseMBID, mbid)
}

// deepingMaintenance 执行精选维护与轨道修正。
func (s *serviceImpl) DeepingMaintenance(ctx context.Context, albumID int64) error {
	return deepingMaintenance(ctx, albumID)
}

func buildMBTrackPositionKey(discNumber, trackNumber int8) string {
	if trackNumber <= 0 {
		return ""
	}
	if discNumber <= 0 {
		discNumber = 1
	}
	return fmt.Sprintf("%d|%d", discNumber, trackNumber)
}

func normalizeMBTrackLookupKey(title string) string {
	title = strings.TrimSpace(common.UnityFixAll(title))
	if title == "" {
		return ""
	}
	return strings.ToLower(common.ConversionSimplifiedFx(title))
}

func extractReleaseGroupFirstReleaseDate(release musicbrainzws2.Release) string {
	if release.ReleaseGroup == nil {
		return ""
	}
	return strings.TrimSpace(release.ReleaseGroup.FirstReleaseDate.String())
}

func matchMBTrackByTitle(
	ta *model.TrackAlbum,
	trackObj *model.Track,
	mbTrackMapByName map[string][]mbTrackInfo,
	processedRecordingIDs map[string]bool,
) (mbTrackInfo, bool, bool, string) {
	type titleMatchResult struct {
		track   mbTrackInfo
		found   bool
		certain bool
		source  string
		key     string
	}

	best := titleMatchResult{}

	for _, candidate := range []struct {
		title  string
		source string
	}{
		{title: ta.Track, source: "track_album_title"},
		{title: trackObj.Track, source: "track_title"},
	} {
		key := normalizeMBTrackLookupKey(candidate.title)
		if key == "" {
			continue
		}

		infos, ok := mbTrackMapByName[key]
		if !ok || len(infos) == 0 {
			continue
		}

		current := titleMatchResult{
			found:  true,
			source: candidate.source,
			key:    key,
		}
		if len(infos) == 1 {
			current.track = infos[0]
			current.certain = true
		} else {
			current.track = infos[0]
			for _, info := range infos {
				if !processedRecordingIDs[string(info.Recording.ID)] {
					current.track = info
					break
				}
			}
		}

		if !best.found ||
			(current.certain && !best.certain) ||
			(current.certain == best.certain && len(current.key) > len(best.key)) {
			best = current
		}
	}

	if best.found {
		return best.track, true, best.certain, best.source
	}

	return mbTrackInfo{}, false, false, ""
}

func mbTrackMatchesAnyLocalTitle(mbTrack mbTrackInfo, titles ...string) bool {
	mbKey := normalizeMBTrackLookupKey(mbTrack.Title)
	if mbKey == "" {
		return false
	}

	for _, title := range titles {
		if normalizeMBTrackLookupKey(title) == mbKey {
			return true
		}
	}

	return false
}

func findMBTrackForHeardTrack(
	ta *model.TrackAlbum,
	trackObj *model.Track,
	mbTrackMapByPos map[string]mbTrackInfo,
	mbTrackMapByName map[string][]mbTrackInfo,
	processedRecordingIDs map[string]bool,
) (mbTrackInfo, bool, string) {
	titleMatch, titleFound, titleCertain, titleSource := matchMBTrackByTitle(
		ta,
		trackObj,
		mbTrackMapByName,
		processedRecordingIDs,
	)
	trackPosKey := buildMBTrackPositionKey(trackObj.DiscNumber, trackObj.TrackNumber)

	for _, candidate := range []struct {
		discNumber  int8
		trackNumber int8
		source      string
	}{
		{discNumber: ta.DiscNumber, trackNumber: ta.TrackNumber, source: "track_album_position"},
		{discNumber: trackObj.DiscNumber, trackNumber: trackObj.TrackNumber, source: "track_position"},
	} {
		posKey := buildMBTrackPositionKey(candidate.discNumber, candidate.trackNumber)
		if posKey == "" {
			continue
		}
		if mbTrack, found := mbTrackMapByPos[posKey]; found {
			if candidate.source == "track_album_position" && trackPosKey != "" && trackPosKey != posKey {
				return mbTrack, true, candidate.source
			}
			if candidate.source == "track_album_position" &&
				titleFound &&
				titleCertain &&
				titleSource == "track_title" &&
				normalizeMBTrackLookupKey(trackObj.Track) != "" &&
				normalizeMBTrackLookupKey(trackObj.Track) != normalizeMBTrackLookupKey(ta.Track) &&
				normalizeMBTrackLookupKey(trackObj.Track) != normalizeMBTrackLookupKey(mbTrack.Title) &&
				string(titleMatch.Recording.ID) != string(mbTrack.Recording.ID) {
				return titleMatch, true, titleSource + "_override_position"
			}
			if mbTrackMatchesAnyLocalTitle(mbTrack, ta.Track, trackObj.Track) {
				return mbTrack, true, candidate.source
			}
			if titleFound && titleCertain && string(titleMatch.Recording.ID) != string(mbTrack.Recording.ID) {
				return titleMatch, true, titleSource + "_override_position"
			}
			return mbTrack, true, candidate.source
		}
	}

	if titleFound {
		return titleMatch, true, titleSource
	}

	return mbTrackInfo{}, false, ""
}

/*
// InitializeAlbums from existing tracks 不是使用直接忽略
func InitializeAlbums(ctx context.Context) error {
	log.Info(ctx, "Starting InitializeAlbums from existing tracks")
	// 1. Get all tracks
	tracks, err := model.GetAllTrackPlayCounts(ctx)
	if err != nil {
		log.Error(ctx, "GetAllTrackPlayCounts failed", zap.Error(err))
		return err
	}
	ensuredAlbumCovers := make(map[int64]struct{})

	for _, t := range tracks {
		if t.Album == "" {
			continue
		}
		// 2. Create or Get Album
		album := &model.Album{
			Name:        t.Album,
			Artist:      t.AlbumArtist,
			ReleaseDate: t.ReleaseDate,
			Genre:       t.Genre,
		}
		if album.Artist == "" {
			album.Artist = t.Artist
		}

		if err := model.GetOrCreateAlbum(ctx, album); err != nil {
			log.Warn(ctx, "GetOrCreateAlbum failed", zap.String("album", album.Name), zap.Error(err))
			return err
		}
		if _, covered := ensuredAlbumCovers[album.ID]; !covered {
			if coverErr := artworklogic.EnsureAlbumCover(
				ctx,
				artworklogic.EnsureAlbumCoverInput{
					AlbumID:     album.ID,
					AlbumArtist: album.Artist,
					Artist:      t.Artist,
					Album:       album.Name,
				},
			); coverErr != nil {
				log.Warn(
					ctx,
					"InitializeAlbums ensure album cover err",
					zap.Int64("album_id", album.ID),
					zap.String("album", album.Name),
					zap.Error(coverErr),
				)
			} else {
				ensuredAlbumCovers[album.ID] = struct{}{}
			}
		}

		// 3. Link Track to Album
		ta := &model.TrackAlbum{
			TrackID:                t.ID,
			AlbumID:                album.ID,
			Track:                  t.Track,
			TrackNumber:            t.TrackNumber,
			DiscNumber:             t.DiscNumber,
			MusicBrainzRecordingID: t.MusicBrainzID,
		}
		if err := model.GetOrCreateTrackAlbum(ctx, ta); err != nil {
			log.Warn(
				ctx, "GetOrCreateTrackAlbum failed", zap.Int64("track_id", t.ID), zap.Int64("album_id", album.ID),
				zap.Error(err),
			)
			return err
		}
	}
	log.Info(ctx, "Successfully initialized albums", zap.Int("total_tracks", len(tracks)))
	return nil
}
*/
// escapeLucene escapes special characters in Lucene query syntax
func escapeLucene(in string) string {
	// 针对 MusicBrainz 主要是转义引号、反斜杠和其他 Lucene 特殊字符
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

// SearchMBReleases 统一的 MusicBrainz Release 搜索函数，支持两阶段检索（有发行格式限制时先精准匹配，无结果则宽松回退）。
// 该函数由 pendingalbum 和本包的 searchAndCacheReleases 共享，避免代码重复。
func SearchMBReleases(ctx context.Context, albumName, artistName, releaseType string) (*musicbrainzws2.SearchReleasesResult, error) {
	// 剥离/规整专辑名
	cleanedAlbum, rt := common.ParseAlbumTitleAndReleaseType(albumName)
	if releaseType == "" {
		releaseType = rt
	}

	escapedAlbum := escapeLucene(cleanedAlbum)
	escapedArtist := escapeLucene(artistName)

	var searchRes *musicbrainzws2.SearchReleasesResult

	// 两阶段检索策略：
	// 阶段一：若专辑有明确 release_type（ep/single/lp），先附加 primarytype 约束做精确检索。
	//         这样可以避免 EP "In The Sun" 匹配到同名全长专辑。
	// 阶段二：若阶段一无结果（或无 release_type），回退到不带类型约束的宽松检索。
	if releaseType != "" {
		mbPrimaryType := mbPrimaryTypeFromReleaseType(releaseType)
		queryWithType := fmt.Sprintf(
			"release:\"%s\" AND artist:\"%s\" AND primarytype:%s",
			escapedAlbum, escapedArtist, mbPrimaryType,
		)
		log.Info(ctx, "阶段一：带 primarytype 约束检索", zap.String("query", queryWithType))
		res, err := musicbrainz.SearchReleases(
			ctx, musicbrainzws2.SearchFilter{Query: queryWithType},
			musicbrainzws2.Paginator{Limit: 10},
		)
		if err == nil && len(res.Releases) > 0 {
			searchRes = &res
			log.Info(ctx, "阶段一检索命中", zap.Int("count", len(res.Releases)))
		} else {
			if err != nil {
				log.Warn(ctx, "阶段一 SearchReleases 失败，降级到宽松检索", zap.String("query", queryWithType), zap.Error(err))
			} else {
				log.Info(ctx, "阶段一无结果，降级到宽松检索", zap.String("release_type", releaseType))
			}
		}
	}

	// 阶段二：宽松回退（无 release_type 或阶段一无结果时执行）
	if searchRes == nil {
		query := fmt.Sprintf("release:\"%s\" AND artist:\"%s\"", escapedAlbum, escapedArtist)
		log.Info(ctx, "阶段二：宽松检索", zap.String("query", query))
		res, err := musicbrainz.SearchReleases(
			ctx, musicbrainzws2.SearchFilter{Query: query},
			musicbrainzws2.Paginator{Limit: 10},
		)
		if err != nil {
			log.Error(ctx, "阶段二 SearchReleases 失败", zap.String("query", query), zap.Error(err))
			return nil, err
		}
		searchRes = &res
	}

	return searchRes, nil
}

// searchAndCacheReleases searches for releases and saves them to release_mb
func searchAndCacheReleases(ctx context.Context, albumID int64) error {
	log.Info(ctx, "开始搜索并缓存 MusicBrainz 候选发行版", zap.Int64("album_id", albumID))
	album, err := model.GetAlbum(ctx, albumID)
	if err != nil {
		log.Error(ctx, "GetAlbum failed", zap.Int64("album_id", albumID), zap.Error(err))
		return err
	}

	// 如果状态为3（精选完成），需要清除之前的MB关联数据，重新开始
	if album.SyncStatus == 3 {
		if err := model.InTx(
			ctx, func(tx *gorm.DB) error {
				if err := model.DeleteAlbumReleaseMBByAlbumIDTx(tx, albumID); err != nil {
					return err
				}
				if err := model.ClearTrackAlbumMBRecordingIDByAlbumIDTx(tx, albumID); err != nil {
					return err
				}
				return model.UpdateAlbumSyncStatusTx(tx, albumID, 1)
			},
		); err != nil {
			log.Warn(ctx, "Reset album musicbrainz links failed", zap.Int64("album_id", albumID), zap.Error(err))
		}
		log.Info(ctx, "Reset album sync status from 3 to 1", zap.Int64("album_id", albumID))
	}

	log.Info(
		ctx, "开始检索 MusicBrainz 候选发行版", zap.Int64("album_id", albumID), zap.String("name", album.Name),
		zap.String("artist", album.Artist), zap.String("release_type", album.ReleaseType),
	)

	// 调用统一的 SearchMBReleases
	searchRes, err := SearchMBReleases(ctx, album.Name, album.Artist, album.ReleaseType)
	if err != nil {
		return err
	}

	for _, r := range searchRes.Releases {
		jsonData, _ := json.Marshal(r)
		rmb := &model.ReleaseMB{
			MBID:     string(r.ID),
			AlbumID:  albumID,
			Name:     r.Title,
			JSONData: string(jsonData),
		}
		if err := model.SaveReleaseMB(ctx, rmb); err != nil {
			log.Warn(ctx, "SaveReleaseMB failed", zap.String("mbid", rmb.MBID), zap.Error(err))
			return err
		}
	}

	// 更新专辑状态为初选进行中/完成（此处可根据业务定义，目前先标记为1表示已搜过候选）
	album.SyncStatus = 1
	if err := model.UpdateAlbumSyncStatus(ctx, albumID, 1); err != nil {
		log.Warn(ctx, "Update sync_status failed", zap.Int64("album_id", albumID), zap.Error(err))
	}

	log.Info(
		ctx, "MusicBrainz 候选发行版缓存完成", zap.Int64("album_id", albumID),
		zap.Int("count", len(searchRes.Releases)),
	)
	return nil
}

// mbPrimaryTypeFromReleaseType 将系统内部发行类型枚举（小写）映射为 MusicBrainz Lucene 查询语法
// 中的 primarytype 值（首字母大写）。
// MusicBrainz 支持的 primarytype 有：Album、Single、EP、Other、Broadcast 等。
func mbPrimaryTypeFromReleaseType(releaseType string) string {
	switch releaseType {
	case "ep":
		return "EP"
	case "single":
		return "Single"
	case "lp", "album":
		return "Album"
	default:
		// 未知类型直接原样首字母大写，让 MusicBrainz 自行过滤
		if len(releaseType) > 0 {
			return strings.ToUpper(releaseType[:1]) + releaseType[1:]
		}
		return releaseType
	}
}

// todo list
// media len:2, cap:2 意思是当前专辑有几张碟
// media.position 碟号是多少
// media.track-count 当前碟有个track
// media.track[0].position或者number 为当前track在这个碟中的序号
// 现在的深度维护不支持多张碟的情况
// album表也没有当前专辑有几张碟每张碟的分别的总track数字的记录
// track_album表也没有DiscNumber 只有TrackNumber
// 现在遇到的情况就是the wall 这张专辑在track_album 分别有 TrackNumber 1 1 两首以此类推其他的序号也是两个 深度维护应该按照歌曲名字 补充上DiscNumber
// 参考json在@internal/logic/musicbrainz/lookUpRelease.json

// deepingMaintenance performs a lookup and updates track numbers
func deepingMaintenance(ctx context.Context, albumID int64) error {
	log.Info(ctx, "开始执行 MusicBrainz 深度维护", zap.Int64("album_id", albumID))

	// 1. Get confirmed MBID
	link, err := model.GetAlbumReleaseMBByAlbumID(ctx, albumID)
	if err != nil {
		log.Error(ctx, "GetAlbumReleaseMBByAlbumID failed", zap.Int64("album_id", albumID), zap.Error(err))
		return err
	}
	albumObj, err := model.GetAlbum(ctx, albumID)
	if err != nil {
		log.Error(ctx, "GetAlbum failed", zap.Int64("album_id", albumID), zap.Error(err))
		return err
	}

	// 2. Lookup Release with details
	log.Info(ctx, "Fetching MB release details", zap.String("mbid", link.MBID))
	release, err := musicbrainz.LookupRelease(
		ctx, mbtypes.MBID(link.MBID), musicbrainzws2.IncludesFilter{
			Includes: []string{"recordings", "media", "artist-credits", "genres", "release-groups"},
		},
	)
	if err != nil {
		log.Error(ctx, "LookupRelease failed", zap.String("mbid", link.MBID), zap.Error(err))
		return err
	}

	// 3. 建立映射关系
	mbTrackMapByName := make(map[string][]mbTrackInfo) // 一个名字可能对应多个（多碟重复）
	mbTrackMapByPos := make(map[string]mbTrackInfo)    // 碟号|轨道号 -> 信息 (物理唯一)
	mbTracks := make([]mbTrackInfo, 0)

	totalDiscs := len(release.Media)
	discInfosMap := make(map[int]int)

	for _, medium := range release.Media {
		discInfosMap[medium.Position] = medium.TrackCount
		for _, t := range medium.Tracks {
			org := musicbrainz.TrackTitleWithFeat(t)
			title := common.UnityFixAll(org)
			key := normalizeMBTrackLookupKey(title)
			// 英文 将 Title 转为小写以支持大小写不敏感匹配 (兼容数据库 utf8mb4_unicode_ci)
			t.Title = title
			t.Recording.Title = title

			info := mbTrackInfo{
				Track:      t,
				DiscNumber: int8(medium.Position),
			}
			mbTrackMapByName[key] = append(mbTrackMapByName[key], info)
			mbTrackMapByPos[buildMBTrackPositionKey(info.DiscNumber, int8(info.Position))] = info
			mbTracks = append(mbTracks, info)
		}
	}
	discInfosBytes, _ := json.Marshal(discInfosMap)
	discInfosStr := string(discInfosBytes)
	releaseDate := release.Date.String()
	originalReleaseDate := extractReleaseGroupFirstReleaseDate(release)
	var genreStr string
	if len(release.Genres) > 0 {
		for _, g := range release.Genres {
			if clean := model.NormalizeGenre(nil, g.Name); clean != "" {
				genreStr = clean
				break
			}
		}
	}

	// 开启事务处理所有数据库操作
	err = model.InTx(
		ctx,
		func(tx *gorm.DB) error {
			// A. Update release_mb cache
			jsonData, _ := json.Marshal(release)
			updated, err := model.UpdateReleaseMBJSONDataTx(tx, albumID, link.MBID, string(jsonData))
			if err != nil {
				return err
			}
			if updated {
				log.Info(
					ctx, "Updated release_mb JSON cache", zap.Int64("album_id", albumID), zap.String("mbid", link.MBID),
				)
			}

			// B. 获取此专辑在本地已有的关联
			tas, err := model.GetTrackAlbumsByAlbumTx(tx, albumID)
			if err != nil {
				return err
			}

			processedRecordingIDs := make(map[string]bool)

			// C. 处理本地已听过的歌曲关联
			completedCount := 0
			for _, ta := range tas {
				if ta.TrackID == 0 {
					continue
				}
				trackObj, err := model.GetTrackByIDTx(tx, ta.TrackID)
				if err == nil {
					mbTrack, found, matchSource := findMBTrackForHeardTrack(
						ta,
						trackObj,
						mbTrackMapByPos,
						mbTrackMapByName,
						processedRecordingIDs,
					)

					if found {
						recordingID := string(mbTrack.Recording.ID)
						processedRecordingIDs[recordingID] = true

						log.Info(
							ctx, "Aligning heard track", zap.String("track", trackObj.Track),
							zap.String("recording_id", recordingID), zap.Int("pos", mbTrack.Position),
							zap.Int8("disc", mbTrack.DiscNumber), zap.String("match_source", matchSource),
						)

						// 更新本地 track 元数据
						if err := model.UpdateTrackMusicBrainzPositionTx(
							tx, trackObj.ID, recordingID, mbTrack.DiscNumber, int8(mbTrack.Position),
						); err != nil {
							return err
						}

						// 更新关联表，并自动回收同位置的占位符
						if err := model.UpsertTrackAlbumTx(
							tx, &model.TrackAlbum{
								TrackID:                ta.TrackID,
								AlbumID:                ta.AlbumID,
								TrackNumber:            int8(mbTrack.Position),
								DiscNumber:             mbTrack.DiscNumber,
								Track:                  mbTrack.Title,
								MusicBrainzRecordingID: recordingID,
							},
							false,
						); err != nil {
							return err
						}
						completedCount++
					}
				}
			}

			// D. 处理未听过的歌曲（创建真实 Track + TrackAlbum，播放次数保持 0）
			unheardLinkedCount := 0
			for _, mbTrack := range mbTracks {
				recordingID := string(mbTrack.Recording.ID)
				if processedRecordingIDs[recordingID] {
					continue
				}

				existingTA, err := model.GetTrackAlbumByAlbumAndRecordingIDTx(tx, albumID, recordingID)
				if err == nil {
					if existingTA.ID > 0 && existingTA.TrackID > 0 {
						if err := model.UpdateTrackMusicBrainzPositionTx(
							tx, existingTA.TrackID, recordingID, mbTrack.DiscNumber, int8(mbTrack.Position),
						); err != nil {
							return err
						}
						if err := model.UpsertTrackAlbumTx(
							tx, &model.TrackAlbum{
								TrackID:                existingTA.TrackID,
								AlbumID:                albumID,
								TrackNumber:            int8(mbTrack.Position),
								DiscNumber:             mbTrack.DiscNumber,
								Track:                  mbTrack.Title,
								MusicBrainzRecordingID: recordingID,
							},
							false,
						); err != nil {
							return err
						}
						processedRecordingIDs[recordingID] = true
						completedCount++
						continue
					} else if existingTA.ID > 0 && existingTA.TrackID == 0 { // 历史遗留解决数据=0的情况
						/*if err := model.UpsertTrackAlbumTx(
							tx, &model.TrackAlbum{
								ID:                     existingTA.ID,
								TrackID:                trackObj.ID,
								AlbumID:                albumID,
								TrackNumber:            int8(mbTrack.Position),
								DiscNumber:             mbTrack.DiscNumber,
								Track:                  mbTrack.Title,
								MusicBrainzRecordingID: recordingID,
							},
							false,
						); err != nil {
							return err
						}*/
						log.Warn(ctx, "ta 出现 track 0数据", zap.Any("ta", existingTA))
					}
				} else if !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}

				// 创建歌曲
				trackObj, err := model.GetOrCreateTrackByIdentityTx(
					tx,
					&model.Track{
						Artist:        albumObj.Artist,
						AlbumArtist:   albumObj.Artist,
						Album:         albumObj.Name,
						AlbumSubtitle: albumObj.NameSubtitle,
						Track:         mbTrack.Title,
						TrackNumber:   int8(mbTrack.Position),
						DiscNumber:    mbTrack.DiscNumber,
						Genre:         genreStr,
						ReleaseDate:   releaseDate,
						MusicBrainzID: recordingID,
						Source:        "MusicBrainz",
					},
				)
				if err != nil {
					return err
				}

				if err := model.UpdateTrackMusicBrainzPositionTx(
					tx, trackObj.ID, recordingID, mbTrack.DiscNumber, int8(mbTrack.Position),
				); err != nil {
					return err
				}
				if err := model.UpsertTrackAlbumTx(
					tx, &model.TrackAlbum{
						TrackID:                trackObj.ID,
						Track:                  mbTrack.Title,
						AlbumID:                albumID,
						TrackNumber:            int8(mbTrack.Position),
						DiscNumber:             mbTrack.DiscNumber,
						MusicBrainzRecordingID: recordingID,
					},
					true,
				); err != nil {
					return err
				}

				processedRecordingIDs[recordingID] = true
				unheardLinkedCount++
			}

			// E. 更新专辑状态及元数据
			updateFields := map[string]interface{}{
				"sync_status": 3,
				"total_discs": totalDiscs,
				"disc_infos":  discInfosStr,
			}
			if releaseDate != "" {
				updateFields["release_date"] = releaseDate
			}
			if originalReleaseDate != "" {
				updateFields["original_release_date"] = originalReleaseDate
			}
			if genreStr != "" {
				updateFields["genre"] = genreStr
			}
			if string(release.CountryCode) != "" {
				updateFields["country"] = string(release.CountryCode)
			}
			if release.Status != "" {
				updateFields["status"] = release.Status
			}
			if release.Packaging != "" {
				updateFields["packaging"] = release.Packaging
			}
			if string(release.Barcode) != "" {
				updateFields["barcode"] = string(release.Barcode)
			}

			if err := model.UpdateAlbumFieldsTx(tx, albumID, updateFields); err != nil {
				return err
			}

			log.Info(
				ctx, "deepingMaintenance transaction completed",
				zap.Int64("album_id", albumID),
				zap.Int("aligned_tracks", completedCount),
				zap.Int("linked_unheard_tracks", unheardLinkedCount),
			)

			return nil
		},
	)

	if err != nil {
		log.Error(ctx, "deepingMaintenance failed", zap.Int64("album_id", albumID), zap.Error(err))
		return err
	}
	log.Info(
		ctx,
		"MusicBrainz 深度维护完成",
		zap.Int64("album_id", albumID),
	)
	if coverErr := artworklogic.EnsureAlbumCover(
		ctx,
		artworklogic.EnsureAlbumCoverInput{
			AlbumID:     albumID,
			AlbumArtist: albumObj.Artist,
			Artist:      albumObj.Artist,
			Album:       albumObj.Name,
		},
	); coverErr != nil {
		log.Warn(
			ctx,
			"deepingMaintenance ensure album cover err",
			zap.Int64("album_id", albumID),
			zap.Error(coverErr),
		)
	}

	return nil
}
