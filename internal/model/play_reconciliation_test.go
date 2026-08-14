package model

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestReconcileAlbumPlayCounts(t *testing.T) {
	db := newTrackResolutionTestDB(t, "reconcile_album_play_counts")

	// 1. 初始化专辑数据 (album.play_count 初始为 0)
	album1 := &Album{
		ID:          1001,
		Name:        "冀西南林路行",
		Artist:      "万能青年旅店",
		SyncStatus:  3,
		PlayCount:   0,
		ReleaseDate: "2020-12-22",
	}
	album2 := &Album{
		ID:          1002,
		Name:        "OK Computer",
		Artist:      "Radiohead",
		SyncStatus:  3,
		PlayCount:   0,
		ReleaseDate: "1997-05-21",
	}
	require.NoError(t, db.Create(album1).Error)
	require.NoError(t, db.Create(album2).Error)

	// 2. 插入听歌流水 (track_play_records 真实物理表)
	// album1 有 3 笔流水记录，album2 有 1 笔流水记录
	now := time.Now()
	records := []TrackPlayRecord{
		{
			Artist:          "万能青年旅店",
			Album:           "冀西南林路行",
			Track:           "河北墨麒麟",
			AlbumID:         1001,
			ResolvedTrackID: 501,
			PlayTime:        now.Add(-10 * time.Minute),
			Source:          "Apple Music",
		},
		{
			Artist:          "万能青年旅店",
			Album:           "冀西南林路行",
			Track:           "采石",
			AlbumID:         1001,
			ResolvedTrackID: 502,
			PlayTime:        now.Add(-5 * time.Minute),
			Source:          "Apple Music",
		},
		{
			Artist:          "万能青年旅店",
			Album:           "冀西南林路行",
			Track:           "河北墨麒麟",
			AlbumID:         1001,
			PlayTime:        now.Add(-2 * time.Minute),
			Source:          "Audirvana",
		},
		{
			Artist:          "Radiohead",
			Album:           "OK Computer",
			Track:           "Airbag",
			AlbumID:         1002,
			ResolvedTrackID: 601,
			PlayTime:        now.Add(-1 * time.Minute),
			Source:          "Apple Music",
		},
	}
	require.NoError(t, db.Create(&records).Error)

	// 3. 执行单专辑及全量对账
	ctx := context.Background()

	// 3.1 指定单专辑对账 (album1)
	require.NoError(t, ReconcileAlbumPlayCounts(ctx, 1001))

	var updated1 Album
	require.NoError(t, db.First(&updated1, 1001).Error)
	require.Equal(t, int64(3), updated1.PlayCount)

	var updated2Temp Album
	require.NoError(t, db.First(&updated2Temp, 1002).Error)
	require.Equal(t, int64(0), updated2Temp.PlayCount) // 尚未对账 1002

	// 3.2 全量对账
	require.NoError(t, ReconcileAlbumPlayCounts(ctx))

	var updated2 Album
	require.NoError(t, db.First(&updated2, 1002).Error)
	require.Equal(t, int64(1), updated2.PlayCount)
}

func TestReconcileGenrePlayCounts(t *testing.T) {
	db := newTrackResolutionTestDB(t, "reconcile_genre_play_counts")

	// 1. 初始化标准英文流派库与一条历史脏数据
	require.NoError(t, db.Create(&Genre{Name: "Alternative Rock", NameZh: "另类摇滚"}).Error)
	require.NoError(t, db.Create(&Genre{Name: "Indie Rock", NameZh: "独立摇滚"}).Error)
	require.NoError(t, db.Create(&Genre{Name: "cn-slug-4edff828", NameZh: "当代歌手", PlayCount: 10}).Error)
	require.NoError(t, db.Create(&Genre{Name: "歌曲作者", NameZh: "", PlayCount: 5}).Error)

	// 2. 插入流水包含单标签与多段组合流派数据（多段组合流派按规则只取首个 Segment）
	now := time.Now()
	records := []TrackPlayRecord{
		{Artist: "Radiohead", Track: "Karma Police", Genre: "Alternative Rock, Rock", PlayTime: now},
		{Artist: "Radiohead", Track: "Creep", Genre: "Alternative Rock", PlayTime: now},
		{Artist: "万能青年旅店", Track: "河北墨麒麟", Genre: "Indie Rock", PlayTime: now},
		{Artist: "未知歌手", Track: "未知曲目", Genre: "完全陌生的未归因流派", PlayTime: now},
	}
	require.NoError(t, db.Create(&records).Error)

	// 3. 执行流派对账（直接针对测试 DB 实例）
	require.NoError(t, ReconcileGenrePlayCountsTx(db))

	// 4. 断言纠偏后的 genre 表播放数（多段流派 "Alternative Rock, Rock" 仅取首个 "Alternative Rock" 计数 1 次，全量为 2 次）
	var genre1, genre2, dirty1, dirty2 Genre
	require.NoError(t, db.Where("name = ?", "Alternative Rock").First(&genre1).Error)
	require.Equal(t, int64(2), genre1.PlayCount)

	require.NoError(t, db.Where("name = ?", "Indie Rock").First(&genre2).Error)
	require.Equal(t, int64(1), genre2.PlayCount)

	// 断言历史 cn-slug- 伪流派与中文 Name 脏流派 play_count 被安全置 0
	require.NoError(t, db.Where("name = ?", "cn-slug-4edff828").First(&dirty1).Error)
	require.Equal(t, int64(0), dirty1.PlayCount)
	require.NoError(t, db.Where("name = ?", "歌曲作者").First(&dirty2).Error)
	require.Equal(t, int64(0), dirty2.PlayCount)

	// 断言物理表没有为“完全陌生的未归因流派”自建任何新记录
	var totalGenreCount int64
	require.NoError(t, db.Model(&Genre{}).Count(&totalGenreCount).Error)
	require.Equal(t, int64(4), totalGenreCount) // 仅原先的 4 条（2标准+2历史脏记录）

	// 5. 校验未归因/未匹配流派曝光列表
	unmatched := GetUnmatchedGenres()
	require.Len(t, unmatched, 1)
	require.Equal(t, "完全陌生的未归因流派", unmatched[0].RawGenre)
	require.Equal(t, int64(1), unmatched[0].PlayCount)
}

