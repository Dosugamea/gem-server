# API役割分担の明確化と修正計画（ユーザーAPI/管理API分離）

## 現状の問題点

1. **権限の分離が不適切**

   - ユーザーが自分自身に対して通貨を付与（grant）できる → セキュリティ上の問題
   - ユーザーが自分自身の通貨を消費（consume）できる → 用途が不明確
   - ユーザーAPIと管理APIが混在している

2. **REST APIの設計が不明確**

   - ユーザーIDをパスパラメータで受け取っているが、公開APIとしてはトークンから取得すべき
   - `/auth/token`が誰でも叩ける（認証不要）→ 公開APIとして問題
   - エンドポイントが `/users/{user_id}/...` 形式で、ユーザー自身のリソースにアクセスする設計になっていない

3. **gRPC APIの認証方式が不適切**

   - JWTベースの認証になっているが、マイクロサービス間ならAPIキーの方が適切
   - マイクロサービス向けであることがドキュメントで明確化されていない

4. **役割分担が不明確**

   - REST APIとgRPC APIの使い分けがドキュメントで明確化されていない

## 修正方針

### 1. REST APIの分離（ユーザーAPI / 管理API）

#### 1.1 ユーザーAPI（公開API）- `/api/v1/me/*`

ブラウザから直接叩く公開API。ユーザー自身のリソースにのみアクセス可能。

**エンドポイント:**

- `GET /api/v1/me/balance` - 自分の残高を取得
- `GET /api/v1/me/transactions` - 自分のトランザクション履歴を取得
- `POST /api/v1/codes/redeem` - コードを引き換え（自分のアカウントに付与）
- `POST /api/v1/payment/process` - 決済処理（自分のアカウントから消費）

**注意:** `consume`はユーザーAPIには存在しない。ユーザーは`/payment/process`を通じて間接的に消費する。

**認証:**

- JWTトークン（Bearer認証）
- トークンから取得した`user_id`を自動的に使用
- パスパラメータやリクエストボディに`user_id`は不要

#### 1.2 管理API（内部API）- REST `/api/v1/admin/*` と gRPC `CurrencyService`

管理者や他のマイクロサービス（ゲームサーバーなど）が使用する内部API。

**RESTとgRPCの両方で同じ機能を提供し、どちらを使っても同じことができる。**

**REST API エンドポイント:**

- `POST /api/v1/admin/users/{user_id}/grant` - ユーザーに通貨を付与
- `POST /api/v1/admin/users/{user_id}/consume` - ユーザーの通貨を消費（ゲームサーバーなどがアイテム購入時に呼び出す）
- `GET /api/v1/admin/users/{user_id}/balance` - ユーザーの残高を取得
- `GET /api/v1/admin/users/{user_id}/transactions` - ユーザーのトランザクション履歴を取得

**gRPC API メソッド（既存の`CurrencyService`）:**

- `rpc Grant(GrantRequest) returns (GrantResponse)` - ユーザーに通貨を付与
- `rpc Consume(ConsumeRequest) returns (ConsumeResponse)` - ユーザーの通貨を消費
- `rpc GetBalance(GetBalanceRequest) returns (GetBalanceResponse)` - ユーザーの残高を取得
- `rpc GetTransactionHistory(GetTransactionHistoryRequest) returns (GetTransactionHistoryResponse)` - ユーザーのトランザクション履歴を取得

**機能の対応関係:**

| 機能 | REST API | gRPC API |

|------|----------|----------|

| 通貨付与 | `POST /api/v1/admin/users/{user_id}/grant` | `Grant` |

| 通貨消費 | `POST /api/v1/admin/users/{user_id}/consume` | `Consume` |

| 残高取得 | `GET /api/v1/admin/users/{user_id}/balance` | `GetBalance` |

| 履歴取得 | `GET /api/v1/admin/users/{user_id}/transactions` | `GetTransactionHistory` |

**認証:**

