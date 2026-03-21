package scrobbler

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/common"
	corelog "github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/core/websocket"
	tracklogic "github.com/vincentchyu/sonic-lens/internal/logic/track"
)

type stubTrackService struct {
	tracklogic.TrackService

	probeCalls      int
	nowPlayingCalls int
	thresholdCalls  int
	lastProbeInput  tracklogic.PlaybackEventInput
	lastNowPlaying  tracklogic.PlaybackEventInput
	lastThreshold   tracklogic.PlaybackEventInput
	probeResult     tracklogic.TrackFavoriteProbeResult
	thresholdResult tracklogic.PlaybackThresholdResult
}

func (s *stubTrackService) ProbeAndSyncTrackFavorite(
	ctx context.Context,
	input tracklogic.PlaybackEventInput,
) tracklogic.TrackFavoriteProbeResult {
	s.probeCalls++
	s.lastProbeInput = input
	return s.probeResult
}

func (s *stubTrackService) HandleNowPlayingStarted(ctx context.Context, input tracklogic.PlaybackEventInput) {
	s.nowPlayingCalls++
	s.lastNowPlaying = input
}

func (s *stubTrackService) HandleTrackPlaybackThreshold(
	ctx context.Context,
	input tracklogic.PlaybackEventInput,
) tracklogic.PlaybackThresholdResult {
	s.thresholdCalls++
	s.lastThreshold = input
	return s.thresholdResult
}

type stubPlayerController struct {
	running  bool
	state    string
	favorite bool
}

func (s *stubPlayerController) IsRunning(ctx context.Context) bool           { return s.running }
func (s *stubPlayerController) IsFavorite(ctx context.Context) bool          { return s.favorite }
func (s *stubPlayerController) GetState(ctx context.Context) (string, error) { return s.state, nil }
func (s *stubPlayerController) GetNowPlayingTrackInfo(ctx context.Context) common.PlayerInfoHandler {
	return nil
}
func (s *stubPlayerController) SetFavorite(ctx context.Context) error { return nil }

type stubPlayerInfo struct {
	title       string
	album       string
	artist      string
	position    float64
	duration    int64
	trackNumber int64
	discNumber  int8
}

func (s *stubPlayerInfo) GetTitle() string         { return s.title }
func (s *stubPlayerInfo) GetAlbum() string         { return s.album }
func (s *stubPlayerInfo) GetArtist() string        { return s.artist }
func (s *stubPlayerInfo) GetPosition() float64     { return s.position }
func (s *stubPlayerInfo) GetDuration() int64       { return s.duration }
func (s *stubPlayerInfo) GetUrl() string           { return "" }
func (s *stubPlayerInfo) GetAlbumArtist() string   { return s.artist }
func (s *stubPlayerInfo) GetTrackNumber() int64    { return s.trackNumber }
func (s *stubPlayerInfo) GetGenre() string         { return "" }
func (s *stubPlayerInfo) GetComposer() string      { return "" }
func (s *stubPlayerInfo) GetReleaseDate() string   { return "" }
func (s *stubPlayerInfo) GetMusicBrainzID() string { return "mbid" }
func (s *stubPlayerInfo) GetSource() string        { return string(common.PlayerAppleMusic) }
func (s *stubPlayerInfo) GetBundleID() string      { return "bundle" }
func (s *stubPlayerInfo) GetUniqueID() string      { return "unique" }
func (s *stubPlayerInfo) GetDiscNumber() int8      { return s.discNumber }

func TestProcessPlayingTrackDispatchesEvents(t *testing.T) {
	corelog.Logger = zap.NewNop()

	pushCount := atomic.Uint32{}
	playing := atomic.Bool{}
	cache := &sync.Map{}
	service := &stubTrackService{
		probeResult: tracklogic.TrackFavoriteProbeResult{
			TrackFavoriteProjection: tracklogic.TrackFavoriteProjection{
				AppleMusic:      true,
				LastFM:          true,
				AppleMusicState: common.TrackFavoriteStateFavorited,
				LastFMState:     common.TrackFavoriteStateFavorited,
				FavoriteState:   common.TrackFavoriteStateFavorited,
			},
			Confidence: common.TrackMetadataConfidenceHigh,
		},
		thresholdResult: tracklogic.PlaybackThresholdResult{Scrobbled: true},
	}
	controller := &stubPlayerController{running: true, state: common.PlayerStatePlaying, favorite: true}
	checker := NewBasePlayerChecker(controller, common.PlayerAppleMusic, &pushCount, &playing, cache, service)

	info := &stubPlayerInfo{
		title:       "Track",
		album:       "Album",
		artist:      "Artist",
		position:    70,
		duration:    100,
		trackNumber: 3,
		discNumber:  1,
	}

	checker.processPlayingTrack(context.Background(), info)

	require.Equal(t, 1, service.probeCalls)
	require.Equal(t, 1, service.nowPlayingCalls)
	require.Equal(t, 1, service.thresholdCalls)
	require.True(t, playing.Load())
	require.Equal(t, uint32(1), pushCount.Load())
	require.Equal(t, "Track", checker.previousTrack)
	require.True(t, checker.scrobbledTracks["Track"])
	require.True(t, service.lastProbeInput.ControllerFavoriteKnown)
	require.True(t, service.lastProbeInput.ControllerFavorite)
	require.Equal(t, common.PlayerAppleMusic, service.lastThreshold.PlayerSource)

	cachedValue, ok := cache.Load(common.PlayerAppleMusic)
	require.True(t, ok)
	wsInfo := cachedValue.(*websocket.WsInfo)
	require.Equal(t, "Track", wsInfo.Data.Title)
	require.True(t, wsInfo.Data.AppleMusic)
	require.True(t, wsInfo.Data.LastFM)
	require.Equal(t, common.TrackFavoriteStateFavorited, wsInfo.Data.FavoriteState)
}

