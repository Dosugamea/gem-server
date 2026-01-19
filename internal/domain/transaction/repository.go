package transaction

import (
	"context"
)

// TransactionRepository トランザクションリポジトリインターフェース
type TransactionRepository interface {
	// Save トランザクションを保存
	Save(ctx context.Context, transaction *Transaction) error

	// FindByTransactionID トランザクションIDでトランザクションを取得
	FindByTransactionID(ctx context.Context, transactionID string) (*Transaction, error)

	// FindByUserID ユーザーIDでトランザクション一覧を取得（ページネーション対応）
	FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*Transaction, error)

	// FindByPaymentRequestID PaymentRequest IDでトランザクションを取得
	FindByPaymentRequestID(ctx context.Context, paymentRequestID string) (*Transaction, error)
}
