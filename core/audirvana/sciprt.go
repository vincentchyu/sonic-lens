package audirvana

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/applesciprt"
	"github.com/vincentchyu/sonic-lens/core/cache"
	"github.com/vincentchyu/sonic-lens/core/exec"
	alog "github.com/vincentchyu/sonic-lens/core/log"
)

type (
	TrackBase struct {
		TrackID    string
		Title      string
		Album      string
		Artist     string
		Duration   int64
		Position   float64
		Url        string
		AirfoiLogo string

		MataDataHandle exec.MataDataHandle
	}
	TrackInfo struct {
		TrackBase
	}
)

func IsRunning(ctx context.Context) bool {
	tell, err := applesciprt.Tell(
		common.AppSystemEvents, fmt.Sprintf(
			`set listApplicationProcessNames to name of every application process
			if listApplicationProcessNames contains "%s" then
				set AUDIRVANA_RUNNING_STATE to "true"
			else
				set AUDIRVANA_RUNNING_STATE to "false"
			end if`, common.AppAudirvanaOrigin,
		),
	)
	if err != nil {
		return false
	}

	parseBool, err := strconv.ParseBool(tell)
	if err != nil {
		alog.Warn(ctx, "err:", zap.Error(err))
		return false
	}
	return parseBool
}

// GetState 使用 AppleScript 从 Audirvana Origin 应用获取当前播放器状态。
// 返回播放器状态（common.PlayerState）以及过程中遇到的任何错误。
func GetState(ctx context.Context) (playerState common.PlayerState, err error) {
	result, err := applesciprt.Tell(common.AppAudirvanaOrigin, `set audirvanaState to get player state`)
	if err != nil {
		alog.Warn(ctx, "err:", zap.Error(err))
		return "", err
	}
	return common.PlayerState(result), nil
}

// GetNowPlayingTrackInfo 使用 AppleScript 从 Audirvana Origin 获取当前正在播放的曲目信息。
// 它返回一个指向 TrackInfo 结构体的指针，包含曲目的标题、专辑、艺术家、时长、播放位置和 URL。
// 如果在执行 AppleScript 或解析数据时发生错误，函数会记录警告并返回 nil。
func GetNowPlayingTrackInfo(ctx context.Context) *TrackInfo {
	tell, err := applesciprt.Tell(
		common.AppAudirvanaOrigin,
		`set playingTrack to playing track title
	set playingAlbum to playing track album
	set playingArtist to playing track artist
	set playingDuration to playing track duration
	set playingDuration to playingDuration as string
	set playingPosition to player position
	set playingPosition to playingPosition as string
	set playingUrl to playing track url
	set result to playingTrack & "|" & playingAlbum & "|" & playingArtist & "|" & playingDuration & "|" & playingPosition & "|" & playingUrl`,
	)
	if err != nil {
		alog.Warn(ctx, "err:", zap.Error(err))
		return nil
	}
	split := strings.Split(tell, "|")
	info := &TrackInfo{
		TrackBase: TrackBase{},
	}
	for i, s := range split {
		switch i {
		case 0:
			info.Title = strings.TrimSpace(s)
		case 1:
			info.Album = strings.TrimSpace(s)
		case 2:
			info.Artist = strings.TrimSpace(s)
		case 3:
			parseInt, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
			if err != nil {
				alog.Warn(ctx, "err:", zap.Error(err))
				return nil
			}
			info.Duration = parseInt
		case 4:
			parseInt, err := strconv.ParseFloat(strings.TrimSpace(s), 32)
			if err != nil {
				alog.Warn(ctx, "err:", zap.Error(err))
				return nil
			}
			info.Position = parseInt
		case 5:
			unescape, err := url.PathUnescape(s)
			if err != nil {
				alog.Warn(ctx, "err:", zap.Error(err))
				return nil
			}
			info.Url = strings.TrimSpace(unescape)
		case 6:
			// info.AirfoiLogo = s
		}
	}

	info.MataDataHandle = cache.FindMataDataHandle(context.Background(), info.Url)
	return info
}
