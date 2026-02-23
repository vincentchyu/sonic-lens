package scrobbler

import (
	"context"
	"strings"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/exec"
	"github.com/vincentchyu/sonic-lens/internal/cache"
)

// RoonTrackInfoWrapper 包装 MRMediaNowPlaying 以实现 PlayerInfoHandler 接口
type RoonTrackInfoWrapper struct {
	*exec.MediaControlNowPlayingInfo
	baseWrapper BaseWrapper
}

func (r *RoonTrackInfoWrapper) GetTitle() string {
	return r.baseWrapper.ConversionSimplified(r.Title)
}

func (r *RoonTrackInfoWrapper) GetAlbum() string {
	return r.baseWrapper.ConversionSimplified(r.Album)
}

func (r *RoonTrackInfoWrapper) GetArtist() string {
	splits := strings.Split(r.Artist, ",")
	if len(splits) > 0 {
		return r.baseWrapper.ConversionSimplified(splits[0])
	}
	return r.baseWrapper.ConversionSimplified(r.Artist)
}

func (r *RoonTrackInfoWrapper) GetPosition() float64 {
	return r.ElapsedTimeNow
}

func (r *RoonTrackInfoWrapper) GetDuration() int64 {
	return int64(r.Duration)
}

func (r *RoonTrackInfoWrapper) GetUrl() string {
	return ""
}

// 新增方法实现
func (r *RoonTrackInfoWrapper) GetAlbumArtist() string {
	// Roon没有直接提供专辑艺术家信息，使用普通艺术家作为默认值
	return r.baseWrapper.ConversionSimplified(r.Artist)
}

func (r *RoonTrackInfoWrapper) GetTrackNumber() int64 {
	// Roon没有直接提供曲目编号
	return int64(r.TrackNumber)
}

func (r *RoonTrackInfoWrapper) GetGenre() string {
	// Roon没有直接提供流派信息
	return cache.GetEnglishGenre(r.Genre)
}

func (r *RoonTrackInfoWrapper) GetComposer() string {
	// Roon没有直接提供作曲家信息
	return ""
}

func (r *RoonTrackInfoWrapper) GetReleaseDate() string {
	// Roon没有直接提供发布日期
	return ""
}

func (r *RoonTrackInfoWrapper) GetMusicBrainzID() string {
	// Roon没有直接提供MusicBrainz ID
	return ""
}

func (r *RoonTrackInfoWrapper) GetSource() string {
	return "Roon"
}

func (r *RoonTrackInfoWrapper) GetBundleID() string {
	// 从BundleIdentifier获取
	return r.BundleIdentifier
}

func (r *RoonTrackInfoWrapper) GetUniqueID() string {
	// Roon没有直接提供唯一标识符
	return r.ContentItemIdentifier
}

// RoonPlayerController Roon播放器控制器
type RoonPlayerController struct{}

func (r *RoonPlayerController) IsRunning(ctx context.Context) bool {
	// todo
	playing, err := exec.GetMediaControlNowPlaying(ctx)
	if err != nil {
		return false
	}
	return playing.BundleIdentifier == exec.MRMediaNowPlayingAppRoon
}

func (r *RoonPlayerController) GetState(ctx context.Context) (string, error) {
	playing, err := exec.GetMediaControlNowPlaying(ctx)
	if err != nil {
		return "", err
	}
	if playing.Playing {
		return common.PlayerStatePlaying, nil
	}
	return common.PlayerStateStopped, nil
}

func (r *RoonPlayerController) GetNowPlayingTrackInfo(ctx context.Context) PlayerInfoHandler {
	playing, err := exec.GetMediaControlNowPlaying(ctx)
	if err != nil {
		return nil
	}
	return &RoonTrackInfoWrapper{playing, BaseWrapper{}}
}

func (a *RoonPlayerController) SetFavorite(ctx context.Context) error {
	return nil
}
func (a *RoonPlayerController) IsFavorite(ctx context.Context) bool {
	return false
}
