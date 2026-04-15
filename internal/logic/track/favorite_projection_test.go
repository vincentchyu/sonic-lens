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
	originalGetTrackWithSubtitle := modelGetTrackByIdentityWithSubtitle
	originalGetPending := modelGetPendingTrackFavoriteSnapshot
	originalResolveTrack := modelResolveTrackForFavoriteProjection
	t.Cleanup(func() {
		modelGetTrackByIdentity = originalGetTrack
		modelGetTrackByIdentityWithSubtitle = originalGetTrackWithSubtitle
		modelGetPendingTrackFavoriteSnapshot = originalGetPending
		modelResolveTrackForFavoriteProjection = originalResolveTrack
	})

	modelGetTrackByIdentity = func(ctx context.Context, artist, album, track string, trackNumber, discNumber int8) (*model.Track, error) {
		return nil, errors.New("低信任场景不应读取稳定曲目")
	}
	modelGetTrackByIdentityWithSubtitle = func(
		ctx context.Context, artist, album, albumSubtitle, track string, trackNumber, discNumber int8,
	) (*model.Track, error) {
		return nil, errors.New("低信任场景不应读取稳定曲目")
	}
	modelResolveTrackForFavoriteProjection = func(
		ctx context.Context, artist, album, track string, metadata model.TrackMetadata,
	) (*model.Track, model.TrackIdentity, error) {
		return nil, model.TrackIdentity{}, errors.New("无弱线索时不应触发弱匹配")
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

func TestBuildFavoriteProjectionLowConfidenceUsesWeakResolvedStableTrack(t *testing.T) {
	originalGetTrack := modelGetTrackByIdentity
	originalGetTrackWithSubtitle := modelGetTrackByIdentityWithSubtitle
	originalGetPending := modelGetPendingTrackFavoriteSnapshot
	originalResolveTrack := modelResolveTrackForFavoriteProjection
	t.Cleanup(func() {
		modelGetTrackByIdentity = originalGetTrack
		modelGetTrackByIdentityWithSubtitle = originalGetTrackWithSubtitle
		modelGetPendingTrackFavoriteSnapshot = originalGetPending
		modelResolveTrackForFavoriteProjection = originalResolveTrack
	})

	modelGetTrackByIdentity = func(ctx context.Context, artist, album, track string, trackNumber, discNumber int8) (*model.Track, error) {
		return nil, errors.New("低信任场景应走弱匹配，不应直接按稳定身份读取")
	}
	modelGetTrackByIdentityWithSubtitle = func(
		ctx context.Context, artist, album, albumSubtitle, track string, trackNumber, discNumber int8,
	) (*model.Track, error) {
		return nil, errors.New("低信任场景应走弱匹配，不应直接按稳定身份读取")
	}
	modelResolveTrackForFavoriteProjection = func(
		ctx context.Context, artist, album, track string, metadata model.TrackMetadata,
	) (*model.Track, model.TrackIdentity, error) {
		require.Equal(t, int64(311), metadata.Duration)
		return &model.Track{
				Artist:          artist,
				Album:           album,
				Track:           track,
				TrackNumber:     0,
				DiscNumber:      1,
				IsAppleMusicFav: true,
				IsLastFmFav:     true,
			},
			model.TrackIdentity{
				Artist:      artist,
				Album:       album,
				Track:       track,
				TrackNumber: 0,
				DiscNumber:  1,
			},
			nil
	}
	modelGetPendingTrackFavoriteSnapshot = func(ctx context.Context, identity model.TrackIdentity) (*model.TrackFavoritePendingSnapshot, error) {
		require.Equal(t, int8(1), identity.DiscNumber)
		return &model.TrackFavoritePendingSnapshot{}, nil
	}

	service := &TrackServiceImpl{}
	projection, err := service.buildFavoriteProjection(context.Background(), FavoriteProjectionInput{
		Artist: "Artist",
		Album:  "Album",
		Track:  "Track",
		Metadata: model.TrackMetadata{
			Confidence: common.TrackMetadataConfidenceLow,
			Duration:   311,
		},
	})

	require.NoError(t, err)
	require.True(t, projection.AppleMusic)
	require.True(t, projection.LastFM)
	require.Equal(t, common.TrackFavoriteStateFavorited, projection.AppleMusicState)
	require.Equal(t, common.TrackFavoriteStateFavorited, projection.LastFMState)
	require.Equal(t, common.TrackFavoriteStateFavorited, projection.FavoriteState)
}

func TestBuildFavoriteProjectionPendingUnfavoriteOverridesStableFact(t *testing.T) {
	originalGetTrack := modelGetTrackByIdentity
	originalGetTrackWithSubtitle := modelGetTrackByIdentityWithSubtitle
	originalGetPending := modelGetPendingTrackFavoriteSnapshot
	originalResolveTrack := modelResolveTrackForFavoriteProjection
	t.Cleanup(func() {
		modelGetTrackByIdentity = originalGetTrack
		modelGetTrackByIdentityWithSubtitle = originalGetTrackWithSubtitle
		modelGetPendingTrackFavoriteSnapshot = originalGetPending
		modelResolveTrackForFavoriteProjection = originalResolveTrack
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
	modelGetTrackByIdentityWithSubtitle = func(
		ctx context.Context, artist, album, albumSubtitle, track string, trackNumber, discNumber int8,
	) (*model.Track, error) {
		return &model.Track{
			Artist:          artist,
			Album:           album,
			AlbumSubtitle:   albumSubtitle,
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
