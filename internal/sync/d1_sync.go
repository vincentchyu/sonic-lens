package d1sync

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/XSAM/otelsql"
	_ "github.com/peterheb/cfd1"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/config"
	coredb "github.com/vincentchyu/sonic-lens/core/db"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/core/telemetry"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

// D1Client D1 客户端封装
type D1Client struct {
	db          *sql.DB
	cfg         *config.CloudflareConfig
	syncRunning int32
}

const d1MaxParamsPerStatement = 31
const d1ExecMaxRetries = 5
const d1TrackUpsertFields = 14
const d1PlayRecordUpsertFields = 20
const d1TopAlbumUpsertFields = 8
const d1DriverName = "cfd1"

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

func isRetryableD1Error(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	retryableKeywords := []string{
		"unexpected eof",
		"timeout",
		"temporarily unavailable",
		"connection reset",
		"connection refused",
		"broken pipe",
		"server closed idle connection",
		"429",
		"502",
		"503",
		"504",
	}
	for _, keyword := range retryableKeywords {
		if strings.Contains(msg, keyword) {
			return true
		}
	}
	return false
}

func d1OpenOptions() []otelsql.Option {
	return []otelsql.Option{
		otelsql.WithTracerProvider(telemetry.GetTracerProvider()),
		otelsql.WithMeterProvider(telemetry.GetMeterProvider()),
		otelsql.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.name", "cloudflare_d1"),
			attribute.String("db.operation.mode", "rest_api"),
		),
	}
}

func (c *D1Client) execWithRetry(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	var lastErr error
	for attempt := 1; attempt <= d1ExecMaxRetries; attempt++ {
		result, err := c.db.ExecContext(ctx, query, args...)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !isRetryableD1Error(err) || attempt == d1ExecMaxRetries {
			break
		}

		backoff := time.Duration(attempt) * time.Second
		log.Warn(
			ctx, "D1 exec failed, retrying",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", d1ExecMaxRetries),
			zap.Duration("backoff", backoff),
			zap.Error(err),
		)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}
	return nil, lastErr
}

func (c *D1Client) ensureSchema(ctx context.Context) error {
	if err := c.ensureSyncMetadataSchema(ctx); err != nil {
		return err
	}
	if err := c.ensureTracksSchema(ctx); err != nil {
		return err
	}
	if err := c.ensureTrackPlayRecordsSchema(ctx); err != nil {
		return err
	}
	if err := c.ensureTopAlbumStatSchema(ctx); err != nil {
		return err
	}
	return nil
}

func (c *D1Client) ensureSyncMetadataSchema(ctx context.Context) error {
	_, err := c.execWithRetry(
		ctx, `
		CREATE TABLE IF NOT EXISTS sync_metadata (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			table_name TEXT NOT NULL UNIQUE,
			last_sync_time TEXT NOT NULL,
			sync_count INTEGER DEFAULT 0,
			last_error TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)
	`,
	)
	if err != nil {
		return fmt.Errorf("failed to ensure sync_metadata schema: %w", err)
	}
	if err := c.seedSyncMetadataBootstrap(ctx); err != nil {
		return err
	}
	return nil
}

func (c *D1Client) seedSyncMetadataBootstrap(ctx context.Context) error {
	bootstrapAt := time.Now().Add(-72 * time.Hour).Format(time.RFC3339)
	now := time.Now().Format(time.RFC3339)
	tableNames := []string{"tracks", "track_play_records", "genres", "dashboard_stats"}

	for _, tableName := range tableNames {
		_, err := c.execWithRetry(
			ctx,
			`
			INSERT OR IGNORE INTO sync_metadata (
				table_name, last_sync_time, sync_count, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?)
		`,
			tableName, bootstrapAt, 0, now, now,
		)
		if err != nil {
			return fmt.Errorf("failed to seed sync_metadata for %s: %w", tableName, err)
		}
	}

	return nil
}

