package artwork

import (
	"fmt"
	"strings"

	"github.com/vincentchyu/sonic-lens/common"
)

// BuildAlbumArtworkSeed 基于专辑三元组身份构建封面对象键种子，需与 scrobbler 逻辑保持一致。
// 当 albumSubtitle 为空时，返回 artist|album，与历史旧数据 100% 保持向后兼容；
// 当 albumSubtitle 非空时，返回 artist|album|subtitle，实现多版本封面物理隔离。
func BuildAlbumArtworkSeed(albumArtist, artist, album, albumSubtitle string) string {
	resolvedArtist := strings.TrimSpace(albumArtist)
	if resolvedArtist == "" {
		resolvedArtist = strings.TrimSpace(artist)
	}

	cleanArtist := strings.ToLower(common.ConversionSimplifiedFx(common.UnityFixAll(resolvedArtist)))
	cleanAlbum := strings.ToLower(common.ConversionSimplifiedFx(common.UnityFixAll(strings.TrimSpace(album))))
	cleanSubtitle := strings.ToLower(common.ConversionSimplifiedFx(common.UnityFixAll(strings.TrimSpace(albumSubtitle))))

	if cleanSubtitle == "" {
		return fmt.Sprintf("%s|%s", cleanArtist, cleanAlbum)
	}
	return fmt.Sprintf("%s|%s|%s", cleanArtist, cleanAlbum, cleanSubtitle)
}
