package common

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateTrackInfo(t *testing.T) {
	ctx := context.Background()

	// 测试有效的参数
	err := ValidateTrackInfo(ctx, "Artist Name", "Album Title", "Track Name")
	assert.NoError(t, err)

	// 测试空的艺术家名称
	err = ValidateTrackInfo(ctx, "", "Album Title", "Track Name")
	assert.Error(t, err)
	assert.Equal(t, "艺术家名称不能为空", err.Error())

	// 测试空的专辑名称
	err = ValidateTrackInfo(ctx, "Artist Name", "", "Track Name")
	assert.Error(t, err)
	assert.Equal(t, "专辑名称不能为空", err.Error())

	// 测试空的曲目名称
	err = ValidateTrackInfo(ctx, "Artist Name", "Album Title", "")
	assert.Error(t, err)
	assert.Equal(t, "歌曲名称不能为空", err.Error())

	// 测试只包含空格的艺术家名称
	err = ValidateTrackInfo(ctx, "   ", "Album Title", "Track Name")
	assert.Error(t, err)
	assert.Equal(t, "艺术家名称不能为空", err.Error())

	// 测试只包含空格的专辑名称
	err = ValidateTrackInfo(ctx, "Artist Name", "   ", "Track Name")
	assert.Error(t, err)
	assert.Equal(t, "专辑名称不能为空", err.Error())

	// 测试只包含空格的曲目名称
	err = ValidateTrackInfo(ctx, "Artist Name", "Album Title", "   ")
	assert.Error(t, err)
	assert.Equal(t, "歌曲名称不能为空", err.Error())
}

func TestUnityFeatFix(t *testing.T) {
	testCases := []string{
		"Hikky Burr (feat. Bill Cosby)",
		"太阳 (feat. Jukka Ahonen)",
		"太阳(feat.Jukka Ahonen)",
		"Somethin' (Feat. Anyone)",
		"Let it be (defeating)",
		"No feat here",
		"Song (Original Mix) (feat. Artist)",
		"万能青年旅店",
	}

	for _, tc := range testCases {
		fmt.Printf("'%s' -> '%s'\n", tc, UnityFeatFix(tc))
	}
}
