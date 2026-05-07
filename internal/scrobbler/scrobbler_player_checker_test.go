package scrobbler

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/audirvana"
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
	running       bool
	state         string
	favorite      bool
	favoriteCalls int
}

func (s *stubPlayerController) IsRunning(ctx context.Context) bool { return s.running }
func (s *stubPlayerController) IsFavorite(ctx context.Context) bool {
	s.favoriteCalls++
	return s.favorite
}
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
	sampleRate  int64
	trackNumber int64
	discNumber  int8
}

func (s *stubPlayerInfo) GetTitle() string                                  { return s.title }
func (s *stubPlayerInfo) GetAlbum() string                                  { return s.album }
func (s *stubPlayerInfo) GetAlbumSubtitle() string                          { return "" }
func (s *stubPlayerInfo) GetAlbumTitleMetadata() *common.AlbumTitleMetadata { return nil }
func (s *stubPlayerInfo) GetArtist() string                                 { return s.artist }
func (s *stubPlayerInfo) GetPosition() float64                              { return s.position }
func (s *stubPlayerInfo) GetDuration() int64                                { return s.duration }
func (s *stubPlayerInfo) GetSampleRate() int64                              { return s.sampleRate }
func (s *stubPlayerInfo) GetUrl() string                                    { return "" }
func (s *stubPlayerInfo) GetAlbumArtist() string                            { return s.artist }
func (s *stubPlayerInfo) GetTrackNumber() int64                             { return s.trackNumber }
func (s *stubPlayerInfo) GetGenre() string                                  { return "" }
func (s *stubPlayerInfo) GetComposer() string                               { return "" }
func (s *stubPlayerInfo) GetReleaseDate() string                            { return "" }
func (s *stubPlayerInfo) GetOriginalReleaseDate() string                    { return "" }
func (s *stubPlayerInfo) GetMusicBrainzID() string                          { return "mbid" }
func (s *stubPlayerInfo) GetSource() string                                 { return string(common.PlayerAppleMusic) }
func (s *stubPlayerInfo) GetBundleID() string                               { return "bundle" }
func (s *stubPlayerInfo) GetUniqueID() string                               { return "unique" }
func (s *stubPlayerInfo) GetDiscNumber() int8                               { return s.discNumber }

type foobarStubPlayerInfo struct {
	stubPlayerInfo
}

func (s *foobarStubPlayerInfo) GetSource() string   { return string(common.PlayerFoobar2000) }
func (s *foobarStubPlayerInfo) GetBundleID() string { return "com.foobar2000.mac" }

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
		sampleRate:  44100,
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
	require.Equal(t, int64(44100), wsInfo.Data.SampleRate)
}

func TestProcessPlayingTrackUsesFoobar2000Source(t *testing.T) {
	corelog.Logger = zap.NewNop()

	pushCount := atomic.Uint32{}
	playing := atomic.Bool{}
	cache := &sync.Map{}
	service := &stubTrackService{
		thresholdResult: tracklogic.PlaybackThresholdResult{Scrobbled: false},
	}
	controller := &stubPlayerController{running: true, state: common.PlayerStatePlaying}
	checker := NewBasePlayerChecker(controller, common.PlayerFoobar2000, &pushCount, &playing, cache, service)

	info := &foobarStubPlayerInfo{
		stubPlayerInfo: stubPlayerInfo{
			title:       "Track",
			album:       "Album",
			artist:      "Artist",
			position:    70,
			duration:    100,
			sampleRate:  44100,
			trackNumber: 3,
			discNumber:  1,
		},
	}

	checker.processPlayingTrack(context.Background(), info)

	require.Equal(t, common.PlayerFoobar2000, service.lastNowPlaying.PlayerSource)
	require.Equal(t, common.PlayerFoobar2000, service.lastThreshold.PlayerSource)

	cachedValue, ok := cache.Load(common.PlayerFoobar2000)
	require.True(t, ok)
	wsInfo := cachedValue.(*websocket.WsInfo)
	require.Equal(t, "Foobar2000", wsInfo.Source)
}

