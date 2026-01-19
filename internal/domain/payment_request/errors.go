package payment_request

import "errors"

var (
	// ErrPaymentRequestNotFound PaymentRequestが見つからないエラー
	ErrPaymentRequestNotFound = errors.New("payment request not found")
	// ErrPaymentRequestAlreadyProcessed 既に処理済みエラー
	ErrPaymentRequestAlreadyProcessed = errors.New("payment request already processed")
	// ErrInvalidPaymentRequest 無効なPaymentRequestエラー
	ErrInvalidPaymentRequest = errors.New("invalid payment request")
)
