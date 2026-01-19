# セキュリティレビューレポート

**プロジェクト**: 仮想通貨管理サービス (gem-server)  
**レビュー日**: 2024年  
**レビュー対象**: ソースコード全体  
**レビュアー**: AI Security Reviewer

---

## エグゼクティブサマリー

本レポートは、仮想通貨管理サービスのソースコードに対して実施したセキュリティレビューの結果をまとめたものです。全体的に、基本的なセキュリティ対策は実装されていますが、本番環境に向けて改善が必要な項目がいくつか見つかりました。

**総合評価**: ⚠️ **中リスク** - 本番環境へのデプロイ前に改善推奨

---

## 1. セキュリティ強み（実装済みの対策）

### 1.1 SQLインジェクション対策 ✅
- **状態**: 適切に実装済み
- **詳細**: すべてのデータベースクエリでプリペアドステートメント（`?`プレースホルダー）を使用
- **確認箇所**: 
  - `internal/infrastructure/persistence/mysql/currency_repository.go`
  - `internal/infrastructure/persistence/mysql/transaction_repository.go`
  - `internal/infrastructure/persistence/mysql/payment_request_repository.go`
  - `internal/infrastructure/persistence/mysql/redemption_code_repository.go`

### 1.2 認証・認可の実装 ✅
- **状態**: JWT認証が実装済み
- **詳細**: 
  - REST APIとgRPCの両方でJWT認証を実装
  - ユーザーIDの検証ロジックが各ハンドラーで実装されている
  - トークンとリクエストのuser_idの一致確認を実施
- **確認箇所**:
  - `internal/presentation/rest/middleware/auth.go`
  - `internal/presentation/grpc/interceptor/auth.go`
  - 各ハンドラーでのuser_id検証

### 1.3 データ整合性保証 ✅
- **状態**: 適切に実装済み
- **詳細**:
  - データベーストランザクションによる整合性保証
  - 楽観的ロック（versionカラム）による同時更新制御
  - 二重決済防止（PaymentRequest IDの一意性）
  - 冪等性保証（Transaction ID、PaymentRequest ID）
- **確認箇所**:
  - `internal/infrastructure/persistence/mysql/transaction_manager.go`
  - `internal/application/payment/service.go` (ProcessPayment)

### 1.4 エラーハンドリング ✅
- **状態**: 適切に実装済み
- **詳細**:
  - 内部エラーをクライアントに漏洩させない設計
  - 標準化されたエラーレスポンス形式
  - ドメインエラーの適切なマッピング
- **確認箇所**:
  - `internal/presentation/rest/middleware/error_handler.go`

### 1.5 Dockerfileのセキュリティ ✅
- **状態**: 適切に実装済み
- **詳細**:
  - 非rootユーザー（appuser）での実行
  - マルチステージビルドによるイメージサイズ削減
  - 最小限のパッケージのみインストール
- **確認箇所**: `Dockerfile`

---

## 2. 重要なセキュリティ問題（要修正）

### 2.1 CORS設定が過度に緩い 🔴 **高リスク**

**問題**:
```go
// internal/presentation/rest/router.go:80-84
e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
    AllowOrigins: []string{"*"}, // 本番環境では適切に設定
    AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.OPTIONS},
    AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
}))
```

**リスク**:
- ワイルドカード（`*`）により、任意のオリジンからのリクエストを許可
- CSRF攻撃のリスクが高まる
- 本番環境では重大なセキュリティ問題

**推奨対策**:
```go
// 環境変数から許可オリジンを取得
allowedOrigins := strings.Split(getEnv("CORS_ALLOWED_ORIGINS", ""), ",")
if len(allowedOrigins) == 0 || allowedOrigins[0] == "" {
    // 本番環境ではエラーにする
    if cfg.Environment == "production" {
        return fmt.Errorf("CORS_ALLOWED_ORIGINS must be set in production")
    }
    allowedOrigins = []string{"http://localhost:3000"} // 開発環境のみ
}

e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
    AllowOrigins: allowedOrigins,
    AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.OPTIONS},
    AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
    AllowCredentials: true, // 必要に応じて
}))
```

**優先度**: 🔴 **高** - 本番環境デプロイ前に必須

---

### 2.2 レート制限が未実装 🔴 **高リスク**

**問題**:
- APIエンドポイントに対するレート制限が実装されていない
- ブルートフォース攻撃やDoS攻撃に対する防御がない
- 特に認証エンドポイント（`/api/v1/auth/token`）が保護されていない

