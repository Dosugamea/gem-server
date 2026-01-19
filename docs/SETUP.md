# セットアップガイド

本ドキュメントでは、開発環境のセットアップ手順を説明します。

## 目次

- [前提条件](#前提条件)
- [ローカル環境でのセットアップ](#ローカル環境でのセットアップ)
- [Dockerを使用したセットアップ](#dockerを使用したセットアップ)
- [環境変数の設定](#環境変数の設定)
- [データベースマイグレーション](#データベースマイグレーション)
- [動作確認](#動作確認)

## 前提条件

以下のソフトウェアがインストールされている必要があります:

- **Go**: 1.21以上
- **MySQL**: 8.0以上（ローカル環境の場合）
- **Git**: 最新版
- **Docker & Docker Compose**: 最新版（Dockerを使用する場合）

### Goのインストール確認

```powershell
go version
```

### MySQLのインストール確認（ローカル環境の場合）

```powershell
mysql --version
```

### Dockerのインストール確認

```powershell
docker --version
docker-compose --version
```

## ローカル環境でのセットアップ

### 1. リポジトリのクローン

```powershell
git clone https://github.com/your-org/gem-server.git
cd gem-server
```

### 2. 依存パッケージのインストール

```powershell
go mod download
```

### 3. 環境変数の設定

`.env`ファイルを作成し、必要な環境変数を設定します:

```powershell
cp .env.example .env
# .envファイルを編集
```

詳細は[環境変数の設定](#環境変数の設定)を参照してください。

### 4. データベースのセットアップ

MySQLデータベースを作成します:

```powershell
mysql -u root -p
```

```sql
CREATE DATABASE gem_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'gem_user'@'localhost' IDENTIFIED BY 'gem_password';
GRANT ALL PRIVILEGES ON gem_db.* TO 'gem_user'@'localhost';
FLUSH PRIVILEGES;
EXIT;
```

### 5. マイグレーションの実行

```powershell
# PowerShellスクリプトを使用
.\scripts\migrate.ps1

# または直接実行
migrate -path ./migrations -database "mysql://gem_user:gem_password@tcp(localhost:3306)/gem_db?multiStatements=true" up
```

### 6. アプリケーションのビルド

```powershell
go build -o bin/gem-server.exe ./cmd/server
```

### 7. アプリケーションの起動

```powershell
.\bin\gem-server.exe
```

または、直接実行:

```powershell
go run ./cmd/server
```

## Dockerを使用したセットアップ（推奨）

Docker Composeを使用すると、すべての依存サービス（MySQL、Redis、Jaegerなど）が自動的にセットアップされます。

### 1. リポジトリのクローン

```powershell
git clone https://github.com/your-org/gem-server.git
cd gem-server
```

### 2. 環境変数の設定（オプション）

本番環境やカスタム設定が必要な場合のみ、`.env`ファイルを作成します:

```powershell
cp .env.example .env
# .envファイルを編集
```

### 3. Docker Composeで起動

```powershell
docker-compose up -d
```

これにより、以下のサービスが起動します:

- **MySQL**: ポート3306
- **Redis**: ポート6379（オプション）
- **Jaeger**: ポート16686（トレーシングUI）
- **アプリケーション**: ポート8080

### 4. ログの確認

```powershell
# すべてのサービスのログ
docker-compose logs -f

# アプリケーションのログのみ
docker-compose logs -f app

# MySQLのログのみ
docker-compose logs -f mysql
```

### 5. サービスの停止

```powershell
# サービスを停止（データは保持）
docker-compose stop

# サービスを停止してボリュームも削除
docker-compose down -v
```

### 6. サービスの再起動

```powershell
docker-compose restart
```

## 環境変数の設定

### 必須環境変数

| 変数名 | 説明 | デフォルト値 | 例 |
|--------|------|--------------|-----|
| `DB_HOST` | MySQLホスト | `localhost` | `mysql` (Docker) |
| `DB_PORT` | MySQLポート | `3306` | `3306` |
| `DB_USER` | MySQLユーザー名 | `gem_user` | `gem_user` |
| `DB_PASSWORD` | MySQLパスワード | - | `gem_password` |
| `DB_NAME` | データベース名 | `gem_db` | `gem_db` |
| `JWT_SECRET` | JWT署名用シークレット | - | `your-secret-key` |

### オプション環境変数

| 変数名 | 説明 | デフォルト値 | 例 |
|--------|------|--------------|-----|
| `ENVIRONMENT` | 環境（development/staging/production） | `development` | `production` |
| `SERVER_PORT` | サーバーポート | `8080` | `8080` |
| `JWT_EXPIRATION` | JWT有効期限 | `24h` | `24h` |
| `REDIS_HOST` | Redisホスト | `localhost` | `redis` |
| `REDIS_PORT` | Redisポート | `6379` | `6379` |
| `REDIS_ENABLED` | Redis有効化 | `false` | `true` |
| `OTEL_ENABLED` | OpenTelemetry有効化 | `true` | `true` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLPエンドポイント | `http://jaeger:4318` | `http://jaeger:4318` |

### .envファイルの例

```env
# サーバー設定
ENVIRONMENT=development
SERVER_PORT=8080

# データベース設定
DB_HOST=localhost
DB_PORT=3306
DB_USER=gem_user
DB_PASSWORD=gem_password
DB_NAME=gem_db
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5

# JWT設定
JWT_SECRET=your-secret-key-change-in-production
JWT_EXPIRATION=24h
JWT_ISSUER=gem-server

# Redis設定（オプション）
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_ENABLED=false

# OpenTelemetry設定
OTEL_ENABLED=true
OTEL_SERVICE_NAME=gem-server
OTEL_SERVICE_VERSION=1.0.0
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
OTEL_EXPORTER_OTLP_INSECURE=true
```

## データベースマイグレーション

### マイグレーションの実行

```powershell
# PowerShellスクリプトを使用（推奨）
.\scripts\migrate.ps1

# または直接実行
migrate -path ./migrations -database "mysql://gem_user:gem_password@tcp(localhost:3306)/gem_db?multiStatements=true" up
```

### マイグレーションのロールバック

```powershell
migrate -path ./migrations -database "mysql://gem_user:gem_password@tcp(localhost:3306)/gem_db?multiStatements=true" down
```

### マイグレーションの状態確認

```powershell
migrate -path ./migrations -database "mysql://gem_user:gem_password@tcp(localhost:3306)/gem_db?multiStatements=true" version
```

### 新しいマイグレーションファイルの作成

```powershell
migrate create -ext sql -dir ./migrations -seq migration_name
```

これにより、`migrations/000002_migration_name.up.sql` と `migrations/000002_migration_name.down.sql` が作成されます。

## 動作確認

### 1. ヘルスチェック

```powershell
curl http://localhost:8080/health
```

**期待されるレスポンス:**

```json
{
  "status": "ok"
}
```

### 2. APIドキュメントの確認

ブラウザで以下のURLにアクセス:

- **Swagger UI**: http://localhost:8080/swagger
- **ReDoc**: http://localhost:8080/redoc
- **OpenAPI仕様**: http://localhost:8080/openapi.yaml

### 3. 認証トークンの取得

```powershell
curl -X POST http://localhost:8080/api/v1/auth/token `
  -H "Content-Type: application/json" `
  -d '{\"user_id\": \"test_user\"}'
```

### 4. 通貨残高の取得

```powershell
# トークンを取得（上記のコマンドで取得）
$token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

curl -X GET http://localhost:8080/api/v1/users/test_user/balance `
  -H "Authorization: Bearer $token"
```

## トラブルシューティング

### MySQL接続エラー

**エラー:** `dial tcp: lookup mysql: no such host`

**解決策:**
- Docker Composeを使用している場合、サービス名が正しいか確認
- ローカル環境の場合、`DB_HOST=localhost` を確認

### マイグレーションエラー

**エラー:** `Error: migration failed`

**解決策:**
- データベースが存在するか確認
- ユーザーに適切な権限があるか確認
- 既存のマイグレーション状態を確認: `migrate version`

### ポートが既に使用されている

**エラー:** `bind: address already in use`

**解決策:**
- 既存のプロセスを停止
- または、`SERVER_PORT` 環境変数で別のポートを指定

### Docker Composeでサービスが起動しない

**解決策:**
- ログを確認: `docker-compose logs`
- ボリュームをクリーンアップ: `docker-compose down -v`
- 再起動: `docker-compose up -d`

## 次のステップ

セットアップが完了したら、以下のドキュメントを参照してください:

- [開発ガイドライン](DEVELOPMENT.md)
- [API使用例](api-examples.md)
- [アーキテクチャドキュメント](../.cursor/plans/doc-02-アーキテクチャ.md)
