# トラブルシューティングガイド

本ドキュメントでは、よくある問題とその解決方法を説明します。

## 目次

- [接続エラー](#接続エラー)
- [認証エラー](#認証エラー)
- [データベースエラー](#データベースエラー)
- [パフォーマンス問題](#パフォーマンス問題)
- [デプロイメントエラー](#デプロイメントエラー)
- [ログとデバッグ](#ログとデバッグ)

## 接続エラー

### MySQL接続エラー

**エラー:** `dial tcp: lookup mysql: no such host`

**原因:**
- ホスト名が正しくない
- ネットワーク設定の問題
- Docker Composeのサービス名が間違っている

**解決策:**

1. ホスト名を確認:
   ```powershell
   # Docker Composeの場合
   DB_HOST=mysql
   
   # ローカル環境の場合
   DB_HOST=localhost
   ```

2. ネットワーク接続を確認:
   ```powershell
   # Docker Composeの場合
   docker-compose ps
   docker network ls
   ```

3. 接続テスト:
   ```powershell
   # Docker Composeの場合
   docker-compose exec mysql mysql -u gem_user -p gem_db
   
   # ローカル環境の場合
   mysql -u gem_user -p gem_db
   ```

**エラー:** `Access denied for user`

**原因:**
- ユーザー名またはパスワードが間違っている
- ユーザーに適切な権限がない

**解決策:**

1. 認証情報を確認:
   ```powershell
   # .envファイルを確認
   cat .env | grep DB_
   ```

2. ユーザーの権限を確認:
   ```sql
   SHOW GRANTS FOR 'gem_user'@'%';
   ```

3. ユーザーを再作成:
   ```sql
   DROP USER IF EXISTS 'gem_user'@'%';
   CREATE USER 'gem_user'@'%' IDENTIFIED BY 'gem_password';
   GRANT ALL PRIVILEGES ON gem_db.* TO 'gem_user'@'%';
   FLUSH PRIVILEGES;
   ```

### Redis接続エラー

**エラー:** `dial tcp: lookup redis: no such host`

**解決策:**

1. Redisが起動しているか確認:
   ```powershell
   docker-compose ps redis
   ```

2. 接続テスト:
   ```powershell
   docker-compose exec redis redis-cli ping
   # 期待される応答: PONG
   ```

3. `REDIS_ENABLED=false` に設定してRedisを無効化（Redisが不要な場合）

## 認証エラー

### JWTトークンが無効

**エラー:** `401 Unauthorized`

**原因:**
- トークンが期限切れ
- トークンの署名が無効
- トークンが正しく送信されていない

**解決策:**

1. トークンを再取得:
   ```powershell
   curl -X POST http://localhost:8080/api/v1/auth/token `
     -H "Content-Type: application/json" `
     -d '{\"user_id\": \"user123\"}'
   ```

2. ヘッダーが正しく設定されているか確認:
   ```powershell
   # 正しい形式
   Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
   
   # 誤り: Bearerの前にスペースがない、またはBearerが欠けている
   Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
   ```

3. JWT_SECRETが一致しているか確認:
   ```powershell
   # 環境変数を確認
   echo $env:JWT_SECRET
   ```

### ユーザーIDの検証エラー

**エラー:** `user_id mismatch`

**原因:**
- トークン内のユーザーIDとリクエストパスのユーザーIDが一致しない

**解決策:**

1. トークン内のユーザーIDを確認（JWTデコードツールを使用）
2. リクエストパスのユーザーIDと一致しているか確認

## データベースエラー

### 楽観的ロックエラー

**エラー:** `optimistic lock conflict`

**原因:**
- 同時更新による競合
- リトライロジックが機能していない

**解決策:**

1. リトライロジックが実装されているか確認
2. リトライ回数を増やす（必要に応じて）
3. ログを確認して競合の原因を特定

### トランザクションエラー

**エラー:** `transaction rollback`

**原因:**
- データベースの制約違反
- デッドロック
- 接続タイムアウト

**解決策:**

1. エラーログを確認:
   ```powershell
   docker logs gem-server | grep -i error
   ```

2. データベースのログを確認:
   ```powershell
   docker-compose logs mysql | grep -i error
   ```

3. トランザクションのタイムアウト設定を確認:
   ```env
   DB_CONN_MAX_LIFETIME=5m
   ```

### マイグレーションエラー

**エラー:** `migration failed`

**解決策:**

1. マイグレーションの状態を確認:
   ```powershell
   migrate -path ./migrations `
     -database "mysql://gem_user:gem_password@tcp(localhost:3306)/gem_db?multiStatements=true" `
     version
   ```

2. マイグレーションファイルの構文を確認:
   ```sql
   -- マイグレーションファイルを直接実行してテスト
   mysql -u gem_user -p gem_db < migrations/000001_init_schema.up.sql
   ```

3. データベースのバックアップから復元（必要に応じて）

## パフォーマンス問題

### レスポンス時間が遅い

**原因:**
- データベースクエリが遅い
- N+1問題
- インデックスが不足している

**解決策:**

1. スロークエリログを確認:
   ```sql
   SET GLOBAL slow_query_log = 'ON';
   SET GLOBAL long_query_time = 1;
   ```

2. クエリの実行計画を確認:
   ```sql
   EXPLAIN SELECT * FROM currency_balances WHERE user_id = 'user123';
   ```

3. インデックスを追加:
   ```sql
   CREATE INDEX idx_user_currency ON currency_balances(user_id, currency_type);
   ```

4. プロファイリングツールを使用してボトルネックを特定

### メモリ使用量が高い

**原因:**
- メモリリーク
- 大きなデータセットの読み込み
- ゴルーチンのリーク

**解決策:**

1. メモリプロファイルを取得:
   ```powershell
   go tool pprof http://localhost:8080/debug/pprof/heap
   ```

2. ゴルーチンの数を確認:
   ```powershell
   curl http://localhost:8080/debug/pprof/goroutine?debug=1
   ```

3. データベース接続プールの設定を確認:
   ```env
   DB_MAX_OPEN_CONNS=25
   DB_MAX_IDLE_CONNS=5
   ```

### データベース接続プールの枯渇

**エラー:** `too many connections`

**解決策:**

1. 接続プールの設定を調整:
   ```env
   DB_MAX_OPEN_CONNS=50
   DB_MAX_IDLE_CONNS=10
   DB_CONN_MAX_LIFETIME=5m
   ```

2. データベースの最大接続数を確認:
   ```sql
   SHOW VARIABLES LIKE 'max_connections';
   ```

3. アイドル接続をタイムアウト:
   ```env
   DB_CONN_MAX_IDLE_TIME=10m
   ```

## デプロイメントエラー

### コンテナが起動しない

**エラー:** `container exited with code 1`

**解決策:**

1. ログを確認:
   ```powershell
   docker logs gem-server
   ```

2. 環境変数を確認:
   ```powershell
   docker-compose config
   ```

3. ヘルスチェックを確認:
   ```powershell
   docker inspect gem-server | grep -A 10 Health
   ```

### マイグレーションが失敗する

**解決策:**

1. マイグレーションを手動で実行してエラーを確認
2. データベースのバックアップから復元
3. マイグレーションファイルの構文を確認

### ポートが既に使用されている

**エラー:** `bind: address already in use`

**解決策:**

1. 使用中のポートを確認:
   ```powershell
   netstat -ano | findstr :8080
   ```

2. プロセスを終了:
   ```powershell
   taskkill /PID <PID> /F
   ```

3. 別のポートを使用:
   ```env
   SERVER_PORT=8081
   ```

## ログとデバッグ

### ログの確認

```powershell
# Docker Composeの場合
docker-compose logs -f app

# 特定の時間範囲のログ
docker-compose logs --since 1h app

# エラーログのみ
docker-compose logs app | grep -i error

# Kubernetesの場合
kubectl logs -f deployment/gem-server
```

### デバッグモードの有効化

```env
ENVIRONMENT=development
LOG_LEVEL=DEBUG
```

### トレーシングの確認

1. Jaeger UIにアクセス: http://localhost:16686
2. サービス名 `gem-server` を選択
3. エラーが発生しているトレースを確認
4. スパンの詳細を確認して問題を特定

### プロファイリング

```powershell
# CPUプロファイル
go tool pprof http://localhost:8080/debug/pprof/profile

# メモリプロファイル
go tool pprof http://localhost:8080/debug/pprof/heap

# ゴルーチンプロファイル
go tool pprof http://localhost:8080/debug/pprof/goroutine
```

## よくある問題のクイックリファレンス

| 問題 | 原因 | 解決策 |
|------|------|--------|
| MySQL接続エラー | ホスト名が間違っている | `DB_HOST`を確認 |
| JWT認証エラー | トークンが期限切れ | トークンを再取得 |
| 楽観的ロックエラー | 同時更新 | リトライロジックを確認 |
| レスポンスが遅い | インデックス不足 | インデックスを追加 |
| 接続プール枯渇 | 接続数が不足 | 接続プール設定を調整 |
| コンテナが起動しない | 環境変数エラー | ログを確認 |

## サポート

問題が解決しない場合:

1. ログとエラーメッセージを収集
2. 再現手順を記録
3. 環境情報を記録（OS、Goバージョン、Dockerバージョンなど）
4. 問題を報告（GitHub Issuesなど）