func (c *D1Client) tableExists(ctx context.Context, tableName string) (bool, error) {
	var count int
	err := c.db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?",
		tableName,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (c *D1Client) tableHasColumn(ctx context.Context, tableName, columnName string) (bool, error) {
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name = ?", tableName)
	err := c.db.QueryRowContext(ctx, query, columnName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (c *D1Client) ensureTracksSchema(ctx context.Context) error {
	exists, err := c.tableExists(ctx, "tracks")
	if err != nil {
		return err
	}
	if !exists {
		return c.createTracksTable(ctx)
	}

	hasTrackNumber, err := c.tableHasColumn(ctx, "tracks", "track_number")
	if err != nil {
		return err
	}
	hasDiscNumber, err := c.tableHasColumn(ctx, "tracks", "disc_number")
	if err != nil {
		return err
	}
	if hasTrackNumber && hasDiscNumber {
		return c.ensureTracksIndexes(ctx)
	}

	return c.migrateTracksTable(ctx, hasTrackNumber, hasDiscNumber)
}

func (c *D1Client) createTracksTable(ctx context.Context) error {
	if _, err := c.execWithRetry(
		ctx, `
		CREATE TABLE IF NOT EXISTS tracks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			artist TEXT NOT NULL,
			album TEXT NOT NULL,
			track TEXT NOT NULL,
			album_artist TEXT,
			play_count INTEGER DEFAULT 0,
			genre TEXT,
			duration INTEGER,
			source TEXT,
			track_number INTEGER DEFAULT 0,
			disc_number INTEGER DEFAULT 1,
			is_apple_music_fav INTEGER DEFAULT 0,
			is_last_fm_fav INTEGER DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE (artist, album, track, disc_number, track_number)
		)
	`,
	); err != nil {
		return fmt.Errorf("failed to create tracks table: %w", err)
	}
	return c.ensureTracksIndexes(ctx)
}

func (c *D1Client) ensureTracksIndexes(ctx context.Context) error {
	statements := []string{
		"CREATE INDEX IF NOT EXISTS idx_tracks_artist ON tracks(artist)",
		"CREATE INDEX IF NOT EXISTS idx_tracks_album ON tracks(album)",
		"CREATE INDEX IF NOT EXISTS idx_tracks_genre ON tracks(genre)",
		"CREATE INDEX IF NOT EXISTS idx_tracks_source ON tracks(source)",
		"CREATE INDEX IF NOT EXISTS idx_tracks_play_count ON tracks(play_count DESC)",
		"CREATE INDEX IF NOT EXISTS idx_tracks_genre_play_count ON tracks(genre, play_count DESC)",
		"CREATE INDEX IF NOT EXISTS idx_tracks_artist_track ON tracks(artist, track)",
	}
	for _, stmt := range statements {
		if _, err := c.execWithRetry(ctx, stmt); err != nil {
			return fmt.Errorf("failed to ensure tracks index: %w", err)
		}
	}
	return nil
}

func (c *D1Client) migrateTracksTable(ctx context.Context, hasTrackNumber, hasDiscNumber bool) error {
	log.Info(
		ctx, "Migrating D1 tracks schema",
		zap.Bool("has_track_number", hasTrackNumber),
		zap.Bool("has_disc_number", hasDiscNumber),
	)

	if _, err := c.execWithRetry(ctx, "DROP TABLE IF EXISTS tracks_legacy_backup"); err != nil {
		return fmt.Errorf("failed to cleanup tracks backup table: %w", err)
	}
	if _, err := c.execWithRetry(ctx, "ALTER TABLE tracks RENAME TO tracks_legacy_backup"); err != nil {
		return fmt.Errorf("failed to rename tracks table: %w", err)
	}
	if err := c.createTracksTable(ctx); err != nil {
		return err
	}

	trackNumberExpr := "0"
	if hasTrackNumber {
		trackNumberExpr = "COALESCE(track_number, 0)"
	}
	discNumberExpr := "1"
	if hasDiscNumber {
		discNumberExpr = "COALESCE(NULLIF(disc_number, 0), 1)"
	}

	copySQL := fmt.Sprintf(
		`INSERT INTO tracks (
			artist, album, track, album_artist, play_count, genre, duration, source,
			track_number, disc_number, is_apple_music_fav, is_last_fm_fav, created_at, updated_at
		)
		SELECT
			artist, album, track, album_artist, COALESCE(play_count, 0), genre, duration, source,
			%s, %s, COALESCE(is_apple_music_fav, 0), COALESCE(is_last_fm_fav, 0), created_at, updated_at
		FROM tracks_legacy_backup`,
		trackNumberExpr, discNumberExpr,
	)
	if _, err := c.execWithRetry(ctx, copySQL); err != nil {
		return fmt.Errorf("failed to copy tracks data into new schema: %w", err)
	}
	if _, err := c.execWithRetry(ctx, "DROP TABLE tracks_legacy_backup"); err != nil {
		return fmt.Errorf("failed to drop tracks legacy table: %w", err)
	}
	return nil
}

func (c *D1Client) ensureTrackPlayRecordsSchema(ctx context.Context) error {
	exists, err := c.tableExists(ctx, "track_play_records")
	if err != nil {
		return err
	}
	if !exists {
		return c.createTrackPlayRecordsTable(ctx)
	}

	hasTrackNumber, err := c.tableHasColumn(ctx, "track_play_records", "track_number")
	if err != nil {
		return err
	}
	hasDiscNumber, err := c.tableHasColumn(ctx, "track_play_records", "disc_number")
	if err != nil {
		return err
	}
	hasResolvedTrackID, err := c.tableHasColumn(ctx, "track_play_records", "resolved_track_id")
	if err != nil {
		return err
	}
	hasResolutionStatus, err := c.tableHasColumn(ctx, "track_play_records", "resolution_status")
	if err != nil {
		return err
	}
	hasResolutionConfidence, err := c.tableHasColumn(ctx, "track_play_records", "resolution_confidence")
	if err != nil {
		return err
	}
	hasLibraryApplied, err := c.tableHasColumn(ctx, "track_play_records", "library_applied")
	if err != nil {
		return err
	}
	hasAlbumID, err := c.tableHasColumn(ctx, "track_play_records", "album_id")
	if err != nil {
		return err
	}
	hasCoverArtPath, err := c.tableHasColumn(ctx, "track_play_records", "cover_art_path")
	if err != nil {
		return err
	}
	hasMusicBrainzID, err := c.tableHasColumn(ctx, "track_play_records", "music_brainz_id")
	if err != nil {
		return err
	}
	hasAlbumSubtitle, err := c.tableHasColumn(ctx, "track_play_records", "album_subtitle")
	if err != nil {
		return err
	}
	if hasTrackNumber && hasDiscNumber && hasResolvedTrackID && hasResolutionStatus &&
		hasResolutionConfidence && hasLibraryApplied && hasAlbumID && hasCoverArtPath && hasMusicBrainzID && hasAlbumSubtitle {
		return c.ensureTrackPlayRecordsIndexes(ctx)
	}

	return c.migrateTrackPlayRecordsTable(
		ctx,
		hasTrackNumber,
		hasDiscNumber,
		hasResolvedTrackID,
		hasResolutionStatus,
		hasResolutionConfidence,
		hasLibraryApplied,
		hasAlbumID,
		hasCoverArtPath,
		hasMusicBrainzID,
		hasAlbumSubtitle,
	)
}

func (c *D1Client) createTrackPlayRecordsTable(ctx context.Context) error {
	if _, err := c.execWithRetry(
		ctx, `
		CREATE TABLE IF NOT EXISTS track_play_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			artist TEXT NOT NULL,
			album_artist TEXT,
			album TEXT NOT NULL,
			album_subtitle TEXT,
			track TEXT NOT NULL,
			album_id INTEGER DEFAULT 0,
			duration INTEGER,
			play_time TEXT NOT NULL,
				scrobbled INTEGER DEFAULT 0,
				track_number INTEGER DEFAULT 0,
				disc_number INTEGER DEFAULT 1,
				music_brainz_id TEXT,
				source TEXT,
				cover_art_path TEXT,
				resolved_track_id INTEGER DEFAULT 0,
				resolution_status TEXT NOT NULL DEFAULT 'pending',
				resolution_confidence INTEGER DEFAULT 0,
				library_applied INTEGER DEFAULT 0,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			)
	`,
	); err != nil {
		return fmt.Errorf("failed to create track_play_records table: %w", err)
	}
	return c.ensureTrackPlayRecordsIndexes(ctx)
}

func (c *D1Client) ensureTrackPlayRecordsIndexes(ctx context.Context) error {
	statements := []string{
		"CREATE INDEX IF NOT EXISTS idx_play_records_play_time ON track_play_records(play_time DESC)",
		"CREATE INDEX IF NOT EXISTS idx_play_records_artist ON track_play_records(artist)",
		"CREATE INDEX IF NOT EXISTS idx_play_records_source ON track_play_records(source)",
		"CREATE INDEX IF NOT EXISTS idx_play_records_album_id ON track_play_records(album_id)",
		"CREATE INDEX IF NOT EXISTS idx_play_records_music_brainz_id ON track_play_records(music_brainz_id)",
		"CREATE INDEX IF NOT EXISTS idx_play_records_resolved_track_id ON track_play_records(resolved_track_id)",
		"CREATE INDEX IF NOT EXISTS idx_play_records_resolution_status ON track_play_records(resolution_status)",
		"CREATE INDEX IF NOT EXISTS idx_play_records_library_applied ON track_play_records(library_applied)",
	}
	for _, stmt := range statements {
		if _, err := c.execWithRetry(ctx, stmt); err != nil {
			return fmt.Errorf("failed to ensure track_play_records index: %w", err)
		}
	}
	return nil
}

func (c *D1Client) migrateTrackPlayRecordsTable(
	ctx context.Context,
	hasTrackNumber, hasDiscNumber, hasResolvedTrackID, hasResolutionStatus,
	hasResolutionConfidence, hasLibraryApplied, hasAlbumID, hasCoverArtPath, hasMusicBrainzID, hasAlbumSubtitle bool,
) error {
	log.Info(
		ctx, "Migrating D1 track_play_records schema",
		zap.Bool("has_track_number", hasTrackNumber),
		zap.Bool("has_disc_number", hasDiscNumber),
		zap.Bool("has_resolved_track_id", hasResolvedTrackID),
		zap.Bool("has_resolution_status", hasResolutionStatus),
		zap.Bool("has_resolution_confidence", hasResolutionConfidence),
		zap.Bool("has_library_applied", hasLibraryApplied),
		zap.Bool("has_album_id", hasAlbumID),
		zap.Bool("has_cover_art_path", hasCoverArtPath),
		zap.Bool("has_music_brainz_id", hasMusicBrainzID),
		zap.Bool("has_album_subtitle", hasAlbumSubtitle),
	)

	if _, err := c.execWithRetry(ctx, "DROP TABLE IF EXISTS track_play_records_legacy_backup"); err != nil {
		return fmt.Errorf("failed to cleanup track_play_records backup table: %w", err)
	}
	if _, err := c.execWithRetry(ctx, "ALTER TABLE track_play_records RENAME TO track_play_records_legacy_backup"); err != nil {
		return fmt.Errorf("failed to rename track_play_records table: %w", err)
	}
	if err := c.createTrackPlayRecordsTable(ctx); err != nil {
		return err
	}

	trackNumberExpr := "0"
	if hasTrackNumber {
		trackNumberExpr = "COALESCE(track_number, 0)"
	}
	discNumberExpr := "1"
	if hasDiscNumber {
		discNumberExpr = "COALESCE(NULLIF(disc_number, 0), 1)"
	}
	resolvedTrackIDExpr := "0"
	if hasResolvedTrackID {
		resolvedTrackIDExpr = "COALESCE(resolved_track_id, 0)"
	}
	resolutionStatusExpr := "'pending'"
	if hasResolutionStatus {
		resolutionStatusExpr = "COALESCE(NULLIF(resolution_status, ''), 'pending')"
	}
	resolutionConfidenceExpr := "0"
	if hasResolutionConfidence {
		resolutionConfidenceExpr = "COALESCE(resolution_confidence, 0)"
	}
	libraryAppliedExpr := "0"
	if hasLibraryApplied {
		libraryAppliedExpr = "COALESCE(library_applied, 0)"
	}
	albumIDExpr := "0"
	if hasAlbumID {
		albumIDExpr = "COALESCE(album_id, 0)"
	}
	musicBrainzIDExpr := "''"
	if hasMusicBrainzID {
		musicBrainzIDExpr = "COALESCE(music_brainz_id, '')"
	}
	coverArtPathExpr := "''"
	if hasCoverArtPath {
		coverArtPathExpr = "COALESCE(cover_art_path, '')"
	}
	albumSubtitleExpr := "''"
	if hasAlbumSubtitle {
		albumSubtitleExpr = "COALESCE(album_subtitle, '')"
	}

	copySQL := fmt.Sprintf(
		`INSERT INTO track_play_records (
			artist, album_artist, album, album_subtitle, track, album_id, duration, play_time, scrobbled,
			track_number, disc_number, music_brainz_id, source, cover_art_path, resolved_track_id, resolution_status, resolution_confidence, library_applied, created_at, updated_at
		)
		SELECT
			artist, album_artist, album, %s, track, %s, duration, play_time, COALESCE(scrobbled, 0),
			%s, %s, %s, source, %s, %s, %s, %s, %s, created_at, updated_at
		FROM track_play_records_legacy_backup`,
		albumSubtitleExpr,
		albumIDExpr,
		trackNumberExpr,
		discNumberExpr,
		musicBrainzIDExpr,
		coverArtPathExpr,
		resolvedTrackIDExpr,
		resolutionStatusExpr,
		resolutionConfidenceExpr,
		libraryAppliedExpr,
	)
	if _, err := c.execWithRetry(ctx, copySQL); err != nil {
		return fmt.Errorf("failed to copy track_play_records data into new schema: %w", err)
	}
	if _, err := c.execWithRetry(ctx, "DROP TABLE track_play_records_legacy_backup"); err != nil {
		return fmt.Errorf("failed to drop track_play_records legacy table: %w", err)
	}
	return nil
}

func (c *D1Client) ensureTopAlbumStatSchema(ctx context.Context) error {
	exists, err := c.tableExists(ctx, "top_album_stat")
	if err != nil {
		return err
	}
	if !exists {
		if _, err := c.execWithRetry(
			ctx, `
		CREATE TABLE IF NOT EXISTS top_album_stat (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			period_days INTEGER NOT NULL,
			album_id INTEGER DEFAULT 0,
			album TEXT NOT NULL,
			album_subtitle TEXT,
			artist TEXT DEFAULT '',
			play_count INTEGER DEFAULT 0,
			rank INTEGER NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE (period_days, album, artist)
		)
	`,
		); err != nil {
			return fmt.Errorf("failed to create top_album_stat table: %w", err)
		}
	}

	hasAlbumID, err := c.tableHasColumn(ctx, "top_album_stat", "album_id")
	if err != nil {
		return err
	}
	if !hasAlbumID {
		if _, err := c.execWithRetry(ctx, "ALTER TABLE top_album_stat ADD COLUMN album_id INTEGER DEFAULT 0"); err != nil {
			return fmt.Errorf("failed to add album_id column to top_album_stat: %w", err)
		}
	}
	hasAlbumSubtitle, err := c.tableHasColumn(ctx, "top_album_stat", "album_subtitle")
	if err != nil {
		return err
	}
	if !hasAlbumSubtitle {
		if _, err := c.execWithRetry(ctx, "ALTER TABLE top_album_stat ADD COLUMN album_subtitle TEXT"); err != nil {
			return fmt.Errorf("failed to add album_subtitle column to top_album_stat: %w", err)
		}
	}

	if _, err := c.execWithRetry(ctx, "CREATE INDEX IF NOT EXISTS idx_top_album_period_rank ON top_album_stat(period_days, rank)"); err != nil {
		return fmt.Errorf("failed to ensure top_album_stat index: %w", err)
	}
	return nil
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

	db, err := otelsql.Open(d1DriverName, dsn, d1OpenOptions()...)
	if err != nil {
		return nil, fmt.Errorf("failed to open D1 connection: %w", err)
	}
	if config.ConfigObj.Telemetry.DBStatsMetricsEnabled {
		if err := coredb.RegisterDBStatsMetrics(db, "sqlite"); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to register D1 db stats metrics: %w", err)
		}
	}

	// 测试连接
	// D1 API error 7403: The given account is not valid or is not authorized to access this service
	// listing databases: listing databases (page 1): D1 API error 10000: Authentication error
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping D1 database: %w", err)
	}

	client := &D1Client{
		db:  db,
		cfg: cfg,
	}

	if err := client.ensureSchema(context.Background()); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ensure D1 schema: %w", err)
	}

	return client, nil
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
	// 14 params per row -> floor(31/14)=2
	batchSize := batchSizeByParams(d1TrackUpsertFields)
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
	const numFields = d1TrackUpsertFields
	placeholders := make([]string, len(tracks))
	args := make([]interface{}, 0, len(tracks)*numFields)

	for i, track := range tracks {
		placeholders[i] = "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
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
			track.TrackNumber,
			track.DiscNumber,
			boolToInt(track.IsAppleMusicFav),
			boolToInt(track.IsLastFmFav),
			track.CreatedAt.Format(time.RFC3339),
			track.UpdatedAt.Format(time.RFC3339),
		)
	}

	query := fmt.Sprintf(
		`
		INSERT OR REPLACE INTO tracks (
			artist, album, track, album_artist, play_count, genre, duration, source, track_number, disc_number,
			is_apple_music_fav, is_last_fm_fav, created_at, updated_at
		) VALUES %s
	`, strings.Join(placeholders, ", "),
	)

	if _, err := c.execWithRetry(ctx, query, args...); err != nil {
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
	// 18 params per row -> floor(31/18)=1
	batchSize := batchSizeByParams(d1PlayRecordUpsertFields)
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

	const numFields = d1PlayRecordUpsertFields
	placeholders := make([]string, len(records))
	args := make([]interface{}, 0, len(records)*numFields)

	for i, record := range records {
		placeholders[i] = "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
		args = append(
			args,
			record.Artist,
			record.AlbumArtist,
			record.Album,
			record.AlbumSubtitle,
			record.Track,
			record.AlbumID,
			record.Duration,
			record.PlayTime.Format(time.RFC3339),
			boolToInt(record.Scrobbled),
			record.TrackNumber,
			record.DiscNumber,
			record.MusicBrainzID,
			record.Source,
			record.CoverArtPath,
			record.ResolvedTrackID,
			record.ResolutionStatus,
			record.ResolutionConfidence,
			boolToInt(record.LibraryApplied),
			record.CreatedAt.Format(time.RFC3339),
			record.UpdatedAt.Format(time.RFC3339),
		)
	}

	query := fmt.Sprintf(
		`
		INSERT OR REPLACE INTO track_play_records (
			artist, album_artist, album, album_subtitle, track, album_id, duration, play_time, scrobbled, track_number, disc_number, music_brainz_id, source, cover_art_path, resolved_track_id, resolution_status, resolution_confidence, library_applied, created_at, updated_at
		) VALUES %s
	`, strings.Join(placeholders, ", "),
	)

	if _, err := c.execWithRetry(ctx, query, args...); err != nil {
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
		if _, err := c.execWithRetry(ctx, query, lastSyncAt); err != nil {
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
		if _, err := c.execWithRetry(ctx, "DELETE FROM "+table); err != nil {
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
	_, err := c.execWithRetry(ctx, query, args...)
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
	args := make([]interface{}, 0, len(rows)*3)
	for i, row := range rows {
		placeholders[i] = "(?, ?, ?)"
		args = append(args, row.Source, row.Count, row.UpdatedAt.Format(time.RFC3339))
	}
	query := fmt.Sprintf(
		"INSERT OR REPLACE INTO play_source_stat (source, count, updated_at) VALUES %s",
		strings.Join(placeholders, ", "),
	)
	_, err := c.execWithRetry(ctx, query, args...)
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
	_, err := c.execWithRetry(ctx, query, args...)
	return err
}

func (c *D1Client) batchUpsertTopAlbumStats(ctx context.Context, rows []*model.TopAlbumStat) error {
	if len(rows) == 0 {
		return nil
	}
	// 7 params per row -> floor(31/7)=4
	batchSize := batchSizeByParams(d1TopAlbumUpsertFields)
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
	args := make([]interface{}, 0, len(rows)*d1TopAlbumUpsertFields)
	for i, row := range rows {
		placeholders[i] = "(?, ?, ?, ?, ?, ?, ?, ?)"
		args = append(args, row.PeriodDays, row.AlbumID, row.Album, row.AlbumSubtitle, row.Artist, row.PlayCount, row.Rank, row.UpdatedAt.Format(time.RFC3339))
	}
	query := fmt.Sprintf(
		"INSERT OR REPLACE INTO top_album_stat (period_days, album_id, album, album_subtitle, artist, play_count, rank, updated_at) VALUES %s",
		strings.Join(placeholders, ", "),
	)
	_, err := c.execWithRetry(ctx, query, args...)
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
	_, err := c.execWithRetry(ctx, query, args...)
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
	_, err := c.execWithRetry(ctx, query, args...)
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
	_, err := c.execWithRetry(ctx, query, args...)
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

	if _, err := c.execWithRetry(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to batch upsert genres: %w", err)
	}
	return nil
}

// getLastSyncTime 获取最后同步时间
func (c *D1Client) getLastSyncTime(ctx context.Context, tableName string) (time.Time, error) {
	var lastSyncTimeStr string
	err := c.db.QueryRowContext(
		ctx,
		"SELECT COALESCE(MAX(last_sync_time), '') FROM sync_metadata WHERE table_name = ?",
		tableName,
	).Scan(&lastSyncTimeStr)

	if errors.Is(err, sql.ErrNoRows) || lastSyncTimeStr == "" {
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

	_, err := c.execWithRetry(
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

func (c *D1Client) tryBeginSync() bool {
	return atomic.CompareAndSwapInt32(&c.syncRunning, 0, 1)
}

func (c *D1Client) endSync() {
	atomic.StoreInt32(&c.syncRunning, 0)
}

// SyncAll 同步所有数据
func (c *D1Client) SyncAll(ctx context.Context, incremental bool) error {
	if !c.tryBeginSync() {
		return ErrD1SyncAlreadyRunning
	}
	defer c.endSync()

	log.Info(ctx, "Starting D1 sync", zap.Bool("incremental", incremental))

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

	log.Info(ctx, "D1 sync completed successfully", zap.Bool("incremental", incremental))
	return nil
}

var ErrD1SyncAlreadyRunning = fmt.Errorf("D1 sync already running")
