package redis

import (
	"context"
	"net"
	"time"

	"github.com/redis/go-redis/v9"
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

		conn, err := hook(ctx, network, addr)

		// Log dial operation
		h.logger.LogDial(ctx, start, network, addr, err)

		return conn, err
	}
}

func (h redisHook) ProcessHook(hook redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		start := time.Now()

		err := hook(ctx, cmd)

		// Log command execution
		h.logger.LogCommand(ctx, start, cmd, err)

		return err
	}
}

func (h redisHook) ProcessPipelineHook(hook redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		start := time.Now()

		err := hook(ctx, cmds)

		// Log pipeline execution
		h.logger.LogPipeline(ctx, start, cmds, err)

		return err
	}
}
