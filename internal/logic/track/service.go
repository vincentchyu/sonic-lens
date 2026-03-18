package track

import (
	"context"
	"errors"
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/core/applemusic"
	"github.com/vincentchyu/sonic-lens/core/lastfm"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

var (
	appleMusicSetFavorite                = applemusic.SetFavorite
	lastfmSetFavorite                    = lastfm.SetFavorite
	modelGetTrackByIdentity              = model.GetTrackByIdentity
	modelInsertTrackPlayRecord           = model.InsertTrackPlayRecord
	modelProcessTrackPlayRecord          = model.ProcessTrackPlayRecord
	modelSetAppleMusicFavorite           = model.SetAppleMusicFavorite
	modelSetLastFmFavorite               = model.SetLastFmFavorite
	modelGetAppleMusicFavorite           = model.GetAppleMusicFavorite
	modelGetAppleMusicFavoriteByIdentity = model.GetAppleMusicFavoriteByIdentity
	modelGetLastFmFavorite               = model.GetLastFmFavorite
	modelGetLastFmFavoriteByIdentity     = model.GetLastFmFavoriteByIdentity
)

// TrackService 定义曲目相关服务接口
type TrackService interface {
	GetTrackPlayCounts(ctx context.Context, limit, offset int, keyword string) ([]*model.Track, error)
	GetTrack(ctx context.Context, artist, album, track string) (*model.Track, error)
	GetTrackByIdentity(
		ctx context.Context, artist, album, track string, trackNumber, discNumber int8,
	) (*model.Track, error)
	InsertTrackPlayRecord(ctx context.Context, record *model.TrackPlayRecord) error
	ResolveTrackPlayRecord(
		ctx context.Context, recordID int64, artist, album, track string, metadata model.TrackMetadata,
	) error
	ProcessTrackPlayRecord(ctx context.Context, recordID int64, metadata model.TrackMetadata) error
	IncrementTrackPlayCount(params model.IncrementTrackPlayCountParams) error
	GetTotalPlayCount(ctx context.Context) (int64, error)
	GetTrackCounts(ctx context.Context) (int64, error)
	GetArtistCounts(ctx context.Context) (int64, error)
	GetAlbumCounts(ctx context.Context) (int64, error)
	GetRecentPlayRecords(ctx context.Context, limit int) ([]*model.TrackPlayRecord, error)
	// GetRecentPlayRecordsByDays 获取指定天数内的播放记录
	GetRecentPlayRecordsByDays(ctx context.Context, days int) (map[string][]*model.TrackPlayRecord, error)
	// GetPlayTrendByDays 获取指定天数的趋势聚合数据
	GetPlayTrendByDays(ctx context.Context, days int) (map[string]int, map[string]*model.HourlyPlayTrendData, error)
	// GetTopArtistsByPlayCount 获取按播放次数统计的热门艺术家
	GetTopArtistsByPlayCount(ctx context.Context, limit int) ([]map[string]interface{}, error)
	// GetTopArtistsByTrackCount 获取按曲目数统计的热门艺术家
	GetTopArtistsByTrackCount(ctx context.Context, limit int) ([]map[string]interface{}, error)
	// GetTrackPlayCountsByPeriod 获取指定时间段内的曲目播放统计
	GetTrackPlayCountsByPeriod(ctx context.Context, limit, offset int, period string, keyword string) ([]*model.Track, error)
	// GetPlayCountsBySource 获取按来源统计的播放次数
	GetPlayCountsBySource(ctx context.Context) (map[string]int64, error)
	// GetUnscrobbledRecordsWithPagination 分页获取未同步到Last.fm的播放记录
	GetUnscrobbledRecordsWithPagination(ctx context.Context, limit, offset int) ([]*model.TrackPlayRecord, error)
	// GetUnscrobbledRecordsCount 获取未同步到Last.fm的播放记录总数
	GetUnscrobbledRecordsCount(ctx context.Context) (int64, error)
	// SyncUnscrobbledRecords 同步未上报的数据到Last.fm并更新状态
	SyncUnscrobbledRecords(ctx context.Context, limit int) ([]*model.TrackPlayRecord, error)
	// SyncSelectedUnscrobbledRecords 同步选中的未同步记录到Last.fm
	SyncSelectedUnscrobbledRecords(ctx context.Context, ids []int64) (
		successCount int, failedRecords []*model.TrackPlayRecord, err error,
	)
	// SetAppleMusicFavorite 设置Apple Music喜欢状态
	SetAppleMusicFavorite(params model.SetFavoriteParams) error
	// SetLastFmFavorite 设置Last.fm喜欢状态
	SetLastFmFavorite(params model.SetFavoriteParams) error
	// GetAppleMusicFavorite 获取Apple Music喜欢状态
	GetAppleMusicFavorite(ctx context.Context, artist, album, track string) (bool, error)
	GetAppleMusicFavoriteByIdentity(
		ctx context.Context, artist, album, track string, trackNumber, discNumber int8,
	) (bool, error)
	// GetLastFmFavorite 获取Last.fm喜欢状态
	GetLastFmFavorite(ctx context.Context, artist, album, track string) (bool, error)
	GetLastFmFavoriteByIdentity(
		ctx context.Context, artist, album, track string, trackNumber, discNumber int8,
	) (bool, error)
	// SetTrackFavorite 设置曲目喜欢状态
	SetTrackFavorite(
		ctx context.Context, artist, album, track, source string, isFavorite bool, metadata model.TrackMetadata,
	) (
		appleMusicFav bool, lastFmFav bool, err error,
	)
	// GetTopAlbumsByPlayCount 获取按播放次数统计的热门专辑
	GetTopAlbumsByPlayCount(ctx context.Context, days int, limit int) ([]*model.TopAlbum, error)
	// Genre related methods
	// GetAlbums 获取专辑列表（分页）
	GetAlbums(ctx context.Context, limit, offset int, keyword string) ([]*model.Album, error)
	// GetAlbumsCount 获取专辑总数
	GetAlbumsCount(ctx context.Context, keyword string) (int64, error)
	// GetTracksOrderedByAlbum 按专辑排序获取曲目列表（分页）
	GetTracksOrderedByAlbum(ctx context.Context, limit, offset int, keyword string) ([]*model.Track, error)
	// GetTracksOrderedByAlbumCount 获取按专辑排序的曲目总数
	GetTracksOrderedByAlbumCount(ctx context.Context, keyword string) (int64, error)
	// GetLibrarySyncDelta 获取资料库同步增量
	GetLibrarySyncDelta(ctx context.Context, sinceVersion int64) (*model.LibrarySyncDelta, error)
	// GetTrackAlbumByTrackID 获取曲目当前首条专辑绑定，用于上层展示关联状态。
	GetTrackAlbumByTrackID(ctx context.Context, trackID int64) (*model.TrackAlbum, error)
	// GetAlbumDetail 获取专辑详情及其曲目列表。
	GetAlbumDetail(ctx context.Context, albumID int64) (*model.AlbumDetail, error)
	// DeleteTrackAlbumLink 删除人工修复指定的曲目专辑关联。
	DeleteTrackAlbumLink(ctx context.Context, trackID, albumID int64) error
	ProbeAndSyncTrackFavorite(ctx context.Context, input PlaybackEventInput) TrackFavoriteProbeResult
	HandleNowPlayingStarted(ctx context.Context, input PlaybackEventInput)
	HandleTrackPlaybackThreshold(ctx context.Context, input PlaybackEventInput) PlaybackThresholdResult
}

// TrackServiceImpl 实现TrackService接口
type TrackServiceImpl struct {
	favoriteProbeMu sync.Mutex
	lastLikeKey     string
	lastLikeProbe   favoriteProbeState
}

// NewTrackService 创建TrackService实例
func NewTrackService() TrackService {
	return &TrackServiceImpl{}
}

// GetTrackPlayCounts 获取曲目播放统计列表
func (s *TrackServiceImpl) GetTrackPlayCounts(ctx context.Context, limit, offset int, keyword string) (
	[]*model.Track, error,
) {
	return model.GetTracks(ctx, limit, offset, keyword)
}

// GetTrackPlayCount 获取特定曲目的播放统计
func (s *TrackServiceImpl) GetTrack(ctx context.Context, artist, album, track string) (
	*model.Track, error,
) {
	return model.GetTrack(ctx, artist, album, track)
}

func (s *TrackServiceImpl) GetTrackByIdentity(
	ctx context.Context, artist, album, track string, trackNumber, discNumber int8,
) (*model.Track, error) {
	return modelGetTrackByIdentity(ctx, artist, album, track, trackNumber, discNumber)
}

func (s *TrackServiceImpl) InsertTrackPlayRecord(ctx context.Context, record *model.TrackPlayRecord) error {
	return modelInsertTrackPlayRecord(ctx, record)
}

func (s *TrackServiceImpl) ResolveTrackPlayRecord(
	ctx context.Context, recordID int64, artist, album, track string, metadata model.TrackMetadata,
) error {
	return model.ResolveTrackPlayRecord(ctx, recordID, artist, album, track, metadata)
}

func (s *TrackServiceImpl) ProcessTrackPlayRecord(
	ctx context.Context, recordID int64, metadata model.TrackMetadata,
) error {
	return modelProcessTrackPlayRecord(ctx, recordID, metadata)
}

func (s *TrackServiceImpl) IncrementTrackPlayCount(params model.IncrementTrackPlayCountParams) error {
	return model.IncrementTrackPlayCount(params)
}

// GetTotalPlayCount 获取总播放次数
func (s *TrackServiceImpl) GetTotalPlayCount(ctx context.Context) (int64, error) {
	return model.GetTotalPlayCount(ctx)
}

// GetTrackCounts 获取曲目总数
func (s *TrackServiceImpl) GetTrackCounts(ctx context.Context) (int64, error) {
	return model.GetTrackCounts(ctx)
}

// GetArtistCounts 获取艺术家总数
func (s *TrackServiceImpl) GetArtistCounts(ctx context.Context) (int64, error) {
	return model.GetArtistCounts(ctx)
}

// GetAlbumCounts 获取专辑总数
func (s *TrackServiceImpl) GetAlbumCounts(ctx context.Context) (int64, error) {
	return model.GetAlbumCounts(ctx)
}

// GetRecentPlayRecords 获取最近播放记录
func (s *TrackServiceImpl) GetRecentPlayRecords(ctx context.Context, limit int) ([]*model.TrackPlayRecord, error) {
	return model.GetRecentPlayRecords(ctx, limit)
}

// GetRecentPlayRecordsByDays 获取指定天数内的播放记录
func (s *TrackServiceImpl) GetRecentPlayRecordsByDays(
	ctx context.Context, days int,
) (map[string][]*model.TrackPlayRecord, error) {
	return model.GetRecentPlayRecordsByDays(ctx, days)
}

func (s *TrackServiceImpl) GetPlayTrendByDays(
	ctx context.Context, days int,
) (map[string]int, map[string]*model.HourlyPlayTrendData, error) {
	return model.GetPlayTrendFromStatByDays(ctx, days)
}

// GetTopArtistsByPlayCount 获取按播放次数统计的热门艺术家
func (s *TrackServiceImpl) GetTopArtistsByPlayCount(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	return model.GetTopArtistsByPlayCount(ctx, limit)
}

// GetTopArtistsByTrackCount 获取按曲目数统计的热门艺术家
func (s *TrackServiceImpl) GetTopArtistsByTrackCount(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	return model.GetTopArtistsByTrackCount(ctx, limit)
}

// GetTrackPlayCountsByPeriod 获取指定时间段内的曲目播放统计
func (s *TrackServiceImpl) GetTrackPlayCountsByPeriod(
	ctx context.Context, limit, offset int, period string, keyword string,
) ([]*model.Track, error) {
	return model.GetTracksByPeriod(ctx, limit, offset, period, keyword)
}

// GetPlayCountsBySource 获取按来源统计的播放次数
func (s *TrackServiceImpl) GetPlayCountsBySource(ctx context.Context) (map[string]int64, error) {
	return model.GetPlayCountsBySource(ctx)
}

// GetTopAlbumsByPlayCount 获取按播放次数统计的热门专辑
func (s *TrackServiceImpl) GetTopAlbumsByPlayCount(ctx context.Context, days int, limit int) (
	[]*model.TopAlbum, error,
) {
	return model.GetTopAlbumsByPlayCount(ctx, days, limit)
}

// GetUnscrobbledRecordsWithPagination 分页获取未同步到Last.fm的播放记录
func (s *TrackServiceImpl) GetUnscrobbledRecordsWithPagination(
	ctx context.Context, limit, offset int,
) ([]*model.TrackPlayRecord, error) {
	return model.GetUnscrobbledRecordsWithPagination(ctx, limit, offset)
}

// GetUnscrobbledRecordsCount 获取未同步到Last.fm的播放记录总数
func (s *TrackServiceImpl) GetUnscrobbledRecordsCount(ctx context.Context) (int64, error) {
	return model.GetUnscrobbledRecordsCount(ctx)
}

// SyncUnscrobbledRecords 同步未上报的数据到Last.fm并更新状态
func (s *TrackServiceImpl) SyncUnscrobbledRecords(ctx context.Context, limit int) ([]*model.TrackPlayRecord, error) {
	return nil, nil
}

// SyncSelectedUnscrobbledRecords 同步选中的未同步记录到Last.fm
func (s *TrackServiceImpl) SyncSelectedUnscrobbledRecords(ctx context.Context, ids []int64) (
	successCount int, failedRecords []*model.TrackPlayRecord, err error,
) {
	// 获取指定ID的未同步记录
	records, err := model.GetUnscrobbledRecordsByIds(ctx, ids)
	if err != nil {
		return 0, nil, err
	}
	if len(records) == 0 {
		return 0, nil, nil
	}

	var successIDs []int64

	for _, record := range records {
		// 创建Last.fm同步请求
		req := &lastfm.PushTrackScrobbleReq{
			Artist:             record.Artist,
			AlbumArtist:        record.AlbumArtist,
			Track:              record.Track,
			Album:              record.Album,
			Duration:           record.Duration,
			Timestamp:          record.PlayTime.Unix(),
			MusicBrainzTrackID: record.MusicBrainzID,
			TrackNumber:        int64(record.TrackNumber),
		}

		_, err := lastfm.PushTrackScrobble(ctx, req)
		if err != nil {
			log.Error(ctx, "Failed to scrobble track", zap.String("track", record.Track), zap.Error(err))
			failedRecords = append(failedRecords, record)
			continue
		}

		successIDs = append(successIDs, record.ID)
	}

	// 批量更新成功同步的记录状态
	if len(successIDs) > 0 {
		if err := model.BatchUpdateScrobbledStatus(ctx, successIDs, true); err != nil {
			return 0, nil, err
		}
	}

	return len(successIDs), failedRecords, nil
}

// SetAppleMusicFavorite 设置Apple Music喜欢状态
func (s *TrackServiceImpl) SetAppleMusicFavorite(
	params model.SetFavoriteParams,
) error {
	err := appleMusicSetFavorite(params.Ctx, params.IsFavorite)
	if err != nil {
		return err
	}
	return modelSetAppleMusicFavorite(params)
}

// SetLastFmFavorite 设置Last.fm喜欢状态
func (s *TrackServiceImpl) SetLastFmFavorite(params model.SetFavoriteParams) error {
	err := lastfmSetFavorite(params.Ctx, params.Artist, params.Track, params.IsFavorite)
	if err != nil {
		return err
	}
	return modelSetLastFmFavorite(params)
}

// GetAppleMusicFavorite 获取Apple Music喜欢状态
func (s *TrackServiceImpl) GetAppleMusicFavorite(ctx context.Context, artist, album, track string) (bool, error) {
	return modelGetAppleMusicFavorite(ctx, artist, album, track)
}

func (s *TrackServiceImpl) GetAppleMusicFavoriteByIdentity(
	ctx context.Context, artist, album, track string, trackNumber, discNumber int8,
) (bool, error) {
	return modelGetAppleMusicFavoriteByIdentity(ctx, artist, album, track, trackNumber, discNumber)
}

// GetLastFmFavorite 获取Last.fm喜欢状态
func (s *TrackServiceImpl) GetLastFmFavorite(ctx context.Context, artist, album, track string) (bool, error) {
	return modelGetLastFmFavorite(ctx, artist, album, track)
}

func (s *TrackServiceImpl) GetLastFmFavoriteByIdentity(
	ctx context.Context, artist, album, track string, trackNumber, discNumber int8,
) (bool, error) {
	return modelGetLastFmFavoriteByIdentity(ctx, artist, album, track, trackNumber, discNumber)
}

// SetTrackFavorite 设置曲目喜欢状态
func (s *TrackServiceImpl) SetTrackFavorite(
	ctx context.Context, artist, album, track, source string, isFavorite bool, metadata model.TrackMetadata,
) (appleMusicFav bool, lastFmFav bool, err error) {
	params := model.SetFavoriteParams{
		Ctx:           ctx,
		Artist:        artist,
		Album:         album,
		Track:         track,
		IsFavorite:    isFavorite,
		TrackMetadata: metadata,
	}

	var callErr error
	// Apple Music 来源维持双写：Apple + Last.fm。
	if strings.EqualFold(source, model.TrackFavoriteEventSourceAppleMusic) {
		if amErr := s.SetAppleMusicFavorite(params); amErr != nil {
			callErr = errors.Join(callErr, amErr)
		}
	}
	// Last.fm 仍是系统统一收藏标记位。
	if lfmErr := s.SetLastFmFavorite(params); lfmErr != nil {
		callErr = errors.Join(callErr, lfmErr)
	}

	// 获取更新后的最终状态，确保返回给前端的数据是准确的
	appleMusicFav, _ = s.GetAppleMusicFavoriteByIdentity(
		ctx, artist, album, track, metadata.TrackNumber, metadata.DiscNumber,
	)
	lastFmFav, _ = s.GetLastFmFavoriteByIdentity(
		ctx, artist, album, track, metadata.TrackNumber, metadata.DiscNumber,
	)

	return appleMusicFav, lastFmFav, callErr
}

// GetAllGenres 获取所有流派（分页）
func (s *TrackServiceImpl) GetAllGenres(ctx context.Context, limit, offset int) ([]*model.Genre, error) {
	return model.GetAllGenres(ctx, limit, offset)
}

// GetGenreByName 根据名称获取流派
func (s *TrackServiceImpl) GetGenreByName(ctx context.Context, name string) (*model.Genre, error) {
	return model.GetGenreByName(ctx, name)
}

// GetGenreCount 获取流派总数
func (s *TrackServiceImpl) GetGenreCount(ctx context.Context) (int64, error) {
	return model.GetGenreCount(ctx)
}

// GetTopGenresByPlayCount 获取按播放次数排序的流派
func (s *TrackServiceImpl) GetTopGenresByPlayCount(ctx context.Context, limit int) ([]*model.Genre, error) {
	return model.GetTopGenresByPlayCount(ctx, limit)
}

// GetAlbums 获取专辑列表
func (s *TrackServiceImpl) GetAlbums(ctx context.Context, limit, offset int, keyword string) ([]*model.Album, error) {
	return model.GetAlbums(ctx, limit, offset, keyword)
}

// GetAlbumsCount 获取专辑总数
func (s *TrackServiceImpl) GetAlbumsCount(ctx context.Context, keyword string) (int64, error) {
	return model.GetAlbumsCount(ctx, keyword)
}

// GetTracksOrderedByAlbum 获取曲目列表
func (s *TrackServiceImpl) GetTracksOrderedByAlbum(ctx context.Context, limit, offset int, keyword string) ([]*model.Track, error) {
	return model.GetTracksOrderedByAlbum(ctx, limit, offset, keyword)
}

// GetTracksOrderedByAlbumCount 获取曲目总数
func (s *TrackServiceImpl) GetTracksOrderedByAlbumCount(ctx context.Context, keyword string) (int64, error) {
	return model.GetTracksOrderedByAlbumCount(ctx, keyword)
}

func (s *TrackServiceImpl) GetLibrarySyncDelta(ctx context.Context, sinceVersion int64) (*model.LibrarySyncDelta, error) {
	return model.GetLibrarySyncDelta(ctx, sinceVersion)
}

// GetTrackAlbumByTrackID 获取曲目当前首条专辑绑定。
func (s *TrackServiceImpl) GetTrackAlbumByTrackID(ctx context.Context, trackID int64) (*model.TrackAlbum, error) {
	return model.GetTrackAlbumByTrackID(ctx, trackID)
}

// GetAlbumDetail 获取专辑详情及其曲目列表。
func (s *TrackServiceImpl) GetAlbumDetail(ctx context.Context, albumID int64) (*model.AlbumDetail, error) {
	return model.GetAlbumWithTracks(ctx, albumID)
}

// DeleteTrackAlbumLink 删除指定的曲目专辑关联。
func (s *TrackServiceImpl) DeleteTrackAlbumLink(ctx context.Context, trackID, albumID int64) error {
	return model.DeleteTrackAlbumLink(ctx, trackID, albumID)
}
