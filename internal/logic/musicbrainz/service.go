package musicbrainz

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uploadedlobster.com/mbtypes"
	"go.uploadedlobster.com/musicbrainzws2"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/core/musicbrainz"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

// InitializeAlbums from existing tracks
func InitializeAlbums(ctx context.Context) error {
	log.Info(ctx, "Starting InitializeAlbums from existing tracks")
	// 1. Get all tracks
	tracks, err := model.GetAllTrackPlayCounts(ctx)
	if err != nil {
		log.Error(ctx, "GetAllTrackPlayCounts failed", zap.Error(err))
		return err
	}

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

		// 3. Link Track to Album
		ta := &model.TrackAlbum{
			TrackID:     t.ID,
			AlbumID:     album.ID,
			TrackNumber: t.TrackNumber,
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

// SearchAndCacheReleases searches for releases and saves them to release_mb
func SearchAndCacheReleases(ctx context.Context, albumID int64) error {
	album, err := model.GetAlbum(ctx, albumID)
	if err != nil {
		log.Error(ctx, "GetAlbum failed", zap.Int64("album_id", albumID), zap.Error(err))
		return err
	}

	log.Info(
		ctx, "Searching candidates for album", zap.Int64("album_id", albumID), zap.String("name", album.Name),
		zap.String("artist", album.Artist),
	)

	client := musicbrainz.GetClient()

	// Search - Escape names to avoid Lucene query errors (e.g. quotes in album names)
	escapedAlbum := escapeLucene(album.Name)
	escapedArtist := escapeLucene(album.Artist)
	query := fmt.Sprintf("release:\"%s\" AND artist:\"%s\"", escapedAlbum, escapedArtist)

	searchRes, err := client.SearchReleases(
		ctx, musicbrainzws2.SearchFilter{
			Query: query,
		}, musicbrainzws2.Paginator{Limit: 10},
	)
	if err != nil {
		log.Error(ctx, "SearchReleases failed", zap.String("query", query), zap.Error(err))
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
	if err := model.GetDB().WithContext(ctx).Model(album).Update("sync_status", 1).Error; err != nil {
		log.Warn(ctx, "Update sync_status failed", zap.Int64("album_id", albumID), zap.Error(err))
	}

	log.Info(
		ctx, "Successfully cached candidates", zap.Int64("album_id", albumID),
		zap.Int("count", len(searchRes.Releases)),
	)
	return nil
}

// DeepingMaintenance performs a lookup and updates track numbers
func DeepingMaintenance(ctx context.Context, albumID int64) error {
	log.Info(ctx, "Starting DeepingMaintenance", zap.Int64("album_id", albumID))

	// 1. Get confirmed MBID
	link, err := model.GetAlbumReleaseMBByAlbumID(ctx, albumID)
	if err != nil {
		log.Error(ctx, "GetAlbumReleaseMBByAlbumID failed", zap.Int64("album_id", albumID), zap.Error(err))
		return err
	}

	client := musicbrainz.GetClient()

	// 2. Lookup Release with details
	log.Info(ctx, "Fetching MB release details", zap.String("mbid", link.MBID))
	release, err := client.LookupRelease(
		ctx, mbtypes.MBID(link.MBID), musicbrainzws2.IncludesFilter{
			Includes: []string{"recordings", "media", "genres"},
		},
	)
	if err != nil {
		log.Error(ctx, "LookupRelease failed", zap.String("mbid", link.MBID), zap.Error(err))
		return err
	}

	// 3. 建立映射关系
	mbTrackMapByName := make(map[string]musicbrainzws2.Track)
	mbTracks := make([]musicbrainzws2.Track, 0)
	for _, medium := range release.Media {
		for _, t := range medium.Tracks {
			key := ""
			// 判断是否是是中文 中文转简体
			if common.IsExistsChineseSimplified(t.Title) {
				conversionSimplified := common.ConversionSimplifiedFx(t.Title)
				key = strings.ToLower(conversionSimplified)
			} else {
				key = strings.ToLower(t.Title)
			}
			// 英文 将 Title 转为小写以支持大小写不敏感匹配 (兼容数据库 utf8mb4_unicode_ci)
			t.Title = key
			t.Recording.Title = key
			mbTrackMapByName[key] = t
			mbTracks = append(mbTracks, t)
		}
	}

	// 开启事务处理所有数据库操作
	err = model.GetDB().WithContext(ctx).Transaction(
		func(tx *gorm.DB) error {
			// A. Update release_mb cache
			jsonData, _ := json.Marshal(release)
			var rmb model.ReleaseMB
			if err := tx.Where("album_id = ? AND mbid = ?", albumID, link.MBID).First(&rmb).Error; err == nil {
				rmb.JSONData = string(jsonData)
				if err := tx.Save(&rmb).Error; err != nil {
					return err
				}
				log.Info(
					ctx, "Updated release_mb JSON cache", zap.Int64("album_id", albumID), zap.String("mbid", link.MBID),
				)
			}

			// B. 获取此专辑在本地已有的关联
			var tas []*model.TrackAlbum
			if err := tx.Where("album_id = ?", albumID).Find(&tas).Error; err != nil {
				return err
			}

			processedRecordingIDs := make(map[string]bool)

			// C. 处理本地已听过的歌曲关联
			completedCount := 0
			for _, ta := range tas {
				if ta.TrackID == 0 {
					continue
				}
				var trackObj model.Track
				if err := tx.First(&trackObj, ta.TrackID).Error; err == nil {
					// 大小写不敏感匹配
					key := ""
					// 判断是否是是中文 中文转简体
					if common.IsExistsChineseSimplified(trackObj.Track) {
						conversionSimplified := common.ConversionSimplifiedFx(trackObj.Track)
						key = strings.ToLower(conversionSimplified)
					} else {
						key = strings.ToLower(trackObj.Track)
					}

					if mbTrack, ok := mbTrackMapByName[key]; ok {
						recordingID := string(mbTrack.Recording.ID)
						processedRecordingIDs[recordingID] = true

						log.Info(
							ctx, "Aligning heard track", zap.String("track", trackObj.Track),
							zap.String("recording_id", recordingID), zap.Int("pos", mbTrack.Position),
						)

						// 更新本地 track 元数据
						if err := tx.Model(&trackObj).Updates(
							map[string]interface{}{
								"music_brainz_id": recordingID,
								"track_number":    int8(mbTrack.Position),
							},
						).Error; err != nil {
							return err
						}

						// 更新关联表
						if err := tx.Model(ta).Updates(
							map[string]interface{}{
								"track_number":    int8(mbTrack.Position),
								"track":           mbTrack.Title,
								"mb_recording_id": recordingID,
							},
						).Error; err != nil {
							return err
						}
						completedCount++
					}
				}
			}

			// D. 处理未听过的歌曲（创建占位符记录 TrackID=0）
			placeholderCount := 0
			for _, mbTrack := range mbTracks {
				recordingID := string(mbTrack.Recording.ID)
				if processedRecordingIDs[recordingID] {
					continue
				}

				// 尝试匹配尚未建立 TrackID 关联但名称吻合的占位符 (大小写不敏感)
				var ta model.TrackAlbum
				if err := tx.Where(
					"album_id = ? AND LOWER(track) = ? AND track_id = 0", albumID, strings.ToLower(mbTrack.Title),
				).First(&ta).Error; err == nil {
					// 发现占位符，校正数据
					ta.TrackNumber = int8(mbTrack.Position)
					ta.MusicBrainzRecordingID = recordingID
					if err := tx.Save(&ta).Error; err != nil {
						return err
					}
					log.Info(
						ctx, "Aligned existing placeholder", zap.String("track", ta.Track),
						zap.String("recording_id", recordingID), zap.Int("pos", int(ta.TrackNumber)),
					)
					processedRecordingIDs[recordingID] = true // Mark as processed
					completedCount++
					continue // Move to next mbTrack
				}

				// 检查是否已经存在该录音 ID 的关联
				var count int64
				tx.Model(&model.TrackAlbum{}).Where(
					"album_id = ? AND mb_recording_id = ?", albumID, recordingID,
				).Count(&count)
				if count == 0 {
					newTA := &model.TrackAlbum{
						TrackID:                0,
						Track:                  mbTrack.Title,
						AlbumID:                albumID,
						TrackNumber:            int8(mbTrack.Position),
						MusicBrainzRecordingID: recordingID,
					}
					if err := tx.Create(newTA).Error; err != nil {
						return err
					}
					placeholderCount++
				}
			}

			// E. 更新专辑状态及元数据
			var genreStr string
			if len(release.Genres) > 0 {
				var genres []string
				for _, g := range release.Genres {
					genres = append(genres, g.Name)
				}
				genreStr = strings.Join(genres, ",")
			}

			updateFields := map[string]interface{}{
				"sync_status": 3,
			}
			if release.Date.String() != "" {
				updateFields["release_date"] = release.Date.String()
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

			if err := tx.Model(&model.Album{}).Where("id = ?", albumID).Updates(updateFields).Error; err != nil {
				return err
			}

			log.Info(
				ctx, "DeepingMaintenance transaction completed",
				zap.Int64("album_id", albumID),
				zap.Int("aligned_tracks", completedCount),
				zap.Int("new_placeholders", placeholderCount),
			)

			return nil
		},
	)

	if err != nil {
		log.Error(ctx, "DeepingMaintenance failed", zap.Int64("album_id", albumID), zap.Error(err))
	}

	return err
}
