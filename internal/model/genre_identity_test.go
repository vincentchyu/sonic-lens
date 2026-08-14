package model

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vincentchyu/sonic-lens/common"
)

func TestExtractPrimaryGenreTag(t *testing.T) {
	require.Equal(t, "Alternative Rock", extractPrimaryGenreTag("Alternative Rock,Electronic,Electronica,Experimental,Experimental Rock,Idm,Post-rock,Progressive Rock,Rock"))
	require.Equal(t, "Rock", extractPrimaryGenreTag("Rock / Pop / Funk"))
	require.Equal(t, "Pop", extractPrimaryGenreTag(" Pop ; Jazz "))
	require.Equal(t, "Jazz", extractPrimaryGenreTag("Jazz|Blues"))
}

func TestResolveStrictGenreIdentity(t *testing.T) {
	db := newTrackResolutionTestDB(t, "strict_genre_identity")

	// 1. 初始化标准权威流派
	require.NoError(t, db.Create(&Genre{Name: "World Music", NameZh: "世界音乐"}).Error)
	require.NoError(t, db.Create(&Genre{Name: "Rock", NameZh: "摇滚"}).Error)

	// 2. 中文可通过数据库/缓存匹配出英文 Name
	matched, eng, zh := ResolveStrictGenreIdentity(db, "世界音乐")
	require.True(t, matched)
	require.Equal(t, "World Music", eng)
	require.Equal(t, "世界音乐", zh)

	matched, eng, zh = ResolveStrictGenreIdentity(db, "摇滚")
	require.True(t, matched)
	require.Equal(t, "Rock", eng)
	require.Equal(t, "摇滚", zh)

	// 3. 未能寻址英文 Name 的未已知中文，应当返回 matched = false，待前端人工干预
	matched, eng, zh = ResolveStrictGenreIdentity(db, "未归因流派名称")
	require.False(t, matched)
	require.Equal(t, "", eng)
	require.Equal(t, "未归因流派名称", zh)
	require.True(t, common.IsExistsChineseSimplified(zh))

	// 4. 英文未在物理库中匹配，应当返回 matched = false，待人工干预
	matched, eng, zh = ResolveStrictGenreIdentity(db, "CompletelyUnknownEnglishGenre123")
	require.False(t, matched)
	require.Equal(t, "CompletelyUnknownEnglishGenre123", eng)
}
