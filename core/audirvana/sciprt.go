package audirvana

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"

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
		AppName    string

		MataDataHandle exec.MataDataHandle
	}
	TrackInfo struct {
		TrackBase
	}
)

var (
	tellFn     = applesciprt.Tell
	runningApp atomic.Pointer[string]
)

// SupportedApps 列出支持的 Audirvana 应用名称（按探测优先级排序：Studio 优先，其次 Origin）。
var SupportedApps = []string{
	common.AppAudirvanaStudio,
	common.AppAudirvanaOrigin,
}

// ResolveRunningApp 探测并返回当前正在运行的 Audirvana 应用名称（"Audirvana Studio" 或 "Audirvana Origin"）。
// 如果两者均未运行，返回空字符串。
func ResolveRunningApp(ctx context.Context) string {
	script := fmt.Sprintf(
		`set listApplicationProcessNames to name of every application process
		if listApplicationProcessNames contains "%s" then
			set AUDIRVANA_APP to "%s"
		else if listApplicationProcessNames contains "%s" then
			set AUDIRVANA_APP to "%s"
		else
			set AUDIRVANA_APP to ""
		end if
		return AUDIRVANA_APP`,
		common.AppAudirvanaStudio, common.AppAudirvanaStudio,
		common.AppAudirvanaOrigin, common.AppAudirvanaOrigin,
	)

	tell, err := tellFn(ctx, common.AppSystemEvents, script)
	if err != nil {
		runningApp.Store(nil)
		return ""
	}

	app := strings.TrimSpace(tell)
	if slices.Contains(SupportedApps, app) {
		runningApp.Store(&app)
		return app
	}

	runningApp.Store(nil)
	return ""
}

// CurrentOrResolveRunningApp 优先获取已记录的运行中应用名称，若未记录则主动探测一次。
func CurrentOrResolveRunningApp(ctx context.Context) string {
	if p := runningApp.Load(); p != nil && *p != "" {
		return *p
	}
	return ResolveRunningApp(ctx)
}

// ClearRunningAppCache 清空当前运行的应用名称缓存（主要供测试或重试恢复使用）。
func ClearRunningAppCache() {
	runningApp.Store(nil)
}

// IsRunning 使用 AppleScript 检查是否有支持的 Audirvana 应用（Studio 或 Origin）正在运行。
func IsRunning(ctx context.Context) bool {
	return ResolveRunningApp(ctx) != ""
}

// GetState 使用 AppleScript 从正在运行的 Audirvana 应用获取当前播放器状态。
// 返回播放器状态（common.PlayerState）以及过程中遇到的任何错误。
func GetState(ctx context.Context) (playerState common.PlayerState, err error) {
	app := CurrentOrResolveRunningApp(ctx)
	if app == "" {
		return "", fmt.Errorf("audirvana is not running")
	}

	result, err := tellFn(ctx, app, `set audirvanaState to get player state`)
	if err != nil {
		ClearRunningAppCache()
		alog.Warn(ctx, "获取 Audirvana 播放状态失败", zap.String("app", app), zap.Error(err))
		return "", err
	}
	return common.PlayerState(strings.TrimSpace(result)), nil
}

// GetNowPlayingTrackInfo 使用 AppleScript 从正在运行的 Audirvana 应用获取当前正在播放的曲目信息。
// 它返回一个指向 TrackInfo 结构体的指针，包含曲目的标题、专辑、艺术家、时长、播放位置、URL 以及应用标识。
// 如果在执行 AppleScript 或解析数据时发生错误，函数会记录警告并返回 nil。
func GetNowPlayingTrackInfo(ctx context.Context) *TrackInfo {
	app := CurrentOrResolveRunningApp(ctx)
	if app == "" {
		return nil
	}

	tell, err := tellFn(
		ctx,
		app,
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
		ClearRunningAppCache()
		alog.Warn(ctx, "获取 Audirvana 播放曲目失败", zap.String("app", app), zap.Error(err))
		return nil
	}
	split := strings.Split(tell, "|")
	info := &TrackInfo{
		TrackBase: TrackBase{
			AppName: app,
		},
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
			duration, err := parseAudirvanaDuration(strings.TrimSpace(s))
			if err != nil {
				alog.Warn(ctx, "Audirvana 解析曲目时长失败", zap.String("raw_duration", strings.TrimSpace(s)), zap.Error(err))
				return nil
			}
			info.Duration = duration
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

	info.MataDataHandle = cache.FindMataDataHandle(ctx, info.Url)
	if info.MataDataHandle == nil {
		alog.Warn(
			ctx,
			"Audirvana 当前曲目未拿到本地元数据句柄",
			zap.String("app", app),
			zap.String("title", info.Title),
			zap.String("album", info.Album),
			zap.String("artist", info.Artist),
			zap.String("url", info.Url),
		)
	}
	return info
}

func parseAudirvanaDuration(raw string) (int64, error) {
	if raw == "" {
		return 0, fmt.Errorf("empty duration")
	}

	if parseInt, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return parseInt, nil
	}

	parseFloat, err := strconv.ParseFloat(strings.ReplaceAll(raw, ",", "."), 64)
	if err != nil {
		return 0, err
	}

	return int64(math.Round(parseFloat)), nil
}
