package currency

// Currency 通貨エンティティ
type Currency struct {
	userID       string
	currencyType CurrencyType
	balance      int64 // 整数値（小数点なし）、マイナス値を許可
	version      int   // 楽観的ロック用
}

// NewCurrency 新しいCurrencyエンティティを作成
func NewCurrency(userID string, currencyType CurrencyType, balance int64, version int) *Currency {
	return &Currency{
		userID:       userID,
		currencyType: currencyType,
		balance:      balance,
		version:      version,
	}
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
	c.balance += amount
	c.version++
	return nil
}

// Consume 通貨を消費する（マイナス残高を許可しないバージョン）
func (c *Currency) Consume(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
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
	c.balance -= amount
	c.version++
	return nil
}

// IncrementVersion バージョンをインクリメント（楽観的ロック用）
func (c *Currency) IncrementVersion() {
	c.version++
}
