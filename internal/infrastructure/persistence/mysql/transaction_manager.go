package mysql

import (
	"context"
	"database/sql"
)

// TransactionManager トランザクション管理を提供
type TransactionManager struct {
	db *DB
}

// NewTransactionManager 新しいトランザクションマネージャーを作成
func NewTransactionManager(db *DB) *TransactionManager {
	return &TransactionManager{db: db}
}

// WithTransaction トランザクション内で関数を実行
func (tm *TransactionManager) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := tm.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	err = fn(tx)
	return err
}
