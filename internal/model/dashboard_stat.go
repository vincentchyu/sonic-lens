package model

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/config"
	artworkcore "github.com/vincentchyu/sonic-lens/core/artwork"
	"github.com/vincentchyu/sonic-lens/core/objectstorage"
)

const (
	defaultDashboardStatRefreshMinutes  = 10
	defaultDashboardHeavyRefreshMinutes = 60
	defaultDashboardTopN                = 10
	defaultDashboardTrendDays           = 180
)

var dashboardAlbumPeriods = []int{7, 30, 90, 365}

var dashboardTrackRankObjectStorageGet = objectstorage.Get
var dashboardArtistProfileObjectStorageGet = objectstorage.Get

type DashboardStat struct {
	ID          int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	TotalPlays  int64     `gorm:"column:total_plays;type:bigint;default:0" json:"total_plays"`
	TotalTracks int64     `gorm:"column:total_tracks;type:bigint;default:0" json:"total_tracks"`
	TotalArtist int64     `gorm:"column:total_artist;type:bigint;default:0" json:"total_artist"`
	TotalAlbums int64     `gorm:"column:total_albums;type:bigint;default:0" json:"total_albums"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

func (DashboardStat) TableName() string {
	return "dashboard_stat"
}

type PlaySourceStat struct {
	ID        int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	Source    string    `gorm:"column:source;type:varchar(100);not null;uniqueIndex:uk_source" json:"source"`
	Count     int64     `gorm:"column:count;type:bigint;default:0" json:"count"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

func (PlaySourceStat) TableName() string {
	return "play_source_stat"
}

type TopArtistStat struct {
	ID          int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	PeriodDays  int       `gorm:"column:period_days;type:int;not null;default:0" json:"period_days"`
	MetricType  string    `gorm:"column:metric_type;type:varchar(20);not null" json:"metric_type"`
	Artist      string    `gorm:"column:artist;type:varchar(255);not null" json:"artist"`
	MetricValue int64     `gorm:"column:metric_value;type:bigint;default:0" json:"metric_value"`
	Rank        int       `gorm:"column:rank;type:int;not null" json:"rank"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

func (TopArtistStat) TableName() string {
	return "top_artist_stat"
}

type TopAlbumStat struct {
	ID            int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	PeriodDays    int       `gorm:"column:period_days;type:int;not null" json:"period_days"`
	AlbumID       int64     `gorm:"column:album_id;type:bigint;index" json:"album_id"` // 新增关联
	Album         string    `gorm:"column:album;type:varchar(255);not null" json:"album"`
	AlbumSubtitle string    `gorm:"column:album_subtitle;type:varchar(255)" json:"album_subtitle"`
	Artist        string    `gorm:"column:artist;type:varchar(255);default:''" json:"artist"`
	PlayCount     int64     `gorm:"column:play_count;type:bigint;default:0" json:"play_count"`
	Rank          int       `gorm:"column:rank;type:int;not null" json:"rank"`
	UpdatedAt     time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

func (TopAlbumStat) TableName() string {
	return "top_album_stat"
}

type TopGenreStat struct {
	ID              int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	GenreName       string    `gorm:"column:genre_name;type:varchar(255);not null;uniqueIndex:uk_genre_name" json:"genre_name"`
	GenreNameZh     string    `gorm:"column:genre_name_zh;type:varchar(255);default:''" json:"genre_name_zh"`
	TrackGenreCount int64     `gorm:"column:track_genre_count;type:bigint;default:0" json:"track_genre_count"`
	GenreCount      int64     `gorm:"column:genre_count;type:bigint;default:0" json:"genre_count"`
	Rank            int       `gorm:"column:rank;type:int;not null" json:"rank"`
	UpdatedAt       time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

func (TopGenreStat) TableName() string {
	return "top_genre_stat"
}

type PlayTrendDailyStat struct {
	StatDate  time.Time `gorm:"column:stat_date;type:date;primaryKey" json:"stat_date"`
	PlayCount int64     `gorm:"column:play_count;type:bigint;default:0" json:"play_count"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

func (PlayTrendDailyStat) TableName() string {
	return "play_trend_daily_stat"
}

type PlayTrendHourlyStat struct {
	StatDate  time.Time `gorm:"column:stat_date;type:date;primaryKey" json:"stat_date"`
	Hour      int       `gorm:"column:hour;type:tinyint;primaryKey" json:"hour"`
	PlayCount int64     `gorm:"column:play_count;type:bigint;default:0" json:"play_count"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

func (PlayTrendHourlyStat) TableName() string {
	return "play_trend_hourly_stat"
}

type TrackRankStat struct {
	ID                int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	PeriodType        string    `gorm:"column:period_type;type:varchar(20);not null" json:"period_type"`
	TrackID           int64     `gorm:"column:track_id;type:bigint;default:0;index:idx_track_rank_track_id" json:"track_id"`
	Artist            string    `gorm:"column:artist;type:varchar(255);not null" json:"artist"`
	Album             string    `gorm:"column:album;type:varchar(255);not null" json:"album"`
	Track             string    `gorm:"column:track;type:varchar(255);not null" json:"track"`
	TrackNumber       int8      `gorm:"column:track_number;type:tinyint" json:"track_number"`
	DiscNumber        int8      `gorm:"column:disc_number;type:tinyint" json:"disc_number"`
	PlayCount         int64     `gorm:"column:play_count;type:bigint;default:0" json:"play_count"`
	Rank              int       `gorm:"column:rank;type:int;not null" json:"rank"`
	CoverArtURL       string    `gorm:"column:cover_art_url;type:varchar(1024)" json:"cover_art_url"`
	CoverArtMime      string    `gorm:"column:cover_art_mime;type:varchar(128)" json:"cover_art_mime"`
	CoverArtObjectKey string    `gorm:"column:cover_art_object_key;type:varchar(512)" json:"cover_art_object_key"`
	UpdatedAt         time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

func (TrackRankStat) TableName() string {
	return "track_rank_stat"
}

type DashboardStatRuntimeConfig struct {
	Enabled              bool
	IntervalMinutes      int
	HeavyIntervalMinutes int
	HeavyOnlyOnNewPlay   bool
	TopN                 int
	TrendDays            int
	HourlyTrendDays      int
}

func GetDashboardStatRuntimeConfig() DashboardStatRuntimeConfig {
	cfg := config.ConfigObj.Dashboard

	runtimeCfg := DashboardStatRuntimeConfig{
		Enabled:            cfg.StatRefreshEnabled,
		HeavyOnlyOnNewPlay: cfg.HeavyStatOnlyOnNewPlay,
	}
	if runtimeCfg.IntervalMinutes = cfg.StatRefreshIntervalMinutes; runtimeCfg.IntervalMinutes <= 0 {
		runtimeCfg.IntervalMinutes = defaultDashboardStatRefreshMinutes
	}
	if runtimeCfg.HeavyIntervalMinutes = cfg.HeavyStatRefreshIntervalMinutes; runtimeCfg.HeavyIntervalMinutes <= 0 {
		runtimeCfg.HeavyIntervalMinutes = defaultDashboardHeavyRefreshMinutes
	}
	if runtimeCfg.HeavyIntervalMinutes < runtimeCfg.IntervalMinutes {
		runtimeCfg.HeavyIntervalMinutes = runtimeCfg.IntervalMinutes
	}
	if runtimeCfg.TopN = cfg.TopN; runtimeCfg.TopN <= 0 {
		runtimeCfg.TopN = defaultDashboardTopN
	}
	if runtimeCfg.TrendDays = cfg.TrendDays; runtimeCfg.TrendDays <= 0 {
		runtimeCfg.TrendDays = defaultDashboardTrendDays
	}
	if runtimeCfg.HourlyTrendDays = cfg.HourlyTrendDays; runtimeCfg.HourlyTrendDays <= 0 {
		// 未显式配置时，小时趋势窗口与日趋势窗口保持一致，避免口径不一致。
		runtimeCfg.HourlyTrendDays = runtimeCfg.TrendDays
	}
	if runtimeCfg.HourlyTrendDays > runtimeCfg.TrendDays {
		runtimeCfg.HourlyTrendDays = runtimeCfg.TrendDays
	}

	return runtimeCfg
}

func GetDashboardStatConfig() (enabled bool, intervalMinutes int, topN int, trendDays int) {
	runtimeCfg := GetDashboardStatRuntimeConfig()
	return runtimeCfg.Enabled, runtimeCfg.IntervalMinutes, runtimeCfg.TopN, runtimeCfg.TrendDays
}

func EnsureDashboardStatSchema(ctx context.Context) error {
	db := GetDB().WithContext(ctx)
	migrator := db.Migrator()

	if err := ensureTableAndColumns(
		migrator, &DashboardStat{},
		[]string{"ID", "TotalPlays", "TotalTracks", "TotalArtist", "TotalAlbums", "UpdatedAt"},
	); err != nil {
		return err
	}
	if err := ensureTableAndColumns(
		migrator, &PlaySourceStat{}, []string{"ID", "Source", "Count", "UpdatedAt"},
	); err != nil {
		return err
	}
	if err := ensureTableAndColumns(
		migrator, &TopArtistStat{},
		[]string{"ID", "PeriodDays", "MetricType", "Artist", "MetricValue", "Rank", "UpdatedAt"},
	); err != nil {
		return err
	}
	if err := ensureTableAndColumns(
		migrator, &ArtistProfile{},
		[]string{
			"ID", "ArtistName", "NormalizedArtistKey",
			"AvatarURL", "AvatarMime", "AvatarObjectKey",
			"CreatedAt", "UpdatedAt",
		},
	); err != nil {
		return err
	}
	if err := ensureTableAndColumns(
		migrator, &TopAlbumStat{},
		[]string{"ID", "PeriodDays", "AlbumID", "Album", "Artist", "PlayCount", "Rank", "UpdatedAt"},
	); err != nil {
		return err
	}
	if err := ensureTableAndColumns(
		migrator, &TopGenreStat{},
		[]string{"ID", "GenreName", "GenreNameZh", "TrackGenreCount", "GenreCount", "Rank", "UpdatedAt"},
	); err != nil {
		return err
	}
	if err := ensureTableAndColumns(
		migrator, &PlayTrendDailyStat{}, []string{"StatDate", "PlayCount", "UpdatedAt"},
	); err != nil {
		return err
	}
	if err := ensureTableAndColumns(
		migrator, &PlayTrendHourlyStat{}, []string{"StatDate", "Hour", "PlayCount", "UpdatedAt"},
	); err != nil {
		return err
	}
	if err := ensureTableAndColumns(
		migrator, &TrackRankStat{},
		[]string{
			"ID", "PeriodType", "TrackID", "Artist", "Album", "Track",
			"TrackNumber", "DiscNumber", "PlayCount", "Rank",
			"CoverArtURL", "CoverArtMime", "CoverArtObjectKey", "UpdatedAt",
		},
	); err != nil {
		return err
	}
	if err := ensureTrackRankStatIndexes(ctx); err != nil {
		return err
	}
	if err := ensureArtistProfileIndexes(ctx); err != nil {
		return err
	}
	/*if err := ensureDefaultArtistProfiles(ctx); err != nil {
		return err
	}*/

	return nil
}

func EnsureArtistProfileSchema(ctx context.Context) error {
	db := GetDB().WithContext(ctx)
	migrator := db.Migrator()

	if err := ensureTableAndColumns(
		migrator, &ArtistProfile{},
		[]string{
			"ID", "ArtistName", "NormalizedArtistKey",
			"AvatarURL", "AvatarMime", "AvatarObjectKey",
			"CreatedAt", "UpdatedAt",
		},
	); err != nil {
		return err
	}
	if err := ensureArtistProfileIndexes(ctx); err != nil {
		return err
	}
	/*if err := ensureDefaultArtistProfiles(ctx); err != nil {
		return err
	}*/

	return nil
}

func ensureTableAndColumns(migrator gorm.Migrator, model interface{}, columns []string) error {
	if !migrator.HasTable(model) {
		if err := migrator.CreateTable(model); err != nil {
			return err
		}
	}
	for _, col := range columns {
		if !migrator.HasColumn(model, col) {
			if err := migrator.AddColumn(model, col); err != nil {
				return err
			}
		}
	}
	return nil
}

func RefreshDashboardStats(ctx context.Context) error {
	runtimeCfg := GetDashboardStatRuntimeConfig()
	if err := RefreshDashboardStatsLight(ctx); err != nil {
		return err
	}
	return refreshDashboardStatsHeavyWithOptions(ctx, runtimeCfg.TopN, runtimeCfg.TrendDays, runtimeCfg.HourlyTrendDays)
}

func RefreshDashboardStatsLight(ctx context.Context) error {
	if err := EnsureDashboardStatSchema(ctx); err != nil {
		return err
	}
	cfg := GetDashboardStatRuntimeConfig()
	if err := refreshDashboardStatsLightOnly(ctx, cfg.TopN); err != nil {
		if isLegacyDashboardSchemaError(err) {
			if rebuildErr := rebuildBrokenDashboardStatTable(ctx, err); rebuildErr != nil {
				return fmt.Errorf("rebuild broken dashboard stat table failed: %w", rebuildErr)
			}
			return refreshDashboardStatsLightOnly(ctx, cfg.TopN)
		}
		return err
	}
	return nil
}

func RefreshDashboardStatsHeavy(ctx context.Context) error {
	cfg := GetDashboardStatRuntimeConfig()
	return refreshDashboardStatsHeavyWithOptions(ctx, cfg.TopN, cfg.TrendDays, cfg.HourlyTrendDays)
}

func isLegacyDashboardSchemaError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unknown column") ||
		strings.Contains(msg, "doesn't have a default value") ||
		strings.Contains(msg, "doesn't exist") ||
		strings.Contains(msg, "no such table")
}

func rebuildBrokenDashboardStatTable(ctx context.Context, rootErr error) error {
	db := GetDB().WithContext(ctx)
	migrator := db.Migrator()
	msg := strings.ToLower(rootErr.Error())

	type tableModel struct {
		name  string
		model interface{}
	}
	tables := []tableModel{
		{name: "dashboard_stat", model: &DashboardStat{}},
		{name: "play_source_stat", model: &PlaySourceStat{}},
		{name: "top_artist_stat", model: &TopArtistStat{}},
		{name: "top_album_stat", model: &TopAlbumStat{}},
		{name: "top_genre_stat", model: &TopGenreStat{}},
		{name: "play_trend_daily_stat", model: &PlayTrendDailyStat{}},
		{name: "play_trend_hourly_stat", model: &PlayTrendHourlyStat{}},
		{name: "track_rank_stat", model: &TrackRankStat{}},
	}

	for _, t := range tables {
		if strings.Contains(msg, t.name) {
			if err := migrator.DropTable(t.model); err != nil {
				return err
			}
			return db.AutoMigrate(t.model)
		}
	}

	// 无法定位具体表时，不做任何 destructive 操作
	return rootErr
}

func refreshDashboardStatsLightOnly(ctx context.Context, topN int) error {
	periods := []string{"all"}
	db := GetDB().WithContext(ctx)
	if err := db.Transaction(
		func(tx *gorm.DB) error {
			if err := refreshDashboardOverview(tx); err != nil {
				return err
			}
			if err := refreshPlaySourceStats(tx); err != nil {
				return err
			}
			if err := refreshTrackRankStats(tx, topN, periods); err != nil {
				return err
			}
			return nil
		},
	); err != nil {
		return err
	}
	return backfillTrackRankStatArtwork(ctx, periods)
}

func refreshDashboardStatsHeavyWithOptions(ctx context.Context, topN, trendDays, hourlyTrendDays int) error {
	if err := EnsureDashboardStatSchema(ctx); err != nil {
		return err
	}
	periods := []string{"all", "week", "month"}
	db := GetDB().WithContext(ctx)
	err := db.Transaction(
		func(tx *gorm.DB) error {
			if err := refreshTopArtistStats(tx, topN); err != nil {
				return err
			}
			if err := refreshTopAlbumStats(tx, topN); err != nil {
				return err
			}
			if err := refreshTopGenreStats(tx, topN); err != nil {
				return err
			}
			if err := refreshTrendStats(tx, trendDays, hourlyTrendDays); err != nil {
				return err
			}
			if err := refreshTrackRankStats(tx, topN, periods); err != nil {
				return err
			}
			return nil
		},
	)
	if err != nil && isLegacyDashboardSchemaError(err) {
		if rebuildErr := rebuildBrokenDashboardStatTable(ctx, err); rebuildErr != nil {
			return fmt.Errorf("rebuild broken dashboard stat table failed: %w", rebuildErr)
		}
		return db.Transaction(
			func(tx *gorm.DB) error {
				if err := refreshTopArtistStats(tx, topN); err != nil {
					return err
				}
				if err := refreshTopAlbumStats(tx, topN); err != nil {
					return err
				}
				if err := refreshTopGenreStats(tx, topN); err != nil {
					return err
				}
				if err := refreshTrendStats(tx, trendDays, hourlyTrendDays); err != nil {
					return err
				}
				if err := refreshTrackRankStats(tx, topN, periods); err != nil {
					return err
				}
				return nil
			},
		)
	}
	if err != nil {
		return err
	}
	return backfillTrackRankStatArtwork(ctx, periods)
}

func refreshDashboardOverview(tx *gorm.DB) error {
	var totalPlays int64
	var totalTracks int64
	var totalArtists int64
	var totalAlbums int64

	if err := tx.Model(&Track{}).Select("COALESCE(SUM(play_count), 0)").Scan(&totalPlays).Error; err != nil {
		return err
	}
	if err := tx.Model(&Track{}).Count(&totalTracks).Error; err != nil {
		return err
	}
	if err := tx.Model(&Track{}).Distinct("artist").Count(&totalArtists).Error; err != nil {
		return err
	}
	if err := tx.Model(&Track{}).Distinct("album").Count(&totalAlbums).Error; err != nil {
		return err
	}

	row := &DashboardStat{
		ID:          1,
		TotalPlays:  totalPlays,
		TotalTracks: totalTracks,
		TotalArtist: totalArtists,
		TotalAlbums: totalAlbums,
	}
	return tx.Clauses(
		clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns(
				[]string{
					"total_plays", "total_tracks", "total_artist", "total_albums", "updated_at",
				},
			),
		},
	).Create(row).Error
}

func refreshPlaySourceStats(tx *gorm.DB) error {
	type sourceRow struct {
		Source string
		Count  int64
	}
	var rows []sourceRow
	if err := tx.Model(&TrackPlayRecord{}).
		Select("source, COUNT(*) as count").
		Group("source").
		Find(&rows).Error; err != nil {
		return err
	}

	if err := tx.Where("1 = 1").Delete(&PlaySourceStat{}).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	items := make([]PlaySourceStat, 0, len(rows))
	for _, row := range rows {
		if row.Source == "" {
			continue
		}
		items = append(items, PlaySourceStat{Source: row.Source, Count: row.Count})
	}
	if len(items) == 0 {
		return nil
	}
	return tx.Create(&items).Error
}

func refreshTopArtistStats(tx *gorm.DB, topN int) error {
	type topArtistRow struct {
		Artist      string
		MetricValue int64
	}

	loadAndStore := func(metricType, selectSQL, orderSQL string) error {
		var rows []topArtistRow
		if err := tx.Model(&Track{}).
			Select(selectSQL).
			Group("artist").
			Order(orderSQL).
			Limit(topN).
			Find(&rows).Error; err != nil {
			return err
		}

		if err := tx.Where(
			"period_days = ? AND metric_type = ?", 0, metricType,
		).Delete(&TopArtistStat{}).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}

		items := make([]TopArtistStat, 0, len(rows))
		for i, row := range rows {
			if row.Artist == "" {
				continue
			}
			items = append(
				items,
				TopArtistStat{
					PeriodDays:  0,
					MetricType:  metricType,
					Artist:      row.Artist,
					MetricValue: row.MetricValue,
					Rank:        i + 1,
				},
			)
		}
		if len(items) == 0 {
			return nil
		}
		return tx.Create(&items).Error
	}

	if err := loadAndStore("plays", "artist, SUM(play_count) as metric_value", "metric_value DESC"); err != nil {
		return err
	}
	if err := loadAndStore("tracks", "artist, COUNT(*) as metric_value", "metric_value DESC"); err != nil {
		return err
	}
	return nil
}

func refreshTopAlbumStats(tx *gorm.DB, topN int) error {
	type topAlbumRow struct {
		AlbumID       int64
		Album         string
		AlbumSubtitle string
		Artist        string
		PlayCount     int64
	}
	now := time.Now()

	for _, days := range dashboardAlbumPeriods {
		startTime := now.AddDate(0, 0, -days)
		var rows []topAlbumRow
		if err := tx.Table("track_play_records AS tpr").
			Where("play_time >= ?", startTime).
			Select(
				"tpr.album_id, MIN(tpr.album) as album, MIN(COALESCE(a.name_subtitle, tpr.album_subtitle, '')) as album_subtitle, MIN(tpr.artist) as artist, COUNT(*) as play_count",
			).
			Joins("LEFT JOIN album AS a ON a.id = tpr.album_id").
			Group("tpr.album_id").
			Having("album_id > 0").
			Order("play_count DESC").
			Limit(topN).
			Find(&rows).Error; err != nil {
			return err
		}

		if err := tx.Where("period_days = ?", days).Delete(&TopAlbumStat{}).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			continue
		}

		items := make([]TopAlbumStat, 0, len(rows))
		for i, row := range rows {
			items = append(
				items,
				TopAlbumStat{
					PeriodDays:    days,
					AlbumID:       row.AlbumID,
					Album:         row.Album,
					AlbumSubtitle: row.AlbumSubtitle,
					Artist:        row.Artist,
					PlayCount:     row.PlayCount,
					Rank:          i + 1,
				},
			)
		}
		if len(items) == 0 {
			continue
		}
		if err := tx.Create(&items).Error; err != nil {
			return err
		}
	}
	return nil
}

func refreshTopGenreStats(tx *gorm.DB, topN int) error {
	type topGenreRow struct {
		GenreName       string
		TrackGenreCount int64
		GenreNameZh     string
		GenreCount      int64
	}
	var rows []topGenreRow
	if err := tx.Table("track as t").
		Select(
			"t.genre as genre_name, COALESCE(SUM(t.play_count), 0) as track_genre_count, " +
				"COALESCE(g.name_zh, '') as genre_name_zh, COALESCE(g.play_count, 0) as genre_count",
		).
		Joins("LEFT JOIN genre as g ON t.genre = g.name").
		Where("t.genre != ''").
		Group("t.genre, g.name_zh, g.play_count").
		Order("track_genre_count DESC").
		Limit(topN).
		Find(&rows).Error; err != nil {
		return err
	}

	if err := tx.Where("1 = 1").Delete(&TopGenreStat{}).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	items := make([]TopGenreStat, 0, len(rows))
	for i, row := range rows {
		if row.GenreName == "" {
			continue
		}
		items = append(
			items,
			TopGenreStat{
				GenreName:       row.GenreName,
				GenreNameZh:     row.GenreNameZh,
				TrackGenreCount: row.TrackGenreCount,
				GenreCount:      row.GenreCount,
				Rank:            i + 1,
			},
		)
	}
	if len(items) == 0 {
		return nil
	}
	return tx.Create(&items).Error
}

func refreshTrendStats(tx *gorm.DB, trendDays, hourlyTrendDays int) error {
	now := time.Now()
	startDaily := startOfDay(now.AddDate(0, 0, -trendDays))
	startHourly := startOfDay(now.AddDate(0, 0, -hourlyTrendDays))

	type dailyRow struct {
		StatDate  string
		PlayCount int64
	}
	type hourlyRow struct {
		StatDate  string
		Hour      int
		PlayCount int64
	}

	var dailyRows []dailyRow
	var hourlyRows []hourlyRow
	var dailySQL string
	var hourlySQL string

	if config.ConfigObj.Database.Type == string(common.DatabaseTypeMySQL) {
		dailySQL = "SELECT DATE_FORMAT(play_time, '%Y-%m-%d') as stat_date, COUNT(*) as play_count FROM track_play_records WHERE play_time >= ? GROUP BY DATE_FORMAT(play_time, '%Y-%m-%d')"
		hourlySQL = "SELECT DATE_FORMAT(play_time, '%Y-%m-%d') as stat_date, HOUR(play_time) as hour, COUNT(*) as play_count FROM track_play_records WHERE play_time >= ? GROUP BY DATE_FORMAT(play_time, '%Y-%m-%d'), HOUR(play_time)"
	} else {
		dailySQL = "SELECT date(play_time) as stat_date, COUNT(*) as play_count FROM track_play_records WHERE play_time >= ? GROUP BY date(play_time)"
		hourlySQL = "SELECT date(play_time) as stat_date, CAST(strftime('%H', play_time) as integer) as hour, COUNT(*) as play_count FROM track_play_records WHERE play_time >= ? GROUP BY date(play_time), strftime('%H', play_time)"
	}

	if err := tx.Raw(dailySQL, startDaily).Scan(&dailyRows).Error; err != nil {
		return err
	}
	if err := tx.Raw(hourlySQL, startHourly).Scan(&hourlyRows).Error; err != nil {
		return err
	}

	if err := tx.Where("1 = 1").Delete(&PlayTrendDailyStat{}).Error; err != nil {
		return err
	}
	if err := tx.Where("1 = 1").Delete(&PlayTrendHourlyStat{}).Error; err != nil {
		return err
	}

	dailyItems := make([]PlayTrendDailyStat, 0, len(dailyRows))
	for _, row := range dailyRows {
		d, err := parseDateOnly(row.StatDate)
		if err != nil {
			return err
		}
		dailyItems = append(dailyItems, PlayTrendDailyStat{StatDate: d, PlayCount: row.PlayCount})
	}
	if len(dailyItems) > 0 {
		if err := tx.Create(&dailyItems).Error; err != nil {
			return err
		}
	}

	hourlyItems := make([]PlayTrendHourlyStat, 0, len(hourlyRows))
	for _, row := range hourlyRows {
		d, err := parseDateOnly(row.StatDate)
		if err != nil {
			return err
		}
		h := row.Hour
		if h < 0 {
			h = 0
		}
		if h > 23 {
			h = 23
		}
		hourlyItems = append(hourlyItems, PlayTrendHourlyStat{StatDate: d, Hour: h, PlayCount: row.PlayCount})
	}
	if len(hourlyItems) > 0 {
		if err := tx.Create(&hourlyItems).Error; err != nil {
			return err
		}
	}

	return nil
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func HasNewTrackPlayRecordsSince(ctx context.Context, since time.Time) (bool, time.Time, error) {
	var latest time.Time
	if err := GetDB().WithContext(ctx).
		Model(&TrackPlayRecord{}).
		Select("MAX(updated_at)").
		Scan(&latest).Error; err != nil {
		return false, time.Time{}, err
	}

	if latest.IsZero() {
		return false, time.Time{}, nil
	}
	if since.IsZero() {
		return true, latest, nil
	}
	return latest.After(since), latest, nil
}

func normalizeTrackRankPeriod(period string) string {
	switch strings.ToLower(strings.TrimSpace(period)) {
	case "week":
		return "week"
	case "month":
		return "month"
	default:
		return "all"
	}
}

func normalizeTrackPeriodByDays(days int) string {
	switch normalizeAlbumPeriod(days) {
	case 7:
		return "week"
	case 30, 90:
		return "month"
	default:
		return "all"
	}
}

func GetTrackPlayCountsFromStat(ctx context.Context, period string, limit, offset int, keyword string) (
	[]*Track, error,
) {
	normalized := normalizeTrackRankPeriod(period)
	var rows []*TrackRankStat

	query := GetDB().WithContext(ctx).Model(&TrackRankStat{}).Where("period_type = ?", normalized)
	if keyword != "" {
		if config.ConfigObj.Database.Type == string(common.DatabaseTypeMySQL) {
			query = query.Where("MATCH(track, artist, album) AGAINST(? IN BOOLEAN MODE)", keyword)
		} else {
			kw := "%" + keyword + "%"
			query = query.Where("track LIKE ? OR artist LIKE ? OR album LIKE ?", kw, kw, kw)
		}
	}
	if err := query.Order("`rank` ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]*Track, 0, len(rows))
	for _, row := range rows {
		result = append(
			result,
			&Track{
				Artist:      row.Artist,
				Album:       row.Album,
				Track:       row.Track,
				TrackNumber: row.TrackNumber,
				DiscNumber:  row.DiscNumber,
				PlayCount:   int(row.PlayCount),
			},
		)
	}
	return result, nil
}

func GetTopTracksByPlayCountFromStat(ctx context.Context, days int, limit int) ([]*TopTrack, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	period := normalizeTrackPeriodByDays(days)

	type topTrackRow struct {
		TrackID           int64
		Track             string
		Album             string
		Artist            string
		PlayCount         int64
		Rank              int
		CoverArtURL       string
		CoverArtMime      string
		CoverArtObjectKey string
	}

	var rows []topTrackRow
	err := GetDB().WithContext(ctx).
		Model(&TrackRankStat{}).
		Select("track_id, track, album, artist, play_count, `rank`, cover_art_url, cover_art_mime, cover_art_object_key").
		Where("period_type = ?", period).
		Where("TRIM(COALESCE(track, '')) <> ''").
		Order("play_count DESC, `rank` ASC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]*TopTrack, 0, len(rows))
	for index, row := range rows {
		rank := row.Rank
		if rank <= 0 {
			rank = index + 1
		}
		result = append(
			result, &TopTrack{
				TrackID:           row.TrackID,
				Track:             row.Track,
				Album:             row.Album,
				Artist:            row.Artist,
				PlayCount:         int(row.PlayCount),
				Rank:              rank,
				CoverArtURL:       row.CoverArtURL,
				CoverArtMime:      row.CoverArtMime,
				CoverArtObjectKey: row.CoverArtObjectKey,
			},
		)
	}
	return result, nil
}

func refreshTrackRankStats(tx *gorm.DB, topN int, periods []string) error {
	topN = topN * 10
	type aggRow struct {
		TrackID           int64  `gorm:"column:track_id;type:bigint" json:"track_id"`
		Artist            string `gorm:"column:artist;type:varchar(255);not null" json:"artist"`
		Album             string `gorm:"column:album;type:varchar(255);not null" json:"album"`
		Track             string `gorm:"column:track;type:varchar(255);not null" json:"track"`
		TrackNumber       int8   `gorm:"column:track_number;type:tinyint" json:"track_number"`
		DiscNumber        int8   `gorm:"column:disc_number;type:tinyint" json:"disc_number"`
		PlayCount         int64  `gorm:"column:play_count;type:bigint;default:0" json:"play_count"`
		CoverArtURL       string `gorm:"column:cover_art_url;type:varchar(1024)" json:"cover_art_url"`
		CoverArtMime      string `gorm:"column:cover_art_mime;type:varchar(128)" json:"cover_art_mime"`
		CoverArtObjectKey string `gorm:"column:cover_art_object_key;type:varchar(512)" json:"cover_art_object_key"`
	}

	for _, p := range periods {
		period := normalizeTrackRankPeriod(p)
		var rows []aggRow

		switch period {
		case "all":
			if err := tx.Table("track AS t").
				Select(
					"t.id AS track_id, t.artist, t.album, t.track, t.track_number, t.disc_number, t.play_count, " +
						"COALESCE(aa.cover_art_url, ab.cover_art_url, '') AS cover_art_url, " +
						"COALESCE(aa.cover_art_mime, ab.cover_art_mime, '') AS cover_art_mime, " +
						"COALESCE(aa.cover_art_object_key, ab.cover_art_object_key, '') AS cover_art_object_key",
				).
				Joins(
					"LEFT JOIN track_album AS ta ON ta.id = (" +
						"SELECT ta2.id FROM track_album AS ta2 " +
						"WHERE ta2.track_id = t.id AND ta2.track_number = t.track_number AND ta2.disc_number = t.disc_number " +
						"ORDER BY ta2.id ASC LIMIT 1" +
						")",
				).
				Joins("LEFT JOIN album AS aa ON aa.id = ta.album_id").
				Joins(
					"LEFT JOIN (" +
						"SELECT artist, name, MIN(id) AS id FROM album GROUP BY artist, name" +
						") AS ac ON ac.artist = t.artist AND ac.name = t.album",
				).
				Joins("LEFT JOIN album AS ab ON ab.id = ac.id").
				Order("t.play_count DESC").
				Limit(topN).
				Find(&rows).Error; err != nil {
				return err
			}
		case "week", "month":
			startTime := time.Now().AddDate(0, 0, -7)
			if period == "month" {
				startTime = time.Now().AddDate(0, -1, 0)
			}
			if err := tx.Table("track_play_records AS tpr").
				Where("play_time >= ?", startTime).
				Select(
					"COALESCE(MAX(CASE WHEN tpr.resolved_track_id > 0 THEN tpr.resolved_track_id ELSE 0 END), 0) AS track_id, " +
						"tpr.artist, tpr.album, tpr.track, tpr.track_number, tpr.disc_number, COUNT(*) AS play_count, " +
						"COALESCE(MAX(aa.cover_art_url), MAX(ab.cover_art_url), '') AS cover_art_url, " +
						"COALESCE(MAX(aa.cover_art_mime), MAX(ab.cover_art_mime), '') AS cover_art_mime, " +
						"COALESCE(MAX(aa.cover_art_object_key), MAX(ab.cover_art_object_key), '') AS cover_art_object_key",
				).
				Joins("LEFT JOIN album AS aa ON aa.id = tpr.album_id AND tpr.album_id > 0").
				Joins(
					"LEFT JOIN (" +
						"SELECT artist, name, MIN(id) AS id FROM album GROUP BY artist, name" +
						") AS ac ON ac.artist = tpr.artist AND ac.name = tpr.album",
				).
				Joins("LEFT JOIN album AS ab ON ab.id = ac.id").
				Group("tpr.artist, tpr.album, tpr.track, tpr.track_number, tpr.disc_number").
				Order("COUNT(*) DESC").
				Limit(topN).
				Find(&rows).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("period_type = ?", period).Delete(&TrackRankStat{}).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			continue
		}

		items := make([]TrackRankStat, 0, len(rows))
		for i, row := range rows {
			if row.Track == "" {
				continue
			}
			items = append(
				items,
				TrackRankStat{
					PeriodType:        period,
					TrackID:           row.TrackID,
					Artist:            row.Artist,
					Album:             row.Album,
					Track:             row.Track,
					TrackNumber:       row.TrackNumber,
					DiscNumber:        row.DiscNumber,
					PlayCount:         row.PlayCount,
					Rank:              i + 1,
					CoverArtURL:       row.CoverArtURL,
					CoverArtMime:      row.CoverArtMime,
					CoverArtObjectKey: row.CoverArtObjectKey,
				},
			)
		}
		if len(items) == 0 {
			continue
		}
		if err := tx.Create(&items).Error; err != nil {
			return err
		}
	}

	return nil
}

type trackRankCoverSeed struct {
	Artist string `gorm:"column:artist"`
	Album  string `gorm:"column:album"`
}

type trackRankCoverUpdate struct {
	CoverArtURL       string
	CoverArtMime      string
	CoverArtObjectKey string
}

func backfillTrackRankStatArtwork(ctx context.Context, periods []string) error {
	normalizedPeriods := normalizeTrackRankPeriods(periods)
	if len(normalizedPeriods) == 0 {
		return nil
	}

	provider := dashboardTrackRankObjectStorageGet()
	if provider == nil {
		return nil
	}

	var seeds []trackRankCoverSeed
	if err := GetDB().WithContext(ctx).
		Model(&TrackRankStat{}).
		Select("artist, album").
		Where("period_type IN ?", normalizedPeriods).
		Where("TRIM(COALESCE(album, '')) <> ''").
		Where(
			"TRIM(COALESCE(cover_art_url, '')) = '' OR TRIM(COALESCE(cover_art_mime, '')) = '' OR TRIM(COALESCE(cover_art_object_key, '')) = ''",
		).
		Group("artist, album").
		Scan(&seeds).Error; err != nil {
		return err
	}

	db := GetDB().WithContext(ctx)
	for _, seed := range seeds {
		update, ok, err := resolveTrackRankCoverUpdate(ctx, provider, seed.Artist, seed.Album)
		if err != nil || !ok {
			continue
		}

		if err := db.Model(&TrackRankStat{}).
			Where("period_type IN ?", normalizedPeriods).
			Where("artist = ? AND album = ?", seed.Artist, seed.Album).
			Where(
				"TRIM(COALESCE(cover_art_url, '')) = '' OR TRIM(COALESCE(cover_art_mime, '')) = '' OR TRIM(COALESCE(cover_art_object_key, '')) = ''",
			).
			Updates(
				map[string]interface{}{
					"cover_art_url":        update.CoverArtURL,
					"cover_art_mime":       update.CoverArtMime,
					"cover_art_object_key": update.CoverArtObjectKey,
				},
			).Error; err != nil {
			return err
		}
	}

	return nil
}

func resolveTrackRankCoverUpdate(
	ctx context.Context, provider objectstorage.Provider, artist, album string,
) (trackRankCoverUpdate, bool, error) {
	artist = strings.TrimSpace(artist)
	album = strings.TrimSpace(album)
	if provider == nil || album == "" {
		return trackRankCoverUpdate{}, false, nil
	}

	seed := artworkcore.BuildAlbumArtworkSeed(artist, artist, album)
	objectKey := strings.TrimSpace(provider.BuildOriginalObjectKey(seed))
	if objectKey == "" {
		return trackRankCoverUpdate{}, false, nil
	}

	exists, contentType, err := provider.CheckObjectExists(ctx, objectKey)
	if err != nil || !exists {
		return trackRankCoverUpdate{}, false, err
	}

	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		contentType = "image/jpeg"
	}

	return trackRankCoverUpdate{
		CoverArtURL:       provider.GetObjectCDNURL(objectKey),
		CoverArtMime:      contentType,
		CoverArtObjectKey: objectKey,
	}, true, nil
}

func normalizeTrackRankPeriods(periods []string) []string {
	seen := make(map[string]struct{}, len(periods))
	result := make([]string, 0, len(periods))
	for _, period := range periods {
		normalized := normalizeTrackRankPeriod(period)
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func ensureTrackRankStatIndexes(ctx context.Context) error {
	if config.ConfigObj.Database.Type != string(common.DatabaseTypeMySQL) {
		return nil
	}
	db := GetDB().WithContext(ctx)
	migrator := db.Migrator()

	if !migrator.HasIndex(&TrackRankStat{}, "idx_track_rank_period_rank") {
		if err := db.Exec("ALTER TABLE track_rank_stat ADD INDEX idx_track_rank_period_rank (period_type, `rank`)").Error; err != nil {
			return err
		}
	}
	if !migrator.HasIndex(&TrackRankStat{}, "idx_track_rank_track_id") {
		if err := db.Exec("ALTER TABLE track_rank_stat ADD INDEX idx_track_rank_track_id (track_id)").Error; err != nil {
			return err
		}
	}
	if !migrator.HasIndex(&TrackRankStat{}, "uk_track_rank_period_track") {
		// utf8mb4 下联合唯一索引总长度不能超过 3072 bytes，使用前缀索引规避超长。
		if err := db.Exec(
			"ALTER TABLE track_rank_stat ADD UNIQUE KEY uk_track_rank_period_track (period_type, artist(191), album(191), track(191), disc_number, track_number)",
		).Error; err != nil {
			return err
		}
	}
	if !migrator.HasIndex(&TrackRankStat{}, "idx_track_rank_stat_fts") {
		if err := db.Exec("ALTER TABLE track_rank_stat ADD FULLTEXT idx_track_rank_stat_fts(track, artist, album) WITH PARSER ngram").Error; err != nil {
			return err
		}
	}
	return nil
}

func parseDateOnly(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("parse date failed: empty value")
	}

	layouts := []string{
		"2006-01-02",
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.999999",
	}
	for _, layout := range layouts {
		if d, err := time.Parse(layout, s); err == nil {
			return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local), nil
		}
	}

	if len(s) >= 10 {
		if d, err := time.Parse("2006-01-02", s[:10]); err == nil {
			return d, nil
		}
	}
	return time.Time{}, fmt.Errorf("parse date %q failed", s)
}

func GetDashboardOverviewFromStat(ctx context.Context) (*DashboardStat, error) {
	var stat DashboardStat
	err := GetDB().WithContext(ctx).Where("id = ?", 1).First(&stat).Error
	return &stat, err
}

func GetPlayCountsBySourceFromStat(ctx context.Context) (map[string]int64, error) {
	var rows []*PlaySourceStat
	if err := GetDB().WithContext(ctx).Order("count DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(rows))
	for _, row := range rows {
		result[row.Source] = row.Count
	}
	return result, nil
}

func GetTopArtistsFromStat(ctx context.Context, metricType string, limit int) ([]map[string]interface{}, error) {
	var rows []*TopArtistStat
	if err := GetDB().WithContext(ctx).
		Where("period_days = ? AND metric_type = ?", 0, metricType).
		Order("`rank` ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		item := map[string]interface{}{"artist": row.Artist, "rank": row.Rank}
		if metricType == "tracks" {
			item["track_count"] = row.MetricValue
		} else {
			item["play_count"] = row.MetricValue
		}
		result = append(result, item)
	}
	return result, nil
}

func normalizeAlbumPeriod(days int) int {
	switch days {
	case 7, 30, 90, 365:
		return days
	default:
		if days <= 14 {
			return 7
		}
		if days <= 60 {
			return 30
		}
		if days <= 180 {
			return 90
		}
		return 365
	}
}

func GetTopAlbumsByPlayCountFromStat(ctx context.Context, days int, limit int) ([]*TopAlbum, error) {
	period := normalizeAlbumPeriod(days)
	type topAlbumRow struct {
		AlbumID           int64
		Album             string
		AlbumSubtitle     string
		Artist            string
		PlayCount         int64
		CoverArtURL       string
		CoverArtMime      string
		CoverArtObjectKey string
	}
	var rows []topAlbumRow
	if err := GetDB().WithContext(ctx).
		Table("top_album_stat AS tas").
		Select(
			"tas.album_id, tas.album, COALESCE(a.name_subtitle, tas.album_subtitle, '') AS album_subtitle, tas.artist, tas.play_count, "+
				"COALESCE(a.cover_art_url, '') AS cover_art_url, "+
				"COALESCE(a.cover_art_mime, '') AS cover_art_mime, "+
				"COALESCE(a.cover_art_object_key, '') AS cover_art_object_key",
		).
		Joins("LEFT JOIN album AS a ON a.id = tas.album_id").
		Where("tas.period_days = ?", period).
		Order("tas.`rank` ASC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]*TopAlbum, 0, len(rows))
	for _, row := range rows {
		result = append(
			result,
			&TopAlbum{
				AlbumID:           row.AlbumID,
				Album:             row.Album,
				AlbumSubtitle:     row.AlbumSubtitle,
				Artist:            row.Artist,
				PlayCount:         int(row.PlayCount),
				CoverArtURL:       row.CoverArtURL,
				CoverArtMime:      row.CoverArtMime,
				CoverArtObjectKey: row.CoverArtObjectKey,
			},
		)
	}
	return result, nil
}

func GetTopGenresWithDetailsFromStat(ctx context.Context, limit int) ([]*TopGenre, error) {
	var rows []*TopGenreStat
	if err := GetDB().WithContext(ctx).Order("`rank` ASC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]*TopGenre, 0, len(rows))
	for _, row := range rows {
		result = append(
			result,
			&TopGenre{
				TrackGenreName:  row.GenreName,
				TrackGenreCount: row.TrackGenreCount,
				GenreNameZh:     row.GenreNameZh,
				GenreCount:      row.GenreCount,
				Rank:            row.Rank,
			},
		)
	}
	return result, nil
}

func GetPlayTrendFromStatByDays(ctx context.Context, days int) (
	map[string]int, map[string]*HourlyPlayTrendData, error,
) {
	startDate := time.Now().AddDate(0, 0, -days)
	dateKey := startDate.Format("2006-01-02")

	type dailyRow struct {
		StatDate  string
		PlayCount int64
	}
	type hourlyRow struct {
		StatDate  string
		Hour      int
		PlayCount int64
	}

	var dailySQL string
	var hourlySQL string
	if config.ConfigObj.Database.Type == string(common.DatabaseTypeMySQL) {
		dailySQL = "SELECT DATE(stat_date) as stat_date, play_count FROM play_trend_daily_stat WHERE stat_date >= ?"
		hourlySQL = "SELECT DATE(stat_date) as stat_date, hour, play_count FROM play_trend_hourly_stat WHERE stat_date >= ?"
	} else {
		dailySQL = "SELECT date(stat_date) as stat_date, play_count FROM play_trend_daily_stat WHERE stat_date >= ?"
		hourlySQL = "SELECT date(stat_date) as stat_date, hour, play_count FROM play_trend_hourly_stat WHERE stat_date >= ?"
	}

	var dailyRows []dailyRow
	var hourlyRows []hourlyRow

	if err := GetDB().WithContext(ctx).Raw(dailySQL, dateKey).Scan(&dailyRows).Error; err != nil {
		return nil, nil, err
	}
	if err := GetDB().WithContext(ctx).Raw(hourlySQL, dateKey).Scan(&hourlyRows).Error; err != nil {
		return nil, nil, err
	}

	daily := make(map[string]int, len(dailyRows))
	hourly := make(map[string]*HourlyPlayTrendData, len(hourlyRows))

	for _, row := range dailyRows {
		normalizedDate := normalizeTrendDateKey(row.StatDate)
		if normalizedDate == "" {
			continue
		}
		playCount := int(row.PlayCount)
		if playCount < 0 {
			playCount = 0
		}
		// 合并同一天的重复键（如 2026-02-01 与 2026-02-01T00:00:00+08:00）
		if old, exists := daily[normalizedDate]; !exists || playCount > old {
			daily[normalizedDate] = playCount
		}
		if _, ok := hourly[normalizedDate]; !ok {
			hourly[normalizedDate] = &HourlyPlayTrendData{
				Date:   normalizedDate,
				Total:  daily[normalizedDate],
				Hourly: make(map[int]int),
			}
		} else {
			hourly[normalizedDate].Total = daily[normalizedDate]
		}
	}

	for _, row := range hourlyRows {
		normalizedDate := normalizeTrendDateKey(row.StatDate)
		if normalizedDate == "" {
			continue
		}
		h := row.Hour
		if h < 0 {
			h = 0
		}
		if h > 23 {
			h = 23
		}
		playCount := int(row.PlayCount)
		if playCount <= 0 {
			continue
		}
		dateObj, ok := hourly[normalizedDate]
		if !ok {
			dateObj = &HourlyPlayTrendData{
				Date:   normalizedDate,
				Total:  daily[normalizedDate],
				Hourly: make(map[int]int),
			}
			hourly[normalizedDate] = dateObj
		}
		// 重复小时记录时使用较大值，避免脏数据重复叠加
		if old, exists := dateObj.Hourly[h]; !exists || playCount > old {
			dateObj.Hourly[h] = playCount
		}
	}

	return daily, hourly, nil
}

func normalizeTrendDateKey(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if d, err := parseDateOnly(raw); err == nil {
		return d.Format("2006-01-02")
	}
	if len(raw) >= 10 {
		return raw[:10]
	}
	return raw
}

func IsDashboardStatReady(ctx context.Context) bool {
	var count int64
	err := GetDB().WithContext(ctx).Model(&DashboardStat{}).Count(&count).Error
	return err == nil && count > 0
}

func GetDashboardStatsUpdatedSince(ctx context.Context, since time.Time) ([]*DashboardStat, error) {
	var rows []*DashboardStat
	query := GetDB().WithContext(ctx).Model(&DashboardStat{})
	if !since.IsZero() {
		query = query.Where("updated_at >= ?", since)
	}
	err := query.Order("updated_at ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func GetPlaySourceStatsUpdatedSince(ctx context.Context, since time.Time) ([]*PlaySourceStat, error) {
	var rows []*PlaySourceStat
	query := GetDB().WithContext(ctx).Model(&PlaySourceStat{})
	if !since.IsZero() {
		query = query.Where("updated_at >= ?", since)
	}
	err := query.Order("updated_at ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func GetTopArtistStatsUpdatedSince(ctx context.Context, since time.Time) ([]*TopArtistStat, error) {
	var rows []*TopArtistStat
	query := GetDB().WithContext(ctx).Model(&TopArtistStat{})
	if !since.IsZero() {
		query = query.Where("updated_at >= ?", since)
	}
	err := query.Order("updated_at ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func GetTopAlbumStatsUpdatedSince(ctx context.Context, since time.Time) ([]*TopAlbumStat, error) {
	var rows []*TopAlbumStat
	query := GetDB().WithContext(ctx).Model(&TopAlbumStat{})
	if !since.IsZero() {
		query = query.Where("updated_at >= ?", since)
	}
	err := query.Order("updated_at ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func GetTopGenreStatsUpdatedSince(ctx context.Context, since time.Time) ([]*TopGenreStat, error) {
	var rows []*TopGenreStat
	query := GetDB().WithContext(ctx).Model(&TopGenreStat{})
	if !since.IsZero() {
		query = query.Where("updated_at >= ?", since)
	}
	err := query.Order("updated_at ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func GetPlayTrendDailyStatsUpdatedSince(ctx context.Context, since time.Time) ([]*PlayTrendDailyStat, error) {
	var rows []*PlayTrendDailyStat
	query := GetDB().WithContext(ctx).Model(&PlayTrendDailyStat{})
	if !since.IsZero() {
		query = query.Where("updated_at >= ?", since)
	}
	err := query.Order("stat_date ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func GetPlayTrendHourlyStatsUpdatedSince(ctx context.Context, since time.Time) ([]*PlayTrendHourlyStat, error) {
	var rows []*PlayTrendHourlyStat
	query := GetDB().WithContext(ctx).Model(&PlayTrendHourlyStat{})
	if !since.IsZero() {
		query = query.Where("updated_at >= ?", since)
	}
	err := query.Order("stat_date ASC, hour ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
