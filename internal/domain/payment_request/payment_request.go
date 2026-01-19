package payment_request

import (
	"gem-server/internal/domain/currency"
	"time"
)

// PaymentRequest PaymentRequestエンティティ
type PaymentRequest struct {
	paymentRequestID  string
	userID            string
	amount            int64  // 整数値（小数点なし）
	currency          string // 通貨コード（例: "JPY"）
	currencyType      currency.CurrencyType
	status            PaymentRequestStatus
	paymentMethodData map[string]interface{} // PaymentRequestのmethodData
	details           map[string]interface{} // PaymentRequestのdetails
	response          map[string]interface{} // PaymentResponse
	createdAt         time.Time
	updatedAt         time.Time
}

// PaymentRequestStatus PaymentRequestのステータス
type PaymentRequestStatus string

const (
	PaymentRequestStatusPending   PaymentRequestStatus = "pending"   // 処理中
	PaymentRequestStatusCompleted PaymentRequestStatus = "completed" // 完了
	PaymentRequestStatusFailed    PaymentRequestStatus = "failed"    // 失敗
	PaymentRequestStatusCancelled PaymentRequestStatus = "cancelled" // キャンセル
)

// String 文字列表現を返す
func (prs PaymentRequestStatus) String() string {
	return string(prs)
}

// NewPaymentRequest 新しいPaymentRequestエンティティを作成
func NewPaymentRequest(
	paymentRequestID string,
	userID string,
	amount int64,
	currency string,
	currencyType currency.CurrencyType,
) *PaymentRequest {
	now := time.Now()
	return &PaymentRequest{
		paymentRequestID:  paymentRequestID,
		userID:            userID,
		amount:            amount,
		currency:          currency,
		currencyType:      currencyType,
		status:            PaymentRequestStatusPending,
		paymentMethodData: make(map[string]interface{}),
		details:           make(map[string]interface{}),
		response:          make(map[string]interface{}),
		createdAt:         now,
		updatedAt:         now,
	}
}

// PaymentRequestID PaymentRequest IDを返す
func (pr *PaymentRequest) PaymentRequestID() string {
	return pr.paymentRequestID
}

// UserID ユーザーIDを返す
func (pr *PaymentRequest) UserID() string {
	return pr.userID
}

// Amount 金額を返す
func (pr *PaymentRequest) Amount() int64 {
	return pr.amount
}

// Currency 通貨コードを返す
func (pr *PaymentRequest) Currency() string {
	return pr.currency
}

// CurrencyType 通貨タイプを返す
func (pr *PaymentRequest) CurrencyType() currency.CurrencyType {
	return pr.currencyType
}

// Status ステータスを返す
func (pr *PaymentRequest) Status() PaymentRequestStatus {
	return pr.status
}

// PaymentMethodData PaymentMethodDataを返す
func (pr *PaymentRequest) PaymentMethodData() map[string]interface{} {
	return pr.paymentMethodData
}

// Details Detailsを返す
func (pr *PaymentRequest) Details() map[string]interface{} {
	return pr.details
}

// Response Responseを返す
func (pr *PaymentRequest) Response() map[string]interface{} {
	return pr.response
}

// CreatedAt 作成日時を返す
func (pr *PaymentRequest) CreatedAt() time.Time {
	return pr.createdAt
}

// UpdatedAt 更新日時を返す
func (pr *PaymentRequest) UpdatedAt() time.Time {
	return pr.updatedAt
}

// SetPaymentMethodData PaymentMethodDataを設定
func (pr *PaymentRequest) SetPaymentMethodData(data map[string]interface{}) {
	pr.paymentMethodData = data
	pr.updatedAt = time.Now()
}

// SetDetails Detailsを設定
func (pr *PaymentRequest) SetDetails(details map[string]interface{}) {
	pr.details = details
	pr.updatedAt = time.Now()
}

// SetResponse Responseを設定
func (pr *PaymentRequest) SetResponse(response map[string]interface{}) {
	pr.response = response
	pr.updatedAt = time.Now()
}

// Complete 決済を完了状態にする
func (pr *PaymentRequest) Complete() {
	pr.status = PaymentRequestStatusCompleted
	pr.updatedAt = time.Now()
}

// Fail 決済を失敗状態にする
func (pr *PaymentRequest) Fail() {
	pr.status = PaymentRequestStatusFailed
	pr.updatedAt = time.Now()
}

// Cancel 決済をキャンセル状態にする
func (pr *PaymentRequest) Cancel() {
	pr.status = PaymentRequestStatusCancelled
	pr.updatedAt = time.Now()
}

// IsCompleted 完了状態かどうかを返す
func (pr *PaymentRequest) IsCompleted() bool {
	return pr.status == PaymentRequestStatusCompleted
}

// IsPending 処理中状態かどうかを返す
func (pr *PaymentRequest) IsPending() bool {
	return pr.status == PaymentRequestStatusPending
}