**リスク**:
- ブルートフォース攻撃による不正アクセス
- DoS攻撃によるサービス停止
- リソースの過剰消費

**推奨対策**:
1. レート制限ミドルウェアの実装
2. IPアドレスベースのレート制限
3. ユーザーIDベースのレート制限（認証後）
4. Redisを使用した分散レート制限（推奨）

**実装例**:
```go
// internal/presentation/rest/middleware/rate_limit.go
func RateLimitMiddleware(redisClient *redis.Client) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // IPアドレスを取得
            ip := c.RealIP()
            
            // レート制限チェック（例: 1分間に100リクエスト）
            key := fmt.Sprintf("rate_limit:%s", ip)
            count, err := redisClient.Incr(ctx, key).Result()
            if err != nil {
                // Redisエラーの場合は許可（フォールバック）
                return next(c)
            }
            
            if count == 1 {
                redisClient.Expire(ctx, key, time.Minute)
            }
            
            if count > 100 {
                return c.JSON(429, ErrorResponse{
                    Error:   "rate_limit_exceeded",
                    Message: "Too many requests",
                })
            }
            
            return next(c)
        }
    }
}
```

**優先度**: 🔴 **高** - 本番環境デプロイ前に必須

---

### 2.3 セキュリティヘッダーが未設定 🟡 **中リスク**

**問題**:
- セキュリティ関連のHTTPヘッダーが設定されていない
- XSS、クリックジャッキング、MIMEタイプスニッフィングなどの攻撃に対する防御がない

**リスク**:
- XSS攻撃
- クリックジャッキング攻撃
- MIMEタイプスニッフィング攻撃

**推奨対策**:
```go
// internal/presentation/rest/middleware/security_headers.go
func SecurityHeadersMiddleware() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // XSS保護
            c.Response().Header().Set("X-XSS-Protection", "1; mode=block")
            
            // クリックジャッキング保護
            c.Response().Header().Set("X-Frame-Options", "DENY")
            
            // MIMEタイプスニッフィング保護
            c.Response().Header().Set("X-Content-Type-Options", "nosniff")
            
            // コンテンツセキュリティポリシー（必要に応じて調整）
            c.Response().Header().Set("Content-Security-Policy", 
                "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
            
            // Strict-Transport-Security（HTTPS使用時）
            if c.Scheme() == "https" {
                c.Response().Header().Set("Strict-Transport-Security", 
                    "max-age=31536000; includeSubDomains")
            }
            
            // Referrer-Policy
            c.Response().Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
            
            return next(c)
        }
    }
}
```

**優先度**: 🟡 **中** - 本番環境デプロイ前に推奨

---

### 2.4 JWTトークンの検証が不完全 🟡 **中リスク**

**問題**:
```go
// internal/presentation/rest/middleware/auth.go:41-47
token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
    // 署名アルゴリズムの確認
    if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
        return nil, jwt.ErrSignatureInvalid
    }
    return []byte(cfg.Secret), nil
})

if err != nil || !token.Valid {
    // エラーハンドリング
}
```

**リスク**:
- `exp`（有効期限）クレームの明示的な検証がない
- `nbf`（Not Before）クレームの検証がない
- `iss`（発行者）クレームの検証がない

**推奨対策**:
```go
// JWT検証オプションを追加
token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
    if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
        return nil, jwt.ErrSignatureInvalid
    }
    return []byte(cfg.Secret), nil
}, jwt.WithValidMethods([]string{"HS256"}), // アルゴリズムの明示的な指定
   jwt.WithExpirationRequired(),            // expクレームを必須に
   jwt.WithIssuer(cfg.Issuer))             // issクレームの検証

if err != nil {
    // エラーハンドリング
}

// クレームの明示的な検証
claims, ok := token.Claims.(jwt.MapClaims)
if !ok {
    return c.JSON(401, ErrorResponse{
        Error:   "unauthorized",
        Message: "Invalid token claims",
    })
}

// expクレームの検証（念のため）
if exp, ok := claims["exp"].(float64); ok {
    if time.Now().Unix() > int64(exp) {
        return c.JSON(401, ErrorResponse{
            Error:   "unauthorized",
            Message: "Token expired",
        })
    }
}
```

**優先度**: 🟡 **中** - 本番環境デプロイ前に推奨

---

## 3. 改善推奨事項（中優先度）

### 3.1 入力検証の強化 🟡 **中リスク**

**問題**:
- 一部の入力値に対して検証が不十分
- 文字列長の制限がない
- 不正な文字のサニタイズがない

