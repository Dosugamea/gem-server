# API使用例

本ドキュメントでは、仮想通貨管理APIの使用例を提供します。

## 目次

- [認証](#認証)
- [通貨残高取得](#通貨残高取得)
- [通貨付与](#通貨付与)
- [通貨消費](#通貨消費)
- [決済処理](#決済処理)
- [コード引き換え](#コード引き換え)
- [トランザクション履歴取得](#トランザクション履歴取得)

## 認証

### JWTトークン生成

```bash
curl -X POST http://localhost:8080/api/v1/auth/token \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123"
  }'
```

**レスポンス例:**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 86400,
  "token_type": "Bearer"
}
```

**注意:** 以降のAPIリクエストでは、取得したトークンを `Authorization: Bearer {token}` ヘッダーに含めてください。

## 通貨残高取得

### 基本的な使用例

```bash
curl -X GET http://localhost:8080/api/v1/users/user123/balance \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

**レスポンス例:**

```json
{
  "user_id": "user123",
  "balances": {
    "paid": "1000",
    "free": "500"
  }
}
```

### JavaScriptでの使用例

```javascript
async function getBalance(userId, token) {
  const response = await fetch(`http://localhost:8080/api/v1/users/${userId}/balance`, {
    method: 'GET',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    }
  });
  
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }
  
  const data = await response.json();
  return data;
}

// 使用例
const token = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...';
getBalance('user123', token)
  .then(balance => {
    console.log('有償通貨:', balance.balances.paid);
    console.log('無償通貨:', balance.balances.free);
  })
  .catch(error => console.error('エラー:', error));
```

## 通貨付与

### 無償通貨の付与

```bash
curl -X POST http://localhost:8080/api/v1/users/user123/grant \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{
    "currency_type": "free",
    "amount": "100",
    "reason": "イベント報酬",
    "metadata": {
      "event_id": "event_001",
      "campaign_id": "summer_2024"
    }
  }'
```

**レスポンス例:**

```json
{
  "transaction_id": "txn_1234567890",
  "balance_after": "600",
  "status": "completed"
}
```

### 有償通貨の付与（補填など）

```bash
curl -X POST http://localhost:8080/api/v1/users/user123/grant \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{
    "currency_type": "paid",
    "amount": "500",
    "reason": "返金処理",
    "metadata": {
      "refund_id": "refund_001",
      "original_transaction_id": "txn_0000000001"
    }
  }'
```

### Pythonでの使用例

```python
import requests

def grant_currency(user_id, token, currency_type, amount, reason=None, metadata=None):
    url = f"http://localhost:8080/api/v1/users/{user_id}/grant"
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }
    payload = {
        "currency_type": currency_type,
        "amount": str(amount),  # 整数値を文字列で指定
        "reason": reason,
        "metadata": metadata or {}
    }
    
    response = requests.post(url, json=payload, headers=headers)
    response.raise_for_status()
    return response.json()

# 使用例
token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
result = grant_currency(
    user_id="user123",
    token=token,
    currency_type="free",
    amount=100,
    reason="ログインボーナス",
    metadata={"day": 1}
)
print(f"トランザクションID: {result['transaction_id']}")
print(f"残高: {result['balance_after']}")
```

## 通貨消費

### 指定通貨タイプでの消費

```bash
curl -X POST http://localhost:8080/api/v1/users/user123/consume \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{
    "currency_type": "paid",
    "amount": "50",
    "item_id": "item_001",
    "metadata": {
      "purchase_id": "purchase_001",
      "item_name": "プレミアムパック"
    }
  }'
```

**レスポンス例:**

```json
{
  "transaction_id": "txn_1234567891",
  "balance_after": "950",
  "status": "completed"
}
```

### 優先順位付き消費（無料通貨優先）

無料通貨を優先して使用し、不足分を有料通貨で補う消費処理。

```bash
curl -X POST http://localhost:8080/api/v1/users/user123/consume \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{
    "currency_type": "auto",
    "amount": "150",
    "item_id": "item_002",
    "use_priority": true,
    "metadata": {
      "purchase_id": "purchase_002",
      "item_name": "スペシャルパック"
    }
  }'
