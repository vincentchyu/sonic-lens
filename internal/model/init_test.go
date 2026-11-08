package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/config"
)

func TestConfigureMySQLConnectionPool(t *testing.T) {
	t.Run("配置默认值生效", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		config.ConfigObj.Database.Mysql = config.MysqlConfig{}
		err = configureMySQLConnectionPool(db)
		require.NoError(t, err)

		sqlDB, err := db.DB()
		require.NoError(t, err)

		stats := sqlDB.Stats()
		assert.Equal(t, 100, stats.MaxOpenConnections)
	})

	t.Run("配置自定义值生效", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
		require.NoError(t, err)

		config.ConfigObj.Database.Mysql = config.MysqlConfig{
			MaxOpenConns:           50,
			MaxIdleConns:           5,
			ConnMaxLifetimeMinutes: 30,
			ConnMaxIdleTimeMinutes: 5,
		}
		t.Cleanup(func() {
			config.ConfigObj.Database.Mysql = config.MysqlConfig{}
		})

		err = configureMySQLConnectionPool(db)
		require.NoError(t, err)

		sqlDB, err := db.DB()
		require.NoError(t, err)

		stats := sqlDB.Stats()
		assert.Equal(t, 50, stats.MaxOpenConnections)
	})

	t.Run("nil DB 安全返回", func(t *testing.T) {
		assert.NoError(t, configureMySQLConnectionPool(nil))
	})
}
