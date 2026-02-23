package lyrics

import (
	"context"
)

// LyricsProvider 定义歌词提供者的通用接口
type LyricsProvider interface {
	// GetName 返回提供者的名称
	GetName() string
	// GetLyrics 获取指定艺术家与曲目的歌词文本
	GetLyrics(ctx context.Context, artist, album, track string) (string, error)
}
