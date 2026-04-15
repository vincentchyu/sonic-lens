package scrobbler

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/applemusic"
	"github.com/vincentchyu/sonic-lens/core/exec"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/internal/cache"
)

// AppleMusicTrackInfoWrapper 包装 AppleMusic TrackInfo 以实现 PlayerInfoHandler 接口
type AppleMusicTrackInfoWrapper struct {
	*applemusic.TrackInfo
	baseWrapper BaseWrapper
}

func (a *AppleMusicTrackInfoWrapper) GetTitle() string {
	return a.baseWrapper.ConversionSimplified(common.UnityFixAll(common.TrackCustomFit(a.Title)))
}

func (a *AppleMusicTrackInfoWrapper) GetAlbum() string {
	name, _ := common.ParseAlbumTitleAndSubtitle(a.Album)
	return a.baseWrapper.ConversionSimplified(name)
}

func (a *AppleMusicTrackInfoWrapper) GetAlbumSubtitle() string {
	_, subtitle := common.ParseAlbumTitleAndSubtitle(a.Album)
	return a.baseWrapper.ConversionSimplified(subtitle)
}
func (a *AppleMusicTrackInfoWrapper) GetAlbumTitleMetadata() *common.AlbumTitleMetadata {
	titleMetadata := common.ParseAlbumTitleMetadata(a.Album)
	return &titleMetadata
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

func (a *AppleMusicTrackInfoWrapper) GetSampleRate() int64 {
	return int64(a.SampleRate)
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

func (a *AppleMusicTrackInfoWrapper) GetOriginalReleaseDate() string {
	return ""
}

func (a *AppleMusicTrackInfoWrapper) GetMusicBrainzID() string {
	// Apple Music没有直接提供MusicBrainz ID
	return ""
}

func (a *AppleMusicTrackInfoWrapper) GetSource() string {
	return string(common.PlayerAppleMusic)
}

func (a *AppleMusicTrackInfoWrapper) GetBundleID() string {
	return a.BundleIdentifier
}

func (a *AppleMusicTrackInfoWrapper) GetUniqueID() string {
	return fmt.Sprintf("%d", a.DatabaseID)
}

func (a *AppleMusicTrackInfoWrapper) GetDiscNumber() int8 {
	return int8(a.DiscNumber)
}

func (a *AppleMusicTrackInfoWrapper) GetArtwork(ctx context.Context) (*common.ArtworkData, error) {
	ctx, span := startPlayerControllerSpan(ctx, common.PlayerAppleMusic, "get_artwork")
	defer span.End()
	cacheKey := a.GetArtworkKey(ctx)
	controlNowPlayingImg, err := exec.GetMediaControlNowPlayingImg(ctx)
	if err != nil {
		return nil, err
	}

	return &common.ArtworkData{
		Data:     controlNowPlayingImg.ArtworkData,
		MimeType: controlNowPlayingImg.ArtworkMimeType,
		CacheKey: cacheKey,
	}, nil
}
func (a *AppleMusicTrackInfoWrapper) GetArtworkKey(ctx context.Context) string {
	cacheKey := fmt.Sprintf("%s:%s:%s:%s", a.GetSource(), a.GetUniqueID(), a.Album, a.Artist)
	return cacheKey
}

// AppleMusicPlayerController Apple Music播放器控制器
type AppleMusicPlayerController struct{}

func (a *AppleMusicPlayerController) IsRunning(ctx context.Context) bool {
	spanCtx, span := startPlayerControllerSpan(ctx, common.PlayerAppleMusic, "is_running")
	defer span.End()

	running := applemusic.IsRunning(spanCtx)
	return running
}

func (a *AppleMusicPlayerController) GetState(ctx context.Context) (string, error) {
	spanCtx, span := startPlayerControllerSpan(ctx, common.PlayerAppleMusic, "get_state")
	defer span.End()

	state, err := applemusic.GetState(spanCtx)
	markPlayerSpanError(span, err)
	return string(state), err
}

func (a *AppleMusicPlayerController) GetNowPlayingTrackInfo(ctx context.Context) common.PlayerInfoHandler {
	spanCtx, span := startPlayerControllerSpan(ctx, common.PlayerAppleMusic, "get_now_playing")
	defer span.End()

	info := applemusic.GetNowPlayingTrackInfoV2(spanCtx)
	if info == nil {
		return nil
	}
	return &AppleMusicTrackInfoWrapper{info, BaseWrapper{}}
}

func (a *AppleMusicPlayerController) SetFavorite(ctx context.Context) error {
	spanCtx, span := startPlayerControllerSpan(ctx, common.PlayerAppleMusic, "set_favorite")
	defer span.End()

	favorite := a.IsFavorite(spanCtx)
	if !favorite {
		err := applemusic.SetFavorite(spanCtx, true)
		if err != nil {
			log.Warn(spanCtx, "AppleMusicPlayerController SetFavorite", zap.Error(err))
			markPlayerSpanError(span, err)
			return err
		}
	}
	return nil
}
func (a *AppleMusicPlayerController) IsFavorite(ctx context.Context) bool {
	spanCtx, span := startPlayerControllerSpan(ctx, common.PlayerAppleMusic, "is_favorite")
	defer span.End()

	favorite, err := applemusic.IsFavorite(spanCtx)
	if err != nil {
		log.Warn(spanCtx, "AppleMusicPlayerController IsFavorite", zap.Error(err))
		markPlayerSpanError(span, err)
		return false
	}
	return favorite
}
