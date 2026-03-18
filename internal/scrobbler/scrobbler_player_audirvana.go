package scrobbler

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/audirvana"
	"github.com/vincentchyu/sonic-lens/core/exec"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/internal/cache"
)

// AudirvanaTrackInfoWrapper 包装 Audirvana TrackInfo 以实现 common.PlayerInfoHandler 接口
type AudirvanaTrackInfoWrapper struct {
	*audirvana.TrackInfo
	baseWrapper BaseWrapper
}

func (a *AudirvanaTrackInfoWrapper) GetTitle() string {
	return a.baseWrapper.ConversionSimplified(common.UnityFixAll(common.TrackCustomFit(a.Title)))
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
	if a.MataDataHandle != nil {
		if trackNumber := a.MataDataHandle.GetTrackNumber(); trackNumber > 0 {
			return trackNumber
		}
	}
	trackNumber, _ := a.parseTrackPositionFromURL()
	return trackNumber
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
func (a *AudirvanaTrackInfoWrapper) GetDiscNumber() int8 {
	if a.MataDataHandle != nil {
		if discNumber := a.MataDataHandle.GetDiscNumber(); discNumber > 0 {
			return discNumber
		}
	}
	_, discNumber := a.parseTrackPositionFromURL()
	if discNumber > 0 {
		return int8(discNumber)
	}
	return 1
}

func (a *AudirvanaTrackInfoWrapper) LogResolvedPosition(ctx context.Context) {
	if a == nil {
		return
	}

	metadataTrackNumber := int64(0)
	metadataDiscNumber := int8(0)
	if a.MataDataHandle != nil {
		metadataTrackNumber = a.MataDataHandle.GetTrackNumber()
		metadataDiscNumber = a.MataDataHandle.GetDiscNumber()
	}

	resolvedTrackNumber := a.GetTrackNumber()
	resolvedDiscNumber := a.GetDiscNumber()
	if metadataTrackNumber > 0 && metadataDiscNumber > 0 {
		return
	}

	log.Info(
		ctx, "Audirvana 曲目信息使用兜底解析",
		zap.String("title", a.Title),
		zap.String("album", a.Album),
		zap.String("url", a.Url),
		zap.Int64("metadata_track_number", metadataTrackNumber),
		zap.Int8("metadata_disc_number", metadataDiscNumber),
		zap.Int64("resolved_track_number", resolvedTrackNumber),
		zap.Int8("resolved_disc_number", resolvedDiscNumber),
	)
}

func (a *AudirvanaTrackInfoWrapper) parseTrackPositionFromURL() (int64, int64) {
	if a == nil || a.Url == "" {
		return 0, 0
	}

	rawPath := a.Url
	if parsedURL, err := url.Parse(a.Url); err == nil && parsedURL.Path != "" {
		if unescapedPath, err := url.PathUnescape(parsedURL.Path); err == nil {
			rawPath = unescapedPath
		} else {
			rawPath = parsedURL.Path
		}
	}

	fileName := filepath.Base(rawPath)
	fileName = strings.TrimSuffix(fileName, filepath.Ext(fileName))
	if fileName == "" {
		return 0, 0
	}

	parts := strings.Fields(fileName)
	if len(parts) == 0 {
		return 0, 0
	}

	prefix := strings.TrimSpace(parts[0])
	if prefix == "" {
		return 0, 0
	}

	if strings.Contains(prefix, "-") {
		segments := strings.SplitN(prefix, "-", 2)
		if len(segments) == 2 {
			discNumber, discErr := strconv.ParseInt(strings.TrimSpace(segments[0]), 10, 64)
			trackNumber, trackErr := strconv.ParseInt(strings.TrimSpace(segments[1]), 10, 64)
			if discErr == nil && trackErr == nil {
				return trackNumber, discNumber
			}
		}
	}

	trackNumber, err := strconv.ParseInt(prefix, 10, 64)
	if err == nil {
		return trackNumber, 1
	}

	return 0, 0
}

func (a *AudirvanaTrackInfoWrapper) GetArtwork(ctx context.Context) (*common.ArtworkData, error) {
	// url 后缀是.m4a  应该是在这个 字段里 "CoverArt": "(Binary data 1907659 bytes, use -b option to extract)",
	// exiftool -json '/Users/vincent/Documents/个人/多媒体/音乐/CD/海朋森/成长小说/01 春风.m4a'

	cacheKey := a.GetArtworkKey(ctx)
	// 对于 m4a 文件，即使 HasPicture 返回 false，也尝试提取，因为 exiftool JSON 输出可能不包含该字段
	isM4A := strings.HasSuffix(strings.ToLower(a.Url), ".m4a")
	if (a.MataDataHandle == nil || !a.MataDataHandle.HasPicture()) && !isM4A {
		return nil, nil
	}

	data, err := exec.ExtractEmbeddedArtwork(ctx, a.Url)
	if err != nil {
		if isM4A {
			log.Warn(ctx, "M4A 封面提取失败", zap.String("url", a.Url), zap.Error(err))
			return nil, nil // 兜底返回，不中断流程
		}
		return nil, err
	}

	return &common.ArtworkData{
		Data:     data,
		MimeType: a.MataDataHandle.GetPictureMimeType(),
		CacheKey: cacheKey,
	}, nil
}
func (a *AudirvanaTrackInfoWrapper) GetArtworkKey(ctx context.Context) string {
	cacheKey := fmt.Sprintf("%s:%s:%s", a.GetSource(), a.GetUniqueID(), a.Url)
	return cacheKey
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

func (a *AudirvanaPlayerController) GetNowPlayingTrackInfo(ctx context.Context) common.PlayerInfoHandler {
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
