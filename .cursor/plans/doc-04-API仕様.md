---
name: タスク04 - API仕様
overview: REST APIとgRPC APIの仕様を定義します
---

# タスク04: API仕様

## 4.1 OpenAPI仕様（Echo Framework）

Echo Frameworkを使用してREST APIを実装し、OpenAPI 3.0仕様に準拠したAPIドキュメントを提供します。

### 4.1.1 OpenAPI仕様ファイル構造

```yaml
# openapi/spec.yaml
openapi: 3.0.3
info:
  title: Virtual Currency Management API
  version: 1.0.0
  description: PaymentRequest API対応の仮想通貨管理API
servers:
  - url: https://api.yourdomain.com/v1
    description: Production server
paths:
  /users/{user_id}/balance:
    get:
      summary: 通貨残高取得
      operationId: getBalance
      parameters:
        - name: user_id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: 成功
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/BalanceResponse'
  /users/{user_id}/grant:
    post:
      summary: 通貨付与
      operationId: grantCurrency
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/GrantRequest'
      responses:
        '200':
          description: 成功
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/GrantResponse'
  /codes/redeem:
    post:
      summary: コード引き換え
      operationId: redeemCode
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/RedeemCodeRequest'
      responses:
        '200':
          description: 成功
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/RedeemCodeResponse'
        '400':
          description: 無効なコードまたは既に使用済み
        '404':
          description: コードが見つからない
components:
  schemas:
    BalanceResponse:
      type: object
      properties:
        user_id:
          type: string
        balances:
          type: object
          properties:
            paid:
              type: string
            free:
              type: string
    GrantRequest:
      type: object
      required:
        - currency_type
        - amount
      properties:
        currency_type:
          type: string
          enum: [paid, free]
        amount:
          type: string
        reason:
          type: string
        metadata:
          type: object
    RedeemCodeRequest:
      type: object
      required:
        - code
        - user_id
      properties:
        code:
          type: string
          description: 引き換えコード
        user_id:
          type: string
          description: ユーザーID
    RedeemCodeResponse:
      type: object
      properties:
        redemption_id:
          type: string
        transaction_id:
          type: string
        code:
          type: string
        currency_type:
          type: string
          enum: [paid, free]
        amount:
          type: string
        balance_after:
          type: string
        status:
          type: string
```

### 4.1.2 Echo Frameworkでの実装

```go
// presentation/rest/handler/currency_handler.go
package handler

import (
    "github.com/labstack/echo/v4"
    "github.com/getkin/kin-openapi/openapi3"
    "github.com/oapi-codegen/echo-middleware"
)

func SetupRoutes(e *echo.Echo, swagger *openapi3.T) {
    // OpenAPI仕様の読み込み
    e.Use(echomiddleware.OapiRequestValidator(swagger))
    
    // ルーティング
    api := e.Group("/api/v1")
    api.GET("/users/:user_id/balance", getBalance)
    api.POST("/users/:user_id/grant", grantCurrency)
    api.POST("/users/:user_id/consume", consumeCurrency)
    api.POST("/payment/process", processPayment)
    api.POST("/codes/redeem", redeemCode)
    api.GET("/users/:user_id/transactions", getTransactionHistory)
}

// OpenAPI仕様の自動生成（コード生成ツール使用）
// go generate で実行
//go:generate oapi-codegen -generate types,server -package openapi openapi/spec.yaml > openapi/generated.go
```

### 4.1.3 Swagger UI / ReDoc統合

```go
// presentation/rest/router.go
import (
    "github.com/swaggo/echo-swagger"
    "github.com/swaggo/swag"
)

func SetupSwagger(e *echo.Echo) {
    // Swagger UI
    e.GET("/swagger/*", echoSwagger.WrapHandler)
    
    // ReDoc
    e.Static("/redoc", "./docs/redoc")
    
    // OpenAPI仕様ファイルの配信
    e.GET("/openapi.yaml", func(c echo.Context) error {
        return c.File("./openapi/spec.yaml")
    })
    e.GET("/openapi.json", func(c echo.Context) error {
        return c.File("./openapi/spec.json")
    })
}
```

## 4.2 REST API エンドポイント

### 4.2.1 通貨残高取得

```
GET /api/v1/users/{user_id}/balance
Authorization: Bearer {token}

Response:
{
  "user_id": "user123",
  "balances": {
    "paid": "1000",
    "free": "500"
  }
}
```

