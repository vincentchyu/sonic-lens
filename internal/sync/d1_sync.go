package d1sync

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/peterheb/cfd1"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

// D1Client D1 客户端封装
type D1Client struct {
	db  *sql.DB
	cfg *config.CloudflareConfig
}

const d1MaxParamsPerStatement = 31

func batchSizeByParams(paramsPerRow int) int {
	if paramsPerRow <= 0 {
		return 1
	}
	size := d1MaxParamsPerStatement / paramsPerRow
	if size < 1 {
		return 1
	}
	return size
}

// NewD1Client 创建 D1 客户端
func NewD1Client(cfg *config.CloudflareConfig) (*D1Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cloudflare config is nil")
	}
	// db, err := sql.Open("cfd1",
	//    "d1://your-account-id:your-api-token@database-name-or-UUID")
	dsn := fmt.Sprintf(
		"d1://%s:%s@%s",
		cfg.AccountID, cfg.APIToken, cfg.D1DatabaseID,
	)

	db, err := sql.Open("cfd1", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open D1 connection: %w", err)
	}

	// 测试连接
	// D1 API error 7403: The given account is not valid or is not authorized to access this service
	// listing databases: listing databases (page 1): D1 API error 10000: Authentication error
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping D1 database: %w", err)
	}

	return &D1Client{
		db:  db,
		cfg: cfg,
	}, nil
}

// Close 关闭 D1 连接
func (c *D1Client) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// SyncTracks 同步曲目数据到 D1
func (c *D1Client) SyncTracks(ctx context.Context, incremental bool) error {
	log.Info(ctx, "Starting D1 tracks sync", zap.Bool("incremental", incremental))

	// 获取最后同步时间
	var lastSyncTime time.Time
	var err error
	lastSyncTime, err = c.getLastSyncTime(ctx, "tracks")
	if err != nil {
		log.Warn(ctx, "Failed to get last sync time, performing full sync", zap.Error(err))
		incremental = false
	}
	if !lastSyncTime.IsZero() {
		incremental = true
		log.Info(ctx, "Starting D1 tracks sync lastSyncTime", zap.Bool("incremental", incremental))
	}

	// 从本地数据库获取曲目数据
	tracks, err := c.getTracksFromLocal(ctx, incremental, lastSyncTime)
	if err != nil {
		return fmt.Errorf("failed to get tracks from local db: %w", err)
	}

	if len(tracks) == 0 {
		log.Info(ctx, "No tracks to sync")
		return nil
	}

	log.Info(ctx, "Got tracks from local db", zap.Int("count", len(tracks)))

	// 批量同步到 D1
	if err := c.batchUpsertTracks(ctx, tracks); err != nil {
		return fmt.Errorf("failed to batch upsert tracks: %w", err)
	}

	// 更新同步元数据
	if err := c.updateSyncMetadata(ctx, "tracks", len(tracks)); err != nil {
		log.Warn(ctx, "Failed to update sync metadata", zap.Error(err))
	}

	log.Info(ctx, "D1 tracks sync completed", zap.Int("synced_count", len(tracks)))
	return nil
}

// getTracksFromLocal 从本地数据库获取曲目数据
func (c *D1Client) getTracksFromLocal(ctx context.Context, incremental bool, lastSyncTime time.Time) (
	[]*model.Track, error,
) {
	if incremental {
		// 增量同步:仅获取自上次同步后更新的记录
		log.Info(ctx, "Performing incremental sync", zap.Time("last_sync_time", lastSyncTime))
		return model.GetTracksUpdatedSince(ctx, lastSyncTime)
	}

	// 全量同步:获取所有记录
	log.Info(ctx, "Performing full sync")
	return model.GetAllTrackPlayCounts(ctx)
}

// batchUpsertTracks 批量插入或更新曲目数据
func (c *D1Client) batchUpsertTracks(ctx context.Context, tracks []*model.Track) error {
	// D1 单次事务限制,使用批量处理
	// 12 params per row -> floor(31/12)=2
	batchSize := batchSizeByParams(12)
	totalBatches := (len(tracks) + batchSize - 1) / batchSize

	for i := 0; i < len(tracks); i += batchSize {
		end := i + batchSize
		if end > len(tracks) {
			end = len(tracks)
		}

		batch := tracks[i:end]
		currentBatch := (i / batchSize) + 1

		log.Info(
			ctx, "Syncing batch",
			zap.Int("batch", currentBatch),
			zap.Int("total_batches", totalBatches),
			zap.Int("batch_size", len(batch)),
		)

		if err := c.upsertTracksBatch(ctx, batch); err != nil {
			return fmt.Errorf("failed to upsert batch %d: %w", currentBatch, err)
		}
	}

	return nil
}

