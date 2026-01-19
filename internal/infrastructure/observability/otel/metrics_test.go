package otel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric/noop"
)

func TestNewMetrics(t *testing.T) {
	// Noopメータープロバイダーを使用
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := NewMetrics("test-meter")
	require.NoError(t, err)
	assert.NotNil(t, metrics)

	assert.NotNil(t, metrics.TransactionCount)
	assert.NotNil(t, metrics.CurrencyBalance)
	assert.NotNil(t, metrics.NegativeBalanceCount)
	assert.NotNil(t, metrics.RequestCount)
	assert.NotNil(t, metrics.ResponseTime)
	assert.NotNil(t, metrics.ErrorCount)
}

func TestMetrics_RecordTransaction(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := NewMetrics("test-meter")
	require.NoError(t, err)

	ctx := context.Background()

	// トランザクションを記録
	metrics.RecordTransaction(ctx, "grant", "paid")

	// エラーが発生しないことを確認
}

func TestMetrics_RecordCurrencyBalance(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := NewMetrics("test-meter")
	require.NoError(t, err)

	ctx := context.Background()

	// 通貨残高を記録
	metrics.RecordCurrencyBalance(ctx, "user123", "paid", 1000)

	// エラーが発生しないことを確認
}

func TestMetrics_RecordNegativeBalance(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := NewMetrics("test-meter")
	require.NoError(t, err)

	ctx := context.Background()

	// マイナス残高を記録
	metrics.RecordNegativeBalance(ctx, "user123", "paid")

	// エラーが発生しないことを確認
}

func TestMetrics_RecordRequest(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := NewMetrics("test-meter")
	require.NoError(t, err)

	ctx := context.Background()

	// リクエストを記録
	metrics.RecordRequest(ctx, "GET", "/api/v1/balance")

	// エラーが発生しないことを確認
}

func TestMetrics_RecordResponseTime(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := NewMetrics("test-meter")
	require.NoError(t, err)

	ctx := context.Background()

	// レスポンス時間を記録
	metrics.RecordResponseTime(ctx, "GET", "/api/v1/balance", 0.123)

	// エラーが発生しないことを確認
}

func TestMetrics_RecordError(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := NewMetrics("test-meter")
	require.NoError(t, err)

	ctx := context.Background()

	// エラーを記録
	metrics.RecordError(ctx, "database_error")

	// エラーが発生しないことを確認
}

func TestMetrics_RecordTransactionWithDifferentTypes(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := NewMetrics("test-meter")
	require.NoError(t, err)

	ctx := context.Background()

	// 異なるトランザクションタイプを記録
	metrics.RecordTransaction(ctx, "grant", "paid")
	metrics.RecordTransaction(ctx, "consume", "free")
	metrics.RecordTransaction(ctx, "consume", "paid")

	// エラーが発生しないことを確認
}

func TestMetrics_RecordCurrencyBalanceWithDifferentUsers(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := NewMetrics("test-meter")
	require.NoError(t, err)

	ctx := context.Background()

	// 異なるユーザーの残高を記録
	metrics.RecordCurrencyBalance(ctx, "user1", "paid", 1000)
	metrics.RecordCurrencyBalance(ctx, "user2", "free", 500)
	metrics.RecordCurrencyBalance(ctx, "user1", "free", 2000)

	// エラーが発生しないことを確認
}

func TestMetrics_RecordRequestWithDifferentMethods(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := NewMetrics("test-meter")
	require.NoError(t, err)

	ctx := context.Background()

	// 異なるHTTPメソッドを記録
	metrics.RecordRequest(ctx, "GET", "/api/v1/balance")
	metrics.RecordRequest(ctx, "POST", "/api/v1/grant")
	metrics.RecordRequest(ctx, "PUT", "/api/v1/consume")

	// エラーが発生しないことを確認
}

func TestMetrics_RecordResponseTimeWithDifferentPaths(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := NewMetrics("test-meter")
	require.NoError(t, err)

	ctx := context.Background()

	// 異なるパスとレスポンス時間を記録
	metrics.RecordResponseTime(ctx, "GET", "/api/v1/balance", 0.05)
	metrics.RecordResponseTime(ctx, "POST", "/api/v1/grant", 0.15)
	metrics.RecordResponseTime(ctx, "PUT", "/api/v1/consume", 0.25)

	// エラーが発生しないことを確認
}

func TestMetrics_RecordErrorWithDifferentTypes(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := NewMetrics("test-meter")
	require.NoError(t, err)

	ctx := context.Background()

	// 異なるエラータイプを記録
	metrics.RecordError(ctx, "database_error")
	metrics.RecordError(ctx, "validation_error")
	metrics.RecordError(ctx, "not_found_error")

	// エラーが発生しないことを確認
}

func TestMetrics_MultipleCalls(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := NewMetrics("test-meter")
	require.NoError(t, err)

	ctx := context.Background()

	// 複数回メトリクスを記録
	for i := 0; i < 10; i++ {
		metrics.RecordTransaction(ctx, "grant", "paid")
		metrics.RecordCurrencyBalance(ctx, "user123", "paid", int64(100*i))
		metrics.RecordRequest(ctx, "GET", "/api/v1/balance")
		metrics.RecordResponseTime(ctx, "GET", "/api/v1/balance", 0.1)
	}

	// エラーが発生しないことを確認
}

func TestNewMetrics_ErrorHandling(t *testing.T) {
	// メータープロバイダーが設定されていない場合でも、エラーが発生しないことを確認
	// （実際にはnoopメータープロバイダーが使用される）
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := NewMetrics("test-meter")
	require.NoError(t, err)
	assert.NotNil(t, metrics)
}
