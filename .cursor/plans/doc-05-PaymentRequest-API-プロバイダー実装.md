---
name: タスク05 - PaymentRequest API プロバイダー実装
overview: PaymentRequest APIのプロバイダー側（決済アプリ側）の実装を定義します
---

# タスク05: PaymentRequest API プロバイダー実装

本システムはPaymentRequest APIのプロバイダー側（決済アプリ側）として機能します。`https://yourdomain.com/pay` という決済方法を提供し、マーチャントサイトがこの決済方法を使用できるようにします。

## 5.1 実装コンポーネント

### 5.1.1 Payment Method Manifest

決済方法を定義するマニフェストファイル。`/pay/payment-manifest.json` として配信します。

**public/pay/payment-manifest.json**

```json
{
  "default_applications": ["https://yourdomain.com/pay/manifest.json"],
  "supported_origins": [
    "https://yourdomain.com",
    "https://game.yourdomain.com"
  ]
}
```

**HTTPヘッダーでの参照**

```go
// Echo Frameworkでの実装例
e.GET("/pay", func(c echo.Context) error {
    c.Response().Header().Set("Link", 
        `<https://yourdomain.com/pay/payment-manifest.json>; rel="payment-method-manifest"`)
    return c.File("public/pay/index.html")
})
```

### 5.1.2 Web App Manifest

決済アプリの設定を定義するマニフェストファイル。

**public/pay/manifest.json**

```json
{
  "name": "YourGame Currency",
  "short_name": "GameCurrency",
  "icons": [{
    "src": "icon.png",
    "sizes": "48x48",
    "type": "image/png"
  }],
  "serviceworker": {
    "src": "sw-payment-handler.js",
    "use_cache": false
  }
}
```

### 5.1.3 Service Worker実装

決済リクエストを処理するService Worker。

**public/pay/sw-payment-handler.js**

主要なイベントハンドラ：

- `canmakepayment`: 決済可能かどうかの確認
- `paymentrequest`: 決済リクエストの処理
- `message`: 決済アプリウィンドウからのメッセージ受信

実装の詳細は `docs/payment-request-api.md` を参照してください。

### 5.1.4 決済アプリウィンドウ

ユーザーが決済を承認するためのUI。

**public/pay/index.html**: 決済アプリウィンドウのHTML

**public/pay/payment-app.js**: 決済アプリのJavaScript

主要な機能：

- ユーザー認証
- ユーザー残高の表示
- 決済情報の表示
- 決済承認/キャンセル
- Service Workerとの通信
- バックエンドAPIとの通信（残高確認）

## 5.2 ディレクトリ構造

```
public/
└── pay/                          # Payment Handler関連ファイル
    ├── payment-manifest.json     # Payment Method Manifest
    ├── manifest.json              # Web App Manifest
    ├── sw-payment-handler.js      # Service Worker
    ├── index.html                 # 決済アプリウィンドウ
    ├── payment-app.js             # 決済アプリJavaScript
    └── icon.png                   # アイコン
```

## 5.3 マーチャント側の使用例（参考）

マーチャントサイトは以下のように本システムの決済方法を使用します：

```javascript
// マーチャントサイト側の実装例（参考）
const supportedMethods = [{
  supportedMethods: 'https://yourdomain.com/pay'
}];

const details = {
  total: {
    label: '有償通貨購入',
    amount: {
      currency: 'JPY',
      value: '1000.00'
    }
  }
};

const request = new PaymentRequest(supportedMethods, details);
const response = await request.show();
// 決済処理APIを呼び出す
```

**注意**: 上記はマーチャント側の実装例です。本システムが実装するのは**プロバイダー側（決済アプリ側）**です。
