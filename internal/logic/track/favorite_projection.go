package track

import (
	"context"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

// FavoriteProjectionInput 描述收藏态投影视图所需的身份与元数据。
type FavoriteProjectionInput struct {
	Artist        string
	Album         string
	AlbumSubtitle string
	Track         string
	TrackNumber   int8
	DiscNumber    int8
	Metadata      model.TrackMetadata
}

// TrackFavoriteProjection 表示给 API/WS 统一消费的收藏态只读投影。
type TrackFavoriteProjection struct {
	AppleMusic      bool                      `json:"apple_music"`
	LastFM          bool                      `json:"lastfm"`
	AppleMusicState common.TrackFavoriteState `json:"apple_music_state"`
	LastFMState     common.TrackFavoriteState `json:"lastfm_state"`
	FavoriteState   common.TrackFavoriteState `json:"favorite_state"`
}

func (s *TrackServiceImpl) buildFavoriteProjection(
	ctx context.Context,
	input FavoriteProjectionInput,
) (TrackFavoriteProjection, error) {
	projection := TrackFavoriteProjection{
		AppleMusicState: common.TrackFavoriteStateNotFavorited,
		LastFMState:     common.TrackFavoriteStateNotFavorited,
		FavoriteState:   common.TrackFavoriteStateNotFavorited,
	}

	metadata := input.Metadata
	if metadata.TrackNumber == 0 {
		metadata.TrackNumber = input.TrackNumber
	}
	if metadata.DiscNumber == 0 {
		metadata.DiscNumber = input.DiscNumber
	}

	identity := model.TrackIdentity{
		Artist:        input.Artist,
		Album:         input.Album,
		AlbumSubtitle: input.AlbumSubtitle,
		Track:         input.Track,
		TrackNumber:   input.TrackNumber,
		DiscNumber:    input.DiscNumber,
	}

	trackAppleMusic := false
	trackLastFM := false
	if s.canSafelyLookupCurrentTrack(metadata) {
		currentTrack, err := modelGetTrackByIdentityWithSubtitle(
			ctx, input.Artist, input.Album, input.AlbumSubtitle, input.Track, input.TrackNumber, input.DiscNumber,
		)
		if err != nil {
			log.Debug(
				ctx,
				"读取稳定收藏态失败",
				zap.Error(err),
				zap.String("artist", input.Artist),
				zap.String("album", input.Album),
				zap.String("track", input.Track),
			)
		} else if currentTrack != nil {
			trackAppleMusic = currentTrack.IsAppleMusicFav
			trackLastFM = currentTrack.IsLastFmFav
		}
	} else if canWeaklyResolveFavoriteProjectionTrack(metadata) {
		currentTrack, resolvedIdentity, err := modelResolveTrackForFavoriteProjection(
			ctx,
			input.Artist,
			input.Album,
			input.Track,
			metadata,
		)
		if err != nil {
			log.Debug(
				ctx,
				"低信任收藏态弱匹配失败",
				zap.Error(err),
				zap.String("artist", input.Artist),
				zap.String("album", input.Album),
				zap.String("track", input.Track),
			)
		} else if currentTrack != nil {
			identity = resolvedIdentity
			trackAppleMusic = currentTrack.IsAppleMusicFav
			trackLastFM = currentTrack.IsLastFmFav
		}
	}

	pendingSnapshot, err := modelGetPendingTrackFavoriteSnapshot(ctx, identity)
	if err != nil {
		return projection, err
	}
	if pendingSnapshot == nil {
		pendingSnapshot = &model.TrackFavoritePendingSnapshot{}
	}

	projection.AppleMusicState = buildSourceFavoriteState(
		trackAppleMusic,
		pendingSnapshot.AppleMusicKnown,
		pendingSnapshot.AppleMusicFavorite,
	)
	projection.LastFMState = buildSourceFavoriteState(
		trackLastFM,
		pendingSnapshot.LastFmKnown,
		pendingSnapshot.LastFmFavorite,
	)
	projection.AppleMusic = projection.AppleMusicState.IsFavoritedEffective()
	projection.LastFM = projection.LastFMState.IsFavoritedEffective()
	projection.FavoriteState = aggregateFavoriteState(projection.AppleMusicState, projection.LastFMState)

	return projection, nil
}

func canWeaklyResolveFavoriteProjectionTrack(metadata model.TrackMetadata) bool {
	return metadata.Duration > 0 || metadata.UniqueID != "" || metadata.MusicBrainzID != ""
}

func buildSourceFavoriteState(
	stableFavorite bool,
	pendingKnown bool,
	pendingFavorite bool,
) common.TrackFavoriteState {
	if pendingKnown && pendingFavorite != stableFavorite {
		if pendingFavorite {
			return common.TrackFavoriteStateFavoritePending
		}
		return common.TrackFavoriteStateUnfavoritePending
	}
	if stableFavorite {
		return common.TrackFavoriteStateFavorited
	}
	return common.TrackFavoriteStateNotFavorited
}

func aggregateFavoriteState(states ...common.TrackFavoriteState) common.TrackFavoriteState {
	hasFavorited := false
	for _, state := range states {
		switch state {
		case common.TrackFavoriteStateUnfavoritePending:
			return common.TrackFavoriteStateUnfavoritePending
		case common.TrackFavoriteStateFavoritePending:
			return common.TrackFavoriteStateFavoritePending
		case common.TrackFavoriteStateFavorited:
			hasFavorited = true
		}
	}
	if hasFavorited {
		return common.TrackFavoriteStateFavorited
	}
	return common.TrackFavoriteStateNotFavorited
}