func TestHandleStopEventSkipsBroadcastWhenFoobar2000StillPlaying(t *testing.T) {
	corelog.Logger = zap.NewNop()

	pushCount := atomic.Uint32{}
	playing := atomic.Bool{}
	cache := &sync.Map{}
	service := &stubTrackService{}
	controller := &stubPlayerController{}
	checker := NewBasePlayerChecker(controller, common.PlayerAppleMusic, &pushCount, &playing, cache, service)
	checker.currentTrackSpanKey = "Track"
	checker.currentTrack = "Track"
	cache.Store(common.PlayerAppleMusic, &websocket.WsInfo{Source: string(common.PlayerAppleMusic)})
	cache.Store(common.PlayerFoobar2000, &websocket.WsInfo{Source: string(common.PlayerFoobar2000)})
	playing.Store(true)

	checker.handleStopEvent(context.Background())

	require.True(t, playing.Load())
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
		sampleRate:  44100,
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

func TestProcessPlayingTrackCachesControllerFavoriteWithinTTL(t *testing.T) {
	corelog.Logger = zap.NewNop()

	pushCount := atomic.Uint32{}
	playing := atomic.Bool{}
	cache := &sync.Map{}
	service := &stubTrackService{}
	controller := &stubPlayerController{running: true, state: common.PlayerStatePlaying, favorite: true}
	checker := NewBasePlayerChecker(controller, common.PlayerAppleMusic, &pushCount, &playing, cache, service)

	info := &stubPlayerInfo{
		title:       "Track",
		album:       "Album",
		artist:      "Artist",
		position:    10,
		duration:    100,
		trackNumber: 1,
		discNumber:  1,
	}

	checker.processPlayingTrack(context.Background(), info)
	checker.processPlayingTrack(context.Background(), info)
	require.Equal(t, 1, controller.favoriteCalls)

	checker.controllerFavoriteCheckedAt = time.Now().Add(-controllerFavoriteRefreshInterval - time.Second)
	checker.processPlayingTrack(context.Background(), info)
	require.Equal(t, 2, controller.favoriteCalls)
}

func TestProcessPlayingTrackPassesTrackTraceContextToThresholdInput(t *testing.T) {
	corelog.Logger = zap.NewNop()

	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(previous)

	pushCount := atomic.Uint32{}
	playing := atomic.Bool{}
	cache := &sync.Map{}
	service := &stubTrackService{
		thresholdResult: tracklogic.PlaybackThresholdResult{Scrobbled: true},
	}
	controller := &stubPlayerController{running: true, state: common.PlayerStatePlaying, favorite: true}
	checker := NewBasePlayerChecker(controller, common.PlayerAppleMusic, &pushCount, &playing, cache, service)

	info := &stubPlayerInfo{
		title:       "Track Trace",
		album:       "Album",
		artist:      "Artist",
		position:    70,
		duration:    100,
		trackNumber: 1,
		discNumber:  1,
	}

	checker.processPlayingTrack(context.Background(), info)

	require.NotEmpty(t, service.lastThreshold.TraceID)
	require.NotEmpty(t, service.lastThreshold.RootSpanID)
	require.Equal(t, checker.currentTrackSpan.SpanContext().TraceID().String(), service.lastThreshold.TraceID)
	require.Equal(t, checker.currentTrackSpan.SpanContext().SpanID().String(), service.lastThreshold.RootSpanID)
	require.Equal(t, checker.currentTrackSpan.SpanContext().IsSampled(), service.lastThreshold.TraceSampled)
}

func TestProcessPlayingTrackTreatsLoopedTrackAsNewPlaybackSession(t *testing.T) {
	corelog.Logger = zap.NewNop()

	pushCount := atomic.Uint32{}
	playing := atomic.Bool{}
	cache := &sync.Map{}
	service := &stubTrackService{
		thresholdResult: tracklogic.PlaybackThresholdResult{Scrobbled: true},
	}
	controller := &stubPlayerController{running: true, state: common.PlayerStatePlaying, favorite: true}
	checker := NewBasePlayerChecker(controller, common.PlayerAppleMusic, &pushCount, &playing, cache, service)

	nearEnd := &stubPlayerInfo{
		title:       "Loop Track",
		album:       "Album",
		artist:      "Artist",
		position:    70,
		duration:    100,
		trackNumber: 1,
		discNumber:  1,
	}
	restarted := &stubPlayerInfo{
		title:       "Loop Track",
		album:       "Album",
		artist:      "Artist",
		position:    2,
		duration:    100,
		trackNumber: 1,
		discNumber:  1,
	}

	checker.processPlayingTrack(context.Background(), nearEnd)
	require.Equal(t, 1, service.nowPlayingCalls)
	require.Equal(t, 1, service.thresholdCalls)
	require.Equal(t, uint32(1), pushCount.Load())
	require.True(t, checker.scrobbledTracks["Loop Track"])

	checker.processPlayingTrack(context.Background(), restarted)
	require.Equal(t, 2, service.nowPlayingCalls)
	require.Equal(t, 1, service.thresholdCalls)
	require.False(t, checker.scrobbledTracks["Loop Track"])

	checker.processPlayingTrack(context.Background(), nearEnd)
	require.Equal(t, 2, service.thresholdCalls)
	require.Equal(t, uint32(2), pushCount.Load())
	require.True(t, checker.scrobbledTracks["Loop Track"])
}

func TestProcessPlayingTrackAudirvanaWithoutMetadataHandleDoesNotPanic(t *testing.T) {
	corelog.Logger = zap.NewNop()

	pushCount := atomic.Uint32{}
	playing := atomic.Bool{}
	cache := &sync.Map{}
	service := &stubTrackService{
		thresholdResult: tracklogic.PlaybackThresholdResult{Scrobbled: true},
	}
	controller := &stubPlayerController{running: true, state: common.PlayerStatePlaying}
	checker := NewBasePlayerChecker(controller, common.PlayerAudirvana, &pushCount, &playing, cache, service)

	info := &AudirvanaTrackInfoWrapper{
		TrackInfo: &audirvana.TrackInfo{
			TrackBase: audirvana.TrackBase{
				Title:    "Audirvana Track",
				Album:    "Audirvana Album",
				Artist:   "Audirvana Artist",
				Duration: 100,
				Position: 70,
				Url:      "file:///music/audirvana-track.flac",
			},
		},
		baseWrapper: BaseWrapper{},
	}

	require.NotPanics(t, func() {
		checker.processPlayingTrack(context.Background(), info)
	})
	require.Equal(t, 1, service.nowPlayingCalls)
	require.Equal(t, 1, service.thresholdCalls)
	require.Equal(t, common.PlayerAudirvana, service.lastThreshold.PlayerSource)
	require.Equal(t, uint32(1), pushCount.Load())
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

func TestProcessPlayingTrackAddsThresholdSpanAndEvent(t *testing.T) {
	corelog.Logger = zap.NewNop()

	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(previous)

	pushCount := atomic.Uint32{}
	playing := atomic.Bool{}
	cache := &sync.Map{}
	service := &stubTrackService{
		thresholdResult: tracklogic.PlaybackThresholdResult{Scrobbled: true},
	}
	controller := &stubPlayerController{running: true, state: common.PlayerStatePlaying}
	checker := NewBasePlayerChecker(controller, common.PlayerAppleMusic, &pushCount, &playing, cache, service)

	info := &stubPlayerInfo{
		title:       "Threshold Track",
		album:       "Album",
		artist:      "Artist",
		position:    70,
		duration:    100,
		trackNumber: 1,
		discNumber:  1,
	}

	checker.processPlayingTrack(context.Background(), info)
	checker.handleStopEvent(context.Background())

	spans := waitForCheckerEndedSpans(t, recorder, 4)
	require.Contains(t, spanNamesWithoutTrackPlayback(spans), "player.handle_playback_threshold")
	rootSpan := findSpanByNameAndTrackTitle(
		t, spans, string(common.PlayerAppleMusic)+"_TrackPlayback", "Threshold Track",
	)
	require.Contains(t, spanEventNames(rootSpan), "scrobble_threshold_reached")
}

func TestProcessPlayingTrackAddsFavoriteSpanOnlyWhenFavoriteChanges(t *testing.T) {
	corelog.Logger = zap.NewNop()

	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(previous)

	pushCount := atomic.Uint32{}
	playing := atomic.Bool{}
	cache := &sync.Map{}
	service := &stubTrackService{}
	controller := &stubPlayerController{running: true, state: common.PlayerStatePlaying}
	checker := NewBasePlayerChecker(controller, common.PlayerAppleMusic, &pushCount, &playing, cache, service)

	info := &stubPlayerInfo{
		title:       "Track Favorite",
		album:       "Album",
		artist:      "Artist",
		position:    10,
		duration:    100,
		trackNumber: 1,
		discNumber:  1,
	}

	checker.processPlayingTrack(context.Background(), info)
	checker.processPlayingTrack(context.Background(), info)

	service.probeResult.TrackFavoriteProjection = tracklogic.TrackFavoriteProjection{
		AppleMusic:      true,
		AppleMusicState: common.TrackFavoriteStateFavorited,
		FavoriteState:   common.TrackFavoriteStateFavorited,
	}
	checker.processPlayingTrack(context.Background(), info)
	checker.handleStopEvent(context.Background())

	spans := waitForCheckerEndedSpans(t, recorder, 4)
	require.Equal(
		t,
		[]string{
			"player.resolve_now_playing",
			"player.sync_favorite_state",
			"player.sync_favorite_state",
		},
		spanNamesWithoutTrackPlayback(spans),
	)
	rootSpan := findSpanByNameAndTrackTitle(
		t, spans, string(common.PlayerAppleMusic)+"_TrackPlayback", "Track Favorite",
	)
	require.Equal(
		t,
		[]string{"track_started", "favorite_state_changed", "player_stopped"},
		spanEventNames(rootSpan),
	)
}

func TestProcessPlayingTrackCreatesTrackPlaybackSpanBySession(t *testing.T) {
	corelog.Logger = zap.NewNop()

	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(previous)

	pushCount := atomic.Uint32{}
	playing := atomic.Bool{}
	cache := &sync.Map{}
	service := &stubTrackService{}
	controller := &stubPlayerController{running: true, state: common.PlayerStatePlaying}
	checker := NewBasePlayerChecker(controller, common.PlayerAppleMusic, &pushCount, &playing, cache, service)

	firstTrack := &stubPlayerInfo{
		title:       "Track A",
		album:       "Album",
		artist:      "Artist",
		position:    10,
		duration:    100,
		trackNumber: 1,
		discNumber:  1,
	}
	secondTrack := &stubPlayerInfo{
		title:       "Track B",
		album:       "Album",
		artist:      "Artist",
		position:    12,
		duration:    100,
		trackNumber: 2,
		discNumber:  1,
	}

	checker.processPlayingTrack(context.Background(), firstTrack)
	firstSpan := checker.currentTrackSpan
	require.NotNil(t, firstSpan)

	checker.processPlayingTrack(context.Background(), firstTrack)
	require.Same(t, firstSpan, checker.currentTrackSpan)

	checker.processPlayingTrack(context.Background(), secondTrack)
	spans := waitForCheckerEndedSpans(t, recorder, 5)
	require.GreaterOrEqual(t, len(spans), 5)
	rootSpan := findSpanByNameAndTrackTitle(
		t, spans, string(common.PlayerAppleMusic)+"_TrackPlayback", "Track A",
	)
	require.NotNil(t, rootSpan)
	require.Equal(
		t, map[string]string{
			"player.source":             string(common.PlayerAppleMusic),
			"track.title":               "Track A",
			"track.artist":              "Artist",
			"track.album":               "Album",
			"track.album_artist":        "Artist",
			"track.metadata_confidence": "medium",
		}, spanStringAttributes(rootSpan),
	)
	require.Equal(
		t, map[string]int64{
			"track.track_number": 1,
			"track.disc_number":  1,
			"track.duration_sec": 100,
		}, spanInt64Attributes(rootSpan),
	)
	require.ElementsMatch(
		t,
		[]string{
			"player.resolve_now_playing",
			"player.sync_favorite_state",
			"player.resolve_now_playing",
			"player.sync_favorite_state",
		},
		spanNamesWithoutTrackPlayback(spans),
	)
	require.Equal(t, []string{"track_started"}, spanEventNames(rootSpan))
	require.Equal(t, "Track B", checker.previousTrack)
	require.NotSame(t, firstSpan, checker.currentTrackSpan)

	checker.handleStopEvent(context.Background())
	spans = waitForCheckerEndedSpans(t, recorder, 6)
	require.GreaterOrEqual(t, len(spans), 6)
	require.Nil(t, checker.currentTrackSpan)
	require.Nil(t, checker.currentTrackCtx)

	checker.processPlayingTrack(context.Background(), secondTrack)
	spans = waitForCheckerEndedSpans(t, recorder, 8)
	require.GreaterOrEqual(t, len(spans), 8)
	stoppedRootSpan := findSpanByNameAndTrackTitle(
		t, spans, string(common.PlayerAppleMusic)+"_TrackPlayback", "Track B",
	)
	require.NotNil(t, stoppedRootSpan)
	require.Contains(t, spanEventNames(stoppedRootSpan), "player_stopped")
	require.NotNil(t, checker.currentTrackSpan)
	require.Equal(t, "Track B", checker.currentTrackSpanKey)
	require.Equal(t, 3, service.nowPlayingCalls)
}

func waitForCheckerEndedSpans(
	t *testing.T,
	recorder *tracetest.SpanRecorder,
	want int,
) []sdktrace.ReadOnlySpan {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		spans := recorder.Ended()
		if len(spans) >= want {
			return spans
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for %d ended spans", want)
	return nil
}

func findSpanByNameAndTrackTitle(
	t *testing.T,
	spans []sdktrace.ReadOnlySpan,
	name string,
	trackTitle string,
) sdktrace.ReadOnlySpan {
	t.Helper()

	for _, span := range spans {
		if span.Name() != name {
			continue
		}
		if spanStringAttributes(span)["track.title"] == trackTitle {
			return span
		}
	}
	t.Fatalf("span %s with track.title=%s not found", name, trackTitle)
	return nil
}

func spanNamesWithoutTrackPlayback(spans []sdktrace.ReadOnlySpan) []string {
	names := make([]string, 0, len(spans))
	for _, span := range spans {
		if strings.HasSuffix(span.Name(), "_TrackPlayback") {
			continue
		}
		names = append(names, span.Name())
	}
	return names
}

func spanEventNames(span sdktrace.ReadOnlySpan) []string {
	events := span.Events()
	names := make([]string, 0, len(events))
	for _, event := range events {
		names = append(names, event.Name)
	}
	return names
}

func spanStringAttributes(span sdktrace.ReadOnlySpan) map[string]string {
	attrs := make(map[string]string)
	for _, attr := range span.Attributes() {
		switch string(attr.Key) {
		case "player.source", "track.title", "track.artist", "track.album", "track.album_artist",
			"track.metadata_confidence":
			attrs[string(attr.Key)] = attr.Value.AsString()
		}
	}
	return attrs
}

func spanInt64Attributes(span sdktrace.ReadOnlySpan) map[string]int64 {
	attrs := make(map[string]int64)
	for _, attr := range span.Attributes() {
		switch string(attr.Key) {
		case "track.track_number", "track.disc_number", "track.duration_sec":
			attrs[string(attr.Key)] = attr.Value.AsInt64()
		}
	}
	return attrs
}
