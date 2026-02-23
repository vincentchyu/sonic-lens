package log

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var dayRotationCounter int32 = 0 // 记录每天日志文件计数值
var Logger *zap.Logger

// 创建日期分隔的日志文件
func openDailyLogFile(dir string) (zapcore.WriteSyncer, error) {
	var fsyncer zapcore.WriteSyncer
	atomic.AddInt32(&dayRotationCounter, 1)
	logFileName := time.Now().Format("2006-01-02") + ".log"

	filePath := filepath.Join(dir, logFileName)
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
		fsyncer = zapcore.AddSync(io.Discard)
	}
	fsyncer = zapcore.AddSync(f)
	return fsyncer, nil
}

func createLumberJackLogger(filename string) zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   filename, // 文件位置
		MaxSize:    10,       // 进行切割之前,日志文件的最大大小(MB为单位)
		MaxAge:     7,        // 保留旧文件的最大天数
		MaxBackups: 10,       // 保留旧文件的最大个数
		Compress:   true,     // 是否压缩/归档旧文件
		LocalTime:  true,
	}
	// AddSync 将 io.Writer 转换为 WriteSyncer。
	// 它试图变得智能：如果 io.Writer 的具体类型实现了 WriteSyncer，我们将使用现有的 Sync 方法。
	// 如果没有，我们将添加一个无操作同步。

	return zapcore.AddSync(lumberJackLogger)
}

// Debug logs a debug message with context
func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	fields = append(fields, traceFields(ctx)...)
	Logger.Debug(msg, fields...)
}

// Info logs an info message with context
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	fields = append(fields, traceFields(ctx)...)
	Logger.Info(msg, fields...)
}

// Warn logs a warning message with context
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	fields = append(fields, traceFields(ctx)...)
	Logger.Warn(msg, fields...)
}

func ErrorForLog(ctx context.Context, logger *zap.Logger, msg string, fields ...zap.Field) {
	fields = append(fields, traceFields(ctx)...)
	logger.Error(msg, fields...)
}

func InfoForLog(ctx context.Context, logger *zap.Logger, msg string, fields ...zap.Field) {
	fields = append(fields, traceFields(ctx)...)
	logger.Info(msg, fields...)
}

func WarnForLog(ctx context.Context, logger *zap.Logger, msg string, fields ...zap.Field) {
	fields = append(fields, traceFields(ctx)...)
	logger.Warn(msg, fields...)
}

// Error logs an error message with context
func Error(ctx context.Context, msg string, fields ...zap.Field) {
	fields = append(fields, traceFields(ctx)...)
	Logger.Error(msg, fields...)
}

// traceFields extracts trace information from context
func traceFields(ctx context.Context) []zap.Field {
	if ctx == nil {
		return nil
	}

	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return nil
	}

	spanCtx := span.SpanContext()
	if !spanCtx.IsValid() {
		return nil
	}

	return []zap.Field{
		zap.String("trace_id", spanCtx.TraceID().String()),
		zap.String("span_id", spanCtx.SpanID().String()),
	}
}

func LogInit(logPath, infoLevel string, c <-chan struct{}) (*zap.Logger, *zap.Logger) {
	err := os.MkdirAll(logPath, 0755)
	if err != nil && !os.IsExist(err) {
		fmt.Println("Error occurred:", err.Error())
		os.Exit(1)
	}
	atom := zap.NewAtomicLevel()
	switch infoLevel {
	case "debug":
		atom.SetLevel(zap.DebugLevel)
	case "info":
		atom.SetLevel(zap.InfoLevel)
	case "error":
		atom.SetLevel(zap.ErrorLevel)
	case "warn":
		atom.SetLevel(zap.WarnLevel)
	default:
		atom.SetLevel(zap.InfoLevel)

	}
	/*fsyncer, err := openDailyLogFile(logPath)
	if err != nil {
		fmt.Println("Failed to set up logger due to error:", err.Error())
		os.Exit(1)
	}*/
	fileAndStdoutSyncer := zapcore.NewMultiWriteSyncer(
		createLumberJackLogger(logPath+"/go_lastfm-scrobbler.log"),
		zapcore.AddSync(os.Stdout),
	)

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	encoderConfig = zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.DateTime)

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		fileAndStdoutSyncer,
		atom,
	)

	Logger = zap.New(
		core, zap.AddCaller(), zap.AddCallerSkip(1),
		// zap.Development(),
	)
	return zap.New(
			core, zap.AddCaller(), zap.AddCallerSkip(5), zap.Development(),
			// zap.Development(),
		), zap.New(
			core, zap.AddCaller(), zap.AddCallerSkip(8), zap.Development(),
			// zap.Development(),
		)
}
