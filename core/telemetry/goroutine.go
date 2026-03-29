package telemetry

import (
	"context"
	"fmt"
	"runtime/debug"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// GoSafe 在新协程中启动一个子 span，并统一兜底 panic，避免异步任务把进程打崩。
func GoSafe(ctx context.Context, spanName string, fn func(context.Context)) {
	goTraceWithContext(ctx, spanName, false, fn)
}

// GoSafeDetached 在保留 trace 父子关系的同时脱离上游取消信号，适合异步落库等收尾任务。
func GoSafeDetached(ctx context.Context, spanName string, fn func(context.Context)) {
	goTraceWithContext(ctx, spanName, true, fn)
}

// GoOnlySafe 只做安全检查
func GoOnlySafe(ctx context.Context, fn func(context.Context)) {
	goWithContext(ctx, fn)
}

func goTraceWithContext(ctx context.Context, spanName string, detached bool, fn func(context.Context)) {
	if fn == nil {
		return
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if detached {
		ctx = context.WithoutCancel(ctx)
	}

	go func() {
		asyncCtx, span := StartSpan(ctx, spanName, trace.WithSpanKind(trace.SpanKindInternal))
		defer func() {
			if recovered := recover(); recovered != nil {
				err := fmt.Errorf("异步协程 panic: %v", recovered)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				fields := []zap.Field{
					zap.String("span_name", spanName),
					zap.Any("panic", recovered),
					zap.ByteString("stack", debug.Stack()),
				}
				if spanCtx := span.SpanContext(); spanCtx.IsValid() {
					fields = append(
						fields,
						zap.String("trace_id", spanCtx.TraceID().String()),
						zap.String("span_id", spanCtx.SpanID().String()),
					)
				}
				zap.L().Error("异步协程发生 panic", fields...)
			}
			span.End()
		}()

		fn(asyncCtx)
	}()
}

func goWithContext(ctx context.Context, fn func(context.Context)) {
	if fn == nil {
		return
	}

	if ctx == nil {
		ctx = context.Background()
	}

	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				fields := []zap.Field{
					zap.Any("panic", recovered),
					zap.ByteString("stack", debug.Stack()),
				}
				if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
					fields = append(
						fields,
						zap.String("trace_id", spanCtx.TraceID().String()),
						zap.String("span_id", spanCtx.SpanID().String()),
					)
				}
				zap.L().Error("异步协程发生 panic", fields...)
			}
		}()

		fn(ctx)
	}()
}
