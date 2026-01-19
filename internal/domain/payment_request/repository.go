package payment_request

import (
	"context"
)

// PaymentRequestRepository PaymentRequestリポジトリインターフェース
type PaymentRequestRepository interface {
	// Save PaymentRequestを保存
	Save(ctx context.Context, paymentRequest *PaymentRequest) error
	
	// FindByPaymentRequestID PaymentRequest IDでPaymentRequestを取得
	FindByPaymentRequestID(ctx context.Context, paymentRequestID string) (*PaymentRequest, error)
	
	// Update PaymentRequestを更新
	Update(ctx context.Context, paymentRequest *PaymentRequest) error
}
