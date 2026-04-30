//go:build integration
// +build integration

package lyrics

import (
	"context"
	"testing"
)

func TestLrcAPIProvider_GetLyrics(t *testing.T) {
	provider := NewLrcAPIProvider("http://127.0.0.1:28883/api/v1/lyrics/single", "todo")
	ctx := context.Background()

	// 使用一首著名的歌曲进行测试
	artist := "Aaron Neville"
	track := "Please Come Home for Christmas"
	album := "Aaron Neville’s Soulful Christmas"

	lyrics, err := provider.GetLyrics(ctx, artist, album, track)
	if err != nil {
		t.Skipf("LrcAPI request failed (might be network issue): %v", err)
		return
	}

	if lyrics == "" {
		t.Errorf("Expected lyrics, got empty string")
	}
	t.Logf("Fetched lyrics:\n %s", lyrics[:500]+"...")

	artist = "腰"
	track = "硬汉"
	album = "相见恨晚"

	lyrics, err = provider.GetLyrics(ctx, artist, album, track)
	if err != nil {
		t.Skipf("LrcAPI request failed (might be network issue): %v", err)
		return
	}

	if lyrics == "" {
		t.Errorf("Expected lyrics, got empty string")
	}
	t.Logf("Fetched lyrics:\n %s", lyrics[:500]+"...")
}

/*func TestMusixmatchProvider_GetLyrics(t *testing.T) {
	provider := NewMusixmatchProvider()
	ctx := context.Background()

	// 注意：如果没有配置 API Key，这个测试应该跳过或者返回错误
	lyrics, err := provider.GetLyrics(ctx, "Radiohead", "OK Computer", "Karma Police")
	if err != nil {
		t.Logf("Musixmatch provider failed (expected if not configured): %v", err)
	} else {
		t.Logf("Musixmatch lyrics: %s", lyrics)
	}
}*/
