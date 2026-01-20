package handler

// ProcessPaymentRequest 決済処理リクエスト
// @Description 決済処理リクエスト
type ProcessPaymentRequest struct {
	PaymentRequestID string            `json:"payment_request_id" example:"req_123"`
	MethodName       string            `json:"method_name" example:"credit_card"`
	Details          map[string]string `json:"details"`
	Amount           string            `json:"amount" example:"1000"`
	Currency         string            `json:"currency" example:"JPY"`
}

// ProcessPaymentResponse 決済処理レスポンス
// @Description 決済処理レスポンス
type ProcessPaymentResponse struct {
	TransactionID      string              `json:"transaction_id" example:"txn_789"`
	PaymentRequestID   string              `json:"payment_request_id" example:"req_123"`
	ConsumptionDetails []ConsumptionDetail `json:"consumption_details"`
	TotalConsumed      string              `json:"total_consumed" example:"1000"`
	Status             string              `json:"status" example:"completed"`
}