```

**レスポンス例:**

```json
{
  "transaction_id": "txn_1234567892",
  "consumption_details": [
    {
      "currency_type": "free",
      "amount": "100",
      "balance_before": "600",
      "balance_after": "500"
    },
    {
      "currency_type": "paid",
      "amount": "50",
      "balance_before": "950",
      "balance_after": "900"
    }
  ],
  "total_consumed": "150",
  "status": "completed"
}
```

### Goでの使用例

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type ConsumeRequest struct {
    CurrencyType string                 `json:"currency_type"`
    Amount       string                 `json:"amount"`
    ItemID       string                 `json:"item_id,omitempty"`
    UsePriority  bool                   `json:"use_priority,omitempty"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type ConsumeResponse struct {
    TransactionID      string              `json:"transaction_id"`
    ConsumptionDetails []ConsumptionDetail `json:"consumption_details,omitempty"`
    BalanceAfter       string              `json:"balance_after,omitempty"`
    TotalConsumed      string              `json:"total_consumed,omitempty"`
    Status             string              `json:"status"`
}

type ConsumptionDetail struct {
    CurrencyType  string `json:"currency_type"`
    Amount        string `json:"amount"`
    BalanceBefore string `json:"balance_before"`
    BalanceAfter  string `json:"balance_after"`
}

func consumeCurrency(userID, token string, req ConsumeRequest) (*ConsumeResponse, error) {
    url := fmt.Sprintf("http://localhost:8080/api/v1/users/%s/consume", userID)
    
    jsonData, err := json.Marshal(req)
    if err != nil {
        return nil, err
    }
    
    httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, err
    }
    
    httpReq.Header.Set("Authorization", "Bearer "+token)
    httpReq.Header.Set("Content-Type", "application/json")
    
    client := &http.Client{}
    resp, err := client.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
    }
    
    var result ConsumeResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return &result, nil
}

// 使用例
func main() {
    token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
    
    // 優先順位付き消費
    req := ConsumeRequest{
        CurrencyType: "auto",
        Amount:       "150",
        ItemID:       "item_002",
        UsePriority:  true,
        Metadata: map[string]interface{}{
            "purchase_id": "purchase_002",
        },
    }
    
    result, err := consumeCurrency("user123", token, req)
    if err != nil {
        fmt.Printf("エラー: %v\n", err)
        return
    }
    
    fmt.Printf("トランザクションID: %s\n", result.TransactionID)
    fmt.Printf("合計消費額: %s\n", result.TotalConsumed)
    for _, detail := range result.ConsumptionDetails {
        fmt.Printf("  %s通貨: %s (残高: %s -> %s)\n",
            detail.CurrencyType,
            detail.Amount,
            detail.BalanceBefore,
            detail.BalanceAfter)
    }
}
```

## 決済処理

### PaymentRequest API経由の決済処理

マーチャントサイトが決済を完了するために呼び出すAPIです。決済アプリウィンドウから返されたPaymentResponseを受け取り、このAPIを呼び出して決済処理を完了します。

```bash
curl -X POST http://localhost:8080/api/v1/payment/process \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{
    "payment_request_id": "pr_1234567890",
    "user_id": "user123",
    "method_name": "https://yourdomain.com/pay",
    "details": {
      "userId": "user123",
      "transactionId": "txn_abc123",
      "timestamp": "1234567890"
    },
    "amount": "1000",
    "currency": "JPY"
  }'
```

**レスポンス例:**

```json
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
      "balance_before": "900",
      "balance_after": "400"
    }
  ],
  "total_consumed": "1000",
  "status": "completed"
}
```

**注意:** このAPIは二重決済を防ぐため、同じ `payment_request_id` で複数回呼び出すと、最初の呼び出しの結果を返します（冪等性保証）。

## コード引き換え

### プロモーションコードの引き換え

```bash
curl -X POST http://localhost:8080/api/v1/codes/redeem \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{
    "code": "PROMO2024ABC",
    "user_id": "user123"
  }'
