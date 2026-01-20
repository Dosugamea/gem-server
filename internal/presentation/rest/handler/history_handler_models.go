package handler

// TransactionItem トランザクションアイテム
// @Description トランザクションアイテム
type TransactionItem struct {
	TransactionID   string `json:"transaction_id" example:"txn_123"`
	TransactionType string `json:"transaction_type" example:"consume"`
	CurrencyType    string `json:"currency_type" example:"paid"`
	Amount          string `json:"amount" example:"100"`
	BalanceBefore   string `json:"balance_before" example:"1000"`
	BalanceAfter    string `json:"balance_after" example:"900"`
	Status          string `json:"status" example:"completed"`
	CreatedAt       string `json:"created_at" example:"2024-01-01T12:00:00Z"`
}

// TransactionHistoryResponse トランザクション履歴レスポンス
// @Description トランザクション履歴レスポンス
type TransactionHistoryResponse struct {
	Transactions []TransactionItem `json:"transactions"`
	Total        int               `json:"total" example:"1"`
	Limit        int               `json:"limit" example:"50"`
	Offset       int               `json:"offset" example:"0"`
}
