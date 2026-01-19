-- 初期スキーマの作成

-- users（ユーザー）
CREATE TABLE users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- currency_balances（通貨残高）
CREATE TABLE currency_balances (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id VARCHAR(255) NOT NULL,
    currency_type ENUM('paid', 'free') NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0 COMMENT '整数値（小数点なし）、マイナス値を許可',
    version INT NOT NULL DEFAULT 0 COMMENT '楽観的ロック用',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_currency (user_id, currency_type),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- transactions（トランザクション履歴）
CREATE TABLE transactions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    transaction_id VARCHAR(255) UNIQUE NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    transaction_type ENUM('grant', 'consume', 'refund', 'expire', 'compensate') NOT NULL,
    currency_type ENUM('paid', 'free') NOT NULL,
    amount BIGINT NOT NULL COMMENT '整数値（小数点なし）',
    balance_before BIGINT NOT NULL COMMENT '整数値（小数点なし）',
    balance_after BIGINT NOT NULL COMMENT '整数値（小数点なし）',
    status ENUM('pending', 'completed', 'failed', 'cancelled') NOT NULL,
    payment_request_id VARCHAR(255) COMMENT 'PaymentRequest APIのID',
    metadata JSON COMMENT '追加情報（商品ID、理由など）',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    INDEX idx_transaction_id (transaction_id),
    INDEX idx_payment_request_id (payment_request_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- payment_requests（PaymentRequest記録）
CREATE TABLE payment_requests (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    payment_request_id VARCHAR(255) UNIQUE NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    amount BIGINT NOT NULL COMMENT '整数値（小数点なし）',
    currency VARCHAR(10) NOT NULL DEFAULT 'JPY',
    currency_type ENUM('paid', 'free') NOT NULL,
    status ENUM('pending', 'completed', 'failed', 'cancelled') NOT NULL,
    payment_method_data JSON COMMENT 'PaymentRequestのmethodData',
    details JSON COMMENT 'PaymentRequestのdetails',
    response JSON COMMENT 'PaymentResponse',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    INDEX idx_payment_request_id (payment_request_id),
    INDEX idx_user_id (user_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- redemption_codes（引き換えコード）
CREATE TABLE redemption_codes (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    code VARCHAR(255) UNIQUE NOT NULL,
    code_type ENUM('promotion', 'gift', 'event') NOT NULL,
    currency_type ENUM('paid', 'free') NOT NULL,
    amount BIGINT NOT NULL COMMENT '整数値（小数点なし）',
    max_uses INT NOT NULL DEFAULT 0 COMMENT '0 = 無制限',
    current_uses INT NOT NULL DEFAULT 0,
    valid_from TIMESTAMP NOT NULL,
    valid_until TIMESTAMP NOT NULL,
    status ENUM('active', 'expired', 'disabled') NOT NULL DEFAULT 'active',
    metadata JSON COMMENT '追加情報（説明、キャンペーンIDなど）',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_code (code),
    INDEX idx_status (status),
    INDEX idx_valid_until (valid_until)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- code_redemptions（コード引き換え履歴）
CREATE TABLE code_redemptions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    redemption_id VARCHAR(255) UNIQUE NOT NULL,
    code VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    transaction_id VARCHAR(255) NOT NULL COMMENT 'transactionsテーブルへの参照',
    redeemed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    FOREIGN KEY (code) REFERENCES redemption_codes(code) ON DELETE RESTRICT,
    FOREIGN KEY (transaction_id) REFERENCES transactions(transaction_id) ON DELETE RESTRICT,
    INDEX idx_code (code),
    INDEX idx_user_id (user_id),
    INDEX idx_transaction_id (transaction_id),
    UNIQUE KEY uk_user_code (user_id, code) COMMENT '同一ユーザーが同じコードを複数回引き換えできないように'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
