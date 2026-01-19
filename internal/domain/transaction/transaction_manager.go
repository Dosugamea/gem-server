package transaction

import (
	"context"
	"database/sql"
)

// TransactionManager トランザクション管理インターフェース
type TransactionManager interface {
	// WithTransaction トランザクション内で関数を実行
	WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error
}
