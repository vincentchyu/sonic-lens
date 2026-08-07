package common

import (
	"context"
)

type ContextKey string

const (
	ContextKeyJobID ContextKey = "insight_job_id"
)

// ArtworkData 定义播放器专辑封面载荷。
type ArtworkData struct {
	Data     []byte
	MimeType string
	CacheKey string
}

type AlbumTitleVersionType string

const (
	AlbumTitleVersionTypeEdition    AlbumTitleVersionType = "edition"
	AlbumTitleVersionTypeRemaster   AlbumTitleVersionType = "remaster"
	AlbumTitleVersionTypeSoundtrack AlbumTitleVersionType = "soundtrack"
	AlbumTitleVersionTypeMix        AlbumTitleVersionType = "mix"
	AlbumTitleVersionTypeLive       AlbumTitleVersionType = "live"
	AlbumTitleVersionTypeVersion    AlbumTitleVersionType = "version"
	AlbumTitleVersionTypeOther      AlbumTitleVersionType = "other" // 兜底类型
)

type AlbumTitleVersion struct {
	Text          string                `json:"text"`
	Type          AlbumTitleVersionType `json:"type"`
	Bracketed     bool                  `json:"bracketed"`
	Parenthesized bool                  `json:"parenthesized"`
}

type AlbumTitleMetadata struct {
	SourceDisplayTitle     string              `json:"source_display_title"`
	OfficialTitle          string              `json:"official_title"`
	TitleVersions          []AlbumTitleVersion `json:"title_versions"`
	NormalizedDisplayTitle string              `json:"normalized_display_title"`
	// ReleaseType 承载从 Apple Music 连字符尾缀提取的发行类型枚举，
	// 例如 "ep"、"single"、"lp"；如果原标题不含连字符类型后缀则为空字符串。
	// 注意：该字段与 TitleVersions 所描述的"版本说明"（Remaster / Deluxe 等）语义完全独立，
	// 二者不会混用同一字段，以避免概念混淆。
	ReleaseType string `json:"release_type,omitempty"`
}

// PlayerInfoHandler 定义播放器信息接口
type PlayerInfoHandler interface {
	GetTitle() string
	GetAlbum() string
	GetAlbumSubtitle() string
	GetAlbumTitleMetadata() *AlbumTitleMetadata
	GetArtist() string
	GetPosition() float64
	GetDuration() int64
	GetSampleRate() int64
	GetUrl() string // Audirvana 特有

	// 新增的方法以支持Track模型的更多字段
	GetAlbumArtist() string         // 专辑艺术家
	GetTrackNumber() int64          // 曲目编号
	GetGenre() string               // 流派
	GetComposer() string            // 作曲家
	GetReleaseDate() string         // 发布日期
	GetOriginalReleaseDate() string // 原始发布日期
	GetMusicBrainzID() string       // MusicBrainz ID
	GetSource() string              // 数据来源
	GetBundleID() string            // 应用标识符
	GetUniqueID() string            // 唯一标识符
	GetDiscNumber() int8            // 盘位
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
