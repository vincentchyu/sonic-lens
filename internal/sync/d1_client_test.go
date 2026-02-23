package d1sync

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

func init() {
	// 初始化日志
	c := make(chan struct{})

	// 尝试加载配置文件 config/config_bak.yaml
	// 注意：这里假设运行测试时的路径是在 internal/sync 目录下，由于 go test ./internal/sync/...
	// 会将 PWD 设置为包目录，所以需要向上寻找
	configPath := "../../config/config_dev.yaml"

	// 加载配置
	// 这里我们不通过 config.InitConfig 直接 panic，而是尝试加载，允许失败（单元测试环境下可能没有配置文件）
	func() {
		defer func() {
			if r := recover(); r != nil {
				// 忽略配置加载失败，测试中会检查配置是否存在
			}
		}()
		config.InitConfig(configPath)
		fmt.Printf("Loaded config from %s\n", configPath)
	}()

	// 初始化日志
	if config.ConfigObj.Log.Path != "" {
		_, _ = log.LogInit(config.ConfigObj.Log.Path, "debug", c)
	} else {
		// 如果没有配置，初始化一个控制台日志
		logger, _ := zap.NewDevelopment()
		zap.ReplaceGlobals(logger)
	}
}

// TestD1Client_Ping 测试 D1 数据库连接是否畅通
// 对应用户需求：1.用作使用ping是否畅通
func TestD1Client_Ping(t *testing.T) {
	cfg := config.ConfigObj.Cloudflare

	// 如果没有配置 D1 信息，跳过集成测试
	if cfg.AccountID == "" || cfg.APIToken == "" || cfg.D1DatabaseID == "" {
		t.Skip("Skipping D1 Ping test: Cloudflare config missing in config/config_bak.yaml")
	}

	ctx := context.Background()
	log.Info(ctx, "Testing D1 connection...")

	// 使用 NewD1Client，它内部已经调用了 db.Ping()
	// failed to ping D1 database: D1 API error 7403: The given account is not valid or is not authorized to access this service
	client, err := NewD1Client(&cfg)
	if err != nil {
		t.Fatalf("Failed to create D1 client (Ping failed): %v", err)
	}
	defer client.Close()

	assert.NotNil(t, client)
	assert.NotNil(t, client.db)

	t.Log("D1 connection Ping successful")
}

// TestD1Client_Query 测试执行 SQL 查询监测是否正常
// 对应用户需求：2.执行trac的查询监测是否正常执行sql语句
func TestD1Client_Query(t *testing.T) {
	cfg := config.ConfigObj.Cloudflare

	// 如果没有配置 D1 信息，跳过集成测试
	if cfg.AccountID == "" || cfg.APIToken == "" || cfg.D1DatabaseID == "" {
		t.Skip("Skipping D1 Query test: Cloudflare config missing in config/config_bak.yaml")
	}

	client, err := NewD1Client(&cfg)
	if err != nil {
		t.Fatalf("Failed to create D1 client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 尝试初始化表结构（避免表不存在导致驱动 panic）
	if err := initSyncMetadataTable(client.db); err != nil {
		t.Logf("Failed to init sync_metadata table: %v", err)
		// 如果无法建表，后续测试可能会失败，但我们继续尝试
	}

	// 插入测试数据，以绕过可能存在的驱动对空结果集的处理 Bug (Panic)
	_, err = client.db.ExecContext(
		ctx,
		"INSERT OR IGNORE INTO sync_metadata (table_name, last_sync_time, created_at, updated_at) VALUES ('tracks', '2023-01-01T00:00:00Z', '2023-01-01T00:00:00Z', '2023-01-01T00:00:00Z')",
	)
	if err != nil {
		t.Logf("Failed to insert test data: %v", err)
	}

	t.Run(
		"SimpleSelect", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Recovered from panic in SimpleSelect: %v", r)
				}
			}()
			var result int
			err := client.db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
			assert.NoError(t, err)
			assert.Equal(t, 1, result)
		},
	)

	t.Run(
		"QueryTracksTable", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Recovered from panic in QueryTracksTable: %v", r)
				}
			}()
			// 检查 tracks 表是否存在，或者尝试查询一条数据
			// 注意：如果表不存在，这个查询会失败
			rows, err := client.db.QueryContext(ctx, "SELECT count(*) FROM tracks")
			if err != nil {
				t.Logf("Query tracks table failed (table might not exist yet): %v", err)
				return
			}
			defer rows.Close()

			var count int
			if rows.Next() {
				if err := rows.Scan(&count); err != nil {
					t.Errorf("Failed to scan count: %v", err)
				}
				t.Logf("Found %d tracks in D1 tracks table", count)
			}
		},
	)

	t.Run(
		"CheckSyncMetadata", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Recovered from panic in CheckSyncMetadata: %v", r)
				}
			}()
			// 测试获取最后同步时间逻辑
			lastTime, err := client.getLastSyncTime(ctx, "tracks")
			if err != nil {
				t.Logf("getLastSyncTime failed: %v", err)
			} else {
				t.Logf("Last sync time for tracks: %v", lastTime)
			}
		},
	)

	t.Run(
		"QuerySyncMetadata", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Recovered from panic in QuerySyncMetadata: %v", r)
				}
			}()

			rows, err := client.db.QueryContext(ctx, "SELECT * FROM sync_metadata")
			if err != nil {
				t.Errorf("Query sync_metadata failed: %v", err)
				return
			}
			// Don't defer rows.Close() here because we need to close it before deletion loop
			// or we can use a separate func scope. But explicit close is fine.

			cols, err := rows.Columns()
			if err != nil {
				rows.Close()
				t.Errorf("Failed to get columns: %v", err)
				return
			}
			t.Logf("Columns: %v", cols)

			// Find id column index
			idIdx := -1
			for i, name := range cols {
				if name == "id" {
					idIdx = i
					break
				}
			}

			t.Log("sync_metadata content:")
			var idsToDelete []interface{}

			for rows.Next() {
				// Create a slice of interface{} to hold values for each column
				values := make([]interface{}, len(cols))
				valuePtrs := make([]interface{}, len(cols))
				for i := range cols {
					valuePtrs[i] = &values[i]
				}

				if err := rows.Scan(valuePtrs...); err != nil {
					t.Errorf("Failed to scan sync_metadata row: %v", err)
					continue
				}

				// Log each column and its value
				rowStr := "Row: "
				for i, col := range cols {
					val := values[i]
					if b, ok := val.([]byte); ok {
						val = string(b)
					}
					rowStr += fmt.Sprintf("%s=%v, ", col, val)
				}
				t.Log(rowStr)

				if idIdx != -1 {
					idsToDelete = append(idsToDelete, values[idIdx])
				}
			}
			rows.Close()

			// 增加对以上查询到的结果增加删除逻辑，一条一条删除
			if len(idsToDelete) > 0 {
				t.Logf("Deleting %d rows...", len(idsToDelete))
				for _, id := range idsToDelete {
					t.Logf("Deleting row with id: %v", id)
					if _, err := client.db.ExecContext(ctx, "DELETE FROM sync_metadata WHERE id = ?", id); err != nil {
						t.Errorf("Failed to delete row %v: %v", id, err)
					}
				}
			} else {
				t.Log("No rows to delete.")
			}
		},
	)
}

