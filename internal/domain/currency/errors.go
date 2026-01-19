package currency

import "errors"

var (
	// ErrInsufficientBalance 残高不足エラー
	ErrInsufficientBalance = errors.New("insufficient balance")
	// ErrInvalidAmount 無効な金額エラー
	ErrInvalidAmount = errors.New("invalid amount")
	// ErrDuplicateTransaction 重複トランザクションエラー
	ErrDuplicateTransaction = errors.New("duplicate transaction")
	// ErrCurrencyNotFound 通貨が見つからないエラー
	ErrCurrencyNotFound = errors.New("currency not found")
)
