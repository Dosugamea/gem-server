# デプロイ手順書

本ドキュメントでは、本番環境へのデプロイ手順を説明します。

## 目次

- [前提条件](#前提条件)
- [デプロイ前の確認事項](#デプロイ前の確認事項)
- [Dockerを使用したデプロイ](#dockerを使用したデプロイ)
- [Kubernetesを使用したデプロイ](#kubernetesを使用したデプロイ)
- [ロールバック手順](#ロールバック手順)
- [デプロイ後の確認](#デプロイ後の確認)

## 前提条件

- Docker & Docker Compose がインストールされていること
- 本番環境のデータベースがセットアップされていること
- 環境変数が適切に設定されていること
- CI/CDパイプラインが設定されていること（オプション）

## デプロイ前の確認事項

### 1. コードの確認

- [ ] すべてのテストがパスしていること
- [ ] コードレビューが完了していること
- [ ] セキュリティチェックが完了していること

### 2. データベースマイグレーションの確認

- [ ] マイグレーションファイルが正しく作成されていること
- [ ] マイグレーションのロールバック手順が確認されていること
- [ ] データベースのバックアップが取得されていること

### 3. 環境変数の確認

- [ ] 本番環境用の環境変数が設定されていること
- [ ] シークレット（JWT_SECRETなど）が適切に管理されていること
- [ ] データベース接続情報が正しいこと

### 4. 依存サービスの確認

- [ ] MySQLが起動していること
- [ ] Redisが起動していること（使用する場合）
- [ ] Jaeger/OpenTelemetry Collectorが起動していること（使用する場合）

## Dockerを使用したデプロイ

### 1. Dockerイメージのビルド

```powershell
# タグを指定してビルド
docker build -t gem-server:v1.0.0 -t gem-server:latest .

# または、CI/CDパイプラインで自動ビルド
```

### 2. 環境変数の設定

本番環境用の `.env` ファイルを作成:

```env
ENVIRONMENT=production
SERVER_PORT=8080

DB_HOST=production-mysql.example.com
DB_PORT=3306
DB_USER=gem_user_prod
DB_PASSWORD=<secure-password>
DB_NAME=gem_db_prod

JWT_SECRET=<secure-jwt-secret>
JWT_EXPIRATION=24h

OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=https://jaeger.example.com:4318
```

### 3. データベースマイグレーションの実行

```powershell
# マイグレーションを実行
migrate -path ./migrations `
  -database "mysql://gem_user_prod:<password>@tcp(production-mysql.example.com:3306)/gem_db_prod?multiStatements=true" `
  up
```

### 4. アプリケーションの起動

```powershell
# Docker Composeを使用
docker-compose -f docker-compose.prod.yml up -d

# または、直接Dockerで実行
docker run -d `
  --name gem-server `
  --env-file .env.production `
  -p 8080:8080 `
  gem-server:v1.0.0
```

### 5. ヘルスチェック

```powershell
# ヘルスチェックエンドポイントを確認
curl http://localhost:8080/health

# ログを確認
docker logs gem-server
```

## Kubernetesを使用したデプロイ

### 1. Kubernetesマニフェストの作成

`k8s/deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gem-server
  labels:
    app: gem-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: gem-server
  template:
    metadata:
      labels:
        app: gem-server
    spec:
      containers:
      - name: gem-server
        image: gem-server:v1.0.0
        ports:
        - containerPort: 8080
        env:
        - name: DB_HOST
          valueFrom:
            secretKeyRef:
              name: gem-server-secrets
              key: db-host
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: gem-server-secrets
              key: db-password
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: gem-server-secrets
              key: jwt-secret
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: gem-server
spec:
  selector:
    app: gem-server
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

### 2. シークレットの作成

```powershell
kubectl create secret generic gem-server-secrets `
  --from-literal=db-host=production-mysql.example.com `
  --from-literal=db-password=<secure-password> `
  --from-literal=jwt-secret=<secure-jwt-secret>
```

### 3. デプロイの実行

```powershell
# デプロイメントを適用
kubectl apply -f k8s/deployment.yaml

# デプロイメントの状態を確認
kubectl get deployments
kubectl get pods

# ログを確認
kubectl logs -f deployment/gem-server
```

### 4. ロールアウトの確認

```powershell
# ロールアウトの状態を確認
kubectl rollout status deployment/gem-server

# ロールアウト履歴を確認
kubectl rollout history deployment/gem-server
```

## ロールバック手順

### Docker Composeの場合

```powershell
# 前のバージョンのイメージを使用
docker-compose -f docker-compose.prod.yml down
docker-compose -f docker-compose.prod.yml up -d --image gem-server:v0.9.0
```

### Kubernetesの場合

```powershell
# 前のリビジョンにロールバック
kubectl rollout undo deployment/gem-server

# 特定のリビジョンにロールバック
kubectl rollout undo deployment/gem-server --to-revision=2
```

### データベースマイグレーションのロールバック

```powershell
# マイグレーションを1つ戻す
migrate -path ./migrations `
  -database "mysql://gem_user_prod:<password>@tcp(production-mysql.example.com:3306)/gem_db_prod?multiStatements=true" `
  down 1

# すべてのマイグレーションをロールバック（注意: データが失われる可能性があります）
migrate -path ./migrations `
  -database "mysql://gem_user_prod:<password>@tcp(production-mysql.example.com:3306)/gem_db_prod?multiStatements=true" `
  down -all
```

## デプロイ後の確認

### 1. ヘルスチェック

```powershell
# ヘルスチェックエンドポイント
curl http://localhost:8080/health

# 期待されるレスポンス
# {"status":"ok"}
```

### 2. APIエンドポイントの確認

```powershell
# 認証トークンの取得
curl -X POST http://localhost:8080/api/v1/auth/token `
  -H "Content-Type: application/json" `
  -d '{\"user_id\": \"test_user\"}'

# 通貨残高の取得
curl -X GET http://localhost:8080/api/v1/users/test_user/balance `
  -H "Authorization: Bearer <token>"
```

### 3. ログの確認

```powershell
# Dockerの場合
docker logs gem-server

# Kubernetesの場合
kubectl logs -f deployment/gem-server
```

### 4. メトリクスの確認

- Prometheus/Grafanaでメトリクスを確認
- エラー率、レスポンス時間、リクエスト数を確認

### 5. トレーシングの確認

- Jaeger UIでトレースを確認
- リクエストフローが正常に記録されているか確認

## ブルー・グリーンデプロイメント

### 手順

1. **グリーン環境（新バージョン）をデプロイ**
   ```powershell
   docker-compose -f docker-compose.green.yml up -d
   ```

2. **グリーン環境の動作確認**
   - ヘルスチェック
   - スモークテスト
   - パフォーマンステスト

3. **トラフィックをグリーン環境に切り替え**
   - ロードバランサーの設定を変更
   - または、DNS設定を変更

4. **ブルー環境（旧バージョン）を停止**
   ```powershell
   docker-compose -f docker-compose.blue.yml down
   ```

## カナリアデプロイメント

### 手順

1. **カナリア環境（新バージョン）をデプロイ**
   - 少数のレプリカのみデプロイ

2. **トラフィックの一部をカナリア環境にルーティング**
   - 10%のトラフィックをカナリア環境に

3. **監視と確認**
   - エラー率、レスポンス時間を監視
   - 問題がなければ、トラフィックを段階的に増やす

4. **フルロールアウト**
   - 100%のトラフィックを新バージョンに
   - 旧バージョンを停止

## デプロイチェックリスト

デプロイ前に以下を確認してください:

- [ ] すべてのテストがパスしている
- [ ] コードレビューが完了している
- [ ] セキュリティチェックが完了している
- [ ] データベースのバックアップが取得されている
- [ ] マイグレーションスクリプトが確認されている
- [ ] 環境変数が正しく設定されている
- [ ] ログとメトリクスの設定が完了している
- [ ] ロールバック手順が確認されている
- [ ] デプロイ計画が関係者に共有されている

## トラブルシューティング

デプロイ時に問題が発生した場合、[トラブルシューティングガイド](TROUBLESHOOTING.md)を参照してください。
