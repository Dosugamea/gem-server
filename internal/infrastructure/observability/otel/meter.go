package otel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

	"gem-server/internal/infrastructure/config"
)

// InitMeter メーターを初期化
func InitMeter(cfg *config.OpenTelemetryConfig) (func(context.Context) error, error) {
	if !cfg.Enabled {
		// OpenTelemetryが無効な場合は、Noopメーターを使用
		return func(context.Context) error { return nil }, nil
	}

	var exporter metric.Exporter
	var err error

	switch cfg.MetricsExporter {
	case "otlp":
		opts := []otlpmetrichttp.Option{
			otlpmetrichttp.WithEndpoint(cfg.OTLPEndpoint),
		}
		if cfg.OTLPInsecure {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		exporter, err = otlpmetrichttp.New(context.Background(), opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
		}
	case "stdout":
		// 標準出力に出力する場合は、簡易的な実装を使用
		return func(context.Context) error { return nil }, nil
	default:
		return nil, fmt.Errorf("unsupported metrics exporter: %s", cfg.MetricsExporter)
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter)),
		metric.WithResource(res),
	)

	otel.SetMeterProvider(mp)

	return mp.Shutdown, nil
}

// Meter メーターを取得
func Meter(name string) interface{} {
	return otel.Meter(name)
}
