package scrobbler

import (
	"context"
	"fmt"
	"strings"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/exec"
)

var (
	getMediaControlNowPlayingFn    = exec.GetMediaControlNowPlaying
	getMediaControlNowPlayingImgFn = exec.GetMediaControlNowPlayingImg
)

// MediaControlTrackInfoWrapper 包装 media-control 返回结果以实现 PlayerInfoHandler。
type MediaControlTrackInfoWrapper struct {
	*exec.MediaControlNowPlayingInfo
	baseWrapper BaseWrapper
	source      common.PlayerType
}

func (m *MediaControlTrackInfoWrapper) GetTitle() string {
	return m.baseWrapper.ConversionSimplified(common.UnityFixAll(common.TrackCustomFit(m.Title)))
}

func (m *MediaControlTrackInfoWrapper) GetAlbum() string {
	return m.baseWrapper.ConversionSimplified(m.Album)
}

func (m *MediaControlTrackInfoWrapper) GetAlbumSubtitle() string {
	return ""
}

func (m *MediaControlTrackInfoWrapper) GetAlbumTitleMetadata() *common.AlbumTitleMetadata {
	return nil
}

func (m *MediaControlTrackInfoWrapper) GetArtist() string {
	splits := strings.Split(m.Artist, ",")
	if len(splits) > 0 {
		return m.baseWrapper.ConversionSimplified(splits[0])
	}
	return m.baseWrapper.ConversionSimplified(m.Artist)
}

func (m *MediaControlTrackInfoWrapper) GetPosition() float64 {
	return m.ElapsedTimeNow
}

func (m *MediaControlTrackInfoWrapper) GetDuration() int64 {
	return int64(m.Duration)
}

func (m *MediaControlTrackInfoWrapper) GetSampleRate() int64 {
	return 0
}

func (m *MediaControlTrackInfoWrapper) GetUrl() string {
	return ""
}

func (m *MediaControlTrackInfoWrapper) GetAlbumArtist() string {
	return m.baseWrapper.ConversionSimplified(preferredAlbumArtist("", m.GetArtist()))
}

func (m *MediaControlTrackInfoWrapper) GetTrackNumber() int64 {
	return int64(m.TrackNumber)
}

func (m *MediaControlTrackInfoWrapper) GetGenre() string {
	return m.baseWrapper.GetGenre(m.Genre)
}

func (m *MediaControlTrackInfoWrapper) GetComposer() string {
	return ""
}

func (m *MediaControlTrackInfoWrapper) GetReleaseDate() string {
	return ""
}

func (m *MediaControlTrackInfoWrapper) GetOriginalReleaseDate() string {
	return ""
}

func (m *MediaControlTrackInfoWrapper) GetMusicBrainzID() string {
	return ""
}

func (m *MediaControlTrackInfoWrapper) GetSource() string {
	return string(m.source)
}

func (m *MediaControlTrackInfoWrapper) GetBundleID() string {
	return m.BundleIdentifier
}

func (m *MediaControlTrackInfoWrapper) GetUniqueID() string {
	return m.ContentItemIdentifier
}

func (m *MediaControlTrackInfoWrapper) GetDiscNumber() int8 {
	return 1
}

func (m *MediaControlTrackInfoWrapper) GetArtwork(ctx context.Context) (*common.ArtworkData, error) {
	cacheKey := m.GetArtworkKey(ctx)
	controlNowPlayingImg, err := getMediaControlNowPlayingImgFn(ctx)
	if err != nil {
		return nil, err
	}

	return &common.ArtworkData{
		Data:     controlNowPlayingImg.ArtworkData,
		MimeType: controlNowPlayingImg.ArtworkMimeType,
		CacheKey: cacheKey,
	}, nil
}

func (m *MediaControlTrackInfoWrapper) GetArtworkKey(ctx context.Context) string {
	return fmt.Sprintf("%s:%s:%s:%s", m.GetSource(), m.GetUniqueID(), m.Album, m.Artist)
}

// MediaControlPlayerController 通用 media-control 播放器控制器。
type MediaControlPlayerController struct {
	source   common.PlayerType
	bundleID string
}

func (m *MediaControlPlayerController) IsRunning(ctx context.Context) bool {
	spanCtx, span := startPlayerControllerSpan(ctx, m.source, "is_running")
	defer span.End()

	playing, err := getMediaControlNowPlayingFn(spanCtx)
	if err != nil {
		markPlayerSpanError(span, err)
		return false
	}
	return playing.BundleIdentifier == m.bundleID
}

func (m *MediaControlPlayerController) GetState(ctx context.Context) (string, error) {
	spanCtx, span := startPlayerControllerSpan(ctx, m.source, "get_state")
	defer span.End()

	playing, err := getMediaControlNowPlayingFn(spanCtx)
	if err != nil {
		markPlayerSpanError(span, err)
		return "", err
	}
	if playing.Playing {
		return common.PlayerStatePlaying, nil
	}
	return common.PlayerStateStopped, nil
}

func (m *MediaControlPlayerController) GetNowPlayingTrackInfo(ctx context.Context) common.PlayerInfoHandler {
	spanCtx, span := startPlayerControllerSpan(ctx, m.source, "get_now_playing")
	defer span.End()

	playing, err := getMediaControlNowPlayingFn(spanCtx)
	if err != nil {
		markPlayerSpanError(span, err)
		return nil
	}
	return &MediaControlTrackInfoWrapper{
		MediaControlNowPlayingInfo: playing,
		baseWrapper:                BaseWrapper{},
		source:                     m.source,
	}
}

func (m *MediaControlPlayerController) SetFavorite(ctx context.Context) error {
	return nil
}

func (m *MediaControlPlayerController) IsFavorite(ctx context.Context) bool {
	return false
}
