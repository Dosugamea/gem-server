package handler

// BalanceItem 残高アイテム
// @Description 残高アイテム
type BalanceItem struct {
	Paid string `json:"paid" example:"1000"`
	Free string `json:"free" example:"500"`
}

// BalanceResponse 残高レスポンス
// @Description 残高レスポンス
type BalanceResponse struct {
	UserID   string      `json:"user_id" example:"user123"`
	Balances BalanceItem `json:"balances"`
}

// GrantRequest 通貨付与リクエスト
// @Description 通貨付与リクエスト
type GrantRequest struct {
	CurrencyType string                 `json:"currency_type" example:"free" enums:"paid,free"`
	Amount       string                 `json:"amount" example:"100"`
	Reason       string                 `json:"reason" example:"イベント報酬"`
	Requester    string                 `json:"requester" example:"game-server-01"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// GrantResponse 通貨付与レスポンス
// @Description 通貨付与レスポンス
type GrantResponse struct {
	TransactionID string `json:"transaction_id" example:"txn_123"`
	BalanceAfter  string `json:"balance_after" example:"600"`
	Status        string `json:"status" example:"completed"`
}

// ConsumeRequest 通貨消費リクエスト
// @Description 通貨消費リクエスト
type ConsumeRequest struct {
	CurrencyType string                 `json:"currency_type" example:"paid" enums:"paid,free,auto"`
	Amount       string                 `json:"amount" example:"50"`
	ItemID       string                 `json:"item_id" example:"item001"`
	UsePriority  bool                   `json:"use_priority" example:"false"`
	Requester    string                 `json:"requester" example:"game-server-01"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ConsumptionDetail 消費詳細
// @Description 消費詳細
type ConsumptionDetail struct {
	CurrencyType  string `json:"currency_type" example:"paid"`
	Amount        string `json:"amount" example:"50"`
	BalanceBefore string `json:"balance_before" example:"1000"`
	BalanceAfter  string `json:"balance_after" example:"950"`
}

// ConsumeResponse 通貨消費レスポンス
// @Description 通貨消費レスポンス
type ConsumeResponse struct {
	TransactionID      string              `json:"transaction_id" example:"txn_456"`
	ConsumptionDetails []ConsumptionDetail `json:"consumption_details,omitempty"`
	BalanceAfter       string              `json:"balance_after,omitempty" example:"950"`
	TotalConsumed      string              `json:"total_consumed,omitempty" example:"50"`
	Status             string              `json:"status" example:"completed"`
}
