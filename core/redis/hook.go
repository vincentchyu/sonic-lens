package redis

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/vincentchyu/sonic-lens/core/telemetry"
)

const (
	redisTracerName = "github.com/vincentchyu/sonic-lens/redis"
)

// redisHook is a Redis hook that provides logging and tracing
type redisHook struct {
	logger *Logger
}

// NewRedisHook creates a new Redis hook
func NewRedisHook(logger *Logger) redis.Hook {
	return redisHook{
		logger: logger,
	}
}

func (h redisHook) DialHook(hook redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		start := time.Now()

		// Add tracing for dialing
		ctx, span := telemetry.StartSpanForTracerName(
			ctx, redisTracerName, "redis.dial", oteltrace.WithSpanKind(oteltrace.SpanKindClient),
		)
		span.SetAttributes(
			attribute.String("server.address", addr),
			attribute.String("network", network),
		)
		defer func() {
			span.End()
		}()

		conn, err := hook(ctx, network, addr)

		// Log dial operation
		h.logger.LogDial(ctx, start, network, addr, err)

		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		} else {
			span.SetStatus(codes.Ok, "")
		}

		return conn, err
	}
}

func (h redisHook) ProcessHook(hook redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		start := time.Now()

		// Add tracing for command processing
		ctx, span := telemetry.StartSpanForTracerName(
			ctx, redisTracerName, "redis.process", oteltrace.WithSpanKind(oteltrace.SpanKindClient),
		)
		span.SetAttributes(
			attribute.String("db.statement", cmd.String()),
		)
		defer func() {
			span.End()
		}()

		err := hook(ctx, cmd)

		// Log command execution
		h.logger.LogCommand(ctx, start, cmd, err)

		if err != nil && !errors.Is(err, redis.Nil) {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		} else {
			span.SetStatus(codes.Ok, "")
		}

		// Add execution time as span attribute
		span.SetAttributes(
			attribute.String("elapsed", time.Since(start).String()),
		)

		return err
	}
}

func (h redisHook) ProcessPipelineHook(hook redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		start := time.Now()

		// Add tracing for pipeline processing
		ctx, span := telemetry.StartSpanForTracerName(
			ctx, redisTracerName, "redis.pipeline", oteltrace.WithSpanKind(oteltrace.SpanKindClient),
		)

		// Add command count as span attribute
		span.SetAttributes(
			attribute.Int("db.redis.num_cmd", len(cmds)),
		)

		// Add commands as span attribute (truncated if too long)
		if len(cmds) <= 10 {
			cmdStrings := make([]string, len(cmds))
			for i, cmd := range cmds {
				cmdStrings[i] = cmd.String()
			}
			span.SetAttributes(
				attribute.StringSlice("db.statements", cmdStrings),
			)
		}

		defer func() {
			span.End()
		}()

		err := hook(ctx, cmds)

		// Log pipeline execution
		h.logger.LogPipeline(ctx, start, cmds, err)

		if err != nil && !errors.Is(err, redis.Nil) {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		} else {
			span.SetStatus(codes.Ok, "")
		}

		// Add execution time as span attribute
		span.SetAttributes(
			attribute.String("elapsed", time.Since(start).String()),
		)

		return err
	}
}
