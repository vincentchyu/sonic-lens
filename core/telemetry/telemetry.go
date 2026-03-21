package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
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
	// meterProvider is the global meter provider
	meterProvider *sdkmetric.MeterProvider
	// otelLogger is the OpenTelemetry logger bridge
	otelLogger *zap.Logger
	// metricRegistrations keeps callback registrations that need to be released on shutdown.
	metricRegistrations []metric.Registration
	registrationMu      sync.Mutex
	// once ensures that the initialization only happens once
	once sync.Once
)

const (
	defaultMetricInterval = 15 * time.Second
	stdoutBatcher         = "stdout"
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
		zap.L().Info("Telemetry 已禁用")
		return nil
	}

	otel.SetErrorHandler(
		otel.ErrorHandlerFunc(
			func(err error) {
				zap.L().Warn("OpenTelemetry 内部错误", zap.Error(err))
			},
		),
	)

	// Create a resource with service information
	res, err := buildResource(cfg)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	traceExporter, metricExporter, err := buildExporters(cfg)
	if err != nil {
		return err
	}

	traceProviderOptions := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(resolveSampler(cfg.Sampler)))),
	}
	if traceExporter != nil {
		traceProviderOptions = append(traceProviderOptions, sdktrace.WithBatcher(traceExporter))
	}
	tracerProvider = sdktrace.NewTracerProvider(traceProviderOptions...)

	meterProviderOptions := []sdkmetric.Option{
		sdkmetric.WithResource(res),
	}
	if metricExporter != nil {
		meterProviderOptions = append(
			meterProviderOptions,
			sdkmetric.WithReader(
				sdkmetric.NewPeriodicReader(
					metricExporter,
					sdkmetric.WithInterval(resolveMetricInterval(cfg.MetricIntervalSeconds)),
				),
			),
		)
	}
	meterProvider = sdkmetric.NewMeterProvider(meterProviderOptions...)

	// Set the global providers.
	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)

	// Set the global propagator for trace context
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	if cfg.RuntimeMetricsEnabled {
		if err := runtime.Start(runtime.WithMeterProvider(meterProvider)); err != nil {
			return fmt.Errorf("failed to start runtime metrics: %w", err)
		}
	}

	// Use the existing zap logger
	otelLogger = zap.L()
	logTelemetryStartup(cfg)

	return nil
}

func buildResource(cfg config.TelemetryConfig) (*resource.Resource, error) {
	serviceName := resolveServiceName(cfg.Name)
	attributes := []attribute.KeyValue{
		semconv.ServiceNameKey.String(serviceName),
	}
	if serviceVersion := resolveServiceVersion(); serviceVersion != "" {
		attributes = append(attributes, semconv.ServiceVersionKey.String(serviceVersion))
	}
	if environment := strings.TrimSpace(cfg.Environment); environment != "" {
		attributes = append(attributes, attribute.String("deployment.environment.name", environment))
	}

	return resource.New(
		context.Background(),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithContainer(),
		resource.WithHost(),
		resource.WithAttributes(
			attributes...,
		),
	)
}

func buildExporters(
	cfg config.TelemetryConfig,
) (sdktrace.SpanExporter, sdkmetric.Exporter, error) {
	if strings.EqualFold(strings.TrimSpace(cfg.Batcher), stdoutBatcher) {
		traceExporter, err := stdouttrace.New(stdouttrace.WithoutTimestamps())
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create stdout trace exporter: %w", err)
		}
		return traceExporter, nil, nil
	}

	if !shouldUseOTLPExporter(cfg) {
		traceExporter, err := stdouttrace.New(stdouttrace.WithoutTimestamps())
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create stdout trace exporter: %w", err)
		}
		return traceExporter, nil, nil
	}

	traceOptions, metricOptions := buildOTLPOptions(cfg)
	traceExporter, err := otlptracegrpc.New(context.Background(), traceOptions...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create otlp trace exporter: %w", err)
	}
	metricExporter, err := otlpmetricgrpc.New(context.Background(), metricOptions...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create otlp metric exporter: %w", err)
	}

	return traceExporter, metricExporter, nil
}

func buildOTLPOptions(cfg config.TelemetryConfig) ([]otlptracegrpc.Option, []otlpmetricgrpc.Option) {
	traceOptions := make([]otlptracegrpc.Option, 0, 4)
	metricOptions := make([]otlpmetricgrpc.Option, 0, 4)

	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint != "" {
		if strings.Contains(endpoint, "://") {
			traceOptions = append(traceOptions, otlptracegrpc.WithEndpointURL(endpoint))
			metricOptions = append(metricOptions, otlpmetricgrpc.WithEndpointURL(endpoint))
		} else {
			traceOptions = append(traceOptions, otlptracegrpc.WithEndpoint(endpoint))
			metricOptions = append(metricOptions, otlpmetricgrpc.WithEndpoint(endpoint))
		}
	}

	if cfg.Insecure {
		traceOptions = append(traceOptions, otlptracegrpc.WithInsecure())
		metricOptions = append(metricOptions, otlpmetricgrpc.WithInsecure())
	}

	headers := cfg.OtlpHeaders
	if len(headers) > 0 {
		traceOptions = append(traceOptions, otlptracegrpc.WithHeaders(headers))
		metricOptions = append(metricOptions, otlpmetricgrpc.WithHeaders(headers))
	}

	return traceOptions, metricOptions
}

