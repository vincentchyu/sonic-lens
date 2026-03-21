package artwork

import (
	"fmt"
	"strings"

	"github.com/vincentchyu/sonic-lens/common"
)

// BuildAlbumArtworkSeed 基于专辑身份构建封面对象键种子，需与 scrobbler 逻辑保持一致。
func BuildAlbumArtworkSeed(albumArtist, artist, album string) string {
	resolvedArtist := strings.TrimSpace(albumArtist)
	if resolvedArtist == "" {
		resolvedArtist = strings.TrimSpace(artist)
	}

	return fmt.Sprintf(
		"%s|%s",
		strings.ToLower(common.UnityFixAll(resolvedArtist)),
		strings.ToLower(common.UnityFixAll(strings.TrimSpace(album))),
	)
}
