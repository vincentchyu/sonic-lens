package scrobbler

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/telemetry"
)

func startPlayerControllerSpan(
	ctx context.Context,
	source common.PlayerType,
	operation string,
) (context.Context, trace.Span) {
	spanCtx, span := telemetry.StartSpanForTracerName(
		ctx,
		_TracerName,
		"player."+operation,
		trace.WithSpanKind(trace.SpanKindInternal),
	)
	span.SetAttributes(
		attribute.String("player.source", string(source)),
		attribute.String("player.operation", operation),
	)
	return spanCtx, span
}

func markPlayerSpanError(span trace.Span, err error) {
	if err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