// upsertTracksBatch 插入或更新一批曲目
func (c *D1Client) upsertTracksBatch(ctx context.Context, tracks []*model.Track) error {
	if len(tracks) == 0 {
		return nil
	}

	// 字段数量
	const numFields = 12
	placeholders := make([]string, len(tracks))
	args := make([]interface{}, 0, len(tracks)*numFields)

	for i, track := range tracks {
		placeholders[i] = "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
		args = append(
			args,
			track.Artist,
			track.Album,
			track.Track,
			track.AlbumArtist,
			track.PlayCount,
			track.Genre,
			track.Duration,
			track.Source,
			boolToInt(track.IsAppleMusicFav),
			boolToInt(track.IsLastFmFav),
			track.CreatedAt.Format(time.RFC3339),
			track.UpdatedAt.Format(time.RFC3339),
		)
	}

	query := fmt.Sprintf(
		`
		INSERT OR REPLACE INTO tracks (
			artist, album, track, album_artist, play_count, genre, duration, source,
			is_apple_music_fav, is_last_fm_fav, created_at, updated_at
		) VALUES %s
	`, strings.Join(placeholders, ", "),
	)

	if _, err := c.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to batch upsert tracks: %w", err)
	}

	return nil
}

// SyncPlayRecords 同步播放记录到 D1
func (c *D1Client) SyncPlayRecords(ctx context.Context, incremental bool) error {
	log.Info(ctx, "Starting D1 play records sync", zap.Bool("incremental", incremental))

	// 获取最后同步时间
	var lastSyncTime time.Time
	var err error
	lastSyncTime, err = c.getLastSyncTime(ctx, "track_play_records")
	if err != nil {
		log.Warn(ctx, "Failed to get last sync time, performing full sync", zap.Error(err))
		incremental = false
	}
	if !lastSyncTime.IsZero() {
		incremental = true
		log.Info(ctx, "Starting D1 records sync lastSyncTime", zap.Bool("incremental", incremental))
	}

	records, err := c.getPlayRecordsFromLocal(ctx, incremental, lastSyncTime)
	if err != nil {
		return fmt.Errorf("failed to get play records from local db: %w", err)
	}

	if len(records) == 0 {
		log.Info(ctx, "No play records to sync")
		return nil
	}

	log.Info(ctx, "Got play records from local db", zap.Int("count", len(records)))

	if err := c.batchUpsertPlayRecords(ctx, records); err != nil {
		return fmt.Errorf("failed to batch upsert play records: %w", err)
	}

	if err := c.updateSyncMetadata(ctx, "track_play_records", len(records)); err != nil {
		log.Warn(ctx, "Failed to update sync metadata", zap.Error(err))
	}

	log.Info(ctx, "D1 play records sync completed", zap.Int("synced_count", len(records)))
	return nil
}

func (c *D1Client) getPlayRecordsFromLocal(ctx context.Context, incremental bool, lastSyncTime time.Time) (
	[]*model.TrackPlayRecord, error,
) {
	if incremental {
		log.Info(ctx, "Performing incremental sync for play records", zap.Time("last_sync_time", lastSyncTime))
		return model.GetPlayRecordsUpdatedSince(ctx, lastSyncTime)
	}
	log.Info(ctx, "Performing full sync for play records")
	return model.GetPlayRecordsUpdatedSince(ctx, time.Time{})
}

func (c *D1Client) batchUpsertPlayRecords(ctx context.Context, records []*model.TrackPlayRecord) error {
	// 10 params per row -> floor(31/10)=3
	batchSize := batchSizeByParams(10)
	totalBatches := (len(records) + batchSize - 1) / batchSize

	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}
		batch := records[i:end]
		currentBatch := (i / batchSize) + 1

		log.Info(
			ctx, "Syncing play records batch",
			zap.Int("batch", currentBatch),
			zap.Int("total_batches", totalBatches),
			zap.Int("batch_size", len(batch)),
		)

		if err := c.upsertPlayRecordsBatch(ctx, batch); err != nil {
			return fmt.Errorf("failed to upsert play records batch %d: %w", currentBatch, err)
		}
	}
	return nil
}