func TestProcessPlayingTrackReusesArtworkForSameTrack(t *testing.T) {
	corelog.Logger = zap.NewNop()

	originalResolveArtworkFn := resolveArtworkFn
	defer func() {
		resolveArtworkFn = originalResolveArtworkFn
	}()

	resolveCalls := 0
	resolveArtworkFn = func(
		_ *BasePlayerChecker,
		_ context.Context,
		_ common.PlayerInfoHandler,
	) (string, string, string) {
		resolveCalls++
		return "https://cdn.example.com/cover.jpg", "image/jpeg", "object-key-1"
	}

	pushCount := atomic.Uint32{}
	playing := atomic.Bool{}
	cache := &sync.Map{}
	service := &stubTrackService{
		probeResult: tracklogic.TrackFavoriteProbeResult{
			Confidence: common.TrackMetadataConfidenceHigh,
		},
	}
	controller := &stubPlayerController{running: true, state: common.PlayerStatePlaying}
	checker := NewBasePlayerChecker(controller, common.PlayerAppleMusic, &pushCount, &playing, cache, service)

	info := &stubPlayerInfo{
		title:       "Track",
		album:       "Album",
		artist:      "Artist",
		position:    10,
		duration:    100,
		trackNumber: 3,
		discNumber:  1,
	}

	checker.processPlayingTrack(context.Background(), info)
	checker.processPlayingTrack(context.Background(), info)

	require.Equal(t, 1, resolveCalls)
	require.Equal(t, "https://cdn.example.com/cover.jpg", checker.currentArtURL)
	require.Equal(t, "image/jpeg", checker.currentArtMime)
	require.Equal(t, "object-key-1", checker.currentArtObjectKey)
	require.Equal(t, "Track", checker.currentArtTrackKey)
}

func TestProcessPlayingTrackRetriesArtworkAfterFailure(t *testing.T) {
	corelog.Logger = zap.NewNop()

	originalResolveArtworkFn := resolveArtworkFn
	defer func() {
		resolveArtworkFn = originalResolveArtworkFn
	}()

	resolveCalls := 0
	resolveArtworkFn = func(
		_ *BasePlayerChecker,
		_ context.Context,
		_ common.PlayerInfoHandler,
	) (string, string, string) {
		resolveCalls++
		if resolveCalls == 1 {
			return "", "", ""
		}
		return "https://cdn.example.com/cover.jpg", "image/jpeg", "object-key-1"
	}

	pushCount := atomic.Uint32{}
	playing := atomic.Bool{}
	cache := &sync.Map{}
	service := &stubTrackService{
		probeResult: tracklogic.TrackFavoriteProbeResult{
			Confidence: common.TrackMetadataConfidenceHigh,
		},
	}
	controller := &stubPlayerController{running: true, state: common.PlayerStatePlaying}
	checker := NewBasePlayerChecker(controller, common.PlayerAppleMusic, &pushCount, &playing, cache, service)

	info := &stubPlayerInfo{
		title:       "Track",
		album:       "Album",
		artist:      "Artist",
		position:    10,
		duration:    100,
		trackNumber: 3,
		discNumber:  1,
	}

	checker.processPlayingTrack(context.Background(), info)
	checker.processPlayingTrack(context.Background(), info)

	require.Equal(t, 2, resolveCalls)
	require.Equal(t, "https://cdn.example.com/cover.jpg", checker.currentArtURL)
	require.Equal(t, "image/jpeg", checker.currentArtMime)
	require.Equal(t, "object-key-1", checker.currentArtObjectKey)
	require.True(t, checker.currentArtResolved)
}

func TestHandleStopEventClearsAtomicPlayingWhenNoOtherPlayer(t *testing.T) {
	corelog.Logger = zap.NewNop()

	pushCount := atomic.Uint32{}
	playing := atomic.Bool{}
	playing.Store(true)
	cache := &sync.Map{}
	service := &stubTrackService{}
	controller := &stubPlayerController{}
	checker := NewBasePlayerChecker(controller, common.PlayerAppleMusic, &pushCount, &playing, cache, service)

	checker.handleStopEvent(context.Background())

	require.False(t, playing.Load())
}