func resolveServiceName(configured string) string {
	if value := strings.TrimSpace(configured); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME")); value != "" {
		return value
	}
	return TracerName
}

func resolveServiceVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info == nil {
		return ""
	}
	if version := strings.TrimSpace(info.Main.Version); version != "" && version != "(devel)" {
		return version
	}
	return ""
}

func resolveSampler(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	case value == 0:
		return 1
	default:
		return value
	}
}

func resolveMetricInterval(seconds int) time.Duration {
	if seconds <= 0 {
		return defaultMetricInterval
	}
	return time.Duration(seconds) * time.Second
}

func shouldUseOTLPExporter(cfg config.TelemetryConfig) bool {
	return resolveOTLPEndpoint(cfg) != ""
}

func resolveOTLPEndpoint(cfg config.TelemetryConfig) string {
	if value := strings.TrimSpace(cfg.Endpoint); value != "" {
		return value
	}
	for _, key := range []string{
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT",
		"OTEL_EXPORTER_OTLP_ENDPOINT",
	} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func resolveTraceExporterKind(cfg config.TelemetryConfig) string {
	if strings.EqualFold(strings.TrimSpace(cfg.Batcher), stdoutBatcher) {
		return "stdout"
	}
	if shouldUseOTLPExporter(cfg) {
		return "otlp-grpc"
	}
	return "stdout-fallback"
}

func resolveMetricExporterKind(cfg config.TelemetryConfig) string {
	if shouldUseOTLPExporter(cfg) {
		return "otlp-grpc"
	}
	return "disabled"
}

func logTelemetryStartup(cfg config.TelemetryConfig) {
	fields := []zap.Field{
		zap.String("service_name", resolveServiceName(cfg.Name)),
		zap.String("service_version", resolveServiceVersion()),
		zap.String("environment", strings.TrimSpace(cfg.Environment)),
		zap.String("trace_exporter", resolveTraceExporterKind(cfg)),
		zap.String("metric_exporter", resolveMetricExporterKind(cfg)),
		zap.String("otlp_endpoint", resolveOTLPEndpoint(cfg)),
		zap.Bool("otlp_insecure", cfg.Insecure),
		zap.Float64("sampler", resolveSampler(cfg.Sampler)),
		zap.Duration("metric_interval", resolveMetricInterval(cfg.MetricIntervalSeconds)),
		zap.Bool("runtime_metrics_enabled", cfg.RuntimeMetricsEnabled),
		zap.Bool("db_stats_metrics_enabled", cfg.DBStatsMetricsEnabled),
		zap.String("propagators", "tracecontext,baggage"),
	}

	zap.L().Info("Telemetry 启动完成", fields...)
}

// GetTracerForName returns a tracer for the given name
func GetTracerForName(name string) trace.Tracer {
	return otel.GetTracerProvider().Tracer(name)
}

// GetTracer returns a tracer for the given name
func GetTracer() trace.Tracer {
	return otel.GetTracerProvider().Tracer(TracerName)
}

// GetTracerProvider 返回当前全局 tracer provider，便于第三方 instrumentation 复用。
func GetTracerProvider() trace.TracerProvider {
	if tracerProvider != nil {
		return tracerProvider
	}
	return otel.GetTracerProvider()
}

// GetMeterProvider 返回当前全局 meter provider，便于第三方 instrumentation 复用。
func GetMeterProvider() metric.MeterProvider {
	if meterProvider != nil {
		return meterProvider
	}
	return otel.GetMeterProvider()
}

// HasMeterProvider 返回当前是否已经初始化了真实的 meter provider。
func HasMeterProvider() bool {
	return meterProvider != nil
}

// GetLogger returns the OpenTelemetry logger bridge
func GetLogger() *zap.Logger {
	return otelLogger
}

// RegisterMetricRegistration 记录 callback registration，便于 shutdown 时统一释放。
func RegisterMetricRegistration(reg metric.Registration) {
	if reg == nil {
		return
	}
	registrationMu.Lock()
	defer registrationMu.Unlock()

	metricRegistrations = append(metricRegistrations, reg)
}

// Shutdown shuts down the telemetry components
func Shutdown(ctx context.Context) error {
	var shutdownErr error

	registrationMu.Lock()
	for _, reg := range metricRegistrations {
		if reg == nil {
			continue
		}
		shutdownErr = errors.Join(shutdownErr, reg.Unregister())
	}
	metricRegistrations = nil
	registrationMu.Unlock()

	if meterProvider != nil {
		if err := meterProvider.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down meter provider: %v", err)
			shutdownErr = errors.Join(shutdownErr, err)
		}
	}
	if tracerProvider != nil {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
			shutdownErr = errors.Join(shutdownErr, err)
		}
	}
	return shutdownErr
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