func (c *D1Client) upsertPlayRecordsBatch(ctx context.Context, records []*model.TrackPlayRecord) error {
	if len(records) == 0 {
		return nil
	}

	const numFields = 10
	placeholders := make([]string, len(records))
	args := make([]interface{}, 0, len(records)*numFields)

	for i, record := range records {
		placeholders[i] = "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
		args = append(
			args,
			record.Artist,
			record.AlbumArtist,
			record.Album,
			record.Track,
			record.Duration,
			record.PlayTime.Format(time.RFC3339),
			boolToInt(record.Scrobbled),
			record.Source,
			record.CreatedAt.Format(time.RFC3339),
			record.UpdatedAt.Format(time.RFC3339),
		)
	}

	query := fmt.Sprintf(
		`
		INSERT OR REPLACE INTO track_play_records (
			artist, album_artist, album, track, duration, play_time, scrobbled, source, created_at, updated_at
		) VALUES %s
	`, strings.Join(placeholders, ", "),
	)

	if _, err := c.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to batch upsert play records: %w", err)
	}
	return nil
}

// SyncGenres 同步流派数据到 D1
func (c *D1Client) SyncGenres(ctx context.Context, incremental bool) error {
	log.Info(ctx, "Starting D1 genres sync", zap.Bool("incremental", incremental))

	// 获取最后同步时间
	var lastSyncTime time.Time
	var err error
	lastSyncTime, err = c.getLastSyncTime(ctx, "genres")
	if err != nil {
		log.Warn(ctx, "Failed to get last sync time, performing full sync", zap.Error(err))
		incremental = false
	}
	if !lastSyncTime.IsZero() {
		incremental = true
		log.Info(ctx, "Starting D1 tracks sync lastSyncTime", zap.Bool("incremental", incremental))
	}
	genres, err := c.getGenresFromLocal(ctx, incremental, lastSyncTime)
	if err != nil {
		return fmt.Errorf("failed to get genres from local db: %w", err)
	}

	if len(genres) == 0 {
		log.Info(ctx, "No genres to sync")
		return nil
	}

	log.Info(ctx, "Got genres from local db", zap.Int("count", len(genres)))

	if err := c.batchUpsertGenres(ctx, genres); err != nil {
		return fmt.Errorf("failed to batch upsert genres: %w", err)
	}

	if err := c.updateSyncMetadata(ctx, "genres", len(genres)); err != nil {
		log.Warn(ctx, "Failed to update sync metadata", zap.Error(err))
	}

	log.Info(ctx, "D1 genres sync completed", zap.Int("synced_count", len(genres)))
	return nil
}

