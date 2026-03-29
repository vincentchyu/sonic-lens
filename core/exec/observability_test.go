package exec

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestRunCommandCreatesSpan(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(previous)

	previousExecCommand := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "printf ok")
	}
	defer func() {
		execCommand = previousExecCommand
	}()

	ctx, parentSpan := tp.Tracer("test").Start(context.Background(), "parent")
	output, err := runCommand(ctx, "fake-command", "--arg")
	if err != nil {
		t.Fatalf("runCommand returned error: %v", err)
	}
	if output != "ok" {
		t.Fatalf("expected output ok, got %q", output)
	}
	parentSpan.End()

	spans := waitForCommandEndedSpans(t, recorder, 2)
	var found sdktrace.ReadOnlySpan
	for _, span := range spans {
		if span.Name() == "exec.run_command" {
			found = span
			break
		}
	}
	if found == nil {
		t.Fatal("expected exec.run_command span")
	}
	if found.Parent().TraceID() != parentSpan.SpanContext().TraceID() {
		t.Fatal("expected exec.run_command span to inherit parent trace id")
	}
}

func waitForCommandEndedSpans(t *testing.T, recorder *tracetest.SpanRecorder, want int) []sdktrace.ReadOnlySpan {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		spans := recorder.Ended()
		if len(spans) >= want {
			return spans
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for %d ended spans", want)
	return nil
}
