package transaction

import (
	"fmt"
)

// TransactionType トランザクションタイプを表す値オブジェクト
type TransactionType string

const (
	TransactionTypeGrant      TransactionType = "grant"      // 付与
	TransactionTypeConsume    TransactionType = "consume"    // 消費
	TransactionTypeRefund     TransactionType = "refund"     // 返金
	TransactionTypeExpire     TransactionType = "expire"     // 失効
	TransactionTypeCompensate TransactionType = "compensate" // 補填
)

// NewTransactionType 新しいTransactionTypeを作成
func NewTransactionType(s string) (TransactionType, error) {
	switch s {
	case "grant", "consume", "refund", "expire", "compensate":
		return TransactionType(s), nil
	default:
		return "", fmt.Errorf("invalid transaction type: %s", s)
	}
}

// String 文字列表現を返す
func (tt TransactionType) String() string {
	return string(tt)
}

// Valid 有効なトランザクションタイプかどうかを返す
func (tt TransactionType) Valid() bool {
	switch tt {
	case TransactionTypeGrant, TransactionTypeConsume, TransactionTypeRefund, TransactionTypeExpire, TransactionTypeCompensate:
		return true
	default:
		return false
	}
}
