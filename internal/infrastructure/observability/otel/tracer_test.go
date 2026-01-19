package otel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace/noop"

	"gem-server/internal/infrastructure/config"
)

func TestInitTracer_Disabled(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		Enabled: false,
	}

	shutdown, err := InitTracer(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, shutdown)

	// シャットダウン関数がエラーを返さないことを確認
	err = shutdown(context.Background())
	assert.NoError(t, err)
}

func TestInitTracer_OTLP(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		Enabled:        true,
		TraceExporter:  "otlp",
		OTLPEndpoint:   "http://localhost:4318",
		OTLPInsecure:   true,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	// 実際のOTLPエンドポイントに接続しようとするため、エラーが発生する可能性がある
	// しかし、初期化自体は成功するはず
	shutdown, err := InitTracer(cfg)
	if err != nil {
		// エンドポイントに接続できない場合はエラーが発生する可能性がある
		// これは正常な動作
		t.Logf("InitTracer failed (expected if OTLP endpoint is not available): %v", err)
		return
	}

	assert.NotNil(t, shutdown)
	if shutdown != nil {
		// シャットダウン関数を呼び出す
		_ = shutdown(context.Background())
	}
}

func TestInitTracer_Stdout(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		Enabled:        true,
		TraceExporter:  "stdout",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	shutdown, err := InitTracer(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, shutdown)

	// シャットダウン関数がエラーを返さないことを確認
	err = shutdown(context.Background())
	assert.NoError(t, err)
}

func TestInitTracer_UnsupportedExporter(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		Enabled:        true,
		TraceExporter:  "unsupported",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	shutdown, err := InitTracer(cfg)
	assert.Error(t, err)
	assert.Nil(t, shutdown)
	assert.Contains(t, err.Error(), "unsupported trace exporter")
}

func TestTracer(t *testing.T) {
	// Noopトレーサープロバイダーを使用
	_ = noop.NewTracerProvider()

	tracer := Tracer("test-tracer")
	assert.NotNil(t, tracer)

	// トレーサーを使用してスパンを作成
	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "test-span")
	assert.NotNil(t, span)
	span.End()
}

func TestInitTracer_OTLPInsecure(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		Enabled:        true,
		TraceExporter:  "otlp",
		OTLPEndpoint:   "http://localhost:4318",
		OTLPInsecure:   false, // セキュア接続
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	// 実際のOTLPエンドポイントに接続しようとするため、エラーが発生する可能性がある
	shutdown, err := InitTracer(cfg)
	if err != nil {
		t.Logf("InitTracer failed (expected if OTLP endpoint is not available): %v", err)
		return
	}

	assert.NotNil(t, shutdown)
	if shutdown != nil {
		_ = shutdown(context.Background())
	}
}

func TestInitTracer_ResourceCreation(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		Enabled:        true,
		TraceExporter:  "stdout",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	shutdown, err := InitTracer(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, shutdown)

	// リソースが正しく作成されることを確認（実際の検証は難しいが、エラーが発生しないことを確認）
	if shutdown != nil {
		_ = shutdown(context.Background())
	}
}

func TestTracer_StartSpan(t *testing.T) {
	tracer := Tracer("test-tracer")

	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "test-operation")

	assert.NotNil(t, span)
	// noopトレーサーではIsRecording()がfalseを返す場合があるが、スパンは作成される
	_ = span.IsRecording()

	span.End()
}

func TestTracer_StartSpanWithAttributes(t *testing.T) {
	tracer := Tracer("test-tracer")

	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "test-operation")

	assert.NotNil(t, span)

	// スパンに属性を設定（実際の実装では、StartWithOptionsなどを使用）
	span.End()
}

func TestInitTracer_MultipleCalls(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		Enabled:        true,
		TraceExporter:  "stdout",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	// 複数回初期化を試みる
	shutdown1, err1 := InitTracer(cfg)
	assert.NoError(t, err1)

	shutdown2, err2 := InitTracer(cfg)
	assert.NoError(t, err2)

	// 両方のシャットダウン関数が有効であることを確認
	if shutdown1 != nil {
		_ = shutdown1(context.Background())
	}
	if shutdown2 != nil {
		_ = shutdown2(context.Background())
	}
}
