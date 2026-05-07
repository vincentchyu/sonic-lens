package scrobbler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/exec"
)

func TestFoobar2000PlayerControllerUsesBundleIdentifier(t *testing.T) {
	originalGetNowPlaying := getMediaControlNowPlayingFn
	t.Cleanup(func() {
		getMediaControlNowPlayingFn = originalGetNowPlaying
	})

	getMediaControlNowPlayingFn = func(ctx context.Context) (*exec.MediaControlNowPlayingInfo, error) {
		return &exec.MediaControlNowPlayingInfo{
			BundleIdentifier:      exec.MRMediaNowPlayingAppFoobar2000,
			Playing:               true,
			Title:                 "Black Dream",
			Artist:                "Dou Wei",
			Album:                 "黑梦",
			ContentItemIdentifier: "content-id",
		}, nil
	}

	controller := NewFoobar2000PlayerController()
	require.True(t, controller.IsRunning(context.Background()))

	state, err := controller.GetState(context.Background())
	require.NoError(t, err)
	require.Equal(t, common.PlayerStatePlaying, state)

	info := controller.GetNowPlayingTrackInfo(context.Background())
	require.NotNil(t, info)
	require.Equal(t, "Foobar2000", info.GetSource())
	require.Equal(t, exec.MRMediaNowPlayingAppFoobar2000, info.GetBundleID())
	require.Equal(t, "content-id", info.GetUniqueID())
}

func TestNetEasePlayerControllerUsesBundleIdentifier(t *testing.T) {
	originalGetNowPlaying := getMediaControlNowPlayingFn
	t.Cleanup(func() {
		getMediaControlNowPlayingFn = originalGetNowPlaying
	})

	getMediaControlNowPlayingFn = func(ctx context.Context) (*exec.MediaControlNowPlayingInfo, error) {
		return &exec.MediaControlNowPlayingInfo{
			BundleIdentifier:      exec.MRMediaNowPlaying163,
			Playing:               true,
			Title:                 "铸铁旅人",
			Artist:                "虎啸春",
			Album:                 "铸铁旅人",
			ContentItemIdentifier: "content-id-163",
		}, nil
	}

	controller := NewNetEasePlayerController()
	require.True(t, controller.IsRunning(context.Background()))

	state, err := controller.GetState(context.Background())
	require.NoError(t, err)
	require.Equal(t, common.PlayerStatePlaying, state)

	info := controller.GetNowPlayingTrackInfo(context.Background())
	require.NotNil(t, info)
	require.Equal(t, "NetEase Music", info.GetSource())
	require.Equal(t, exec.MRMediaNowPlaying163, info.GetBundleID())
	require.Equal(t, "content-id-163", info.GetUniqueID())
}
