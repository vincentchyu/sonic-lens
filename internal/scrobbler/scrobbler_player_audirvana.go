package scrobbler

import (
	"context"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/audirvana"
	"github.com/vincentchyu/sonic-lens/internal/cache"
)

// AudirvanaTrackInfoWrapper 包装 Audirvana TrackInfo 以实现 PlayerInfoHandler 接口
type AudirvanaTrackInfoWrapper struct {
	*audirvana.TrackInfo
	baseWrapper BaseWrapper
}

func (a *AudirvanaTrackInfoWrapper) GetTitle() string {
	return a.baseWrapper.ConversionSimplified(a.Title)
}

func (a *AudirvanaTrackInfoWrapper) GetAlbum() string {
	return a.baseWrapper.ConversionSimplified(a.MataDataHandle.GetAlbum())
}

func (a *AudirvanaTrackInfoWrapper) GetArtist() string {
	return a.baseWrapper.ConversionSimplified(a.MataDataHandle.GetArtist())
}

func (a *AudirvanaTrackInfoWrapper) GetPosition() float64 {
	return a.Position
}

func (a *AudirvanaTrackInfoWrapper) GetDuration() int64 {
	return a.Duration
}

func (a *AudirvanaTrackInfoWrapper) GetUrl() string {
	return a.Url
}

// 新增方法实现
func (a *AudirvanaTrackInfoWrapper) GetAlbumArtist() string {
	// Audirvana没有直接提供专辑艺术家信息，使用普通艺术家作为默认值
	return a.baseWrapper.ConversionSimplified(a.MataDataHandle.GetArtist())
}

func (a *AudirvanaTrackInfoWrapper) GetTrackNumber() int64 {
	// Audirvana没有直接提供曲目编号
	return a.MataDataHandle.GetTrackNumber()
}

func (a *AudirvanaTrackInfoWrapper) GetGenre() string {
	// Audirvana没有直接提供流派信息
	return cache.GetEnglishGenre(common.GenreCustomFit(a.MataDataHandle.GetGenre()))
}

func (a *AudirvanaTrackInfoWrapper) GetComposer() string {
	// Audirvana没有直接提供作曲家信息
	return a.baseWrapper.ConversionSimplified(a.MataDataHandle.GetComposer())
}

func (a *AudirvanaTrackInfoWrapper) GetReleaseDate() string {
	// Audirvana没有直接提供发布日期
	return a.MataDataHandle.GetReleaseDate()
}

func (a *AudirvanaTrackInfoWrapper) GetMusicBrainzID() string {
	// Audirvana没有直接提供MusicBrainz ID
	return a.MataDataHandle.GetMusicBrainzTrackId()
}

func (a *AudirvanaTrackInfoWrapper) GetSource() string {
	return a.MataDataHandle.GetSource()
}

func (a *AudirvanaTrackInfoWrapper) GetBundleID() string {
	// Audirvana没有直接提供应用标识符
	return a.MataDataHandle.GetBundleID()
}

func (a *AudirvanaTrackInfoWrapper) GetUniqueID() string {
	// 使用URL作为唯一标识符
	return a.MataDataHandle.GetUniqueID()
}

// AudirvanaPlayerController Audirvana播放器控制器
type AudirvanaPlayerController struct{}

func (a *AudirvanaPlayerController) IsRunning(ctx context.Context) bool {
	return audirvana.IsRunning(ctx)
}

func (a *AudirvanaPlayerController) GetState(ctx context.Context) (string, error) {
	state, err := audirvana.GetState(ctx)
	return string(state), err
}

func (a *AudirvanaPlayerController) GetNowPlayingTrackInfo(ctx context.Context) PlayerInfoHandler {
	info := audirvana.GetNowPlayingTrackInfo(ctx)
	if info == nil {
		return nil
	}
	return &AudirvanaTrackInfoWrapper{info, BaseWrapper{}}
}
func (a *AudirvanaPlayerController) SetFavorite(ctx context.Context) error {
	return nil
}

func (a *AudirvanaPlayerController) IsFavorite(ctx context.Context) bool {
	return false
}

// 网易云
/* {
  "playbackRate" : 1,
  "album" : "铸铁旅人",
  "elapsedTimeNow" : 401.89587608909608,
  "elapsedTime" : 297.21600000000001,
  "timestamp" : "2025-09-13T02:53:11Z",
  "bundleIdentifier" : "com.netease.163music",
  "processIdentifier" : 41260,
  "title" : "铸铁旅人",
  "duration" : 520.12697916666662,
  "artist" : "虎啸春",
  "contentItemIdentifier" : "C4B45625-FB20-419B-BFA0-42CCEC333EA4",
  "playing" : true
} */
