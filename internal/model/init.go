package model

import (
	"cmp"
	"errors"
	"time"

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

func isMySQL(db *gorm.DB) bool {
	if db == nil || db.Dialector == nil {
		return true
	}
	return db.Dialector.Name() == "mysql"
}

func InitDB(l *zap.Logger) error {
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
	if err = configureMySQLConnectionPool(GlobalDBForMysql); err != nil {
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

// configureMySQLConnectionPool 配置 MySQL 底层连接池生命周期策略。
// 客户端主动超时（ConnMaxLifetime 与 ConnMaxIdleTime）必须短于 MySQL 服务端的 wait_timeout (默认 8h)，
// 确保在服务端单方面掐断 socket 前平滑回收空闲连接，规避 "closing bad idle connection: unexpected read from socket"。
func configureMySQLConnectionPool(gormDB *gorm.DB) error {
	if gormDB == nil {
		return nil
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		return err
	}

	mysqlCfg := config.ConfigObj.Database.Mysql
	maxOpen := cmp.Or(mysqlCfg.MaxOpenConns, 100)
	maxIdle := cmp.Or(mysqlCfg.MaxIdleConns, 10)
	lifetimeMin := cmp.Or(mysqlCfg.ConnMaxLifetimeMinutes, 60)
	idleTimeMin := cmp.Or(mysqlCfg.ConnMaxIdleTimeMinutes, 10)

	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(time.Duration(lifetimeMin) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(idleTimeMin) * time.Minute)
	return nil
}
