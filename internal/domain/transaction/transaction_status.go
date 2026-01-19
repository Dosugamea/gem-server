package transaction

import (
	"fmt"
)

// TransactionStatus トランザクションステータスを表す値オブジェクト
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"   // 処理中
	TransactionStatusCompleted TransactionStatus = "completed" // 完了
	TransactionStatusFailed    TransactionStatus = "failed"    // 失敗
	TransactionStatusCancelled TransactionStatus = "cancelled" // キャンセル
)

// NewTransactionStatus 新しいTransactionStatusを作成
func NewTransactionStatus(s string) (TransactionStatus, error) {
	switch s {
	case "pending", "completed", "failed", "cancelled":
		return TransactionStatus(s), nil
	default:
		return "", fmt.Errorf("invalid transaction status: %s", s)
	}
}

// String 文字列表現を返す
func (ts TransactionStatus) String() string {
	return string(ts)
}

// Valid 有効なトランザクションステータスかどうかを返す
func (ts TransactionStatus) Valid() bool {
	switch ts {
	case TransactionStatusPending, TransactionStatusCompleted, TransactionStatusFailed, TransactionStatusCancelled:
		return true
	default:
		return false
	}
}

// IsCompleted 完了状態かどうかを返す
func (ts TransactionStatus) IsCompleted() bool {
	return ts == TransactionStatusCompleted
}

// IsFailed 失敗状態かどうかを返す
func (ts TransactionStatus) IsFailed() bool {
	return ts == TransactionStatusFailed
}