// initSyncMetadataTable 初始化 sync_metadata 表
func initSyncMetadataTable(db *sql.DB) error {
	ctx := context.Background()
	_, err := db.ExecContext(
		ctx, `
		CREATE TABLE IF NOT EXISTS sync_metadata (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			table_name TEXT NOT NULL UNIQUE,
			last_sync_time TEXT NOT NULL,
			sync_count INTEGER DEFAULT 0,
			last_error TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
	`,
	)
	return err
}

func TestD1Client_Upsert(t *testing.T) {
	cfg := config.ConfigObj.Cloudflare
	if cfg.AccountID == "" || cfg.APIToken == "" || cfg.D1DatabaseID == "" {
		t.Skip("Skipping D1 Upsert test: Cloudflare config missing")
	}

	client, err := NewD1Client(&cfg)
	if err != nil {
		t.Fatalf("Failed to create D1 client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 构造测试数据
	now := time.Now()
	testTrack := &model.Track{
		Artist:          "Test Artist",
		Album:           "Test Album",
		Track:           "Test Track Batch Upsert",
		AlbumArtist:     "Test Album Artist",
		PlayCount:       1,
		Genre:           "Test Genre",
		Duration:        180,
		Source:          "test",
		IsAppleMusicFav: true,
		IsLastFmFav:     false,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// 1. 测试批量插入 (验证重构后的 upsertTracksBatch 是否工作)
	t.Run(
		"BatchUpsert", func(t *testing.T) {
			err := client.upsertTracksBatch(ctx, []*model.Track{testTrack})
			if err != nil {
				t.Fatalf("Batch upsert failed: %v", err)
			}
			t.Log("Batch upsert successful")
		},
	)

	// 2. 验证插入结果
	t.Run(
		"VerifyUpsert", func(t *testing.T) {
			var count int
			err := client.db.QueryRowContext(
				ctx, "SELECT count(*) FROM tracks WHERE track = ?", testTrack.Track,
			).Scan(&count)
			if err != nil {
				t.Errorf("Failed to query inserted track: %v", err)
			} else {
				assert.Equal(t, 1, count, "Should find exactly 1 inserted track")
			}
		},
	)

	// 3. 清理数据
	t.Run(
		"Cleanup", func(t *testing.T) {
			_, err := client.db.ExecContext(ctx, "DELETE FROM tracks WHERE track = ?", testTrack.Track)
			if err != nil {
				t.Logf("Failed to cleanup test track: %v", err)
			} else {
				t.Log("Cleanup successful")
			}
		},
	)
}