// SyncDashboardStats 同步 dashboard 统计表到 D1
func (c *D1Client) SyncDashboardStats(ctx context.Context) error {
	log.Info(ctx, "Starting D1 dashboard stats sync")
	if err := model.EnsureDashboardStatSchema(ctx); err != nil {
		return fmt.Errorf("failed to ensure dashboard stat schema: %w", err)
	}

	lastSyncTime, err := c.getLastSyncTime(ctx, "dashboard_stats")
	if err != nil {
		log.Warn(ctx, "Failed to get dashboard stats last sync time, using full sync fallback", zap.Error(err))
		lastSyncTime = time.Time{}
	}

	overviewRows, err := model.GetDashboardStatsUpdatedSince(ctx, lastSyncTime)
	if err != nil {
		return fmt.Errorf("failed to get dashboard_stat from local db: %w", err)
	}
	sourceRows, err := model.GetPlaySourceStatsUpdatedSince(ctx, lastSyncTime)
	if err != nil {
		return fmt.Errorf("failed to get play_source_stat from local db: %w", err)
	}
	artistRows, err := model.GetTopArtistStatsUpdatedSince(ctx, lastSyncTime)
	if err != nil {
		return fmt.Errorf("failed to get top_artist_stat from local db: %w", err)
	}
	albumRows, err := model.GetTopAlbumStatsUpdatedSince(ctx, lastSyncTime)
	if err != nil {
		return fmt.Errorf("failed to get top_album_stat from local db: %w", err)
	}
	genreRows, err := model.GetTopGenreStatsUpdatedSince(ctx, lastSyncTime)
	if err != nil {
		return fmt.Errorf("failed to get top_genre_stat from local db: %w", err)
	}
	trendDailyRows, err := model.GetPlayTrendDailyStatsUpdatedSince(ctx, lastSyncTime)
	if err != nil {
		return fmt.Errorf("failed to get play_trend_daily_stat from local db: %w", err)
	}
	trendHourlyRows, err := model.GetPlayTrendHourlyStatsUpdatedSince(ctx, lastSyncTime)
	if err != nil {
		return fmt.Errorf("failed to get play_trend_hourly_stat from local db: %w", err)
	}

	totalCount := len(overviewRows) + len(sourceRows) + len(artistRows) + len(albumRows) + len(genreRows) + len(trendDailyRows) + len(trendHourlyRows)
	if totalCount == 0 {
		log.Info(ctx, "No dashboard stats changes to sync")
		return nil
	}

	// 仅首次同步执行一次全量清理，后续增量同步不再全量 DELETE + 全量 INSERT。
	if lastSyncTime.IsZero() {
		if err := c.resetDashboardStatTables(ctx); err != nil {
			return fmt.Errorf("failed to reset D1 dashboard stat tables: %w", err)
		}
	}

	if err := c.batchUpsertDashboardOverview(ctx, overviewRows); err != nil {
		return fmt.Errorf("failed to sync dashboard overview: %w", err)
	}
	if err := c.batchUpsertPlaySourceStats(ctx, sourceRows); err != nil {
		return fmt.Errorf("failed to sync play source stats: %w", err)
	}
	if err := c.batchUpsertTopArtistStats(ctx, artistRows); err != nil {
		return fmt.Errorf("failed to sync top artist stats: %w", err)
	}
	if err := c.batchUpsertTopAlbumStats(ctx, albumRows); err != nil {
		return fmt.Errorf("failed to sync top album stats: %w", err)
	}
	if err := c.batchUpsertTopGenreStats(ctx, genreRows); err != nil {
		return fmt.Errorf("failed to sync top genre stats: %w", err)
	}
	if err := c.batchUpsertPlayTrendDailyStats(ctx, trendDailyRows); err != nil {
		return fmt.Errorf("failed to sync daily trend stats: %w", err)
	}
	if err := c.batchUpsertPlayTrendHourlyStats(ctx, trendHourlyRows); err != nil {
		return fmt.Errorf("failed to sync hourly trend stats: %w", err)
	}

	// 增量同步模式下，清理遗留旧数据（由本地“删后重建”造成）。
	if !lastSyncTime.IsZero() {
		if err := c.cleanupStaleDashboardStatRows(ctx, lastSyncTime); err != nil {
			return fmt.Errorf("failed to cleanup stale dashboard stats rows: %w", err)
		}
	}

	if err := c.updateSyncMetadata(ctx, "dashboard_stats", totalCount); err != nil {
		log.Warn(ctx, "Failed to update dashboard stats sync metadata", zap.Error(err))
	}

	log.Info(
		ctx, "D1 dashboard stats sync completed",
		zap.Int("dashboard_stat", len(overviewRows)),
		zap.Int("play_source_stat", len(sourceRows)),
		zap.Int("top_artist_stat", len(artistRows)),
		zap.Int("top_album_stat", len(albumRows)),
		zap.Int("top_genre_stat", len(genreRows)),
		zap.Int("play_trend_daily_stat", len(trendDailyRows)),
		zap.Int("play_trend_hourly_stat", len(trendHourlyRows)),
	)
	return nil
}

func (c *D1Client) cleanupStaleDashboardStatRows(ctx context.Context, lastSyncTime time.Time) error {
	lastSyncAt := lastSyncTime.Format(time.RFC3339)
	tables := []string{
		"dashboard_stat",
		"play_source_stat",
		"top_artist_stat",
		"top_album_stat",
		"top_genre_stat",
		"play_trend_daily_stat",
		"play_trend_hourly_stat",
	}

	for _, table := range tables {
		query := fmt.Sprintf("DELETE FROM %s WHERE updated_at < ?", table)
		if _, err := c.db.ExecContext(ctx, query, lastSyncAt); err != nil {
			return fmt.Errorf("failed to cleanup stale rows from table %s: %w", table, err)
		}
	}
	return nil
}

func (c *D1Client) resetDashboardStatTables(ctx context.Context) error {
	tables := []string{
		"dashboard_stat",
		"play_source_stat",
		"top_artist_stat",
		"top_album_stat",
		"top_genre_stat",
		"play_trend_daily_stat",
		"play_trend_hourly_stat",
	}
	for _, table := range tables {
		if _, err := c.db.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return fmt.Errorf("failed to clear table %s: %w", table, err)
		}
	}
	return nil
}