**確認箇所**:
- `internal/presentation/rest/handler/currency_handler.go`
- `internal/presentation/rest/handler/payment_handler.go`
- `internal/presentation/rest/handler/code_redemption_handler.go`

**推奨対策**:
```go
// 入力検証関数の追加
func validateUserID(userID string) error {
    if len(userID) == 0 || len(userID) > 255 {
        return fmt.Errorf("user_id must be between 1 and 255 characters")
    }
    // 不正な文字のチェック（例: SQLインジェクション文字）
    if strings.ContainsAny(userID, "';--") {
        return fmt.Errorf("user_id contains invalid characters")
    }
    return nil
}

func validateAmount(amount int64) error {
    if amount <= 0 {
        return currency.ErrInvalidAmount
    }
    // 最大値の制限（例: 10兆）
    if amount > 10000000000000 {
        return fmt.Errorf("amount exceeds maximum limit")
    }
    return nil
}
```

**優先度**: 🟡 **中**

---

### 3.2 ログに機密情報が含まれる可能性 🟡 **中リスク**

**問題**:
- エラーログに詳細なエラーメッセージが含まれる可能性
- トークンやパスワードがログに出力されるリスク

**確認箇所**:
- `internal/presentation/rest/middleware/auth.go:50-52`
```go
logger.Warn(ctx, "Invalid token", map[string]interface{}{
    "error": err.Error(), // エラーの詳細が含まれる可能性
})
```

**推奨対策**:
```go
// 機密情報をマスクする関数
func maskSensitiveData(data map[string]interface{}) map[string]interface{} {
    masked := make(map[string]interface{})
    sensitiveKeys := []string{"password", "token", "secret", "key", "authorization"}
    
    for k, v := range data {
        for _, sensitiveKey := range sensitiveKeys {
            if strings.Contains(strings.ToLower(k), sensitiveKey) {
                masked[k] = "***MASKED***"
                continue
            }
        }
        masked[k] = v
    }
    return masked
}

// 使用例
logger.Warn(ctx, "Invalid token", maskSensitiveData(map[string]interface{}{
    "error": err.Error(),
}))
```

**優先度**: 🟡 **中**

---

### 3.3 認証エンドポイントの保護不足 🟡 **中リスク**

**問題**:
- `/api/v1/auth/token` エンドポイントが認証不要
- レート制限がない
- ユーザーIDの検証が不十分（任意のuser_idでトークン生成可能）

**確認箇所**:
- `internal/presentation/rest/router.go:120`
- `internal/presentation/rest/handler/auth_handler.go`

**推奨対策**:
1. レート制限の追加（特に厳しく）
2. CAPTCHAの導入（必要に応じて）
3. ユーザーIDの存在確認（データベースで確認）
4. 失敗回数の記録とアカウントロック

**優先度**: 🟡 **中**

---

### 3.4 パスワード/シークレットの管理 🟢 **低リスク**

**状態**: 環境変数から読み込む設計は適切

**改善提案**:
- シークレット管理サービスの使用（AWS Secrets Manager、HashiCorp Vaultなど）
- シークレットのローテーション機能
- 本番環境でのシークレット強度チェック

**優先度**: 🟢 **低**

---

## 4. 依存関係のセキュリティ

### 4.1 依存関係の脆弱性チェック 🔍 **要確認**

**推奨アクション**:
```powershell
# Goのセキュリティチェックツールを使用
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# または、GitHubのDependabotを有効化
```

**確認が必要な主要依存関係**:
- `github.com/labstack/echo/v4` - Webフレームワーク
- `github.com/golang-jwt/jwt/v5` - JWTライブラリ
- `github.com/go-sql-driver/mysql` - MySQLドライバー
- `go.opentelemetry.io/otel` - OpenTelemetry

**優先度**: 🟡 **中** - 定期的な確認が必要

---

## 5. フロントエンド（Payment Handler）のセキュリティ

### 5.1 XSS対策 🟡 **中リスク**

**問題**:
- `public/pay/payment-app.js` でユーザー入力のサニタイズが不十分
- `innerHTML` の使用（確認が必要）

**確認箇所**:
- `public/pay/payment-app.js`
- `public/pay/index.html`

**推奨対策**:
- `textContent` の使用（`innerHTML`の代わりに）
- Content Security Policy (CSP) の設定
- 入力値のサニタイズ

**優先度**: 🟡 **中**

---

### 5.2 認証トークンの保存方法 🟡 **中リスク**

