package track

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

func TestBuildFavoriteProjectionPendingFavoriteWithoutStableTrack(t *testing.T) {
	originalGetTrack := modelGetTrackByIdentity
	originalGetPending := modelGetPendingTrackFavoriteSnapshot
	t.Cleanup(func() {
		modelGetTrackByIdentity = originalGetTrack
		modelGetPendingTrackFavoriteSnapshot = originalGetPending
	})

	modelGetTrackByIdentity = func(ctx context.Context, artist, album, track string, trackNumber, discNumber int8) (*model.Track, error) {
		return nil, errors.New("低信任场景不应读取稳定曲目")
	}
	modelGetPendingTrackFavoriteSnapshot = func(ctx context.Context, identity model.TrackIdentity) (*model.TrackFavoritePendingSnapshot, error) {
		require.Equal(t, "Artist", identity.Artist)
		require.Equal(t, "Album", identity.Album)
		require.Equal(t, "Track", identity.Track)
		return &model.TrackFavoritePendingSnapshot{
			AppleMusicKnown:    true,
			AppleMusicFavorite: true,
		}, nil
	}

	service := &TrackServiceImpl{}
	projection, err := service.buildFavoriteProjection(context.Background(), FavoriteProjectionInput{
		Artist:   "Artist",
		Album:    "Album",
		Track:    "Track",
		Metadata: model.TrackMetadata{Confidence: common.TrackMetadataConfidenceLow},
	})

	require.NoError(t, err)
	require.True(t, projection.AppleMusic)
	require.False(t, projection.LastFM)
	require.Equal(t, common.TrackFavoriteStateFavoritePending, projection.AppleMusicState)
	require.Equal(t, common.TrackFavoriteStateNotFavorited, projection.LastFMState)
	require.Equal(t, common.TrackFavoriteStateFavoritePending, projection.FavoriteState)
}

func TestBuildFavoriteProjectionPendingUnfavoriteOverridesStableFact(t *testing.T) {
	originalGetTrack := modelGetTrackByIdentity
	originalGetPending := modelGetPendingTrackFavoriteSnapshot
	t.Cleanup(func() {
		modelGetTrackByIdentity = originalGetTrack
		modelGetPendingTrackFavoriteSnapshot = originalGetPending
	})

	modelGetTrackByIdentity = func(ctx context.Context, artist, album, track string, trackNumber, discNumber int8) (*model.Track, error) {
		return &model.Track{
			Artist:          artist,
			Album:           album,
			Track:           track,
			TrackNumber:     trackNumber,
			DiscNumber:      discNumber,
			IsAppleMusicFav: true,
		}, nil
	}
	modelGetPendingTrackFavoriteSnapshot = func(ctx context.Context, identity model.TrackIdentity) (*model.TrackFavoritePendingSnapshot, error) {
		return &model.TrackFavoritePendingSnapshot{
			AppleMusicKnown:    true,
			AppleMusicFavorite: false,
		}, nil
	}

	service := &TrackServiceImpl{}
	projection, err := service.buildFavoriteProjection(context.Background(), FavoriteProjectionInput{
		Artist:      "Artist",
		Album:       "Album",
		Track:       "Track",
		TrackNumber: 3,
		DiscNumber:  1,
		Metadata: model.TrackMetadata{
			Confidence:  common.TrackMetadataConfidenceHigh,
			TrackNumber: 3,
			DiscNumber:  1,
		},
	})

	require.NoError(t, err)
	require.False(t, projection.AppleMusic)
	require.Equal(t, common.TrackFavoriteStateUnfavoritePending, projection.AppleMusicState)
	require.Equal(t, common.TrackFavoriteStateUnfavoritePending, projection.FavoriteState)
}
