---
name: タスク07 - テスト戦略・デプロイメント
overview: テスト戦略とデプロイメント、監視・可観測性を定義します
---

# タスク07: テスト戦略・デプロイメント

## 9. テスト戦略

### 9.1 単体テスト

- 各サービス関数のテスト
- データベース操作のモック

### 9.2 統合テスト

- APIエンドポイントのテスト
- データベース統合テスト

### 9.3 E2Eテスト

- PaymentRequest APIプロバイダー側のフローテスト
- 実際のService Workerとブラウザを使用したテスト
- マーチャントサイトからの決済リクエストのシミュレーション
- 決済アプリウィンドウの動作確認

## 10. デプロイメント

### 10.1 環境

- 開発環境
- ステージング環境
- 本番環境

### 10.2 監視と可観測性

#### 10.2.1 OpenTelemetry統合

OpenTelemetryを使用して、分散トレーシング、メトリクス、ログを統合的に管理します。

**トレーシング設定**

```go
// infrastructure/observability/otel/tracer.go
package otel

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/sdk/resource"
    "go.opentelemetry.io/otel/sdk/trace"
)

func InitTracer(serviceName string) (*trace.TracerProvider, error) {
    exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(
        jaeger.WithEndpoint("http://jaeger:14268/api/traces"),
    ))
    if err != nil {
        return nil, err
    }

    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String(serviceName),
        )),
    )
    
    otel.SetTracerProvider(tp)
    return tp, nil
}
```

**メトリクス設定**

```go
// infrastructure/observability/otel/meter.go
package otel

import (
    "go.opentelemetry.io/otel/exporters/prometheus"
    "go.opentelemetry.io/otel/sdk/metric"
)

func InitMeter() (*metric.MeterProvider, error) {
    exporter, err := prometheus.New()
    if err != nil {
        return nil, err
    }

    mp := metric.NewMeterProvider(
        metric.WithReader(exporter),
    )
    
    return mp, nil
}
```

**Echo Middlewareでのトレーシング**

```go
// infrastructure/observability/middleware/tracing.go
package middleware

import (
    "github.com/labstack/echo/v4"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/trace"
)

func TracingMiddleware() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            ctx := otel.GetTextMapPropagator().Extract(
                c.Request().Context(),
                propagation.HeaderCarrier(c.Request().Header),
            )
            
            span := trace.SpanFromContext(ctx)
            span.SetAttributes(
                attribute.String("http.method", c.Request().Method),
                attribute.String("http.url", c.Request().URL.String()),
            )
            
            c.SetRequest(c.Request().WithContext(ctx))
            
            err := next(c)
            
            span.SetAttributes(
                attribute.Int("http.status_code", c.Response().Status),
            )
            
            return err
        }
    }
}
```

**ドメイン層でのトレーシング**

```go
// application/currency/service.go
package currency

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
)

func (s *CurrencyApplicationService) Grant(ctx context.Context, req *GrantRequest) (*GrantResponse, error) {
    tracer := otel.Tracer("currency-service")
    ctx, span := tracer.Start(ctx, "CurrencyApplicationService.Grant")
    defer span.End()
    
    span.SetAttributes(
        attribute.String("user_id", req.UserID),
        attribute.String("currency_type", req.CurrencyType),
        attribute.String("amount", req.Amount),
    )
    
    // ビジネスロジック実行
    result, err := s.currencyDomainService.Grant(ctx, ...)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, err
    }
    
    span.SetAttributes(
        attribute.String("transaction_id", result.TransactionID),
    )
    
    return result, nil
}
```

**メトリクス収集**

```go
// infrastructure/observability/otel/metrics.go
package otel

import (
    "go.opentelemetry.io/otel/metric"
)

var (
    TransactionCounter metric.Int64Counter
    BalanceGauge       metric.Float64ObservableGauge
)

func InitMetrics(meter metric.Meter) error {
    var err error
    
    TransactionCounter, err = meter.Int64Counter(
        "currency_transactions_total",
        metric.WithDescription("Total number of currency transactions"),
    )
    if err != nil {
        return err
    }
    
    BalanceGauge, err = meter.Float64ObservableGauge(
        "currency_balance",
        metric.WithDescription("Current currency balance"),
    )
    
    return err
}
```

#### 10.2.2 監視項目

**トレーシング**

- HTTPリクエスト/レスポンスのトレース
- データベースクエリのトレース
- 外部API呼び出しのトレース
- ビジネスロジックのトレース

**メトリクス**

- リクエスト数（リクエスト/秒）
- レスポンス時間（p50, p95, p99）
- エラー率
- トランザクション数
- 通貨残高の分布
- マイナス残高の発生件数・ユーザー数（マイナス残高が発生した場合の監視）
- データベース接続プールの状態

**ログ**

- 構造化ログ（JSON形式）
- ログレベル（DEBUG, INFO, WARN, ERROR）
- トレースIDとの関連付け
- コンテキスト情報の付与

#### 10.2.3 監視ツール統合

- **Jaeger**: 分散トレーシングの可視化
- **Prometheus**: メトリクスの収集と保存
- **Grafana**: ダッシュボードの作成
- **ELK Stack / Loki**: ログの集約と分析
