package model

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/config"
)

var modelTestNow = time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)

func newModelTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()

	rawDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	db, err := gorm.Open(
		mysql.New(mysql.Config{
			Conn:                      rawDB,
			SkipInitializeWithVersion: true,
		}),
		&gorm.Config{
			SkipDefaultTransaction: true,
			NowFunc: func() time.Time {
				return modelTestNow
			},
		},
	)
	require.NoError(t, err)

	prevConfig := *config.ConfigObj
	prevSQLite := GlobalDBForSqlLite
	prevMySQL := GlobalDBForMysql

	config.ConfigObj.Database.Type = string(common.DatabaseTypeMySQL)
	GlobalDBForSqlLite = nil
	GlobalDBForMysql = db

	t.Cleanup(
		func() {
			_ = rawDB.Close()
			*config.ConfigObj = prevConfig
			GlobalDBForSqlLite = prevSQLite
			GlobalDBForMysql = prevMySQL
		},
	)

	return db, mock
}

// TestInsertTrackPlayRecordValidation 验证曲目信息校验逻辑在不连数据库时仍可工作。
func TestInsertTrackPlayRecordValidation(t *testing.T) {
	ctx := context.Background()

	// 测试验证逻辑 - 有效参数
	err := common.ValidateTrackInfo(ctx, "Artist Name", "Album Title", "Track Name")
	assert.NoError(t, err)

	// 测试验证逻辑 - 无效参数（空艺术家）
	err = common.ValidateTrackInfo(ctx, "", "Album Title", "Track Name")
	assert.Error(t, err)
	assert.Equal(t, "艺术家名称不能为空", err.Error())
}

// TestIncrementTrackPlayCountValidation 验证播放计数前的基础参数校验。
func TestIncrementTrackPlayCountValidation(t *testing.T) {
	ctx := context.Background()

	// 测试验证逻辑 - 有效参数
	err := common.ValidateTrackInfo(ctx, "Artist Name", "Album Title", "Track Name")
	assert.NoError(t, err)

	// 测试验证逻辑 - 无效参数（空艺术家）
	err = common.ValidateTrackInfo(ctx, "", "Album Title", "Track Name")
	assert.Error(t, err)
	assert.Equal(t, "艺术家名称不能为空", err.Error())
}