**問題**:
```javascript
// public/pay/payment-app.js:233
const token = localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token');
```

**リスク**:
- `localStorage` はXSS攻撃に対して脆弱
- トークンが長期間保存される可能性

**推奨対策**:
- `sessionStorage` の優先使用
- トークンの有効期限を短く設定
- HttpOnly Cookieの使用を検討（可能な場合）

**優先度**: 🟡 **中**

---

## 6. データベースセキュリティ

### 6.1 接続文字列の保護 ✅ **適切**

**状態**: 環境変数から読み込む設計は適切

**改善提案**:
- SSL/TLS接続の強制（本番環境）
- 接続文字列の暗号化（オプション）

**優先度**: 🟢 **低**

---

### 6.2 SQLインジェクション対策 ✅ **適切**

**状態**: プリペアドステートメントを使用

**確認**: すべてのクエリで `?` プレースホルダーを使用

---

## 7. 監査とログ

### 7.1 監査ログ ✅ **適切**

**状態**: トランザクション履歴が適切に記録されている

**改善提案**:
- セキュリティイベントの専用ログ（認証失敗、権限エラーなど）
- ログの長期保存と分析

**優先度**: 🟢 **低**

---

## 8. 優先度別アクションプラン

### 🔴 **高優先度（本番環境デプロイ前に必須）**

1. **CORS設定の修正**
   - ワイルドカード（`*`）の削除
   - 許可オリジンの環境変数化
   - 本番環境でのエラーチェック追加

2. **レート制限の実装**
   - IPアドレスベースのレート制限
   - 認証エンドポイントへの厳しい制限
   - Redisを使用した分散レート制限

### 🟡 **中優先度（本番環境デプロイ前に推奨）**

3. **セキュリティヘッダーの追加**
   - X-Frame-Options
   - X-Content-Type-Options
   - Content-Security-Policy
   - Strict-Transport-Security

4. **JWT検証の強化**
   - `exp` クレームの明示的な検証
   - `iss` クレームの検証
   - 検証オプションの追加

5. **入力検証の強化**
   - 文字列長の制限
   - 不正な文字のサニタイズ
   - 数値範囲の検証

6. **ログの機密情報マスキング**
   - トークン、パスワードのマスキング
   - エラーメッセージのサニタイズ

### 🟢 **低優先度（継続的改善）**

7. **依存関係の脆弱性チェック**
   - `govulncheck` の定期実行
   - Dependabotの有効化

8. **フロントエンドのセキュリティ強化**
   - XSS対策の強化
   - トークン保存方法の改善

---

## 9. セキュリティチェックリスト

### 認証・認可
- [x] JWT認証の実装
- [x] ユーザーIDの検証
- [ ] レート制限の実装
- [ ] 認証エンドポイントの保護強化
- [ ] JWT検証の強化

### 入力検証
- [x] 基本的な入力検証
- [ ] 文字列長の制限
- [ ] 不正な文字のサニタイズ
- [ ] 数値範囲の検証

### データ保護
- [x] SQLインジェクション対策
- [x] プリペアドステートメントの使用
- [x] トランザクション管理
- [ ] 機密情報のマスキング（ログ）

### ネットワークセキュリティ
- [ ] CORS設定の適正化
- [ ] セキュリティヘッダーの設定
- [ ] HTTPSの強制（本番環境）

### 監査とログ
- [x] トランザクション履歴の記録
- [x] エラーログの記録
- [ ] セキュリティイベントの専用ログ

### 依存関係
- [ ] 脆弱性スキャンの実施
- [ ] 依存関係の定期更新

---

## 10. 結論

本プロジェクトは、基本的なセキュリティ対策が適切に実装されています。特に、SQLインジェクション対策、認証・認可、データ整合性保証などは良好な状態です。

しかし、本番環境へのデプロイ前に以下の項目を修正することを強く推奨します：

1. **CORS設定の修正**（高優先度）
2. **レート制限の実装**（高優先度）
3. **セキュリティヘッダーの追加**（中優先度）
4. **JWT検証の強化**（中優先度）

これらの修正により、本番環境でのセキュリティリスクを大幅に低減できます。

---

## 11. 参考資料

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [OWASP API Security Top 10](https://owasp.org/www-project-api-security/)
- [Go Security Best Practices](https://go.dev/doc/security/best-practices)
- [Echo Framework Security](https://echo.labstack.com/guide/security/)

---

**レポート作成日**: 2024年  
**次回レビュー推奨日**: 本番環境デプロイ前、または主要機能追加時
