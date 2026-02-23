package telemetry

import (
	"context"
	"fmt"
	"log"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/config"
)

const (
	TracerName = "sonic-lens"
)

var (
	// tracerProvider is the global tracer provider
	tracerProvider *sdktrace.TracerProvider
	// otelLogger is the OpenTelemetry logger bridge
	otelLogger *zap.Logger
	// once ensures that the initialization only happens once
	once sync.Once
)

// Init initializes the OpenTelemetry tracing and logging
func Init(telemetryConfig config.TelemetryConfig) error {
	var initErr error
	once.Do(
		func() {
			initErr = initTelemetry(telemetryConfig)
		},
	)
	return initErr
}

// initTelemetry initializes the OpenTelemetry components
func initTelemetry(cfg config.TelemetryConfig) error {
	if cfg.Disabled {
		return nil
	}

	// Create a resource with service information
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.Name),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Create a trace exporter (stdout only if not disabled and for debugging)
	// Usually in production we'd use Jaeger/OTLP, but here we just manage the stdout noise
	traceExporter, err := stdouttrace.New(
		// stdouttrace.WithPrettyPrint(),
		stdouttrace.WithoutTimestamps(),
	)
	if err != nil {
		return fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create a trace provider
	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExporter),
	)

	// Set the global tracer provider
	otel.SetTracerProvider(tracerProvider)

	// Set the global propagator for trace context
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	// Use the existing zap logger
	otelLogger = zap.L()

	return nil
}

// GetTracerForName returns a tracer for the given name
func GetTracerForName(name string) trace.Tracer {
	return otel.GetTracerProvider().Tracer(name)
}

// GetTracer returns a tracer for the given name
func GetTracer() trace.Tracer {
	return otel.GetTracerProvider().Tracer(TracerName)
}

// GetLogger returns the OpenTelemetry logger bridge
func GetLogger() *zap.Logger {
	return otelLogger
}

// Shutdown shuts down the telemetry components
func Shutdown(ctx context.Context) error {
	if tracerProvider != nil {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
			return err
		}
	}
	return nil
}

// StartSpan starts a new span and returns the context with the span
func StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return TracerFromContext(ctx).Start(ctx, spanName, opts...)
}

func StartSpanForTracerName(ctx context.Context, tracerName, spanName string, opts ...trace.SpanStartOption) (
	context.Context, trace.Span,
) {
	return TracerNameFromContext(ctx, tracerName).Start(ctx, spanName, opts...)
}

// TracerFromContext returns a tracer in ctx, otherwise returns a global tracer.
func TracerFromContext(ctx context.Context) (tracer trace.Tracer) {
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		tracer = span.TracerProvider().Tracer(TracerName)
	} else {
		tracer = otel.Tracer(TracerName)
	}
	return
}

func TracerNameFromContext(ctx context.Context, name string) (tracer trace.Tracer) {
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		tracer = span.TracerProvider().Tracer(name)
	} else {
		tracer = otel.Tracer(name)
	}
	return
}
