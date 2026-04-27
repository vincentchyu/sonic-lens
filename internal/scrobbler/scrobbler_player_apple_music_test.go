package scrobbler

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vincentchyu/sonic-lens/core/applemusic"
	"github.com/vincentchyu/sonic-lens/core/audirvana"
	"github.com/vincentchyu/sonic-lens/core/exec"
)

func TestAppleMusicTrackInfoWrapperGetAlbumArtistPrefersAlbumArtist(t *testing.T) {
	wrapper := &AppleMusicTrackInfoWrapper{
		TrackInfo: &applemusic.TrackInfo{
			TrackBase: applemusic.TrackBase{
				Artist:      "Track Artist",
				AlbumArtist: "Album Artist",
			},
		},
	}

	require.Equal(t, "Album Artist", wrapper.GetAlbumArtist())
}

func TestAppleMusicTrackInfoWrapperGetAlbumArtistFallsBackToArtist(t *testing.T) {
	wrapper := &AppleMusicTrackInfoWrapper{
		TrackInfo: &applemusic.TrackInfo{
			TrackBase: applemusic.TrackBase{
				Artist:      "Track Artist",
				AlbumArtist: "",
			},
		},
	}

	require.Equal(t, "Track Artist", wrapper.GetAlbumArtist())
}

func TestAudirvanaTrackInfoWrapperGetAlbumArtistPrefersMetadataAlbumArtist(t *testing.T) {
	wrapper := &AudirvanaTrackInfoWrapper{
		TrackInfo: &audirvana.TrackInfo{
			TrackBase: audirvana.TrackBase{
				Artist: "Track Artist",
			},
		},
	}
	wrapper.MataDataHandle = exec.ExiftoolInfo{
		"Artist":      "Track Artist",
		"AlbumArtist": "Album Artist",
	}

	require.Equal(t, "Album Artist", wrapper.GetAlbumArtist())
}

func TestAudirvanaTrackInfoWrapperGetAlbumArtistFallsBackToTrackArtist(t *testing.T) {
	wrapper := &AudirvanaTrackInfoWrapper{
		TrackInfo: &audirvana.TrackInfo{
			TrackBase: audirvana.TrackBase{
				Artist: "Track Artist",
			},
		},
	}
	wrapper.MataDataHandle = exec.ExiftoolInfo{
		"Artist": "Track Artist",
	}

	require.Equal(t, "Track Artist", wrapper.GetAlbumArtist())
}
