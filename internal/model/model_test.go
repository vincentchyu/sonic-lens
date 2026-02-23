package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vincentchyu/sonic-lens/common"
)

// Mock functions for testing without database
func TestInsertTrackPlayRecordValidation(t *testing.T) {
	ctx := context.Background()

	// 测试验证逻辑 - 有效参数
	err := common.ValidateTrackInfo(ctx, "Artist Name", "Album Title", "Track Name")
	assert.NoError(t, err)

	// 测试验证逻辑 - 无效参数（空艺术家）
	err = common.ValidateTrackInfo(ctx, "", "Album Title", "Track Name")
	assert.Error(t, err)
	assert.Equal(t, "艺术家名称不能为空", err.Error())
}

func TestIncrementTrackPlayCountValidation(t *testing.T) {
	ctx := context.Background()

	// 测试验证逻辑 - 有效参数
	err := common.ValidateTrackInfo(ctx, "Artist Name", "Album Title", "Track Name")
	assert.NoError(t, err)

	// 测试验证逻辑 - 无效参数（空艺术家）
	err = common.ValidateTrackInfo(ctx, "", "Album Title", "Track Name")
	assert.Error(t, err)
	assert.Equal(t, "艺术家名称不能为空", err.Error())
}
