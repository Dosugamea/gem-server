package transaction

import (
	"gem-server/internal/domain/currency"
	"time"
)

// Transaction トランザクションエンティティ
type Transaction struct {
	transactionID    string
	userID           string
	transactionType  TransactionType
	currencyType     currency.CurrencyType
	amount           int64 // 整数値（小数点なし）
	balanceBefore    int64 // 整数値（小数点なし）
	balanceAfter     int64 // 整数値（小数点なし）
	status           TransactionStatus
	paymentRequestID *string // PaymentRequest APIのID（オプション）
	metadata         map[string]interface{}
	createdAt        time.Time
	updatedAt        time.Time
}

// NewTransaction 新しいTransactionエンティティを作成
func NewTransaction(
	transactionID string,
	userID string,
	transactionType TransactionType,
	currencyType currency.CurrencyType,
	amount int64,
	balanceBefore int64,
	balanceAfter int64,
	status TransactionStatus,
	metadata map[string]interface{},
) *Transaction {
	now := time.Now()
	return &Transaction{
		transactionID:    transactionID,
		userID:           userID,
		transactionType:  transactionType,
		currencyType:     currencyType,
		amount:           amount,
		balanceBefore:    balanceBefore,
		balanceAfter:     balanceAfter,
		status:           status,
		paymentRequestID: nil,
		metadata:         metadata,
		createdAt:        now,
		updatedAt:        now,
	}
}

// TransactionID トランザクションIDを返す
func (t *Transaction) TransactionID() string {
	return t.transactionID
}

// UserID ユーザーIDを返す
func (t *Transaction) UserID() string {
	return t.userID
}

// TransactionType トランザクションタイプを返す
func (t *Transaction) TransactionType() TransactionType {
	return t.transactionType
}

// CurrencyType 通貨タイプを返す
func (t *Transaction) CurrencyType() currency.CurrencyType {
	return t.currencyType
}

// Amount 金額を返す
func (t *Transaction) Amount() int64 {
	return t.amount
}

// BalanceBefore 処理前の残高を返す
func (t *Transaction) BalanceBefore() int64 {
	return t.balanceBefore
}

// BalanceAfter 処理後の残高を返す
func (t *Transaction) BalanceAfter() int64 {
	return t.balanceAfter
}

// Status ステータスを返す
func (t *Transaction) Status() TransactionStatus {
	return t.status
}

// PaymentRequestID PaymentRequest IDを返す
func (t *Transaction) PaymentRequestID() *string {
	return t.paymentRequestID
}

// Metadata メタデータを返す
func (t *Transaction) Metadata() map[string]interface{} {
	return t.metadata
}

// CreatedAt 作成日時を返す
func (t *Transaction) CreatedAt() time.Time {
	return t.createdAt
}

// UpdatedAt 更新日時を返す
func (t *Transaction) UpdatedAt() time.Time {
	return t.updatedAt
}

// SetPaymentRequestID PaymentRequest IDを設定
func (t *Transaction) SetPaymentRequestID(id string) {
	t.paymentRequestID = &id
	t.updatedAt = time.Now()
}

// UpdateStatus ステータスを更新
func (t *Transaction) UpdateStatus(status TransactionStatus) error {
	if !status.Valid() {
		return ErrInvalidTransaction
	}
	t.status = status
	t.updatedAt = time.Now()
	return nil
}
