package history

import "gem-server/internal/domain/transaction"

// GetTransactionHistoryRequest トランザクション履歴取得リクエスト
type GetTransactionHistoryRequest struct {
	UserID          string
	Limit           int
	Offset          int
	CurrencyType    string // optional: "paid" or "free"
	TransactionType string // optional: "grant", "consume", etc.
}

// GetTransactionHistoryResponse トランザクション履歴取得レスポンス
type GetTransactionHistoryResponse struct {
	Transactions []*transaction.Transaction
	Total         int
	Limit         int
	Offset        int
}
