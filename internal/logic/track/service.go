package track

import (
	"context"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/core/applemusic"
	"github.com/vincentchyu/sonic-lens/core/lastfm"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

// TrackService 定义曲目相关服务接口
type TrackService interface {
	GetTrackPlayCounts(ctx context.Context, limit, offset int, keyword string) ([]*model.Track, error)
	GetTrack(ctx context.Context, artist, album, track string) (*model.Track, error)
	InsertTrackPlayRecord(ctx context.Context, record *model.TrackPlayRecord) error
	IncrementTrackPlayCount(params model.IncrementTrackPlayCountParams) error
	GetTotalPlayCount(ctx context.Context) (int64, error)
	GetTrackCounts(ctx context.Context) (int64, error)
	GetArtistCounts(ctx context.Context) (int64, error)
	GetAlbumCounts(ctx context.Context) (int64, error)
	GetRecentPlayRecords(ctx context.Context, limit int) ([]*model.TrackPlayRecord, error)
	// GetRecentPlayRecordsByDays 获取指定天数内的播放记录
	GetRecentPlayRecordsByDays(ctx context.Context, days int) (map[string][]*model.TrackPlayRecord, error)
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
	// GetLastFmFavorite 获取Last.fm喜欢状态
	GetLastFmFavorite(ctx context.Context, artist, album, track string) (bool, error)
	// SetTrackFavorite 设置曲目喜欢状态
	SetTrackFavorite(
		ctx context.Context, artist, album, track, source string, isFavorite bool, metadata model.TrackMetadata,
	) (
		appleMusicFav bool, lastFmFav bool, err error,
	)
	// GetTopAlbumsByPlayCount 获取按播放次数统计的热门专辑
	GetTopAlbumsByPlayCount(ctx context.Context, days int, limit int) ([]*model.TopAlbum, error)
	// Genre related methods
	GetAllGenres(ctx context.Context, limit, offset int) ([]*model.Genre, error)
	GetGenreByName(ctx context.Context, name string) (*model.Genre, error)
	GetGenreCount(ctx context.Context) (int64, error)
	GetTopGenresByPlayCount(ctx context.Context, limit int) ([]*model.Genre, error)
}

// TrackServiceImpl 实现TrackService接口
type TrackServiceImpl struct{}

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

func (s *TrackServiceImpl) InsertTrackPlayRecord(ctx context.Context, record *model.TrackPlayRecord) error {
	return model.InsertTrackPlayRecord(ctx, record)
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
	err := applemusic.SetFavorite(params.Ctx, params.IsFavorite)
	if err != nil {
		return err
	}
	return model.SetAppleMusicFavorite(params)
}

// SetLastFmFavorite 设置Last.fm喜欢状态
func (s *TrackServiceImpl) SetLastFmFavorite(params model.SetFavoriteParams) error {
	err := lastfm.SetFavorite(params.Ctx, params.Artist, params.Track, params.IsFavorite)
	if err != nil {
		return err
	}
	return model.SetLastFmFavorite(params)
}

// GetAppleMusicFavorite 获取Apple Music喜欢状态
func (s *TrackServiceImpl) GetAppleMusicFavorite(ctx context.Context, artist, album, track string) (bool, error) {
	return model.GetAppleMusicFavorite(ctx, artist, album, track)
}

// GetLastFmFavorite 获取Last.fm喜欢状态
func (s *TrackServiceImpl) GetLastFmFavorite(ctx context.Context, artist, album, track string) (bool, error) {
	return model.GetLastFmFavorite(ctx, artist, album, track)
}

// SetTrackFavorite 设置曲目喜欢状态
func (s *TrackServiceImpl) SetTrackFavorite(
	ctx context.Context, artist, album, track, source string, isFavorite bool, metadata model.TrackMetadata,
) (appleMusicFav bool, lastFmFav bool, err error) {
	// 对于Apple Music来源，同时更新Apple Music和Last.fm的喜欢状态
	if source == "Apple Music" {
		// 更新Apple Music喜欢状态
		if err = s.SetAppleMusicFavorite(
			model.SetFavoriteParams{
				Ctx:           ctx,
				Artist:        artist,
				Album:         album,
				Track:         track,
				IsFavorite:    isFavorite,
				TrackMetadata: metadata,
			},
		); err != nil {
			return false, false, err
		}

		// 更新Last.fm喜欢状态
		if err = s.SetLastFmFavorite(
			model.SetFavoriteParams{
				Ctx:           ctx,
				Artist:        artist,
				Album:         album,
				Track:         track,
				IsFavorite:    isFavorite,
				TrackMetadata: metadata,
			},
		); err != nil {
			return false, false, err
		}

		// 获取更新后的状态
		appleMusicFav, _ = s.GetAppleMusicFavorite(ctx, artist, album, track)
		lastFmFav, _ = s.GetLastFmFavorite(ctx, artist, album, track)

		return appleMusicFav, lastFmFav, nil
	}

	// 对于其他来源，只更新数据库中的喜欢状态
	// 这里可以根据需要扩展对其他来源的支持 adirvana roon 支持上报lastfm喜欢状态
	_ = s.SetLastFmFavorite(
		model.SetFavoriteParams{
			Ctx:           ctx,
			Artist:        artist,
			Album:         album,
			Track:         track,
			IsFavorite:    isFavorite,
			TrackMetadata: metadata,
		},
	)
	return false, false, nil
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
