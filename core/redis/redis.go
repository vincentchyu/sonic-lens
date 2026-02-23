package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/config"
	alog "github.com/vincentchyu/sonic-lens/core/log"
)

var redisClient *redis.Client

// InitRedis 初始化Redis客户端
func InitRedis(redisConfig config.RedisConfig, logger *zap.Logger) {
	options := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisConfig.Host, redisConfig.Port),
		Password: redisConfig.Password,
		DB:       redisConfig.DB,
	}

	// 如果配置了密码，设置用户名（如果需要）
	if redisConfig.Password != "" && redisConfig.Username != "" {
		options.Username = redisConfig.Username
	}

	redisClient = redis.NewClient(options)

	// Enable tracing instrumentation (if Redis is available)
	if err := redisotel.InstrumentTracing(
		redisClient, redisotel.WithTracerProvider(otel.GetTracerProvider()),
	); err != nil {
		alog.Warn(context.Background(), "Failed to enable Redis tracing instrumentation", zap.Error(err))
	}

	// Enable metrics instrumentation (if Redis is available)
	if err := redisotel.InstrumentMetrics(redisClient); err != nil {
		alog.Warn(context.Background(), "Failed to enable Redis metrics instrumentation", zap.Error(err))
	}

	// Add custom logging hook

	redisLogger := NewRedisLogger(logger)
	redisHook := NewRedisHook(redisLogger)
	redisClient.AddHook(redisHook)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		alog.Warn(context.Background(), "Failed to connect to Redis (continuing without Redis)", zap.Error(err))
		// 不要panic，而是继续运行但没有Redis支持
		return
	}

	alog.Info(context.Background(), "Successfully connected to Redis with tracing and logging enabled")
}

// GetRedisClient 获取Redis客户端实例
func GetRedisClient() *redis.Client {
	return redisClient
}
