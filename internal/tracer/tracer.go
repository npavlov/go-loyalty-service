package tracer

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"

	"github.com/npavlov/go-loyalty-service/internal/config"
)

type Tracer struct {
	cfg *config.Config
}

// NewTracer creates a new instance of the Tracer.
func NewTracer(cfg *config.Config) *Tracer {
	return &Tracer{cfg: cfg}
}

// InitTracer initializes OpenTelemetry tracing with Jaeger.
func (tr *Tracer) InitTracer(ctx context.Context) (*trace.TracerProvider, error) {
	if tr.cfg.Jaeger == "" {
		return nil, errors.New("missing Jaeger endpoint in config")
	}

	headers := map[string]string{
		"content-type": "application/json",
	}

	exporter, err := otlptrace.New(
		ctx,
		otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(tr.cfg.Jaeger),
			otlptracehttp.WithHeaders(headers),
			otlptracehttp.WithInsecure(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(
			exporter,
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
			trace.WithBatchTimeout(trace.DefaultScheduleDelay*time.Millisecond),
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
		),
		trace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String("go-loyalty-service"),
				semconv.DeploymentEnvironmentKey.String("production"),
			),
		),
	)

	otel.SetTracerProvider(tp)

	return tp, nil
}
