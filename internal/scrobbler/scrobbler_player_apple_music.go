package scrobbler

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/applemusic"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/internal/cache"
)

// AppleMusicTrackInfoWrapper 包装 AppleMusic TrackInfo 以实现 PlayerInfoHandler 接口
type AppleMusicTrackInfoWrapper struct {
	*applemusic.TrackInfo
	baseWrapper BaseWrapper
}

func (a *AppleMusicTrackInfoWrapper) GetTitle() string {
	return a.baseWrapper.ConversionSimplified(a.Title)
}

func (a *AppleMusicTrackInfoWrapper) GetAlbum() string {
	return a.baseWrapper.ConversionSimplified(a.Album)
}

func (a *AppleMusicTrackInfoWrapper) GetArtist() string {
	return a.baseWrapper.ConversionSimplified(common.ArtistCustomFit(a.Artist))
}

func (a *AppleMusicTrackInfoWrapper) GetPosition() float64 {
	return a.Position
}

func (a *AppleMusicTrackInfoWrapper) GetDuration() int64 {
	return a.Duration
}

func (a *AppleMusicTrackInfoWrapper) GetUrl() string {
	return a.Url
}

// 新增方法实现
func (a *AppleMusicTrackInfoWrapper) GetAlbumArtist() string {
	return a.baseWrapper.ConversionSimplified(common.ArtistCustomFit(a.Artist))
}

func (a *AppleMusicTrackInfoWrapper) GetTrackNumber() int64 {
	return int64(a.TrackNumber)
}

func (a *AppleMusicTrackInfoWrapper) GetGenre() string {
	return cache.GetEnglishGenre(common.GenreCustomFit(a.Genre))
}

func (a *AppleMusicTrackInfoWrapper) GetComposer() string {
	return a.baseWrapper.ConversionSimplified(a.Composer)
}

func (a *AppleMusicTrackInfoWrapper) GetReleaseDate() string {
	if !a.ReleaseDate.IsZero() {
		return a.ReleaseDate.Format("2006-01-02")
	}
	return ""
}

func (a *AppleMusicTrackInfoWrapper) GetMusicBrainzID() string {
	// Apple Music没有直接提供MusicBrainz ID
	return ""
}

func (a *AppleMusicTrackInfoWrapper) GetSource() string {
	return fmt.Sprintf("%d", a.DatabaseID)
}

func (a *AppleMusicTrackInfoWrapper) GetBundleID() string {
	return a.BundleIdentifier
}

func (a *AppleMusicTrackInfoWrapper) GetUniqueID() string {
	return fmt.Sprintf("%d", a.DatabaseID)
}

// AppleMusicPlayerController Apple Music播放器控制器
type AppleMusicPlayerController struct{}

func (a *AppleMusicPlayerController) IsRunning(ctx context.Context) bool {
	return applemusic.IsRunning(ctx)
}

func (a *AppleMusicPlayerController) GetState(ctx context.Context) (string, error) {
	state, err := applemusic.GetState(ctx)
	return string(state), err
}

func (a *AppleMusicPlayerController) GetNowPlayingTrackInfo(ctx context.Context) PlayerInfoHandler {
	info := applemusic.GetNowPlayingTrackInfoV2(ctx)
	if info == nil {
		return nil
	}
	return &AppleMusicTrackInfoWrapper{info, BaseWrapper{}}
}

func (a *AppleMusicPlayerController) SetFavorite(ctx context.Context) error {
	favorite := a.IsFavorite(ctx)
	if !favorite {
		err := applemusic.SetFavorite(ctx, true)
		if err != nil {
			log.Warn(ctx, "AppleMusicPlayerController SetFavorite", zap.Error(err))
			return err
		}
	}
	return nil
}
func (a *AppleMusicPlayerController) IsFavorite(ctx context.Context) bool {
	favorite, err := applemusic.IsFavorite(ctx)
	if err != nil {
		log.Warn(ctx, "AppleMusicPlayerController IsFavorite", zap.Error(err))
		return false
	}
	return favorite
}
