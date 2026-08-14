package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBackfillTrackPlayRecordGenres(t *testing.T) {
	db := newTrackResolutionTestDB(t, "backfill_track_play_record_genres")

	// 1. 初始化标准英文流派库与包含流派信息的 album 与 track 资料
	require.NoError(t, db.Create(&Genre{Name: "Progressive Rock", NameZh: "前卫摇滚"}).Error)
	require.NoError(t, db.Create(&Genre{Name: "Indie Rock", NameZh: "独立摇滚"}).Error)
	require.NoError(t, db.Create(&Genre{Name: "Alternative Rock", NameZh: "另类摇滚"}).Error)

	album := Album{
		ID:        5001,
		Artist:    "万能青年旅店",
		Name:      "冀西南林路行",
		Genre:     "Progressive Rock",
		PlayCount: 0,
	}
	require.NoError(t, db.Create(&album).Error)

	trackObj := Track{
		ID:        6001,
		Artist:    "草东没有派对",
		Album:     "丑奴儿",
		Track:     "大风吹",
		Genre:     "Indie Rock",
		PlayCount: 0,
	}
	require.NoError(t, db.Create(&trackObj).Error)

	// 2. 插入 genre 字段缺失的历史听歌流水记录
	now := time.Now()
	records := []TrackPlayRecord{
		{
			ID:        7001,
			Artist:    "万能青年旅店",
			Album:     "冀西南林路行",
			Track:     "采石",
			AlbumID:   5001,
			Genre:     "", // 缺失流派
			PlayTime:  now,
		},
		{
			ID:               7002,
			Artist:           "草东没有派对",
			Album:            "丑奴儿",
			Track:            "大风吹",
			ResolvedTrackID: 6001,
			Genre:            "", // 缺失流派
			PlayTime:         now,
		},
		{
			ID:        7003,
			Artist:    "Radiohead",
			Album:     "OK Computer",
			Track:     "Karma Police",
			Genre:     "Alternative Rock", // 已有流派，不受影响
			PlayTime:  now,
		},
	}
	require.NoError(t, db.Create(&records).Error)

	// 3. 执行对齐回补逻辑
	res, err := BackfillTrackPlayRecordGenresTx(db)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, int64(2), res.BackfilledRecords) // 修正了 2 条流水记录

	// 4. 断言修补后的流水记录 genre 字段
	var rec1, rec2, rec3 TrackPlayRecord
	require.NoError(t, db.First(&rec1, 7001).Error)
	require.Equal(t, "Progressive Rock", rec1.Genre)

	require.NoError(t, db.First(&rec2, 7002).Error)
	require.Equal(t, "Indie Rock", rec2.Genre)

	require.NoError(t, db.First(&rec3, 7003).Error)
	require.Equal(t, "Alternative Rock", rec3.Genre)

	// 5. 校验自动重对账后的 genre 表播放数
	var genreProg, genreIndie, genreAlt Genre
	require.NoError(t, db.Where("name = ?", "Progressive Rock").First(&genreProg).Error)
	require.Equal(t, int64(1), genreProg.PlayCount)

	require.NoError(t, db.Where("name = ?", "Indie Rock").First(&genreIndie).Error)
	require.Equal(t, int64(1), genreIndie.PlayCount)

	require.NoError(t, db.Where("name = ?", "Alternative Rock").First(&genreAlt).Error)
	require.Equal(t, int64(1), genreAlt.PlayCount)
}