func (c *D1Client) batchUpsertDashboardOverview(ctx context.Context, rows []*model.DashboardStat) error {
	if len(rows) == 0 {
		return nil
	}
	// 6 params per row -> floor(31/6)=5
	batchSize := batchSizeByParams(6)
	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		if err := c.upsertDashboardOverviewBatch(ctx, rows[i:end]); err != nil {
			return err
		}
	}
	return nil
}

func (c *D1Client) upsertDashboardOverviewBatch(ctx context.Context, rows []*model.DashboardStat) error {
	placeholders := make([]string, len(rows))
	args := make([]interface{}, 0, len(rows)*6)
	for i, row := range rows {
		placeholders[i] = "(?, ?, ?, ?, ?, ?)"
		args = append(args, row.ID, row.TotalPlays, row.TotalTracks, row.TotalArtist, row.TotalAlbums, row.UpdatedAt.Format(time.RFC3339))
	}
	query := fmt.Sprintf(
		"INSERT OR REPLACE INTO dashboard_stat (id, total_plays, total_tracks, total_artist, total_albums, updated_at) VALUES %s",
		strings.Join(placeholders, ", "),
	)
	_, err := c.db.ExecContext(ctx, query, args...)
	return err
}

func (c *D1Client) batchUpsertPlaySourceStats(ctx context.Context, rows []*model.PlaySourceStat) error {
	if len(rows) == 0 {
		return nil
	}
	// 4 params per row -> floor(31/4)=7
	batchSize := batchSizeByParams(4)
	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		if err := c.upsertPlaySourceStatsBatch(ctx, rows[i:end]); err != nil {
			return err
		}
	}
	return nil
}

func (c *D1Client) upsertPlaySourceStatsBatch(ctx context.Context, rows []*model.PlaySourceStat) error {
	placeholders := make([]string, len(rows))
	args := make([]interface{}, 0, len(rows)*4)
	for i, row := range rows {
		placeholders[i] = "(?, ?, ?, ?)"
		args = append(args, row.ID, row.Source, row.Count, row.UpdatedAt.Format(time.RFC3339))
	}
	query := fmt.Sprintf(
		"INSERT OR REPLACE INTO play_source_stat (id, source, count, updated_at) VALUES %s",
		strings.Join(placeholders, ", "),
	)
	_, err := c.db.ExecContext(ctx, query, args...)
	return err
}

func (c *D1Client) batchUpsertTopArtistStats(ctx context.Context, rows []*model.TopArtistStat) error {
	if len(rows) == 0 {
		return nil
	}
	// 6 params per row -> floor(31/6)=5
	batchSize := batchSizeByParams(6)
	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		if err := c.upsertTopArtistStatsBatch(ctx, rows[i:end]); err != nil {
			return err
		}
	}
	return nil
}

func (c *D1Client) upsertTopArtistStatsBatch(ctx context.Context, rows []*model.TopArtistStat) error {
	placeholders := make([]string, len(rows))
	args := make([]interface{}, 0, len(rows)*6)
	for i, row := range rows {
		placeholders[i] = "(?, ?, ?, ?, ?, ?)"
		args = append(args, row.PeriodDays, row.MetricType, row.Artist, row.MetricValue, row.Rank, row.UpdatedAt.Format(time.RFC3339))
	}
	query := fmt.Sprintf(
		"INSERT OR REPLACE INTO top_artist_stat (period_days, metric_type, artist, metric_value, rank, updated_at) VALUES %s",
		strings.Join(placeholders, ", "),
	)
	_, err := c.db.ExecContext(ctx, query, args...)
	return err
}

func (c *D1Client) batchUpsertTopAlbumStats(ctx context.Context, rows []*model.TopAlbumStat) error {
	if len(rows) == 0 {
		return nil
	}
	// 6 params per row -> floor(31/6)=5
	batchSize := batchSizeByParams(6)
	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		if err := c.upsertTopAlbumStatsBatch(ctx, rows[i:end]); err != nil {
			return err
		}
	}
	return nil
}

