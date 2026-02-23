package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/core/log"
)

// Logger is a custom logger for Redis that uses zap and OpenTelemetry
type Logger struct {
	logger *zap.Logger
}

// NewRedisLogger creates a new Redis logger
func NewRedisLogger(logger *zap.Logger) *Logger {
	return &Logger{
		logger: logger,
	}
}

// LogCommand logs Redis command execution with timing information
func (l *Logger) LogCommand(ctx context.Context, start time.Time, cmd redis.Cmder, err error) {
	elapsed := time.Since(start)

	// Define a slow query threshold (e.g., 100ms)
	slowThreshold := 100 * time.Millisecond

	// Extract command name and args
	cmdString := cmd.String()

	switch {
	case err != nil && !errors.Is(err, redis.Nil):
		log.ErrorForLog(
			ctx, l.logger, "redis error",
			zap.String("command", cmdString),
			zap.Duration("elapsed", elapsed),
			zap.Error(err),
		)
	case elapsed > slowThreshold:
		log.WarnForLog(
			ctx, l.logger, "slow redis command",
			zap.String("command", cmdString),
			zap.Duration("elapsed", elapsed),
		)
	default:
		log.InfoForLog(
			ctx, l.logger, "redis command",
			zap.String("command", cmdString),
			zap.Duration("elapsed", elapsed),
		)
	}
}

// LogPipeline logs Redis pipeline execution with timing information
func (l *Logger) LogPipeline(ctx context.Context, start time.Time, cmds []redis.Cmder, err error) {
	elapsed := time.Since(start)

	// Define a slow query threshold (e.g., 200ms for pipelines)
	slowThreshold := 200 * time.Millisecond

	// Extract command names
	cmdStrings := make([]string, len(cmds))
	for i, cmd := range cmds {
		cmdStrings[i] = cmd.String()
	}

	switch {
	case err != nil && !errors.Is(err, redis.Nil):
		log.ErrorForLog(
			ctx, l.logger, "redis pipeline error",
			zap.Strings("commands", cmdStrings),
			zap.Duration("elapsed", elapsed),
			zap.Error(err),
		)
	case elapsed > slowThreshold:
		log.WarnForLog(
			ctx, l.logger, "slow redis pipeline",
			zap.Strings("commands", cmdStrings),
			zap.Duration("elapsed", elapsed),
		)
	default:
		log.InfoForLog(
			ctx, l.logger, "redis pipeline",
			zap.Strings("commands", cmdStrings),
			zap.Duration("elapsed", elapsed),
		)
	}
}

// LogDial logs Redis connection dialing
func (l *Logger) LogDial(ctx context.Context, start time.Time, network, addr string, err error) {
	elapsed := time.Since(start)

	if err != nil {
		log.ErrorForLog(
			ctx, l.logger, "redis dial error",
			zap.String("network", network),
			zap.String("address", addr),
			zap.Duration("elapsed", elapsed),
			zap.Error(err),
		)
	} else {
		log.InfoForLog(
			ctx, l.logger, "redis dial",
			zap.String("network", network),
			zap.String("address", addr),
			zap.Duration("elapsed", elapsed),
		)
	}
}
