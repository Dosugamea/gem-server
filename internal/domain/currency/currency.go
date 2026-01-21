package currency

import (
	"errors"
	"regexp"
)

var (
	// ErrInvalidUserID ユーザーIDが無効
	ErrInvalidUserID = errors.New("invalid user id")
	// ErrBalanceOutOfRange 残高が範囲外
	ErrBalanceOutOfRange = errors.New("balance out of range")
	// ErrAmountTooLarge 金額が大きすぎる
	ErrAmountTooLarge = errors.New("amount too large")
)

const (
	// MaxAmount 最大金額 (10兆)
	MaxAmount = 10_000_000_000_000
	// MinBalance 最小残高 (-10兆: 一時的なマイナス許容のため)
	MinBalance = -10_000_000_000_000
)

var userIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-\.\@]{1,255}$`)

// Currency 通貨エンティティ
type Currency struct {
	userID       string
	currencyType CurrencyType
	balance      int64 // 整数値（小数点なし）、マイナス値を許可
	version      int   // 楽観的ロック用
}

// NewCurrency 新しいCurrencyエンティティを作成
func NewCurrency(userID string, currencyType CurrencyType, balance int64, version int) (*Currency, error) {
	if !userIDRegex.MatchString(userID) {
		return nil, ErrInvalidUserID
	}
	if balance < MinBalance || balance > MaxAmount {
		return nil, ErrBalanceOutOfRange
	}
	return &Currency{
		userID:       userID,
		currencyType: currencyType,
		balance:      balance,
		version:      version,
	}, nil
}

// UserID ユーザーIDを返す
func (c *Currency) UserID() string {
	return c.userID
}

// CurrencyType 通貨タイプを返す
func (c *Currency) CurrencyType() CurrencyType {
	return c.currencyType
}

// Balance 残高を返す
func (c *Currency) Balance() int64 {
	return c.balance
}

// Version バージョンを返す（楽観的ロック用）
func (c *Currency) Version() int {
	return c.version
}

// Grant 通貨を付与する
func (c *Currency) Grant(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if amount > MaxAmount {
		return ErrAmountTooLarge
	}
	// オーバーフローチェック
	if c.balance > MaxAmount-amount {
		return ErrBalanceOutOfRange
	}
	c.balance += amount
	c.version++
	return nil
}

// Consume 通貨を消費する（マイナス残高を許可しないバージョン）
func (c *Currency) Consume(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if amount > MaxAmount {
		return ErrAmountTooLarge
	}
	if c.balance < amount {
		return ErrInsufficientBalance
	}
	c.balance -= amount
	c.version++
	return nil
}

// ConsumeAllowNegative 通貨を消費する（マイナス残高を許可するバージョン）
// 返金処理、補填処理、手動調整などでマイナス残高が発生する可能性がある場合に使用
func (c *Currency) ConsumeAllowNegative(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if amount > MaxAmount {
		return ErrAmountTooLarge
	}
	// アンダーフローチェック (MinBalanceを下回らないか)
	if c.balance < MinBalance+amount {
		return ErrBalanceOutOfRange
	}
	c.balance -= amount
	c.version++
	return nil
}

// IncrementVersion バージョンをインクリメント（楽観的ロック用）
func (c *Currency) IncrementVersion() {
	c.version++
}

// MustNewCurrency テスト用ヘルパー: NewCurrencyを呼び出し、エラーが発生した場合はpanicする
func MustNewCurrency(userID string, currencyType CurrencyType, balance int64, version int) *Currency {
	c, err := NewCurrency(userID, currencyType, balance, version)
	if err != nil {
		panic(err)
	}
	return c
}
