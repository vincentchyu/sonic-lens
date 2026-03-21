package telemetry

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

func TestGoCreatesChildSpan(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(previous)

	parentCtx, parentSpan := tp.Tracer(TracerName).Start(context.Background(), "parent")
	defer parentSpan.End()

	childCtxCh := make(chan trace.SpanContext, 1)
	GoSafe(
		parentCtx, "child", func(asyncCtx context.Context) {
			childCtxCh <- trace.SpanFromContext(asyncCtx).SpanContext()
		},
	)

	childCtx := <-childCtxCh
	if !childCtx.IsValid() {
		t.Fatal("expected child span context to be valid")
	}
	if childCtx.TraceID() != parentSpan.SpanContext().TraceID() {
		t.Fatal("expected child span to keep parent trace id")
	}
	if childCtx.SpanID() == parentSpan.SpanContext().SpanID() {
		t.Fatal("expected child span id to differ from parent")
	}

	waitForEndedSpans(t, recorder, 1)
}

func TestGoRecoversPanicAndMarksSpanError(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(previous)

	done := make(chan struct{})
	GoSafe(
		context.Background(), "panic-child", func(context.Context) {
			defer close(done)
			panic("boom")
		},
	)
	<-done

	spans := waitForEndedSpans(t, recorder, 1)
	if got := spans[0].Status().Code; got != codes.Error {
		t.Fatalf("expected panic span status error, got %v", got)
	}
	if got := spans[0].Name(); got != "panic-child" {
		t.Fatalf("expected span name panic-child, got %s", got)
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