func TestGetAlbumsByGenreExactMatching(t *testing.T) {
	db := newTrackResolutionTestDB(t, "get_albums_by_genre_exact_matching")

	albums := []Album{
		{ID: 2001, Artist: "Band A", Name: "Album A", Genre: "Alternative Rock, Rock", PlayCount: 100},
		{ID: 2002, Artist: "Band B", Name: "Album B", Genre: "Rock Musical", PlayCount: 50},
		{ID: 2003, Artist: "Band C", Name: "Album C", Genre: "Indie Rock", PlayCount: 30},
	}
	require.NoError(t, db.Create(&albums).Error)

	ctx := context.Background()
	// 检索 "Rock" 专辑列表时，只应该精准匹配 Album A (Genre 包含独立的 Rock 词汇)，绝不应该混入 "Rock Musical" 或 "Indie Rock"
	matchedRock, err := GetAlbumsByGenre(ctx, "Rock", 10, 0, "play_count")
	require.NoError(t, err)
	require.Len(t, matchedRock, 1)
	require.Equal(t, int64(2001), matchedRock[0].ID)

	// 检索 "Rock Musical" 时只应该精准匹配 Album B
	matchedMusical, err := GetAlbumsByGenre(ctx, "Rock Musical", 10, 0, "play_count")
	require.NoError(t, err)
	require.Len(t, matchedMusical, 1)
	require.Equal(t, int64(2002), matchedMusical[0].ID)

	// 测试从 track 表反向归因的专辑匹配
	require.NoError(t, db.Create(&Album{ID: 2004, Artist: "Band D", Name: "Album D", Genre: ""}).Error)
	require.NoError(t, db.Create(&Track{ID: 3001, Artist: "Band D", Album: "Album D", Track: "Track 1", Genre: "Adult Alternative"}).Error)
	require.NoError(t, db.Create(&TrackAlbum{ID: 4001, AlbumID: 2004, TrackID: 3001, TrackNumber: 1, DiscNumber: 1}).Error)

	matchedAdult, err := GetAlbumsByGenre(ctx, "Adult Alternative", 10, 0, "play_count")
	require.NoError(t, err)
	require.Len(t, matchedAdult, 1)
	require.Equal(t, int64(2004), matchedAdult[0].ID)

	// 测试传入带 %20 的未完全解码字符串，DAO 层自动 Unescape 命中
	matchedEncoded, err := GetAlbumsByGenre(ctx, "Adult%20Alternative", 10, 0, "play_count")
	require.NoError(t, err)
	require.Len(t, matchedEncoded, 1)
	require.Equal(t, int64(2004), matchedEncoded[0].ID)

	// 测试传入带 %2520 的二次转义字符串，DAO 层防御性 Unescape 命中
	matchedDoubleEncoded, err := GetAlbumsByGenre(ctx, "Adult%2520Alternative", 10, 0, "play_count")
	require.NoError(t, err)
	require.Len(t, matchedDoubleEncoded, 1)
	require.Equal(t, int64(2004), matchedDoubleEncoded[0].ID)
}

func TestIncrementAlbumPlayCountTx(t *testing.T) {
	db := newTrackResolutionTestDB(t, "increment_album_play_count_tx")

	album := &Album{
		ID:        2001,
		Name:      "生如夏花",
		Artist:    "朴树",
		PlayCount: 5,
	}
	require.NoError(t, db.Create(album).Error)

	require.NoError(t, InTx(context.Background(), func(tx *gorm.DB) error {
		return IncrementAlbumPlayCountTx(tx, 2001)
	}))

	var updated Album
	require.NoError(t, db.First(&updated, 2001).Error)
	require.Equal(t, int64(6), updated.PlayCount)
}

func TestPopulateAlbumsPlayCount(t *testing.T) {
	db := newTrackResolutionTestDB(t, "populate_albums_play_count")

	album := &Album{
		ID:        3001,
		Name:      "草霉声明",
		Artist:    "刺猬",
		PlayCount: 0,
	}
	require.NoError(t, db.Create(album).Error)

	now := time.Now()
	records := []TrackPlayRecord{
		{Artist: "刺猬", Album: "草霉声明", Track: "火车驶向云外", AlbumID: 3001, PlayTime: now},
		{Artist: "刺猬", Album: "草霉声明", Track: "二十一世纪", AlbumID: 3001, PlayTime: now},
	}
	require.NoError(t, db.Create(&records).Error)

	ctx := context.Background()
	albums := []*Album{album}
	PopulateAlbumsPlayCount(ctx, albums)

	require.Equal(t, int64(2), album.PlayCount)
}
