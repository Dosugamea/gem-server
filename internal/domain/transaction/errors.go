package transaction

import "errors"

var (
	// ErrTransactionNotFound トランザクションが見つからないエラー
	ErrTransactionNotFound = errors.New("transaction not found")
	// ErrInvalidTransaction 無効なトランザクションエラー
	ErrInvalidTransaction = errors.New("invalid transaction")
	// ErrDuplicateTransactionID 重複トランザクションIDエラー
	ErrDuplicateTransactionID = errors.New("duplicate transaction id")
)
