package model

import (
	"errors"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormtracing "gorm.io/plugin/opentelemetry/tracing"

	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/db"
)

var GlobalDBForMysql *gorm.DB

var mysqlAutoMigrateModels = []interface{}{
	&TrackPlayRecord{},
	&TrackFavoriteEvent{},
	&PendingAlbumWorkItem{},
	&Track{},
	&Genre{},
	&DashboardStat{},
	&PlaySourceStat{},
	&TopArtistStat{},
	&ArtistProfile{},
	&TopAlbumStat{},
	&TopGenreStat{},
	&PlayTrendDailyStat{},
	&PlayTrendHourlyStat{},
	&TrackRankStat{},
	&TrackInsight{},
	&TrackInsightFeedback{},
	&AlbumInsight{},
	&AlbumInsightFeedback{},
	&LLMCallLog{},
	&InsightJob{},
	&Album{},
	&TrackAlbum{},
	&ReleaseMB{},
	&AlbumReleaseMB{},
	&LibraryChangeLog{},
}

func GetDB() *gorm.DB {
	return GlobalDBForMysql
}

func InitDB(dataSourceName string, l *zap.Logger) error {
	if config.ConfigObj.Database.Mysql.GetMysqlDSN() == "" {
		return errors.New("mysql dsn is required")
	}

	customLogger := db.NewCustomLogger(l)
	var err error
	GlobalDBForMysql, err = gorm.Open(
		mysql.Open(db.MysqlDSN(config.ConfigObj.Database.Mysql.GetMysqlDSN())), &gorm.Config{
			Logger: customLogger,
		},
	)
	if err != nil {
		return err
	}
	if err = enableGORMTelemetry(GlobalDBForMysql, l, "mysql"); err != nil {
		return err
	}
	if config.ConfigObj.IsDev {
		if err = autoMigrateMySQLDevSchema(GlobalDBForMysql); err != nil {
			return err
		}
	}

	return nil
}

func autoMigrateMySQLDevSchema(gormDB *gorm.DB) error {
	return gormDB.AutoMigrate(mysqlAutoMigrateModels...)
}

func enableGORMTelemetry(gormDB *gorm.DB, logger *zap.Logger, dbSystem string) error {
	if gormDB == nil {
		return nil
	}

	if err := gormDB.Use(
		gormtracing.NewPlugin(
			gormtracing.WithTracerProvider(otel.GetTracerProvider()),
			gormtracing.WithDBSystem(dbSystem),
			gormtracing.WithoutMetrics(),
			gormtracing.WithoutQueryVariables(),
		),
	); err != nil {
		return err
	}

	if !config.ConfigObj.Telemetry.DBStatsMetricsEnabled {
		return nil
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return err
	}
	if err := db.RegisterDBStatsMetrics(sqlDB, dbSystem); err != nil && logger != nil {
		logger.Warn("注册数据库连接池指标失败", zap.String("db_system", dbSystem), zap.Error(err))
	}
	return nil
}
