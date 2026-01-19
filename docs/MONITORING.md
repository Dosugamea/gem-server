# 監視・アラート設定ガイド

本ドキュメントでは、システムの監視とアラート設定について説明します。

## 目次

- [監視アーキテクチャ](#監視アーキテクチャ)
- [メトリクス](#メトリクス)
- [ログ](#ログ)
- [トレーシング](#トレーシング)
- [アラート設定](#アラート設定)
- [ダッシュボード](#ダッシュボード)

## 監視アーキテクチャ

本システムは、OpenTelemetryを使用して分散トレーシング、メトリクス、ログを統合的に管理します。

```
┌─────────────┐
│  gem-server │
│  (アプリ)   │
└──────┬──────┘
       │
       ├─── トレース ───> Jaeger
       ├─── メトリクス ──> Prometheus
       └─── ログ ──────> Loki / ELK
```

## メトリクス

### システムメトリクス

#### HTTPリクエストメトリクス

- `http_requests_total`: リクエスト総数（ラベル: method, path, status）
- `http_request_duration_seconds`: リクエスト処理時間（ヒストグラム）
- `http_request_size_bytes`: リクエストサイズ
- `http_response_size_bytes`: レスポンスサイズ

#### データベースメトリクス

- `db_connections_active`: アクティブな接続数
- `db_connections_idle`: アイドル接続数
- `db_query_duration_seconds`: クエリ実行時間
- `db_query_total`: クエリ総数

### ビジネスメトリクス

#### 通貨関連メトリクス

- `currency_transactions_total`: トランザクション総数（ラベル: type, currency_type）
- `currency_balance`: 通貨残高（ゲージ、ラベル: user_id, currency_type）
- `currency_grant_total`: 通貨付与総数
- `currency_consume_total`: 通貨消費総数
- `currency_negative_balance_count`: マイナス残高の発生件数

#### 決済関連メトリクス

- `payment_requests_total`: 決済リクエスト総数
- `payment_success_total`: 決済成功数
- `payment_failure_total`: 決済失敗数
- `payment_amount_total`: 決済金額合計

#### コード引き換えメトリクス

- `redemption_code_redeemed_total`: コード引き換え総数
- `redemption_code_failed_total`: コード引き換え失敗数

### Prometheus設定例

`prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'gem-server'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

## ログ

### ログレベル

- **DEBUG**: デバッグ情報（開発環境のみ）
- **INFO**: 一般的な情報（リクエスト処理、ビジネスイベント）
- **WARN**: 警告（非致命的な問題）
- **ERROR**: エラー（致命的な問題）

### 構造化ログ

すべてのログはJSON形式で出力されます:

```json
{
  "timestamp": "2024-01-01T00:00:00Z",
  "level": "INFO",
  "message": "currency granted",
  "trace_id": "abc123...",
  "span_id": "def456...",
  "user_id": "user123",
  "currency_type": "free",
  "amount": "100"
}
```

### ログ収集

#### Loki設定例

`loki-config.yml`:

```yaml
server:
  http_listen_port: 3100

ingester:
  lifecycler:
    address: 127.0.0.1
    ring:
      kvstore:
        store: inmemory
      replication_factor: 1

schema_config:
  configs:
    - from: 2024-01-01
      store: boltdb
      object_store: filesystem
      schema: v11
      index:
        prefix: index_
        period: 168h

storage_config:
  boltdb:
    directory: /loki/index
  filesystem:
    directory: /loki/chunks

limits_config:
  enforce_metric_name: false
  reject_old_samples: true
  reject_old_samples_max_age: 168h
```

#### Docker ComposeでのLoki統合

```yaml
services:
  loki:
    image: grafana/loki:latest
    ports:
      - "3100:3100"
    volumes:
      - ./loki-config.yml:/etc/loki/local-config.yaml
    command: -config.file=/etc/loki/local-config.yaml

  promtail:
    image: grafana/promtail:latest
    volumes:
      - ./promtail-config.yml:/etc/promtail/config.yml
      - /var/log:/var/log:ro
    command: -config.file=/etc/promtail/config.yml
```

## トレーシング

### Jaeger設定

Jaegerは、OpenTelemetry Collector経由でトレースを受信します。

#### OpenTelemetry Collector設定

`otel-collector-config.yml`:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 1s
    send_batch_size: 1024

exporters:
  jaeger:
    endpoint: jaeger:14250
    tls:
      insecure: true

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [jaeger]
```

### トレースの確認

1. Jaeger UIにアクセス: http://localhost:16686
2. サービス名 `gem-server` を選択
3. トレースを検索・確認

## アラート設定

### Prometheusアラートルール

`alerts.yml`:

```yaml
groups:
  - name: gem-server
    interval: 30s
    rules:
      # エラー率が高い
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "エラー率が高いです"
          description: "5分間のエラー率が5%を超えています"

      # レスポンス時間が遅い
      - alert: HighResponseTime
        expr: histogram_quantile(0.95, http_request_duration_seconds_bucket) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "レスポンス時間が遅いです"
          description: "95パーセンタイルのレスポンス時間が1秒を超えています"

      # データベース接続プールが枯渇
      - alert: DatabaseConnectionPoolExhausted
        expr: db_connections_active / db_connections_max > 0.9
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "データベース接続プールが枯渇しています"
          description: "アクティブな接続数が最大接続数の90%を超えています"

      # マイナス残高の発生
      - alert: NegativeBalanceDetected
        expr: increase(currency_negative_balance_count[1h]) > 0
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "マイナス残高が検出されました"
          description: "過去1時間でマイナス残高が発生しました"

      # 決済失敗率が高い
      - alert: HighPaymentFailureRate
        expr: rate(payment_failure_total[5m]) / rate(payment_requests_total[5m]) > 0.1
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "決済失敗率が高いです"
          description: "5分間の決済失敗率が10%を超えています"
```

### Alertmanager設定

`alertmanager.yml`:

```yaml
global:
  resolve_timeout: 5m

route:
  group_by: ['alertname', 'severity']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h
  receiver: 'default'
  routes:
    - match:
        severity: critical
      receiver: 'critical-alerts'
    - match:
        severity: warning
      receiver: 'warning-alerts'

receivers:
  - name: 'default'
    webhook_configs:
      - url: 'http://webhook-receiver:5001/alerts'

  - name: 'critical-alerts'
    email_configs:
      - to: 'ops-team@example.com'
        from: 'alerts@example.com'
        smarthost: 'smtp.example.com:587'
        auth_username: 'alerts@example.com'
        auth_password: 'password'
    webhook_configs:
      - url: 'http://pagerduty-webhook:5001/alerts'

  - name: 'warning-alerts'
    webhook_configs:
      - url: 'http://slack-webhook:5001/alerts'
```

## ダッシュボード

### Grafanaダッシュボード

#### システムダッシュボード

**HTTPメトリクスパネル:**

- リクエスト数（時系列）
- エラー率（時系列）
- レスポンス時間（p50, p95, p99）
- リクエスト/レスポンスサイズ

**データベースメトリクスパネル:**

- 接続プールの状態
- クエリ実行時間
- クエリ総数

#### ビジネスダッシュボード

**通貨メトリクスパネル:**

- トランザクション数（時系列、タイプ別）
- 通貨残高の分布
- マイナス残高の発生件数
- 付与/消費の推移

**決済メトリクスパネル:**

- 決済リクエスト数
- 決済成功率
- 決済金額合計
- 決済失敗の内訳

### Grafanaダッシュボードのインポート

1. Grafana UIにアクセス: http://localhost:3000
2. 「+」→「Import」を選択
3. ダッシュボードJSONを貼り付け、またはファイルをアップロード
4. データソース（Prometheus）を選択
5. 「Import」をクリック

## 監視チェックリスト

### 日常的な監視項目

- [ ] エラー率が閾値以下か
- [ ] レスポンス時間が正常範囲内か
- [ ] データベース接続プールが正常か
- [ ] マイナス残高が発生していないか
- [ ] 決済成功率が正常か

### 週次レビュー項目

- [ ] メトリクスのトレンドを確認
- [ ] アラートの発生頻度を確認
- [ ] ダッシュボードを更新
- [ ] ログの保持期間を確認

### 月次レビュー項目

- [ ] 監視設定の見直し
- [ ] アラートルールの最適化
- [ ] ダッシュボードの改善
- [ ] 監視コストの確認

## トラブルシューティング

監視に関する問題が発生した場合、[トラブルシューティングガイド](TROUBLESHOOTING.md)を参照してください。