### 4.2.2 通貨付与（無償）

```
POST /api/v1/users/{user_id}/grant
Authorization: Bearer {token}
Content-Type: application/json

Request:
{
  "currency_type": "free",
  "amount": "100.00",
  "reason": "イベント報酬",
  "metadata": {
    "event_id": "event_001"
  }
}

Response:
{
  "transaction_id": "txn_1234567890",
  "balance_after": "600.00",
  "status": "completed"
}
```

### 4.2.3 通貨消費

#### 4.2.3.1 指定通貨タイプでの消費

```
POST /api/v1/users/{user_id}/consume
Authorization: Bearer {token}
Content-Type: application/json

Request:
{
  "currency_type": "paid",
  "amount": "50",
  "item_id": "item_001",
  "metadata": {
    "purchase_id": "purchase_001"
  }
}

Response:
{
  "transaction_id": "txn_1234567891",
  "balance_after": "950",
  "status": "completed"
}
```

#### 4.2.3.2 優先順位付き消費（無料通貨優先）

無料通貨を優先して使用し、不足分を有料通貨で補う消費処理。

```
POST /api/v1/users/{user_id}/consume
Authorization: Bearer {token}
Content-Type: application/json

Request:
{
  "currency_type": "auto",  // "auto"を指定すると優先順位制御が有効
  "amount": "150",
  "item_id": "item_001",
  "use_priority": true,  // 優先順位制御を有効化
  "metadata": {
    "purchase_id": "purchase_001"
  }
}

Response:
{
  "transaction_id": "txn_1234567891",
  "consumption_details": [
    {
      "currency_type": "free",
      "amount": "100",  // 無料通貨から消費
      "balance_before": "100",
      "balance_after": "0"
    },
    {
      "currency_type": "paid",
      "amount": "50",   // 有料通貨から消費（不足分）
      "balance_before": "1000",
      "balance_after": "950"
    }
  ],
  "total_consumed": "150",
  "status": "completed"
}
```

**消費ロジック**:

1. 無料通貨の残高を確認
2. 無料通貨で支払える分を消費
3. 不足分があれば有料通貨から消費
4. 各通貨タイプごとにトランザクション履歴を記録

### 4.2.4 PaymentRequest処理（マーチャントから）

マーチャントサイトが決済を完了するために呼び出すAPIです。マーチャントは決済アプリウィンドウから返されたPaymentResponseを受け取り、このAPIを呼び出して決済処理を完了します。

**注意**: このAPIは決済金額をユーザーの仮想通貨残高から**消費**します。消費は**無料通貨を優先**し、不足分を有料通貨で補います。決済が完了すると、マーチャントサイトはユーザーにデジタル商品（ゲーム内アイテム、コンテンツなど）を提供します。

```
POST /api/v1/payment/process
Authorization: Bearer {token}
Content-Type: application/json

Request:
{
  "payment_request_id": "pr_1234567890",
  "user_id": "user123",
  "method_name": "https://yourdomain.com/pay",
  "details": {
    "userId": "user123",
    "transactionId": "txn_abc123",
    "timestamp": 1234567890
  },
  "amount": "1000",
  "currency": "JPY"
}

Response:
{
  "transaction_id": "txn_1234567892",
  "payment_request_id": "pr_1234567890",
  "consumption_details": [
    {
      "currency_type": "free",
      "amount": "500",
      "balance_before": "500",
      "balance_after": "0"
    },
    {
      "currency_type": "paid",
      "amount": "500",
      "balance_before": "1500",
      "balance_after": "1000"
    }
  ],
  "total_consumed": "1000",
  "status": "completed"
}
```

**注意**: このAPIはマーチャントサイトから呼び出されます。決済アプリウィンドウからは残高確認APIのみを呼び出します。

### 4.2.5 コード引き換え

```
POST /api/v1/codes/redeem
Authorization: Bearer {token}
Content-Type: application/json

Request:
{
  "code": "PROMO2024ABC",
  "user_id": "user123"
}

Response:
{
  "redemption_id": "red_1234567890",
  "transaction_id": "txn_1234567891",
  "code": "PROMO2024ABC",
  "currency_type": "free",
  "amount": "500",
  "balance_after": "1100",
  "status": "completed"
}

エラーレスポンス:
- 400: コードが無効、期限切れ、または既に使用済み
- 404: コードが見つからない
```

