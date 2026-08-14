package model

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRepairAndReconcileTrackPlayRecords(t *testing.T) {
	db := newTrackResolutionTestDB(t, "test_repair_reconcile")
	ctx := context.Background()

	// 1. 准备实体数据 (Track & Album)
	album := &Album{
		Name:         "万能青年旅店",
		NameSubtitle: "",
		Artist:       "万能青年旅店",
		ReleaseDate:  "2010-11-12",
		Genre:        "Indie Rock",
		PlayCount:    0,
	}
	assert.NoError(t, db.Create(album).Error)

	track := &Track{
		Artist:      "万能青年旅店",
		Album:       "万能青年旅店",
		Track:       "十万嬉皮",
		TrackNumber: 7,
		DiscNumber:  1,
		Genre:       "Indie Rock",
		PlayCount:   0,
	}
	assert.NoError(t, db.Create(track).Error)

	ta := &TrackAlbum{
		TrackID:     track.ID,
		AlbumID:     album.ID,
		Track:       "十万嬉皮",
		TrackNumber: 7,
		DiscNumber:  1,
	}
	assert.NoError(t, db.Create(ta).Error)

	// 2. 插入带连字符发行后缀和缺失字段的听歌流水记录
	rec1 := &TrackPlayRecord{
		Artist:           "万能青年旅店",
		Album:            "万能青年旅店 - Single",
		Track:            "十万嬉皮",
		Genre:            "", // 缺失
		AlbumID:          0,  // 缺失
		ResolvedTrackID:  0,  // 缺失
		ResolutionStatus: TrackPlayRecordResolutionPending,
		PlayTime:         time.Now(),
		Source:           "Apple Music",
	}
	assert.NoError(t, db.Create(rec1).Error)

	// 3. 执行修补与归因
	report, err := RepairAndReconcileTrackPlayRecords(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.EqualValues(t, 1, report.TotalProcessed)
	assert.EqualValues(t, 1, report.RepairedTrackID)
	assert.EqualValues(t, 1, report.RepairedAlbumID)
	assert.EqualValues(t, 1, report.RepairedGenre)
	assert.EqualValues(t, 1, report.RepairedReleaseType)

	// 4. 验证修补后的记录属性
	updatedRec, err := GetTrackPlayRecordByID(ctx, rec1.ID)
	assert.NoError(t, err)
	assert.EqualValues(t, track.ID, updatedRec.ResolvedTrackID)
	assert.EqualValues(t, album.ID, updatedRec.AlbumID)
	assert.Equal(t, "Indie Rock", updatedRec.Genre)
	assert.Equal(t, "single", updatedRec.ReleaseType)
	assert.Equal(t, TrackPlayRecordResolutionResolved, updatedRec.ResolutionStatus)

	// 5. 执行全表播放对账校对
	assert.NoError(t, ReconcileTrackPlayCounts(ctx))
	assert.NoError(t, ReconcileAlbumPlayCounts(ctx))

	// 验证 play_count 校对完成
	var updatedTrack Track
	assert.NoError(t, db.First(&updatedTrack, track.ID).Error)
	assert.EqualValues(t, 1, updatedTrack.PlayCount)

	var updatedAlbum Album
	assert.NoError(t, db.First(&updatedAlbum, album.ID).Error)
	assert.EqualValues(t, 1, updatedAlbum.PlayCount)
}
