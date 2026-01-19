package payment

// ProcessPaymentRequest 決済処理リクエスト
type ProcessPaymentRequest struct {
	PaymentRequestID string
	UserID           string
	MethodName       string
	Details          map[string]string
	Amount           int64
	Currency         string
}

// ProcessPaymentResponse 決済処理レスポンス
type ProcessPaymentResponse struct {
	TransactionID      string
	PaymentRequestID   string
	ConsumptionDetails []ConsumptionDetail
	TotalConsumed      int64
	Status             string
}

// ConsumptionDetail 消費詳細
type ConsumptionDetail struct {
	CurrencyType  string
	Amount        int64
	BalanceBefore int64
	BalanceAfter  int64
}
