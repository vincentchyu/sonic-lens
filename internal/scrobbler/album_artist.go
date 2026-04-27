package scrobbler

import "strings"

func preferredAlbumArtist(albumArtist, artist string) string {
	if strings.TrimSpace(albumArtist) != "" {
		return albumArtist
	}
	return artist
}
