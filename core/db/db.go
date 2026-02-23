package db

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"gorm.io/gorm/logger"

	"github.com/vincentchyu/sonic-lens/core/log"
)

// customLogger is a custom logger for GORM that uses zap and OpenTelemetry
type customLogger struct {
	logger *zap.Logger
}

// NewCustomLogger creates a new custom logger
func NewCustomLogger(log *zap.Logger) logger.Interface {
	return &customLogger{
		logger: log,
	}
}

// LogMode sets the logger mode
func (l *customLogger) LogMode(level logger.LogLevel) logger.Interface {
	return l
}

func (l *customLogger) Fields(datas ...interface{}) []zap.Field {
	fields := make([]zap.Field, 0, len(datas))
	for _, data := range datas {
		if v, ok := data.(zap.Field); ok {
			fields = append(fields, v)
		}
	}
	return fields
}

// Info logs info level messages
func (l *customLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	log.InfoForLog(ctx, l.logger, msg, l.Fields(data...)...)
}

// Warn logs warn level messages
func (l *customLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	log.InfoForLog(ctx, l.logger, msg, l.Fields(data...)...)
}

// Error logs error level messages
func (l *customLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	log.ErrorForLog(ctx, l.logger, msg, l.Fields(data...)...)
}

// Trace logs SQL queries and their execution time
func (l *customLogger) Trace(
	ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error,
) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	// Add tracing for SQL queries
	ctx, span := startSpan(ctx, "sql.run")
	defer func() {
		endSpan(span, err)
	}()

	// Add SQL query and execution time as span attributes
	span.SetAttributes(
		attribute.String("sql.query", sql),
		attribute.Int64("rows.affected", rows),
		attribute.String("elapsed", elapsed.String()),
	)

	// Define a slow query threshold (e.g., 200ms)
	slowThreshold := 200 * time.Millisecond

	switch {
	case err != nil:
		l.Error(
			ctx,
			"sql error",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
			zap.Error(err),
		)
		span.SetAttributes(
			attribute.String("error", err.Error()),
		)
	case elapsed > slowThreshold:
		l.Warn(
			ctx,
			"slow sql query",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
		)
	default:
		l.Info(
			ctx,
			"sql query",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
		)
	}
}

// MysqlDSN generates a MySQL DSN from the provided configuration
func MysqlDSN(dsn string) string {
	return dsn
}
