package lyrics

import (
	"context"
	"testing"
)

func TestLrcAPIProvider_GetLyrics(t *testing.T) {
	provider := NewLrcAPIProvider()
	ctx := context.Background()

	// 使用一首著名的歌曲进行测试
	artist := "Radiohead"
	track := "Karma Police"
	album := "OK Computer"

	lyrics, err := provider.GetLyrics(ctx, artist, album, track)
	if err != nil {
		t.Skipf("LrcAPI request failed (might be network issue): %v", err)
		return
	}

	if lyrics == "" {
		t.Errorf("Expected lyrics, got empty string")
	}
	t.Logf("Fetched lyrics: %s", lyrics[:50]+"...")
}

func TestMusixmatchProvider_GetLyrics(t *testing.T) {
	provider := NewMusixmatchProvider()
	ctx := context.Background()

	// 注意：如果没有配置 API Key，这个测试应该跳过或者返回错误
	lyrics, err := provider.GetLyrics(ctx, "Radiohead", "OK Computer", "Karma Police")
	if err != nil {
		t.Logf("Musixmatch provider failed (expected if not configured): %v", err)
	} else {
		t.Logf("Musixmatch lyrics: %s", lyrics)
	}
}
