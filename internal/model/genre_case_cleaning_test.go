package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGenreIdentityCaseProtectionAndReverseCleaning(t *testing.T) {
	db := newTrackResolutionTestDB(t, "genre_case_protection_reverse_clean")

	// 1. 在 genre 物理表中预先初始化规范大写的权威元数据记录 (Alternative Rock, Classic Rock)
	genreAlt := Genre{Name: "Alternative Rock", NameZh: "另类摇滚", PlayCount: 0}
	genreClassic := Genre{Name: "Classic Rock", NameZh: "经典摇滚", PlayCount: 0}
	require.NoError(t, db.Create(&genreAlt).Error)
	require.NoError(t, db.Create(&genreClassic).Error)

	// 2. 插入带有大小写不规范、中文、多段拼接的听歌流水记录
	now := time.Now()
	records := []TrackPlayRecord{
		{Artist: "Band 1", Track: "Song 1", Genre: "Alternative rock", PlayTime: now},        // 小写 rock
		{Artist: "Band 2", Track: "Song 2", Genre: "Alternative Rock, Rock", PlayTime: now},  // 多段拼接
		{Artist: "Band 3", Track: "Song 3", Genre: "另类摇滚", PlayTime: now},                  // 中文
		{Artist: "Band 4", Track: "Song 4", Genre: "classic rock", PlayTime: now},            // 小写 classic rock
	}
	require.NoError(t, db.Create(&records).Error)

	// 3. 执行流派对账与全量反向清洗
	require.NoError(t, ReconcileGenrePlayCountsTx(db))

	// 4. 断言 A：genre 物理表中的权威 Name 绝对未被小写覆写，且 PlayCount 正确累加为 3 和 1
	var checkAlt, checkClassic Genre
	require.NoError(t, db.Where("id = ?", genreAlt.ID).First(&checkAlt).Error)
	require.Equal(t, "Alternative Rock", checkAlt.Name) // 绝对保持首字母大写规范！
	require.Equal(t, int64(3), checkAlt.PlayCount)

	require.NoError(t, db.Where("id = ?", genreClassic.ID).First(&checkClassic).Error)
	require.Equal(t, "Classic Rock", checkClassic.Name)
	require.Equal(t, int64(1), checkClassic.PlayCount)

	// 5. 断言 B：track_play_records 听歌流水表中的原始脏记录已被 100% 反向更正清洗为标准权威 Name
	var cleanRecords []TrackPlayRecord
	require.NoError(t, db.Order("id ASC").Find(&cleanRecords).Error)
	require.Len(t, cleanRecords, 4)

	require.Equal(t, "Alternative Rock", cleanRecords[0].Genre)
	require.Equal(t, "Alternative Rock", cleanRecords[1].Genre)
	require.Equal(t, "Alternative Rock", cleanRecords[2].Genre)
	require.Equal(t, "Classic Rock", cleanRecords[3].Genre)
}

func TestNormalizeGenre(t *testing.T) {
	db := newTrackResolutionTestDB(t, "normalize_genre_test")

	// 预置权威流派
	require.NoError(t, db.Create(&Genre{Name: "Alternative Rock", NameZh: "另类摇滚"}).Error)
	require.NoError(t, db.Create(&Genre{Name: "Pop Rock", NameZh: "流行摇滚"}).Error)

	// 1. 小写提取与 Title Case
	require.Equal(t, "Folk", NormalizeGenre(db, "folk"))
	require.Equal(t, "Chinese Rock", NormalizeGenre(db, "chinese rock"))
	require.Equal(t, "Alternative Rock", NormalizeGenre(db, "Alternative rock"))
	
	// 2. 多段拼接提取首流派并规范化
	require.Equal(t, "Chinese Rock", NormalizeGenre(db, "chinese rock;alternative rock"))
	require.Equal(t, "Folk", NormalizeGenre(db, "folk / rock"))
	require.Equal(t, "Pop", NormalizeGenre(db, "pop, rock, jazz"))

	// 3. 中文权威流派匹配
	require.Equal(t, "Alternative Rock", NormalizeGenre(db, "另类摇滚"))
	require.Equal(t, "Pop Rock", NormalizeGenre(db, "流行摇滚"))

	// 4. 未认证中文标签绝不生成 cn-slug
	require.Equal(t, "未知流派测试", NormalizeGenre(db, "未知流派测试"))
}

