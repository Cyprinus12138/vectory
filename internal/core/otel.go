package core

import (
	"github.com/Cyprinus12138/vectory/internal/utils/logger"
	"github.com/Cyprinus12138/vectory/pkg"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func (a *Vectory) initOTel() error {
	res, err := resource.New(a.ctx,
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(pkg.ClusterName),
			semconv.ServiceInstanceIDKey.String(a.id),
		),
	)
	// Create a http traceExporter.
	traceExporter, err := otlptracehttp.New(a.ctx)
	if err != nil {
		logger.Error("create http traceExporter failed", logger.Err(err))
		return err
	}
	// Create a prometheus metricExporter.
	metricExporter, err := prometheus.New()
	if err != nil {
		logger.Error("create prometheus metricExporter failed", logger.Err(err))
		return err
	}
	// Create a tracer provider with the traceExporter and a res describing this service
	a.traceProvider = trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(res),
	)
	// Create a metric provider with the traceExporter and a res describing this service
	a.meterProvider = metric.NewMeterProvider(
		metric.WithReader(metricExporter),
		metric.WithResource(res),
	)
	// Set the global tracer provider
	otel.SetTracerProvider(a.traceProvider)
	// Set the global metric provider
	otel.SetMeterProvider(a.meterProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return nil
}
