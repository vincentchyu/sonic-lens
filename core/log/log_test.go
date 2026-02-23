package log

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/config"
	telemetry2 "github.com/vincentchyu/sonic-lens/core/telemetry"
)

func TInit() {
	_, _ = LogInit("./.logs", "info", make(<-chan struct{}))
}

func TestLogInit(t *testing.T) {
	TInit()
	// Test with background context (no trace)
	Info(context.Background(), "haha", zap.String("time", time.Now().Format("2006-01-02 15:04:05")))

	// Test traceFields with background context
	fields := traceFields(context.Background())
	if len(fields) != 0 {
		t.Error("trace fields should be empty for background context")
	}
}

func TestTraceLogging(t *testing.T) {
	TInit()
	if err := telemetry2.Init(
		config.TelemetryConfig{
			Name:           "test",
			Endpoint:       "",
			Sampler:        0,
			Batcher:        "",
			OtlpHeaders:    nil,
			OtlpHttpPath:   "",
			OtlpHttpSecure: false,
			Disabled:       false,
		},
	); err != nil {
		t.Fatalf("failed to initialize telemetry: %v", err)
	}
	defer telemetry2.Shutdown(context.Background())

	// Create a context with trace
	ctx := context.Background()
	ctx, span := telemetry2.StartSpan(ctx, "test-span")
	defer span.End()

	// Test logging with trace context
	Info(ctx, "test log with trace", zap.String("time", time.Now().Format("2006-01-02 15:04:05")))

	Debug(
		ctx, "debug", zap.String("time", time.Now().Format("2006-01-02 15:04:05")),
		zap.String("TraceIDFromContext", telemetry2.TraceIDFromContext(ctx)),
	)
	Warn(
		ctx, "warn", zap.String("time", time.Now().Format("2006-01-02 15:04:05")),
		zap.String("SpanIDFromContext", telemetry2.SpanIDFromContext(ctx)),
	)
}
