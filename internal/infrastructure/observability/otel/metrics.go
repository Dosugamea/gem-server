package otel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics メトリクス定義
type Metrics struct {
	// トランザクション数
	TransactionCount metric.Int64Counter
	
	// 通貨残高の分布
	CurrencyBalance metric.Int64Gauge
	
	// マイナス残高の発生件数
	NegativeBalanceCount metric.Int64Counter
	
	// リクエスト数
	RequestCount metric.Int64Counter
	
	// レスポンス時間
	ResponseTime metric.Float64Histogram
	
	// エラー率
	ErrorCount metric.Int64Counter
}

// NewMetrics 新しいMetricsを作成
func NewMetrics(meterName string) (*Metrics, error) {
	meter := otel.Meter(meterName)
	
	transactionCount, err := meter.Int64Counter(
		"transactions_total",
		metric.WithDescription("Total number of transactions"),
	)
	if err != nil {
		return nil, err
	}
	
	currencyBalance, err := meter.Int64Gauge(
		"currency_balance",
		metric.WithDescription("Currency balance"),
	)
	if err != nil {
		return nil, err
	}
	
	negativeBalanceCount, err := meter.Int64Counter(
		"negative_balance_total",
		metric.WithDescription("Total number of negative balance occurrences"),
	)
	if err != nil {
		return nil, err
	}
	
	requestCount, err := meter.Int64Counter(
		"requests_total",
		metric.WithDescription("Total number of requests"),
	)
	if err != nil {
		return nil, err
	}
	
	responseTime, err := meter.Float64Histogram(
		"response_time_seconds",
		metric.WithDescription("Response time in seconds"),
	)
	if err != nil {
		return nil, err
	}
	
	errorCount, err := meter.Int64Counter(
		"errors_total",
		metric.WithDescription("Total number of errors"),
	)
	if err != nil {
		return nil, err
	}
	
	return &Metrics{
		TransactionCount:     transactionCount,
		CurrencyBalance:      currencyBalance,
		NegativeBalanceCount: negativeBalanceCount,
		RequestCount:         requestCount,
		ResponseTime:         responseTime,
		ErrorCount:           errorCount,
	}, nil
}

// RecordTransaction トランザクションを記録
func (m *Metrics) RecordTransaction(ctx context.Context, transactionType, currencyType string) {
	m.TransactionCount.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("transaction_type", transactionType),
			attribute.String("currency_type", currencyType),
		),
	)
}

// RecordCurrencyBalance 通貨残高を記録
func (m *Metrics) RecordCurrencyBalance(ctx context.Context, userID, currencyType string, balance int64) {
	m.CurrencyBalance.Record(ctx, balance,
		metric.WithAttributes(
			attribute.String("user_id", userID),
			attribute.String("currency_type", currencyType),
		),
	)
}

// RecordNegativeBalance マイナス残高の発生を記録
func (m *Metrics) RecordNegativeBalance(ctx context.Context, userID, currencyType string) {
	m.NegativeBalanceCount.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("user_id", userID),
			attribute.String("currency_type", currencyType),
		),
	)
}

// RecordRequest リクエストを記録
func (m *Metrics) RecordRequest(ctx context.Context, method, path string) {
	m.RequestCount.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("path", path),
		),
	)
}

// RecordResponseTime レスポンス時間を記録
func (m *Metrics) RecordResponseTime(ctx context.Context, method, path string, duration float64) {
	m.ResponseTime.Record(ctx, duration,
		metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("path", path),
		),
	)
}

// RecordError エラーを記録
func (m *Metrics) RecordError(ctx context.Context, errorType string) {
	m.ErrorCount.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("error_type", errorType),
		),
	)
}