### 4.2.6 トランザクション履歴取得

```
GET /api/v1/users/{user_id}/transactions
Authorization: Bearer {token}
Query Parameters:
  - limit: 取得件数（デフォルト: 50）
  - offset: オフセット（デフォルト: 0）
  - currency_type: 通貨種別フィルタ（paid/free）
  - transaction_type: 取引種別フィルタ

Response:
{
  "transactions": [
    {
      "transaction_id": "txn_1234567890",
      "transaction_type": "grant",
      "currency_type": "free",
      "amount": "100",
      "balance_before": "500",
      "balance_after": "600",
      "status": "completed",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 100,
  "limit": 50,
  "offset": 0
}
```

## 4.3 gRPC API定義（proto）

```protobuf
syntax = "proto3";

package currency;

service CurrencyService {
  rpc GetBalance(GetBalanceRequest) returns (GetBalanceResponse);
  rpc Grant(GrantRequest) returns (GrantResponse);
  rpc Consume(ConsumeRequest) returns (ConsumeResponse);
  rpc ProcessPayment(ProcessPaymentRequest) returns (ProcessPaymentResponse);
  rpc RedeemCode(RedeemCodeRequest) returns (RedeemCodeResponse);
  rpc GetTransactionHistory(GetTransactionHistoryRequest) returns (GetTransactionHistoryResponse);
}

message GetBalanceRequest {
  string user_id = 1;
}

message GetBalanceResponse {
  string user_id = 1;
  map<string, string> balances = 2; // "paid" => "1000", "free" => "500" (整数値の文字列)
}

message GrantRequest {
  string user_id = 1;
  string currency_type = 2; // "paid" or "free"
  string amount = 3; // 整数値の文字列（例: "100"）
  string reason = 4;
  map<string, string> metadata = 5;
}

message GrantResponse {
  string transaction_id = 1;
  string balance_after = 2; // 整数値の文字列（例: "600"）
  string status = 3;
}

message ConsumeRequest {
  string user_id = 1;
  string currency_type = 2; // "paid", "free", or "auto"
  string amount = 3; // 整数値の文字列（例: "50"）
  string item_id = 4;
  bool use_priority = 5; // 優先順位制御（無料通貨優先）
  map<string, string> metadata = 6;
}

message ConsumeResponse {
  string transaction_id = 1;
  repeated ConsumptionDetail consumption_details = 2; // 優先順位制御使用時
  string balance_after = 3; // 単一通貨タイプ消費時
  string total_consumed = 4; // 優先順位制御使用時
  string status = 5;
}

message ConsumptionDetail {
  string currency_type = 1; // "paid" or "free"
  string amount = 2; // 整数値の文字列
  string balance_before = 3; // 整数値の文字列
  string balance_after = 4; // 整数値の文字列
}

message ProcessPaymentRequest {
  string payment_request_id = 1;
  string user_id = 2;
  string method_name = 3;
  map<string, string> details = 4;
  string amount = 5;
  string currency = 6;
}

message ProcessPaymentResponse {
  string transaction_id = 1;
  string payment_request_id = 2;
  string balance_after = 3;
  string status = 4;
}

message RedeemCodeRequest {
  string code = 1;
  string user_id = 2;
}

message RedeemCodeResponse {
  string redemption_id = 1;
  string transaction_id = 2;
  string code = 3;
  string currency_type = 4;
  string amount = 5;
  string balance_after = 6;
  string status = 7;
}

message GetTransactionHistoryRequest {
  string user_id = 1;
  int32 limit = 2;
  int32 offset = 3;
  string currency_type = 4; // optional
  string transaction_type = 5; // optional
}

message GetTransactionHistoryResponse {
  repeated Transaction transactions = 1;
  int32 total = 2;
  int32 limit = 3;
  int32 offset = 4;
}

message Transaction {
  string transaction_id = 1;
  string transaction_type = 2;
  string currency_type = 3;
  string amount = 4; // 整数値の文字列（例: "100"）
  string balance_before = 5; // 整数値の文字列（例: "500"）
  string balance_after = 6; // 整数値の文字列（例: "600"）
  string status = 7;
  string created_at = 8;
}
```
