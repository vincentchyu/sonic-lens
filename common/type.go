package common

import (
	"context"
)

// ArtworkData 定义播放器专辑封面载荷。
type ArtworkData struct {
	Data     []byte
	MimeType string
	CacheKey string
}

// PlayerInfoHandler 定义播放器信息接口
type PlayerInfoHandler interface {
	GetTitle() string
	GetAlbum() string
	GetArtist() string
	GetPosition() float64
	GetDuration() int64
	GetUrl() string // Audirvana 特有

	// 新增的方法以支持Track模型的更多字段
	GetAlbumArtist() string   // 专辑艺术家
	GetTrackNumber() int64    // 曲目编号
	GetGenre() string         // 流派
	GetComposer() string      // 作曲家
	GetReleaseDate() string   // 发布日期
	GetMusicBrainzID() string // MusicBrainz ID
	GetSource() string        // 数据来源
	GetBundleID() string      // 应用标识符
	GetUniqueID() string      // 唯一标识符
	GetDiscNumber() int8      // 盘位
}

// ArtworkProvider 定义可选的封面能力接口。
type ArtworkProvider interface {
	GetArtwork(ctx context.Context) (*ArtworkData, error)
	GetArtworkKey(ctx context.Context) string
}

// PlayerController 定义播放器控制接口
type PlayerController interface {
	IsRunning(ctx context.Context) bool
	IsFavorite(ctx context.Context) bool
	GetState(ctx context.Context) (string, error)
	GetNowPlayingTrackInfo(ctx context.Context) PlayerInfoHandler
	SetFavorite(ctx context.Context) error
}

// PlayerChecker 定义播放器检查器接口
type PlayerChecker interface {
	CheckPlayingTrack(ctx context.Context, stop <-chan struct{})
}
