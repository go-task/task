package telemetry

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel/sdk/trace"
)

type TracerProviderShutdownFunc func(context.Context) error

func Initialize(ctx context.Context) (TracerProviderShutdownFunc, error) {
	if os.Getenv("OTEL_TRACES_EXPORTER") == "" {
		return func(context.Context) error { return nil }, nil
	}

	spanExporter, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating span exporter: %w", err)
	}

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(spanExporter),
	)
	otel.SetTracerProvider(tracerProvider)

	return tracerProvider.Shutdown, nil
}
