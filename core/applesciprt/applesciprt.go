package applesciprt

import (
	"context"

	"github.com/andybrewer/mack"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/vincentchyu/sonic-lens/core/telemetry"
)

const tracerName = "sonic-lens/core/applesciprt"

var tellFn = mack.Tell

func Tell(ctx context.Context, application string, command string) (result string, err error) {
	_, span := telemetry.StartSpanForTracerName(
		ctx,
		tracerName,
		"applescript.tell",
		trace.WithSpanKind(trace.SpanKindInternal),
	)
	span.SetAttributes(
		attribute.String("script.engine", "applescript"),
		attribute.String("script.operation", "tell"),
		attribute.String("script.application", application),
	)
	defer span.End()

	result, err = tellFn(application, command)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return result, err
	}
	return result, nil
}
