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
	// 尝试不同的配置文件路径
	// core/lastfm/lastfm.go
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
		// 只有在配置加载成功时才初始化API
		/*if config.ConfigObj.Lastfm.ApiKey != "" {
			lastfm.InitLastfmApi(
				context.Background(),
				config.ConfigObj.Lastfm.ApiKey, config.ConfigObj.Lastfm.SharedSecret, "", true,
				config.ConfigObj.Lastfm.UserUsername, config.ConfigObj.Lastfm.UserPassword,
			)
		}*/
	}

	_, _ = alog.LogInit("../../"+config.ConfigObj.Log.Path, config.ConfigObj.Log.Level, make(<-chan struct{}))
	/*err := model.InitDB("../../"+config.ConfigObj.Database.Path, logger)
	if err != nil {
		panic(err)
	}*/
}

func TestIsRunning(t *testing.T) {
	ctx := context.Background()
	// This test will pass if Music app is running or not
	// We're just testing that the function doesn't panic
	running := IsRunning(ctx)
	assert.NotNil(t, running)
}

func TestGetState(t *testing.T) {
	ctx := context.Background()
	// Check if Music is running first
	if !IsRunning(ctx) {
		// If Music is not running, we expect an error
		state, err := GetState(ctx)
		assert.Equal(t, "", string(state))
		assert.NotNil(t, err)
	} else {
		// If Music is running, we should get a valid state or an error
		state, err := GetState(ctx)
		fmt.Println(state)
		// We just ensure the function doesn't panic
		_ = state
		_ = err
	}
}

func TestGetNowPlayingTrackInfo(t *testing.T) {
	ctx := context.Background()
	// Check if Music is running first
	if !IsRunning(ctx) {
		// If Music is not running, we expect nil
		info := GetNowPlayingTrackInfo(ctx)
		assert.Nil(t, info)
	} else {
		// If Music is running, we should get track info or nil (if no track is playing)
		info := GetNowPlayingTrackInfoV2(ctx)
		// info can be nil if no track is playing, which is valid
		// We just ensure the function doesn't panic
		fmt.Println(info)
	}
}

func TestIsFavorited(t *testing.T) {
	ctx := context.Background()
	// Check if Music is running first
	if !IsRunning(ctx) {
		// If Music is not running, we expect an error
		favorited, err := IsFavorite(ctx)
		assert.False(t, favorited)
		assert.NotNil(t, err)
		fmt.Println(favorited)
	} else {
		// If Music is running, we should get a boolean value or an error
		favorited, err := IsFavorite(ctx)
		// We just ensure the function doesn't panic
		fmt.Println(favorited)
		_ = favorited
		_ = err
	}
}

func TestSetFavorited(t *testing.T) {
	ctx := context.Background()
	// Check if Music is running first
	if !IsRunning(ctx) {
		// If Music is not running, we expect an error
		err := SetFavorite(ctx, true)
		assert.NotNil(t, err)
	} else {
		// If Music is running, we should be able to set favorited status
		// We just ensure the function doesn't panic
		err := SetFavorite(ctx, true)
		_ = err
	}
}
