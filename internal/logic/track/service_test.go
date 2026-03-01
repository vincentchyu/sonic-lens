package track

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/lastfm"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

func init() {
	c := make(chan struct{})
	// 尝试不同的配置文件路径
	// core/lastfm/lastfm.go
	configPaths := []string{"../../../config/config_bak.yaml"}
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
		_, _ = log.LogInit(config.ConfigObj.Log.Path, config.ConfigObj.Log.Level, c)
		// 只有在配置加载成功时才初始化API
		if config.ConfigObj.Lastfm.ApiKey != "" {
			lastfm.InitLastfmApi(
				context.Background(),
				config.ConfigObj.Lastfm.ApiKey, config.ConfigObj.Lastfm.SharedSecret, "", true,
				config.ConfigObj.Lastfm.UserUsername, config.ConfigObj.Lastfm.UserPassword,
			)
		}
	}

	logger, _ := log.LogInit("../../../"+config.ConfigObj.Log.Path, config.ConfigObj.Log.Level, make(<-chan struct{}))
	err := model.InitDB("../../../"+config.ConfigObj.Database.Path, logger)
	if err != nil {
		panic(err)
	}
}

func TestName(t *testing.T) {
	ctx := context.Background()
	tracks, err := model.GetTracks(ctx, 500, 0, "")
	if err != nil {
		panic(err)
	}
	for _, track := range tracks {
		time.Sleep(time.Second)
		if !track.IsLastFmFav {
			favorite, err := lastfm.IsFavorite(ctx, track.Artist, track.Track)
			if err != nil {
				log.Warn(ctx, "IsFavorite fail", zap.String("track", track.Track), zap.Error(err))
			}
			if favorite {
				err := model.SetLastFmFavorite(
					model.SetFavoriteParams{
						Ctx:        ctx,
						Artist:     track.Artist,
						Album:      track.Album,
						Track:      track.Track,
						IsFavorite: true,
						TrackMetadata: model.TrackMetadata{
							Source: track.Source,
						},
					},
				)
				if err != nil {
					panic(err)
				}
				fmt.Println(track.Track + "db 喜欢状态更新成功")
			}
		}
	}
}