func (c *D1Client) upsertTopAlbumStatsBatch(ctx context.Context, rows []*model.TopAlbumStat) error {
	placeholders := make([]string, len(rows))
	args := make([]interface{}, 0, len(rows)*6)
	for i, row := range rows {
		placeholders[i] = "(?, ?, ?, ?, ?, ?)"
		args = append(args, row.PeriodDays, row.Album, row.Artist, row.PlayCount, row.Rank, row.UpdatedAt.Format(time.RFC3339))
	}
	query := fmt.Sprintf(
		"INSERT OR REPLACE INTO top_album_stat (period_days, album, artist, play_count, rank, updated_at) VALUES %s",
		strings.Join(placeholders, ", "),
	)
	_, err := c.db.ExecContext(ctx, query, args...)
	return err
}

func (c *D1Client) batchUpsertTopGenreStats(ctx context.Context, rows []*model.TopGenreStat) error {
	if len(rows) == 0 {
		return nil
	}
	// 6 params per row -> floor(31/6)=5
	batchSize := batchSizeByParams(6)
	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		if err := c.upsertTopGenreStatsBatch(ctx, rows[i:end]); err != nil {
			return err
		}
	}
	return nil
}

func (c *D1Client) upsertTopGenreStatsBatch(ctx context.Context, rows []*model.TopGenreStat) error {
	placeholders := make([]string, len(rows))
	args := make([]interface{}, 0, len(rows)*6)
	for i, row := range rows {
		placeholders[i] = "(?, ?, ?, ?, ?, ?)"
		args = append(args, row.GenreName, row.GenreNameZh, row.TrackGenreCount, row.GenreCount, row.Rank, row.UpdatedAt.Format(time.RFC3339))
	}
	query := fmt.Sprintf(
		"INSERT OR REPLACE INTO top_genre_stat (genre_name, genre_name_zh, track_genre_count, genre_count, rank, updated_at) VALUES %s",
		strings.Join(placeholders, ", "),
	)
	_, err := c.db.ExecContext(ctx, query, args...)
	return err
}

func (c *D1Client) batchUpsertPlayTrendDailyStats(ctx context.Context, rows []*model.PlayTrendDailyStat) error {
	if len(rows) == 0 {
		return nil
	}
	// 3 params per row -> floor(31/3)=10
	batchSize := batchSizeByParams(3)
	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		if err := c.upsertPlayTrendDailyStatsBatch(ctx, rows[i:end]); err != nil {
			return err
		}
	}
	return nil
}

func (c *D1Client) upsertPlayTrendDailyStatsBatch(ctx context.Context, rows []*model.PlayTrendDailyStat) error {
	placeholders := make([]string, len(rows))
	args := make([]interface{}, 0, len(rows)*3)
	for i, row := range rows {
		placeholders[i] = "(?, ?, ?)"
		args = append(args, row.StatDate.Format("2006-01-02"), row.PlayCount, row.UpdatedAt.Format(time.RFC3339))
	}
	query := fmt.Sprintf(
		"INSERT OR REPLACE INTO play_trend_daily_stat (stat_date, play_count, updated_at) VALUES %s",
		strings.Join(placeholders, ", "),
	)
	_, err := c.db.ExecContext(ctx, query, args...)
	return err
}

func (c *D1Client) batchUpsertPlayTrendHourlyStats(ctx context.Context, rows []*model.PlayTrendHourlyStat) error {
	if len(rows) == 0 {
		return nil
	}
	// 4 params per row -> floor(31/4)=7
	batchSize := batchSizeByParams(4)
	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		if err := c.upsertPlayTrendHourlyStatsBatch(ctx, rows[i:end]); err != nil {
			return err
		}
	}
	return nil
}

func (c *D1Client) upsertPlayTrendHourlyStatsBatch(ctx context.Context, rows []*model.PlayTrendHourlyStat) error {
	placeholders := make([]string, len(rows))
	args := make([]interface{}, 0, len(rows)*4)
	for i, row := range rows {
		placeholders[i] = "(?, ?, ?, ?)"
		args = append(args, row.StatDate.Format("2006-01-02"), row.Hour, row.PlayCount, row.UpdatedAt.Format(time.RFC3339))
	}
	query := fmt.Sprintf(
		"INSERT OR REPLACE INTO play_trend_hourly_stat (stat_date, hour, play_count, updated_at) VALUES %s",
		strings.Join(placeholders, ", "),
	)
	_, err := c.db.ExecContext(ctx, query, args...)
	return err
}

