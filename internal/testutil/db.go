package testutil

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

var (
	memDBSeq  uint64
	frozenNow = time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
)

// NewMemoryDB 创建一个完全隔离的 SQLite in-memory GORM 数据库实例。
// 支持传入任意数量的建表 DDL，并在测试结束时自动清理资源。
func NewMemoryDB(t *testing.T, schemaSQL ...string) *gorm.DB {
	t.Helper()

	seq := atomic.AddUint64(&memDBSeq, 1)
	dsn := fmt.Sprintf("file:test_mem_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), seq)

	db, err := gorm.Open(
		sqlite.Open(dsn),
		&gorm.Config{
			SkipDefaultTransaction: true,
			NowFunc: func() time.Time {
				return frozenNow
			},
		},
	)
	require.NoError(t, err, "初始化 SQLite 内存数据库失败")

	for _, ddl := range schemaSQL {
		if ddl == "" {
			continue
		}
		require.NoError(t, db.Exec(ddl).Error, "执行测试建表 DDL 失败: %s", ddl)
	}

	sqlDB, err := db.DB()
	if err == nil {
		t.Cleanup(func() {
			_ = sqlDB.Close()
		})
	}

	return db
}

// NewMockDB 创建一个使用 sqlmock 的 GORM MySQL 适配实例。
func NewMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()

	rawDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err, "初始化 sqlmock 失败")

	db, err := gorm.Open(
		mysql.New(
			mysql.Config{
				Conn:                      rawDB,
				SkipInitializeWithVersion: true,
			},
		),
		&gorm.Config{
			SkipDefaultTransaction: true,
			NowFunc: func() time.Time {
				return frozenNow
			},
		},
	)
	require.NoError(t, err, "绑定 GORM 与 sqlmock 失败")

	t.Cleanup(func() {
		_ = rawDB.Close()
	})

	return db, mock
}

// SetupTestGlobalMySQL 临时替换 model.GlobalDBForMysql 为测试 DB，并在测试结束时自动还原。
func SetupTestGlobalMySQL(t *testing.T, db *gorm.DB) {
	t.Helper()

	prevConfig := *config.ConfigObj
	prevMySQL := model.GlobalDBForMysql

	model.GlobalDBForMysql = db

	t.Cleanup(func() {
		*config.ConfigObj = prevConfig
		model.GlobalDBForMysql = prevMySQL
	})
}
