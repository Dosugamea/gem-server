package transaction

import (
	"errors"
	"gem-server/internal/domain/currency"
	"regexp"
	"time"
)

var (
	// ErrInvalidTransactionID トランザクションIDが無効
	ErrInvalidTransactionID = errors.New("invalid transaction id")
	// ErrInvalidUserID ユーザーIDが無効
	ErrInvalidUserID = errors.New("invalid user id")
	// ErrInvalidAmount 金額が無効
	ErrInvalidAmount = errors.New("invalid amount")
	// ErrAmountTooLarge 金額が大きすぎる
	ErrAmountTooLarge = errors.New("amount too large")
	// ErrBalanceOutOfRange 残高が範囲外
	ErrBalanceOutOfRange = errors.New("balance out of range")
)

const (
	// MaxAmount 最大金額 (10兆)
	MaxAmount = 10_000_000_000_000
	// MinBalance 最小残高 (-10兆)
	MinBalance = -10_000_000_000_000
)

var (
	idRegex     = regexp.MustCompile(`^[a-zA-Z0-9_\-\.\@]{1,255}$`)
	userIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-\.\@]{1,255}$`)
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
	requester        *string // リクエスト元（サービス名やユーザーIDなど）
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
) (*Transaction, error) {
	return NewTransactionWithRequester(
		transactionID,
		userID,
		transactionType,
		currencyType,
		amount,
		balanceBefore,
		balanceAfter,
		status,
		nil,
		metadata,
	)
}

// NewTransactionWithRequester 新しいTransactionエンティティを作成（requester指定あり）
func NewTransactionWithRequester(
	transactionID string,
	userID string,
	transactionType TransactionType,
	currencyType currency.CurrencyType,
	amount int64,
	balanceBefore int64,
	balanceAfter int64,
	status TransactionStatus,
	requester *string,
	metadata map[string]interface{},
) (*Transaction, error) {
	if !idRegex.MatchString(transactionID) {
		return nil, ErrInvalidTransactionID
	}
	if !userIDRegex.MatchString(userID) {
		return nil, ErrInvalidUserID
	}
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if amount > MaxAmount {
		return nil, ErrAmountTooLarge
	}
	if balanceBefore < MinBalance || balanceBefore > MaxAmount {
		return nil, ErrBalanceOutOfRange
	}
	if balanceAfter < MinBalance || balanceAfter > MaxAmount {
		return nil, ErrBalanceOutOfRange
	}

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
		requester:        requester,
		metadata:         metadata,
		createdAt:        now,
		updatedAt:        now,
	}, nil
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

// Requester リクエスト元を返す
func (t *Transaction) Requester() *string {
	return t.requester
}

// SetRequester リクエスト元を設定
func (t *Transaction) SetRequester(requester string) {
	t.requester = &requester
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

// MustNewTransaction テスト用ヘルパー: NewTransactionを呼び出し、エラーが発生した場合はpanicする
func MustNewTransaction(
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
	tx, err := NewTransaction(transactionID, userID, transactionType, currencyType, amount, balanceBefore, balanceAfter, status, metadata)
	if err != nil {
		panic(err)
	}
	return tx
}

// MustNewTransactionWithRequester テスト用ヘルパー: NewTransactionWithRequesterを呼び出し、エラーが発生した場合はpanicする
func MustNewTransactionWithRequester(
	transactionID string,
	userID string,
	transactionType TransactionType,
	currencyType currency.CurrencyType,
	amount int64,
	balanceBefore int64,
	balanceAfter int64,
	status TransactionStatus,
	requester *string,
	metadata map[string]interface{},
) *Transaction {
	tx, err := NewTransactionWithRequester(transactionID, userID, transactionType, currencyType, amount, balanceBefore, balanceAfter, status, requester, metadata)
	if err != nil {
		panic(err)
	}
	return tx
}