func (c *D1Client) getGenresFromLocal(ctx context.Context, incremental bool, lastSyncTime time.Time) (
	[]*model.Genre, error,
) {
	if incremental {
		log.Info(ctx, "Performing incremental sync for genres", zap.Time("last_sync_time", lastSyncTime))
		return model.GetGenresUpdatedSince(ctx, lastSyncTime)
	}
	log.Info(ctx, "Performing full sync for genres")
	return model.GetGenresUpdatedSince(ctx, time.Time{})
}

func (c *D1Client) batchUpsertGenres(ctx context.Context, genres []*model.Genre) error {
	// 5 params per row -> floor(31/5)=6
	batchSize := batchSizeByParams(5)
	totalBatches := (len(genres) + batchSize - 1) / batchSize

	for i := 0; i < len(genres); i += batchSize {
		end := i + batchSize
		if end > len(genres) {
			end = len(genres)
		}
		batch := genres[i:end]
		currentBatch := (i / batchSize) + 1

		log.Info(
			ctx, "Syncing genres batch",
			zap.Int("batch", currentBatch),
			zap.Int("total_batches", totalBatches),
			zap.Int("batch_size", len(batch)),
		)

		if err := c.upsertGenresBatch(ctx, batch); err != nil {
			return fmt.Errorf("failed to upsert genres batch %d: %w", currentBatch, err)
		}
	}
	return nil
}

func (c *D1Client) upsertGenresBatch(ctx context.Context, genres []*model.Genre) error {
	if len(genres) == 0 {
		return nil
	}

	const numFields = 5
	placeholders := make([]string, len(genres))
	args := make([]interface{}, 0, len(genres)*numFields)

	for i, genre := range genres {
		placeholders[i] = "(?, ?, ?, ?, ?)"
		args = append(
			args,
			genre.Name,
			genre.NameZh,
			genre.PlayCount,
			genre.CreatedAt.Format(time.RFC3339),
			genre.UpdatedAt.Format(time.RFC3339),
		)
	}

	query := fmt.Sprintf(
		`
		INSERT OR REPLACE INTO genres (
			name, name_zh, play_count, created_at, updated_at
		) VALUES %s
	`, strings.Join(placeholders, ", "),
	)

	if _, err := c.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to batch upsert genres: %w", err)
	}
	return nil
}

// getLastSyncTime 获取最后同步时间
func (c *D1Client) getLastSyncTime(ctx context.Context, tableName string) (time.Time, error) {
	var lastSyncTimeStr string
	err := c.db.QueryRowContext(
		ctx,
		"SELECT last_sync_time FROM sync_metadata WHERE table_name = ?",
		tableName,
	).Scan(&lastSyncTimeStr)

	if errors.Is(err, sql.ErrNoRows) {
		// 首次同步,返回零时间
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}

	return time.Parse(time.RFC3339, lastSyncTimeStr)
}

// updateSyncMetadata 更新同步元数据
func (c *D1Client) updateSyncMetadata(ctx context.Context, tableName string, syncCount int) error {
	now := time.Now()

	_, err := c.db.ExecContext(
		ctx, `
		INSERT OR REPLACE INTO sync_metadata (
			table_name, last_sync_time, sync_count, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?)
	`, tableName, now.Format(time.RFC3339), syncCount, now.Format(time.RFC3339), now.Format(time.RFC3339),
	)

	return err
}

// boolToInt 将 bool 转换为 int
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// SyncAll 同步所有数据
func (c *D1Client) SyncAll(ctx context.Context, incremental bool) error {
	log.Info(ctx, "Starting full D1 sync", zap.Bool("incremental", incremental))

	// 同步曲目数据
	if err := c.SyncTracks(ctx, incremental); err != nil {
		log.Error(ctx, "Failed to sync tracks", zap.Error(err))
		return err
	}

	// 同步播放记录
	if err := c.SyncPlayRecords(ctx, incremental); err != nil {
		log.Error(ctx, "Failed to sync play records", zap.Error(err))
		return err
	}

	// 同步流派数据
	if err := c.SyncGenres(ctx, incremental); err != nil {
		log.Error(ctx, "Failed to sync genres", zap.Error(err))
		return err
	}

	// 同步 dashboard 统计数据
	if err := c.SyncDashboardStats(ctx); err != nil {
		log.Warn(ctx, "Failed to sync dashboard stats, skipping", zap.Error(err))
	}

	log.Info(ctx, "Full D1 sync completed successfully")
	return nil
}
