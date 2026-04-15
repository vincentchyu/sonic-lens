//go:build integration
// +build integration

package d1sync

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

func TestD1Client_UpsertPlayRecords(t *testing.T) {
	cfg := config.ConfigObj.Cloudflare
	if cfg.AccountID == "" || cfg.APIToken == "" || cfg.D1DatabaseID == "" {
		t.Skip("Skipping D1 UpsertPlayRecords test: Cloudflare config missing")
	}

	client, err := NewD1Client(&cfg)
	if err != nil {
		t.Fatalf("Failed to create D1 client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 确保表存在
	err = client.createTrackPlayRecordsTable(ctx)
	if err != nil {
		t.Fatalf("Failed to create track_play_records table: %v", err)
	}

	// 构造测试数据
	now := time.Now()
	testRecord := &model.TrackPlayRecord{
		Artist:               "Test Artist",
		AlbumArtist:          "Test Album Artist",
		Album:                "Test Album",
		Track:                "Test Track Play Record",
		AlbumID:              1,
		Duration:             180,
		PlayTime:             now,
		Scrobbled:            true,
		TrackNumber:          1,
		DiscNumber:           1,
		MusicBrainzID:        "test-mbid",
		Source:               "test",
		CoverArtPath:         "/path/to/cover.jpg",
		ResolvedTrackID:      100,
		ResolutionStatus:     "resolved",
		ResolutionConfidence: 100,
		LibraryApplied:       true,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	// 1. 测试批量插入 (验证修正后的 upsertPlayRecordsBatch 是否工作)
	t.Run("BatchUpsertPlayRecords", func(t *testing.T) {
		err := client.upsertPlayRecordsBatch(ctx, []*model.TrackPlayRecord{testRecord})
		if err != nil {
			t.Fatalf("Batch upsert play records failed: %v", err)
		}
		t.Log("Batch upsert play records successful")
	})

	// 2. 验证插入结果
	t.Run("VerifyUpsertPlayRecords", func(t *testing.T) {
		var count int
		err := client.db.QueryRowContext(
			ctx, "SELECT count(*) FROM track_play_records WHERE track = ?", testRecord.Track,
		).Scan(&count)
		if err != nil {
			t.Errorf("Failed to query inserted play record: %v", err)
		} else {
			assert.Equal(t, 1, count, "Should find exactly 1 inserted play record")
		}
	})

	// 3. 清理数据
	t.Run("CleanupPlayRecords", func(t *testing.T) {
		_, err := client.db.ExecContext(ctx, "DELETE FROM track_play_records WHERE track = ?", testRecord.Track)
		if err != nil {
			t.Logf("Failed to cleanup test play record: %v", err)
		} else {
			t.Log("Cleanup play records successful")
		}
	})
}
