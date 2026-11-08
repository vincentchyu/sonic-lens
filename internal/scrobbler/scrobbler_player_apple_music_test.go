package scrobbler

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vincentchyu/sonic-lens/common"
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

func TestAudirvanaTrackInfoWrapperGetBundleID(t *testing.T) {
	t.Run("Audirvana Studio 默认 BundleID", func(t *testing.T) {
		wrapper := &AudirvanaTrackInfoWrapper{
			TrackInfo: &audirvana.TrackInfo{
				TrackBase: audirvana.TrackBase{
					AppName: common.AppAudirvanaStudio,
				},
			},
		}
		require.Equal(t, common.BundleIDAudirvanaStudio, wrapper.GetBundleID())
	})

	t.Run("Audirvana Origin 默认 BundleID", func(t *testing.T) {
		wrapper := &AudirvanaTrackInfoWrapper{
			TrackInfo: &audirvana.TrackInfo{
				TrackBase: audirvana.TrackBase{
					AppName: common.AppAudirvanaOrigin,
				},
			},
		}
		require.Equal(t, common.BundleIDAudirvanaOrigin, wrapper.GetBundleID())
	})

	t.Run("元数据自带 BundleID 优先", func(t *testing.T) {
		wrapper := &AudirvanaTrackInfoWrapper{
			TrackInfo: &audirvana.TrackInfo{
				TrackBase: audirvana.TrackBase{
					AppName: common.AppAudirvanaStudio,
				},
			},
		}
		wrapper.MataDataHandle = exec.ExiftoolInfo{
			"BundleID": "com.custom.bundle",
		}
		require.Equal(t, "com.custom.bundle", wrapper.GetBundleID())
	})
}
