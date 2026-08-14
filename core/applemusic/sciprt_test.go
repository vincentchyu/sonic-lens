//go:build integration
// +build integration

package applemusic

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vincentchyu/sonic-lens/config"
	alog "github.com/vincentchyu/sonic-lens/core/log"
)

func init() {
	c := make(chan struct{})
	configPaths := []string{"../../config/config_bak.yaml"}
	configLoaded := false
	for _, path := range configPaths {
		func() {
			defer func() {
				if r := recover(); r != nil {
					// 忽略配置加载失败的情况
				}
			}()
			config.InitConfig(path)
			configLoaded = true
		}()
		if configLoaded {
			break
		}
	}
	if configLoaded {
		_, _ = alog.LogInit(config.ConfigObj.Log.Path, config.ConfigObj.Log.Level, c)
	}
}

func TestIsRunning(t *testing.T) {
	ctx := context.Background()
	running := IsRunning(ctx)
	assert.NotNil(t, running)
}

func TestGetState(t *testing.T) {
	ctx := context.Background()
	if !IsRunning(ctx) {
		state, err := GetState(ctx)
		assert.Equal(t, "", string(state))
		assert.NotNil(t, err)
	} else {
		state, err := GetState(ctx)
		fmt.Println(state)
		_ = state
		_ = err
	}
}

func TestGetNowPlayingTrackInfo(t *testing.T) {
	ctx := context.Background()
	if !IsRunning(ctx) {
		info := GetNowPlayingTrackInfo(ctx)
		assert.Nil(t, info)
	} else {
		info := GetNowPlayingTrackInfoV2(ctx)
		fmt.Println(info)
	}
}

func TestIsFavorited(t *testing.T) {
	ctx := context.Background()
	if !IsRunning(ctx) {
		favorited, err := IsFavorite(ctx)
		assert.False(t, favorited)
		assert.NotNil(t, err)
		fmt.Println(favorited)
	} else {
		favorited, err := IsFavorite(ctx)
		fmt.Println(favorited)
		_ = favorited
		_ = err
	}
}

func TestSetFavorited(t *testing.T) {
	ctx := context.Background()
	if !IsRunning(ctx) {
		err := SetFavorite(ctx, true)
		assert.NotNil(t, err)
	} else {
		err := SetFavorite(ctx, true)
		_ = err
	}
}

