package artwork

import (
	"testing"
)

func TestBuildAlbumArtworkSeed(t *testing.T) {
	tests := []struct {
		name          string
		albumArtist   string
		artist        string
		album         string
		albumSubtitle string
		want          string
	}{
		{
			name:          "standard album without subtitle preserves legacy seed",
			albumArtist:   "Dire Straits",
			artist:        "Dire Straits",
			album:         "Brothers In Arms",
			albumSubtitle: "",
			want:          "dire straits|brothers in arms",
		},
		{
			name:          "album with 1996 remastered subtitle generates isolated seed",
			albumArtist:   "Dire Straits",
			artist:        "Dire Straits",
			album:         "Brothers In Arms",
			albumSubtitle: "Remastered 1996",
			want:          "dire straits|brothers in arms|remastered 1996",
		},
		{
			name:          "album with 40th anniversary subtitle generates isolated seed",
			albumArtist:   "Dire Straits",
			artist:        "Dire Straits",
			album:         "Brothers In Arms",
			albumSubtitle: "40th Anniversary",
			want:          "dire straits|brothers in arms|40th anniversary",
		},
		{
			name:          "fallback to artist when albumArtist is empty",
			albumArtist:   "",
			artist:        "Pink Floyd",
			album:         "The Wall",
			albumSubtitle: "Deluxe Edition",
			want:          "pink floyd|the wall|deluxe edition",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got := BuildAlbumArtworkSeed(tt.albumArtist, tt.artist, tt.album, tt.albumSubtitle)
				if got != tt.want {
					t.Errorf("BuildAlbumArtworkSeed() = %q, want %q", got, tt.want)
				}
			},
		)
	}
}
