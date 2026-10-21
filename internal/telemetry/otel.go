package telemetry

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type TracerProviderShutdownFunc func(context.Context) error

// FlushTimeout bounds how long sending the spans may delay Task's exit: a
// collector that is down or slow costs every run at most this.
const FlushTimeout = 2 * time.Second

var (
	providerMu sync.Mutex
	provider   *trace.TracerProvider // set by Initialize when enabled
)

// DefaultServiceName is the spans' service when OTEL_SERVICE_NAME is unset.
const DefaultServiceName = "task"

// Enabled reports whether spans are exported: OTEL_TRACES_EXPORTER is
// "otlp", or it is unset and an OTLP endpoint is
// (OTEL_EXPORTER_OTLP_ENDPOINT, OTEL_EXPORTER_OTLP_TRACES_ENDPOINT). Spans
// go over OTLP/HTTP (protobuf) only: no other exporter, to keep Task's
// dependencies small.
func Enabled() bool {
	switch os.Getenv("OTEL_TRACES_EXPORTER") {
	case "otlp":
		return true
	case "":
		return os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" || os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") != ""
	}
	return false
}

// Initialize sets up tracing when Enabled: the global TracerProvider
// exporting spans over OTLP/HTTP per the OTEL_EXPORTER_OTLP_* environment
// (endpoint, headers, compression, certificate, timeout), and the W3C trace
// context propagator. OpenTelemetry's own errors (e.g. a failed export) go
// to onError, nil to drop them. The returned func flushes and shuts down,
// taking at most FlushTimeout.
func Initialize(ctx context.Context, onError func(error)) (TracerProviderShutdownFunc, error) {
	if !Enabled() {
		return func(context.Context) error { return nil }, nil
	}
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		if onError != nil {
			onError(err)
		}
	}))

	spanExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating span exporter: %w", err)
	}

	// The service is OTEL_SERVICE_NAME, else "task" (not the SDK's
	// "unknown_service:task").
	res, err := resource.Merge(
		resource.NewSchemaless(attribute.String("service.name", DefaultServiceName)),
		resource.Environment(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating resource: %w", err)
	}

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(spanExporter, trace.WithExportTimeout(FlushTimeout)),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	providerMu.Lock()
	provider = tracerProvider
	providerMu.Unlock()

	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), FlushTimeout)
		defer cancel()
		return tracerProvider.Shutdown(ctx)
	}, nil
}

// Flush sends the spans ended so far, taking at most FlushTimeout: for a
// path that exits the process without returning (e.g. watch mode's
// interrupt). Nothing when tracing isn't enabled.
func Flush() {
	providerMu.Lock()
	p := provider
	providerMu.Unlock()
	if p == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), FlushTimeout)
	defer cancel()
	_ = p.ForceFlush(ctx)
}

// envCarrier is the W3C trace context in environment variables
// (TRACEPARENT, TRACESTATE), as a TextMapCarrier; keys upper case.
type envCarrier map[string]string

func (c envCarrier) Get(key string) string { return c[strings.ToUpper(key)] }
func (c envCarrier) Set(key, value string) { c[strings.ToUpper(key)] = value }
func (c envCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// ParentFromEnv is ctx with the span of TRACEPARENT (and TRACESTATE) as its
// parent, when set: the process that ran Task (e.g. a wrapper recording the
// run, or a CI job) gets Task's spans in its trace.
func ParentFromEnv(ctx context.Context) context.Context {
	traceparent := os.Getenv("TRACEPARENT")
	if traceparent == "" {
		return ctx
	}
	carrier := envCarrier{"TRACEPARENT": traceparent, "TRACESTATE": os.Getenv("TRACESTATE")}
	return propagation.TraceContext{}.Extract(ctx, carrier)
}

// Environ is environ (nil: this process's) with TRACEPARENT (and
// TRACESTATE) set to the span of ctx, for a command Task runs: a program it
// starts that reads them (another task, any instrumented tool) puts its
// spans under the command's. environ as it is when ctx has no recording
// span.
func Environ(ctx context.Context, environ []string) []string {
	if !oteltrace.SpanContextFromContext(ctx).IsValid() {
		return environ
	}
	if environ == nil {
		environ = os.Environ()
	}
	carrier := envCarrier{}
	propagation.TraceContext{}.Inject(ctx, carrier)
	out := make([]string, 0, len(environ)+len(carrier))
	for _, kv := range environ {
		// Any case: Windows' environment keys are case-insensitive.
		if key, _, _ := strings.Cut(kv, "="); !strings.EqualFold(key, "TRACEPARENT") && !strings.EqualFold(key, "TRACESTATE") {
			out = append(out, kv)
		}
	}
	for k, v := range carrier {
		out = append(out, k+"="+v)
	}
	return out
}
