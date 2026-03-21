package applesciprt

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTellCreatesSpan(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(previous)

	previousTellFn := tellFn
	tellFn = func(application string, commands ...string) (string, error) {
		return "ok", nil
	}
	defer func() {
		tellFn = previousTellFn
	}()

	ctx, parentSpan := tp.Tracer("test").Start(context.Background(), "parent")
	_, err := Tell(ctx, "Music", "return 1")
	if err != nil {
		t.Fatalf("Tell returned error: %v", err)
	}
	parentSpan.End()

	spans := waitForEndedSpans(t, recorder, 2)
	var found sdktrace.ReadOnlySpan
	for _, span := range spans {
		if span.Name() == "applescript.tell" {
			found = span
			break
		}
	}
	if found == nil {
		t.Fatal("expected applescript.tell span")
	}
	if found.Parent().TraceID() != parentSpan.SpanContext().TraceID() {
		t.Fatal("expected applescript.tell span to inherit parent trace id")
	}
}

func waitForEndedSpans(t *testing.T, recorder *tracetest.SpanRecorder, want int) []sdktrace.ReadOnlySpan {
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