- REST API: APIキー認証（`X-API-Key`ヘッダー）
- gRPC API: APIキー認証（`X-API-Key`メタデータ）
- サービス間認証として使用
- 管理者権限の確認（必要に応じて）

#### 1.3 認証エンドポイント

- `/api/v1/admin/users/{user_id}/issue_token` - 管理APIとして、APIキー認証が必要
- 外部認証サービスと連携する想定で、ドキュメントに明記
- 開発環境用の簡易認証として位置づけ、本番環境では無効化可能にする

### 2. gRPC APIの修正（管理API/マイクロサービス向けに明確化）

#### 2.1 認証方式の変更

- JWT認証を削除
- APIキー認証のみ（`X-API-Key`メタデータ）

#### 2.2 インターセプターの修正

- 既存のJWT認証インターセプターを削除
- APIキー認証用のインターセプターを追加（必須）

#### 2.3 ドキュメントの追加

- gRPC APIが管理API/マイクロサービス向けであることを明記
- REST管理APIとgRPC APIが同じ機能を提供することを明記
- 認証方式の選択肢をドキュメント化
- RESTとgRPCの使い分けガイドラインを追加

### 3. ミドルウェアの追加

#### 3.1 ユーザーAPI用ミドルウェア

- JWT認証ミドルウェア（既存）
- トークンから`user_id`を取得してコンテキストに設定

#### 3.2 管理API用ミドルウェア

- APIキー認証ミドルウェア（新規）
- 管理者権限チェックミドルウェア（必要に応じて）

### 4. ドキュメントの更新

#### 4.1 Swagger/OpenAPI仕様の更新

- ユーザーAPIと管理APIを別々のタグで分類
- エンドポイントパスの変更を反映
- 認証フローの説明を追加
- 公開APIとしての用途を明記

#### 4.2 README/API仕様書の更新

- REST API（ユーザーAPI/管理API）とgRPC APIの役割分担を明確化
- **管理APIはRESTとgRPCの両方で同じ機能を提供することを明記**
- REST管理APIとgRPC APIの機能対応表を追加
- 認証方式の違いを説明
- RESTとgRPCの使い分けガイドラインを追加
- 使用例を追加（RESTとgRPCの両方）

## 実装ファイル

### REST API関連

- `internal/presentation/rest/router.go` - ルーティングの変更（ユーザーAPI/管理APIの分離）
- `internal/presentation/rest/middleware/api_key.go` - APIキー認証ミドルウェアの追加（新規）
- `internal/presentation/rest/handler/currency_handler.go` - ユーザーAPI用ハンドラーの修正
- `internal/presentation/rest/handler/admin_currency_handler.go` - 管理API用ハンドラーの追加（新規、または既存ハンドラーを再利用）
- `internal/presentation/rest/handler/payment_handler.go` - `user_id`の削除
- `internal/presentation/rest/handler/history_handler.go` - エンドポイントの変更
- `internal/presentation/rest/handler/code_redemption_handler.go` - `user_id`の削除
- `docs/swagger.yaml` - Swagger仕様の更新
- `internal/presentation/openapi/spec.yaml` - OpenAPI仕様の更新

### gRPC API関連

- `internal/presentation/grpc/interceptor/auth.go` - JWT認証インターセプターを削除
- `internal/presentation/grpc/interceptor/api_key.go` - APIキー認証インターセプターの追加（新規、必須）
- `internal/presentation/grpc/server.go` - APIキー認証インターセプターのみを使用するように変更
- `internal/infrastructure/config/config.go` - gRPC認証設定と管理API設定の追加（JWT設定を削除）
- `internal/presentation/grpc/proto/currency.proto` - ドキュメントコメントの更新（管理API向けであることを明記）

### ドキュメント

- `README.md` - API役割分担の説明を追加
- `docs/api-examples.md` - 使用例の更新
- `.cursor/plans/doc-04-API仕様.md` - API仕様書の更新

## 実装の詳細

### ユーザーAPIエンドポイントの変更

**変更前:**

