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

```bash
# 依存パッケージのインストール
go mod download

# ビルド
go build -o bin/gem-server cmd/main.go
```

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
go build -o bin/gem-server cmd/main.go
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
go run cmd/main.go
```

## ドキュメント

詳細なドキュメントは以下のディレクトリを参照してください：

- `.cursor/plans/` - 実装計画書と設計ドキュメント
- `docs/` - API仕様書など

## ライセンス

[LICENSE](LICENSE) を参照してください。
