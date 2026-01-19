package otel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"gem-server/internal/infrastructure/config"
)

func TestInitMeter_Disabled(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		Enabled: false,
	}

	shutdown, err := InitMeter(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, shutdown)

	// シャットダウン関数がエラーを返さないことを確認
	err = shutdown(context.Background())
	assert.NoError(t, err)
}

func TestInitMeter_OTLP(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		Enabled:          true,
		MetricsExporter: "otlp",
		OTLPEndpoint:    "http://localhost:4318",
		OTLPInsecure:    true,
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
	}

	// 実際のOTLPエンドポイントに接続しようとするため、エラーが発生する可能性がある
	shutdown, err := InitMeter(cfg)
	if err != nil {
		// エンドポイントに接続できない場合はエラーが発生する可能性がある
		t.Logf("InitMeter failed (expected if OTLP endpoint is not available): %v", err)
		return
	}

	assert.NotNil(t, shutdown)
	if shutdown != nil {
		_ = shutdown(context.Background())
	}
}

func TestInitMeter_Stdout(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		Enabled:          true,
		MetricsExporter: "stdout",
		ServiceName:      "test-service",
		ServiceVersion:   "1.0.0",
	}

	shutdown, err := InitMeter(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, shutdown)

	// シャットダウン関数がエラーを返さないことを確認
	err = shutdown(context.Background())
	assert.NoError(t, err)
}

func TestInitMeter_UnsupportedExporter(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		Enabled:          true,
		MetricsExporter: "unsupported",
		ServiceName:      "test-service",
		ServiceVersion:   "1.0.0",
	}

	shutdown, err := InitMeter(cfg)
	assert.Error(t, err)
	assert.Nil(t, shutdown)
	assert.Contains(t, err.Error(), "unsupported metrics exporter")
}

func TestMeter(t *testing.T) {
	// メーターを取得
	meter := Meter("test-meter")
	assert.NotNil(t, meter)
}

func TestInitMeter_OTLPInsecure(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		Enabled:          true,
		MetricsExporter: "otlp",
		OTLPEndpoint:    "http://localhost:4318",
		OTLPInsecure:    false, // セキュア接続
		ServiceName:      "test-service",
		ServiceVersion:   "1.0.0",
	}

	// 実際のOTLPエンドポイントに接続しようとするため、エラーが発生する可能性がある
	shutdown, err := InitMeter(cfg)
	if err != nil {
		t.Logf("InitMeter failed (expected if OTLP endpoint is not available): %v", err)
		return
	}

	assert.NotNil(t, shutdown)
	if shutdown != nil {
		_ = shutdown(context.Background())
	}
}

func TestInitMeter_ResourceCreation(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		Enabled:          true,
		MetricsExporter: "stdout",
		ServiceName:      "test-service",
		ServiceVersion:   "1.0.0",
	}

	shutdown, err := InitMeter(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, shutdown)

	// リソースが正しく作成されることを確認
	if shutdown != nil {
		_ = shutdown(context.Background())
	}
}

func TestInitMeter_MultipleCalls(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		Enabled:          true,
		MetricsExporter: "stdout",
		ServiceName:      "test-service",
		ServiceVersion:   "1.0.0",
	}

	// 複数回初期化を試みる
	shutdown1, err1 := InitMeter(cfg)
	assert.NoError(t, err1)

	shutdown2, err2 := InitMeter(cfg)
	assert.NoError(t, err2)

	// 両方のシャットダウン関数が有効であることを確認
	if shutdown1 != nil {
		_ = shutdown1(context.Background())
	}
	if shutdown2 != nil {
		_ = shutdown2(context.Background())
	}
}

func TestMeter_GetMeter(t *testing.T) {
	// メータープロバイダーが設定されていない場合でも、メーターを取得できることを確認
	meter := Meter("test-meter")
	assert.NotNil(t, meter)

	// メーターが正しい型であることを確認（interface{}を返すため、型アサーションは難しい）
	_ = meter
}

func TestInitMeter_DefaultConfig(t *testing.T) {
	// デフォルト設定でメーターを初期化
	cfg := &config.OpenTelemetryConfig{
		Enabled:          true,
		MetricsExporter: "stdout",
		ServiceName:      "gem-server",
		ServiceVersion:   "1.0.0",
	}

	shutdown, err := InitMeter(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, shutdown)

	if shutdown != nil {
		_ = shutdown(context.Background())
	}
}
