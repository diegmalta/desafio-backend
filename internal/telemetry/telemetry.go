package telemetry

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Init configures the global TracerProvider and text map propagators.
// If tracesExporter is "none" (case-insensitive), it returns a no-op shutdown and leaves the default noop provider.
func Init(ctx context.Context, serviceName, tracesExporter string) (shutdown func(context.Context) error, err error) {
	shutdown = func(context.Context) error { return nil }
	if strings.EqualFold(strings.TrimSpace(tracesExporter), "none") {
		return shutdown, nil
	}
	if serviceName == "" {
		serviceName = "desafio-backend"
	}
	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(serviceName)),
	)
	if err != nil {
		return nil, err
	}
	exp, err := stdouttrace.New()
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	return tp.Shutdown, nil
}
