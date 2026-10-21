package task_test

import (
	"bytes"
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-task/task/v3"
)

var (
	otelOnce     sync.Once
	otelRecorder *tracetest.SpanRecorder
)

// otelRun runs the task name of testdata/otel under a root span of its own
// (the trace a wrapper or CI job would pass in TRACEPARENT), and returns the
// root span's id, the spans of its trace by name and the run's error.
// Spans are recorded by a global provider every test shares, told apart by
// trace.
func otelRun(t *testing.T, name string) (trace.SpanID, map[string]sdktrace.ReadOnlySpan, error) {
	t.Helper()
	otelOnce.Do(func() {
		otelRecorder = tracetest.NewSpanRecorder()
		otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(otelRecorder)))
	})
	var buf bytes.Buffer
	e := task.NewExecutor(task.WithDir("testdata/otel"), task.WithStdout(&buf), task.WithStderr(&buf), task.WithSilent(true))
	require.NoError(t, e.Setup())
	ctx, root := otel.Tracer("test").Start(context.Background(), "wrapper")
	err := e.Run(ctx, &task.Call{Task: name})
	root.End()
	spans := map[string]sdktrace.ReadOnlySpan{}
	for _, s := range otelRecorder.Ended() {
		if s.SpanContext().TraceID() == root.SpanContext().TraceID() {
			spans[s.Name()] = s
		}
	}
	return root.SpanContext().SpanID(), spans, err
}

func otelAttr(s sdktrace.ReadOnlySpan, key string) attribute.Value {
	for _, a := range s.Attributes() {
		if string(a.Key) == key {
			return a.Value
		}
	}
	return attribute.Value{}
}

func TestOtelTaskJoinsTheCallersTrace(t *testing.T) {
	t.Parallel()
	root, spans, err := otelRun(t, "ok")
	require.NoError(t, err)
	require.Contains(t, spans, "ok")
	assert.Equal(t, root, spans["ok"].Parent().SpanID(), "the task's span is a child of the caller's")
	assert.Equal(t, "executed", otelAttr(spans["ok"], "task.outcome").AsString())
	require.Contains(t, spans, "shell[0]")
	assert.Equal(t, spans["ok"].SpanContext().SpanID(), spans["shell[0]"].Parent().SpanID())
}

func TestOtelFailedTask(t *testing.T) {
	t.Parallel()
	_, spans, err := otelRun(t, "fail")
	require.Error(t, err)
	s := spans["fail"]
	require.NotNil(t, s)
	assert.Equal(t, "failed", otelAttr(s, "task.outcome").AsString())
	assert.Equal(t, int64(3), otelAttr(s, "process.exit.code").AsInt64())
	assert.Equal(t, codes.Error, s.Status().Code)
}

// A dep's failure fails its parent, but the exit code stays the dep's.
func TestOtelFailedDep(t *testing.T) {
	t.Parallel()
	_, spans, err := otelRun(t, "dep-fails")
	require.Error(t, err)
	assert.Equal(t, "failed", otelAttr(spans["dep-fails"], "task.outcome").AsString())
	assert.False(t, otelAttr(spans["dep-fails"], "process.exit.code").Type() == attribute.INT64, "no exit code of its own")
	assert.Equal(t, int64(3), otelAttr(spans["fail"], "process.exit.code").AsInt64())
}

func TestOtelTimeout(t *testing.T) {
	t.Parallel()
	_, spans, err := otelRun(t, "timeout")
	require.Error(t, err)
	assert.Equal(t, "failed", otelAttr(spans["timeout"], "task.outcome").AsString())
	assert.Equal(t, int64(124), otelAttr(spans["timeout"], "process.exit.code").AsInt64())
}

func TestOtelIgnoredError(t *testing.T) {
	t.Parallel()
	_, spans, err := otelRun(t, "ignored")
	require.NoError(t, err)
	assert.Equal(t, "executed", otelAttr(spans["ignored"], "task.outcome").AsString())
	assert.Equal(t, "true", otelAttr(spans["shell[0]"], "command.error_ignored").AsString())
}

// A deferred command's span is in the task's trace, under the task.
func TestOtelDeferredInTheTasksTrace(t *testing.T) {
	t.Parallel()
	_, spans, err := otelRun(t, "deferred")
	require.NoError(t, err)
	var deferred sdktrace.ReadOnlySpan
	for _, s := range spans {
		if otelAttr(s, "command.defer").AsBool() {
			deferred = s
		}
	}
	require.NotNil(t, deferred, "the deferred command's span, in the run's trace")
	assert.Equal(t, spans["deferred"].SpanContext().SpanID(), deferred.Parent().SpanID())
}

// A run: once task called twice runs once; the second call's span says so.
func TestOtelRunOnceDeduplicated(t *testing.T) {
	t.Parallel()
	_, _, err := otelRun(t, "twice")
	require.NoError(t, err)
	var outcomes []string
	for _, s := range otelRecorder.Ended() {
		if s.Name() == "once" && otelAttr(s, "task.name").AsString() == "once" {
			outcomes = append(outcomes, otelAttr(s, "task.outcome").AsString()+"/"+otelAttr(s, "task.skip_reason").AsString())
		}
	}
	assert.Contains(t, outcomes, "executed/")
	assert.Contains(t, outcomes, "skipped/deduplicated")
}

func TestOtelSecretsMasked(t *testing.T) {
	t.Parallel()
	_, spans, err := otelRun(t, "secret")
	require.NoError(t, err)
	cmd := otelAttr(spans["shell[0]"], "command.shell").AsString()
	assert.NotContains(t, cmd, "hunter2")
	assert.Contains(t, cmd, "*****")
}
