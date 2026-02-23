package lyrics

import (
	"context"

	"github.com/vincentchyu/sonic-lens/core/musixmatch"
)

type MusixmatchProvider struct{}

func NewMusixmatchProvider() *MusixmatchProvider {
	return &MusixmatchProvider{}
}

func (p *MusixmatchProvider) GetName() string {
	return "Musixmatch"
}

func (p *MusixmatchProvider) GetLyrics(ctx context.Context, artist, album, track string) (string, error) {
	if !musixmatch.HasClient() {
		return "", nil // 或者返回一个特定的错误，表示客户端未初始化
	}
	return musixmatch.GetLyrics(ctx, artist, track)
}
