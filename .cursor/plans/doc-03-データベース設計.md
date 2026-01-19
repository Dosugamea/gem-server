---
name: タスク03 - データベース設計
overview: データベースのテーブル構成とスキーマを定義します
---

# タスク03: データベース設計

## 3.1 テーブル構成

### users（ユーザー）

```sql
CREATE TABLE users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id)
);
```

### currency_balances（通貨残高）

```sql
CREATE TABLE currency_balances (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id VARCHAR(255) NOT NULL,
    currency_type ENUM('paid', 'free') NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0, -- 整数値（小数点なし）、マイナス値を許可
    version INT NOT NULL DEFAULT 0, -- 楽観的ロック用
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_currency (user_id, currency_type),
    FOREIGN KEY (user_id) REFERENCES users(user_id),
    INDEX idx_user_id (user_id)
);
```

**注意**: `balance`カラムはマイナス値を許可します。運用によっては返金処理、補填処理、手動調整などでマイナス残高が発生する可能性があります。マイナス残高が発生した場合は、トランザクション履歴に記録し、監視・アラートの対象とします。

### transactions（トランザクション履歴）

```sql
CREATE TABLE transactions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    transaction_id VARCHAR(255) UNIQUE NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    transaction_type ENUM('grant', 'consume', 'refund', 'expire', 'compensate') NOT NULL,
    currency_type ENUM('paid', 'free') NOT NULL,
    amount BIGINT NOT NULL, -- 整数値（小数点なし）
    balance_before BIGINT NOT NULL, -- 整数値（小数点なし）
    balance_after BIGINT NOT NULL, -- 整数値（小数点なし）
    status ENUM('pending', 'completed', 'failed', 'cancelled') NOT NULL,
    payment_request_id VARCHAR(255), -- PaymentRequest APIのID
    metadata JSON, -- 追加情報（商品ID、理由など）
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(user_id),
    INDEX idx_user_id (user_id),
    INDEX idx_transaction_id (transaction_id),
    INDEX idx_payment_request_id (payment_request_id),
    INDEX idx_created_at (created_at)
);
```

### payment_requests（PaymentRequest記録）

```sql
CREATE TABLE payment_requests (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    payment_request_id VARCHAR(255) UNIQUE NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    amount BIGINT NOT NULL, -- 整数値（小数点なし）
    currency VARCHAR(10) NOT NULL DEFAULT 'JPY',
    currency_type ENUM('paid', 'free') NOT NULL,
    status ENUM('pending', 'completed', 'failed', 'cancelled') NOT NULL,
    payment_method_data JSON, -- PaymentRequestのmethodData
    details JSON, -- PaymentRequestのdetails
    response JSON, -- PaymentResponse
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(user_id),
    INDEX idx_payment_request_id (payment_request_id),
    INDEX idx_user_id (user_id),
    INDEX idx_status (status)
);
```

### redemption_codes（引き換えコード）

```sql
CREATE TABLE redemption_codes (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    code VARCHAR(255) UNIQUE NOT NULL,
    code_type ENUM('promotion', 'gift', 'event') NOT NULL,
    currency_type ENUM('paid', 'free') NOT NULL,
    amount BIGINT NOT NULL, -- 整数値（小数点なし）
    max_uses INT NOT NULL DEFAULT 0, -- 0 = 無制限
    current_uses INT NOT NULL DEFAULT 0,
    valid_from TIMESTAMP NOT NULL,
    valid_until TIMESTAMP NOT NULL,
    status ENUM('active', 'expired', 'disabled') NOT NULL DEFAULT 'active',
    metadata JSON, -- 追加情報（説明、キャンペーンIDなど）
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_code (code),
    INDEX idx_status (status),
    INDEX idx_valid_until (valid_until)
);
```

### code_redemptions（コード引き換え履歴）

```sql
CREATE TABLE code_redemptions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    redemption_id VARCHAR(255) UNIQUE NOT NULL,
    code VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    transaction_id VARCHAR(255) NOT NULL, -- transactionsテーブルへの参照
    redeemed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(user_id),
    FOREIGN KEY (code) REFERENCES redemption_codes(code),
    FOREIGN KEY (transaction_id) REFERENCES transactions(transaction_id),
    INDEX idx_code (code),
    INDEX idx_user_id (user_id),
    INDEX idx_transaction_id (transaction_id),
    UNIQUE KEY uk_user_code (user_id, code) -- 同一ユーザーが同じコードを複数回引き換えできないように
);
```
