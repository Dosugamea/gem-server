# gem-server

PaymentRequest APIプロバイダーとして機能する仮想通貨管理サービス

## 概要

Webサイト・ゲーム内専用の仮想通貨を管理するマイクロサービス。**PaymentRequest APIのプロバイダー側（決済アプリ側）**として機能し、`https://yourdomain.com/pay` という決済方法を提供します。マーチャントサイトはこの決済方法を `supportedMethods` に指定することで、ユーザーが保有する仮想通貨を使ってデジタル商品（ゲーム内アイテム、コンテンツなど）を購入できるようになります。

## 主要機能

- **PaymentRequest APIプロバイダー**: `https://yourdomain.com/pay` という決済方法を提供
- **Payment Handler実装**: Service Workerと決済アプリウィンドウによる決済処理
- **通貨管理**: 有償通貨（Paid Currency）と無償通貨（Free Currency）の2種類を管理
- **決済処理**: ユーザーが保有する仮想通貨を使ってデジタル商品を購入する際の決済処理
- **付与**: 無償通貨の配布、その他の方法による通貨の付与
- **消費**: 保有している通貨の使用（決済時の消費を含む）
- **コード引き換え**: プロモーションコードやギフトコードを引き換えて通貨を加算
- **補填**: 問題があった際の補填処理
- **失効**: アカウントBANなどによる通貨の失効
- **返金**: 決済返金時の有償通貨回収
- **履歴管理**: 全取引履歴の記録と過去状態の遡及

## 技術スタック

- **バックエンド**: Golang (Echo Framework + gRPC)
- **データベース**: MySQL
- **Payment Handler**: Service Worker + 決済アプリウィンドウ（JavaScript）
- **認証**: JWT トークンベース認証
- **可観測性**: OpenTelemetry (トレーシング、メトリクス、ログ)
- **API仕様**: OpenAPI 3.0 (Swagger/Redoc)

## アーキテクチャ

本システムはドメイン駆動設計（DDD）とクリーンアーキテクチャの原則に基づいて設計されています。

### レイヤー構成

```
internal/
├── domain/                    # ドメイン層
│   ├── currency/             # 通貨ドメイン
│   ├── transaction/          # トランザクションドメイン
│   ├── payment_request/      # PaymentRequestドメイン
│   ├── redemption_code/      # コード引き換えドメイン
│   └── service/              # ドメインサービス
├── application/              # アプリケーション層
│   ├── currency/            # 通貨ユースケース
│   ├── payment/             # 決済ユースケース
│   └── history/             # 履歴ユースケース
├── infrastructure/          # インフラストラクチャ層
│   ├── persistence/        # 永続化
│   ├── observability/       # 可観測性
│   └── config/              # 設定管理
└── presentation/            # プレゼンテーション層
    ├── rest/                # REST API
    ├── grpc/                # gRPC API
    └── payment_handler/     # Payment Handler
```

## セットアップ

### 前提条件

- Go 1.21以上
- MySQL 8.0以上
- Redis（オプション）

### インストール

#### ローカル環境でのセットアップ

```bash
# 依存パッケージのインストール
go mod download

# ビルド
go build -o bin/gem-server ./cmd/server
```

#### Dockerを使用したセットアップ（推奨）

```bash
# Docker Composeを使用して開発環境を起動
docker-compose up -d

# ログを確認
docker-compose logs -f app

# 停止
docker-compose down
```

Docker Composeを使用すると、以下のサービスが自動的に起動します：
- **MySQL**: データベースサーバー（ポート3306）
- **Redis**: キャッシュサーバー（ポート6379、オプション）
- **Jaeger**: トレーシングUI（ポート16686、オプション）
- **アプリケーション**: REST APIサーバー（ポート8080）

マイグレーションは自動的に実行されます。

### 環境変数

`.env`ファイルを作成し、以下の環境変数を設定してください：

```env
# データベース設定
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=password
DB_NAME=gem_db

# JWT設定
JWT_SECRET=your-secret-key

# サーバー設定
SERVER_PORT=8080
```

## 開発

### ビルド

