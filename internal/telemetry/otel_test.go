package telemetry

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

func TestEnabled(t *testing.T) {
	for name, tc := range map[string]struct {
		env  map[string]string
		want bool
	}{
		"nothing set":            {nil, false},
		"otlp":                   {map[string]string{"OTEL_TRACES_EXPORTER": "otlp"}, true},
		"another exporter":       {map[string]string{"OTEL_TRACES_EXPORTER": "console", "OTEL_EXPORTER_OTLP_ENDPOINT": "http://x:4318"}, false},
		"none":                   {map[string]string{"OTEL_TRACES_EXPORTER": "none", "OTEL_EXPORTER_OTLP_ENDPOINT": "http://x:4318"}, false},
		"an endpoint":            {map[string]string{"OTEL_EXPORTER_OTLP_ENDPOINT": "http://x:4318"}, true},
		"a traces endpoint only": {map[string]string{"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": "http://x:4318/v1/traces"}, true},
	} {
		t.Run(name, func(t *testing.T) {
			for _, k := range []string{"OTEL_TRACES_EXPORTER", "OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"} {
				t.Setenv(k, tc.env[k])
			}
			assert.Equal(t, tc.want, Enabled())
		})
	}
}

const traceparent = "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"

func TestParentFromEnv(t *testing.T) {
	t.Setenv("TRACEPARENT", traceparent)
	sc := trace.SpanContextFromContext(ParentFromEnv(context.Background()))
	assert.Equal(t, "0af7651916cd43dd8448eb211c80319c", sc.TraceID().String())
	assert.Equal(t, "b7ad6b7169203331", sc.SpanID().String())
	assert.True(t, sc.IsRemote())

	t.Setenv("TRACEPARENT", "")
	assert.False(t, trace.SpanContextFromContext(ParentFromEnv(context.Background())).IsValid())
}

func TestEnviron(t *testing.T) {
	t.Setenv("TRACEPARENT", "")
	// No span: the environment as it is, nil (the process's) included.
	assert.Nil(t, Environ(context.Background(), nil))

	tid, _ := trace.TraceIDFromHex("0af7651916cd43dd8448eb211c80319c")
	sid, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{TraceID: tid, SpanID: sid, TraceFlags: trace.FlagsSampled}))

	got := Environ(ctx, []string{"FOO=bar", "TRACEPARENT=" + traceparent, "TraceParent=x", "tracestate=y"})
	assert.Contains(t, got, "FOO=bar")
	var traced []string
	for _, kv := range got {
		if key, _, _ := strings.Cut(kv, "="); strings.EqualFold(key, "TRACEPARENT") || strings.EqualFold(key, "TRACESTATE") {
			traced = append(traced, kv)
		}
	}
	assert.Equal(t, []string{"TRACEPARENT=00-0af7651916cd43dd8448eb211c80319c-00f067aa0ba902b7-01"}, traced, "the span's, once, whatever the old keys' case")

	// nil: the process's environment, with the span's TRACEPARENT.
	t.Setenv("FROM_PROCESS", "1")
	got = Environ(ctx, nil)
	assert.True(t, slices.Contains(got, "FROM_PROCESS=1"))
	assert.True(t, slices.Contains(got, "TRACEPARENT=00-0af7651916cd43dd8448eb211c80319c-00f067aa0ba902b7-01"))
}