- `GET /api/v1/users/{user_id}/balance`
- `POST /api/v1/users/{user_id}/grant` ← 削除（管理APIに移動）
- `POST /api/v1/users/{user_id}/consume` ← 削除（管理APIに移動）
- `GET /api/v1/users/{user_id}/transactions`
- `POST /payment/process` (リクエストボディに`user_id`が必要)
- `POST /codes/redeem` (リクエストボディに`user_id`が必要)

**変更後（ユーザーAPI）:**

- `GET /api/v1/me/balance`
- `GET /api/v1/me/transactions`
- `POST /api/v1/payment/process` (リクエストボディから`user_id`を削除)
- `POST /api/v1/codes/redeem` (リクエストボディから`user_id`を削除)

**変更後（管理API - REST）:**

- `POST /api/v1/admin/users/{user_id}/grant`
- `POST /api/v1/admin/users/{user_id}/consume`
- `GET /api/v1/admin/users/{user_id}/balance`
- `GET /api/v1/admin/users/{user_id}/transactions`

**変更後（管理API - gRPC）:**

既存の`CurrencyService`の以下のメソッドを使用（認証方式をAPIキーのみに変更、JWT認証は削除）:

- `Grant` - 通貨付与
- `Consume` - 通貨消費
- `GetBalance` - 残高取得
- `GetTransactionHistory` - 履歴取得

**注意:** REST管理APIとgRPC APIは同じ機能を提供し、どちらを使っても同じことができる。gRPC APIはAPIキー認証のみを使用する。

### 認証フローの実装

#### ユーザーAPI認証フロー

```
1. ユーザーが外部認証サービスで認証
2. 外部認証サービスからJWTトークンを取得
3. ユーザーAPIを呼び出す際にBearerトークンとして送信
4. ミドルウェアがトークンを検証し、user_idをコンテキストに設定
5. ハンドラーがコンテキストからuser_idを取得して使用
```

#### 管理API認証フロー（REST）

```
1. マイクロサービスまたは管理者がAPIキーを設定
2. REST管理APIを呼び出す際にX-API-Keyヘッダーとして送信
3. ミドルウェアがAPIキーを検証
4. 必要に応じて管理者権限をチェック
5. ハンドラーがパスパラメータからuser_idを取得して使用
```

#### 管理API認証フロー（gRPC）

```
1. マイクロサービスまたは管理者がAPIキーを設定
2. gRPC APIを呼び出す際にX-API-Keyメタデータとして送信（必須）
3. インターセプターがAPIキーを検証（JWT認証は使用しない）
4. 必要に応じて管理者権限をチェック
5. ハンドラーがリクエストからuser_idを取得して使用
```

### 設定構造の追加

```go
type AdminAPIConfig struct {
    Enabled  bool
    APIKey   string
    AllowedIPs []string // オプション: IP制限
}

type AdminAPIConfig struct {
    Enabled  bool
    APIKey   string
    AllowedIPs []string // オプション: IP制限
}

// gRPC APIはAPIキー認証のみ（設定構造は不要、直接APIKeyを使用）
```

## 注意事項

1. **セキュリティ**

   - ユーザーAPI: 公開APIとして適切なレート制限やCORS設定
   - 管理API（REST/gRPC）: 内部ネットワークからのみアクセス可能にする（IP制限等）
   - APIキーの管理方法を明確化
   - REST管理APIとgRPC APIで同じAPIキーを使用可能にする

3. **テストの更新**

   - すべてのハンドラーのテストを更新
   - エンドポイント変更に対応した統合テストを追加
   - 認証ミドルウェアのテストを追加

4. **フロントエンドへの影響**

   - `public/pay/payment-app.js`のAPI呼び出しを更新
   - エンドポイントパスの変更に対応

5. **REST管理APIとgRPC APIの機能統一**

   - 両方のAPIが同じアプリケーションサービス層を使用することを確認
   - レスポンス形式の違い（JSON vs protobuf）をドキュメント化
   - エラーハンドリングの統一
   - テストで両方のAPIが同じ動作をすることを確認