```bash
# Makefileを使用（WindowsではWSL2またはMake for Windowsが必要）
make build

# または直接goコマンドを使用
go build -o bin/gem-server ./cmd/server

# Dockerイメージをビルド
docker build -t gem-server:latest .
```

### テスト

```bash
# 全テスト実行
go test ./...

# カバレッジ付きテスト
go test -cover ./...
```

### 実行

```bash
# 開発サーバー起動
go run ./cmd/server

# Docker Composeで実行
docker-compose up

# Dockerイメージから実行
docker run -p 8080:8080 \
  -e DB_HOST=mysql \
  -e DB_USER=gem_user \
  -e DB_PASSWORD=gem_password \
  -e DB_NAME=gem_db \
  -e JWT_SECRET=your-secret-key \
  gem-server:latest
```

## Docker

### Docker Compose

開発環境を簡単にセットアップするには、`docker-compose.yml`を使用します：

```bash
# 全サービスを起動
docker-compose up -d

# 特定のサービスのみ起動
docker-compose up -d mysql redis

# ログを確認
docker-compose logs -f app

# サービスを停止
docker-compose down

# ボリュームも含めて削除
docker-compose down -v
```

### 環境変数の設定

Docker Composeを使用する場合、環境変数は`docker-compose.yml`で設定されています。本番環境では、`.env`ファイルを作成するか、環境変数を直接設定してください：

```bash
# .envファイルを作成
cp .env.example .env
# .envファイルを編集して必要な値を設定

# 環境変数を指定して起動
docker-compose --env-file .env up -d
```

### カスタムDockerイメージ

```bash
# イメージをビルド
docker build -t gem-server:latest .

# イメージを実行
docker run -p 8080:8080 \
  --env-file .env \
  gem-server:latest
```

## CI/CD

GitHub Actionsを使用したCI/CDパイプラインが設定されています（`.github/workflows/ci.yml`）。

### ワークフロー

1. **テスト**: コードのリント、フォーマットチェック、ユニットテストの実行
2. **ビルド**: アプリケーションのビルドと成果物のアップロード
3. **Dockerビルド**: Dockerイメージのビルドとプッシュ（mainブランチのみ）
4. **デプロイ**: 本番環境へのデプロイ（mainブランチへのpush時のみ）

### 必要なシークレット

GitHubリポジトリに以下のシークレットを設定してください：

- `DOCKER_USERNAME`: Docker Hubのユーザー名
- `DOCKER_PASSWORD`: Docker Hubのパスワード
- `DOCKER_REGISTRY`: DockerレジストリのURL（例: `your-registry.io`）

## ドキュメント

詳細なドキュメントは以下のディレクトリを参照してください：

### 開発者向けドキュメント

- [セットアップガイド](docs/SETUP.md) - 開発環境のセットアップ手順
- [開発ガイドライン](docs/DEVELOPMENT.md) - コーディング規約と開発フロー
- [API使用例](docs/api-examples.md) - APIの使用例とサンプルコード

### 運用ドキュメント

- [デプロイ手順書](docs/DEPLOYMENT.md) - 本番環境へのデプロイ手順
- [監視・アラート設定ガイド](docs/MONITORING.md) - 監視とアラートの設定
- [トラブルシューティングガイド](docs/TROUBLESHOOTING.md) - よくある問題と解決方法

### 設計ドキュメント

- `.cursor/plans/` - 実装計画書と設計ドキュメント
  - [システム概要](.cursor/plans/doc-01-システム概要.md)
  - [アーキテクチャ](.cursor/plans/doc-02-アーキテクチャ.md)
  - [データベース設計](.cursor/plans/doc-03-データベース設計.md)
  - [API仕様](.cursor/plans/doc-04-API仕様.md)
  - [PaymentRequest API実装](.cursor/plans/doc-05-PaymentRequest-API-プロバイダー実装.md)

### APIドキュメント

- **Swagger UI**: http://localhost:8080/swagger
- **ReDoc**: http://localhost:8080/redoc
- **OpenAPI仕様**: http://localhost:8080/openapi.yaml

## ライセンス

[LICENSE](LICENSE) を参照してください。