```

**レスポンス例（成功）:**

```json
{
  "redemption_id": "red_1234567890",
  "transaction_id": "txn_1234567891",
  "code": "PROMO2024ABC",
  "currency_type": "free",
  "amount": "500",
  "balance_after": "1100",
  "status": "completed"
}
```

**エラーレスポンス例（無効なコード）:**

```json
{
  "error": "invalid_code",
  "message": "コードが無効、期限切れ、または既に使用済みです",
  "code": "CODE_INVALID"
}
```

**エラーレスポンス例（コードが見つからない）:**

```json
{
  "error": "code_not_found",
  "message": "指定されたコードが見つかりません",
  "code": "CODE_NOT_FOUND"
}
```

## トランザクション履歴取得

### 基本的な使用例

```bash
curl -X GET "http://localhost:8080/api/v1/users/user123/transactions?limit=10&offset=0" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

**レスポンス例:**

```json
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
    },
    {
      "transaction_id": "txn_1234567891",
      "transaction_type": "consume",
      "currency_type": "paid",
      "amount": "50",
      "balance_before": "1000",
      "balance_after": "950",
      "status": "completed",
      "created_at": "2024-01-01T01:00:00Z"
    }
  ],
  "total": 100,
  "limit": 10,
  "offset": 0
}
```

### フィルタリング例

#### 有償通貨のみ取得

```bash
curl -X GET "http://localhost:8080/api/v1/users/user123/transactions?currency_type=paid&limit=20" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

#### 付与トランザクションのみ取得

```bash
curl -X GET "http://localhost:8080/api/v1/users/user123/transactions?transaction_type=grant&limit=20" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

#### 複合フィルタ

```bash
curl -X GET "http://localhost:8080/api/v1/users/user123/transactions?currency_type=free&transaction_type=grant&limit=50&offset=0" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

## エラーハンドリング

### 一般的なエラーレスポンス

すべてのエラーレスポンスは以下の形式です:

```json
{
  "error": "error_code",
  "message": "エラーメッセージ",
  "code": "ERROR_CODE"
}
```

### 主なエラーコード

- `UNAUTHORIZED` (401): 認証トークンが無効または期限切れ
- `BAD_REQUEST` (400): リクエストパラメータが不正
- `NOT_FOUND` (404): リソースが見つからない
- `INSUFFICIENT_BALANCE` (409): 残高不足
- `DUPLICATE_PAYMENT` (409): 二重決済
- `CODE_INVALID` (400): コードが無効、期限切れ、または既に使用済み
- `CODE_NOT_FOUND` (404): コードが見つからない
- `INTERNAL_SERVER_ERROR` (500): サーバー内部エラー

### エラーハンドリングの例（JavaScript）

```javascript
async function handleApiRequest(url, options) {
  try {
    const response = await fetch(url, options);
    
    if (!response.ok) {
      const error = await response.json();
      
      switch (response.status) {
        case 401:
          // 認証エラー - トークンを再取得
          console.error('認証エラー:', error.message);
          // トークン再取得処理
          break;
        case 400:
          // バリデーションエラー
          console.error('リクエストエラー:', error.message);
          break;
        case 404:
          // リソースが見つからない
          console.error('リソースが見つかりません:', error.message);
          break;
        case 409:
          // 競合エラー（残高不足など）
          console.error('競合エラー:', error.message);
          break;
        default:
          console.error('サーバーエラー:', error.message);
      }
      
      throw new Error(error.message);
    }
    
    return await response.json();
  } catch (error) {
    console.error('APIリクエストエラー:', error);
    throw error;
  }
}

// 使用例
const token = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...';
handleApiRequest('http://localhost:8080/api/v1/users/user123/balance', {
  method: 'GET',
  headers: {
    'Authorization': `Bearer ${token}`
  }
})
  .then(data => console.log('成功:', data))
  .catch(error => console.error('失敗:', error));
```

## 補足情報

### 金額の形式

すべての金額は**整数値の文字列**として扱われます。小数点は使用しません。

- ✅ 正しい: `"100"`, `"1000"`, `"0"`
- ❌ 誤り: `100.00`, `"100.00"`, `100`

### 認証トークンの有効期限

デフォルトでは、JWTトークンの有効期限は24時間です。期限切れの場合は、`/auth/token` エンドポイントで新しいトークンを取得してください。

### レート制限

現在、レート制限は実装されていませんが、将来的に追加される可能性があります。大量のリクエストを送信する場合は、適切な間隔を空けてください。

### タイムゾーン

すべての日時はUTCで返されます。必要に応じて、クライアント側でローカルタイムゾーンに変換してください。
