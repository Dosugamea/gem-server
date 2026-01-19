package currency

// GetBalanceRequest 残高取得リクエスト
type GetBalanceRequest struct {
	UserID string
}

// GetBalanceResponse 残高取得レスポンス
type GetBalanceResponse struct {
	UserID   string
	Balances map[string]int64 // "paid" => 1000, "free" => 500
}

// GrantRequest 通貨付与リクエスト
type GrantRequest struct {
	UserID       string
	CurrencyType string // "paid" or "free"
	Amount       int64
	Reason       string
	Metadata     map[string]interface{}
}

// GrantResponse 通貨付与レスポンス
type GrantResponse struct {
	TransactionID string
	BalanceAfter  int64
	Status        string
}

// ConsumeRequest 通貨消費リクエスト
type ConsumeRequest struct {
	UserID       string
	CurrencyType string // "paid", "free", or "auto"
	Amount       int64
	ItemID       string
	UsePriority  bool // 優先順位制御（無料通貨優先）
	Metadata     map[string]interface{}
}

// ConsumeResponse 通貨消費レスポンス
type ConsumeResponse struct {
	TransactionID      string
	ConsumptionDetails []ConsumptionDetail // 優先順位制御使用時
	BalanceAfter       int64               // 単一通貨タイプ消費時
	TotalConsumed      int64               // 優先順位制御使用時
	Status             string
}

// ConsumptionDetail 消費詳細
type ConsumptionDetail struct {
	CurrencyType  string
	Amount        int64
	BalanceBefore int64
	BalanceAfter  int64
}
