package model

import (
	"errors"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/db"
)

var GlobalDBForSqlLite *gorm.DB
var GlobalDBForMysql *gorm.DB

func GetDB() *gorm.DB {
	if config.ConfigObj.Database.Type == string(common.DatabaseTypeMySQL) {
		return GlobalDBForMysql
	}
	return GlobalDBForSqlLite
}

func InitDB(dataSourceName string, l *zap.Logger) error {
	var err error

	// Create custom logger with OpenTelemetry
	customLogger := db.NewCustomLogger(l)
	switch config.ConfigObj.Database.Type {
	case string(common.DatabaseTypeSQLite):
		// Open SQLite database with custom logger
		GlobalDBForSqlLite, err = gorm.Open(
			sqlite.Open(dataSourceName), &gorm.Config{
				Logger: customLogger,
			},
		)
		if err != nil {
			return err
		}

		// Auto migrate the schema for core tables
		if err = GlobalDBForSqlLite.AutoMigrate(&TrackPlayRecord{}); err != nil {
			return err
		}
		if err = GlobalDBForSqlLite.AutoMigrate(&Track{}); err != nil {
			return err
		}
		if err = GlobalDBForSqlLite.AutoMigrate(&Genre{}); err != nil {
			return err
		}
		// Auto migrate the schema for AI insight related tables
		if err = GlobalDBForSqlLite.AutoMigrate(&TrackInsight{}, &TrackInsightFeedback{}); err != nil {
			return err
		}
		// Auto migrate the schema for LLM call log table
		if err = GlobalDBForSqlLite.AutoMigrate(&LLMCallLog{}); err != nil {
			return err
		}
	case string(common.DatabaseTypeMySQL):
		// Open MySQL database with custom logger
		GlobalDBForMysql, err = gorm.Open(
			mysql.Open(db.MysqlDSN(config.ConfigObj.Database.Mysql.GetMysqlDSN())), &gorm.Config{
				Logger: customLogger,
			},
		)
		if err != nil {
			return err
		}
		if config.ConfigObj.IsDev {
			// Auto migrate the schema for core tables
			if err = GlobalDBForMysql.AutoMigrate(&TrackPlayRecord{}); err != nil {
				return err
			}
			if err = GlobalDBForMysql.AutoMigrate(&Track{}); err != nil {
				return err
			}
			if err = GlobalDBForMysql.AutoMigrate(&Genre{}); err != nil {
				return err
			}
			// Auto migrate the schema for AI insight related tables
			if err = GlobalDBForMysql.AutoMigrate(&TrackInsight{}, &TrackInsightFeedback{}); err != nil {
				return err
			}
			// Auto migrate the schema for LLM call log table
			if err = GlobalDBForMysql.AutoMigrate(&LLMCallLog{}); err != nil {
				return err
			}
		}
	default:
		return errors.New("unsupported database type" + config.ConfigObj.Database.Type)
	}

	return nil
}
