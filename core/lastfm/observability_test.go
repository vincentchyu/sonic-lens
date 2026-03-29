package lastfm

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestLastfmWrappersCreateClientSpans(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(previous)

	previousAPI := lastfmApi
	lastfmApi = nil
	defer func() {
		lastfmApi = previousAPI
	}()

	ctx, parentSpan := tp.Tracer("test").Start(context.Background(), "parent")

	_, _ = PushTrackScrobble(
		ctx,
		&PushTrackScrobbleReq{
			Artist:      "artist",
			Track:       "track",
			Album:       "album",
			TrackNumber: 1,
			Timestamp:   time.Now().Unix(),
			Duration:    180,
		},
	)
	_ = TrackUpdateNowPlaying(
		ctx,
		&TrackUpdateNowPlayingReq{
			Artist:   "artist",
			Track:    "track",
			Album:    "album",
			Duration: 180,
		},
	)
	_, _ = IsFavorite(ctx, "artist", "track")
	_ = SetFavorite(ctx, "artist", "track", true)
	parentSpan.End()

	spans := waitForLastfmEndedSpans(t, recorder, 5)
	names := map[string]sdktrace.ReadOnlySpan{}
	for _, span := range spans {
		names[span.Name()] = span
	}

	assertLastfmSpan(t, names["lastfm.track.scrobble"], parentSpan.SpanContext())
	assertLastfmSpan(t, names["lastfm.track.update_now_playing"], parentSpan.SpanContext())
	assertLastfmSpan(t, names["lastfm.track.get_info"], parentSpan.SpanContext())
	assertLastfmSpan(t, names["lastfm.track.love"], parentSpan.SpanContext())
}

func assertLastfmSpan(t *testing.T, span sdktrace.ReadOnlySpan, parent trace.SpanContext) {
	t.Helper()

	if span == nil {
		t.Fatal("expected lastfm span to exist")
	}
	if got := span.SpanKind(); got != trace.SpanKindClient {
		t.Fatalf("expected client span, got %v", got)
	}
	if got := span.Status().Code; got != codes.Error {
		t.Fatalf("expected error status for uninitialized api, got %v", got)
	}
	if span.Parent().TraceID() != parent.TraceID() {
		t.Fatal("expected lastfm span to inherit parent trace id")
	}
}

func waitForLastfmEndedSpans(
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
