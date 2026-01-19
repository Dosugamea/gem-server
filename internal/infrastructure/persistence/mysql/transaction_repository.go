package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"gem-server/internal/domain/currency"
	"gem-server/internal/domain/transaction"
)

// TransactionRepository MySQL実装のTransactionRepository
type TransactionRepository struct {
	db *DB
}

// NewTransactionRepository 新しいTransactionRepositoryを作成
func NewTransactionRepository(db *DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// Save トランザクションを保存
func (r *TransactionRepository) Save(ctx context.Context, t *transaction.Transaction) error {
	query := `
		INSERT INTO transactions (
			transaction_id, user_id, transaction_type, currency_type,
			amount, balance_before, balance_after, status,
			payment_request_id, metadata, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			status = VALUES(status),
			updated_at = VALUES(updated_at)
	`
	
	var metadataJSON []byte
	var err error
	if t.Metadata() != nil {
		metadataJSON, err = json.Marshal(t.Metadata())
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}
	
	paymentRequestID := t.PaymentRequestID()
	var paymentRequestIDValue interface{}
	if paymentRequestID != nil {
		paymentRequestIDValue = *paymentRequestID
	}
	
	_, err = r.db.ExecContext(ctx, query,
		t.TransactionID(),
		t.UserID(),
		t.TransactionType().String(),
		t.CurrencyType().String(),
		t.Amount(),
		t.BalanceBefore(),
		t.BalanceAfter(),
		t.Status().String(),
		paymentRequestIDValue,
		string(metadataJSON),
		t.CreatedAt(),
		t.UpdatedAt(),
	)
	
	if err != nil {
		return fmt.Errorf("failed to save transaction: %w", err)
	}
	
	return nil
}

// FindByTransactionID トランザクションIDでトランザクションを取得
func (r *TransactionRepository) FindByTransactionID(ctx context.Context, transactionID string) (*transaction.Transaction, error) {
	query := `
		SELECT 
			transaction_id, user_id, transaction_type, currency_type,
			amount, balance_before, balance_after, status,
			payment_request_id, metadata, created_at, updated_at
		FROM transactions
		WHERE transaction_id = ?
	`
	
	var dbTransactionID, dbUserID, dbTransactionType, dbCurrencyType string
	var amount, balanceBefore, balanceAfter int64
	var dbStatus string
	var paymentRequestID sql.NullString
	var metadataJSON sql.NullString
	var createdAt, updatedAt time.Time
	
	err := r.db.QueryRowContext(ctx, query, transactionID).Scan(
		&dbTransactionID,
		&dbUserID,
		&dbTransactionType,
		&dbCurrencyType,
		&amount,
		&balanceBefore,
		&balanceAfter,
		&dbStatus,
		&paymentRequestID,
		&metadataJSON,
		&createdAt,
		&updatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, transaction.ErrTransactionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find transaction: %w", err)
	}
	
	tt, err := transaction.NewTransactionType(dbTransactionType)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction type: %w", err)
	}
	
	ct, err := currency.NewCurrencyType(dbCurrencyType)
	if err != nil {
		return nil, fmt.Errorf("invalid currency type: %w", err)
	}
	
	ts, err := transaction.NewTransactionStatus(dbStatus)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction status: %w", err)
	}
	
	var metadata map[string]interface{}
	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}
	
	t := transaction.NewTransaction(
		dbTransactionID,
		dbUserID,
		tt,
		ct,
		amount,
		balanceBefore,
		balanceAfter,
		ts,
		metadata,
	)
	
	if paymentRequestID.Valid {
		t.SetPaymentRequestID(paymentRequestID.String)
	}
	
	return t, nil
}

// FindByUserID ユーザーIDでトランザクション一覧を取得（ページネーション対応）
func (r *TransactionRepository) FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*transaction.Transaction, error) {
	query := `
		SELECT 
			transaction_id, user_id, transaction_type, currency_type,
			amount, balance_before, balance_after, status,
			payment_request_id, metadata, created_at, updated_at
		FROM transactions
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	
	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()
	
	var transactions []*transaction.Transaction
	for rows.Next() {
		var dbTransactionID, dbUserID, dbTransactionType, dbCurrencyType string
		var amount, balanceBefore, balanceAfter int64
		var dbStatus string
		var paymentRequestID sql.NullString
		var metadataJSON sql.NullString
		var createdAt, updatedAt time.Time
		
		if err := rows.Scan(
			&dbTransactionID,
			&dbUserID,
			&dbTransactionType,
			&dbCurrencyType,
			&amount,
			&balanceBefore,
			&balanceAfter,
			&dbStatus,
			&paymentRequestID,
			&metadataJSON,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		
		tt, err := transaction.NewTransactionType(dbTransactionType)
		if err != nil {
			return nil, fmt.Errorf("invalid transaction type: %w", err)
		}
		
		ct, err := currency.NewCurrencyType(dbCurrencyType)
		if err != nil {
			return nil, fmt.Errorf("invalid currency type: %w", err)
		}
		
		ts, err := transaction.NewTransactionStatus(dbStatus)
		if err != nil {
			return nil, fmt.Errorf("invalid transaction status: %w", err)
		}
		
		var metadata map[string]interface{}
		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}
		
		t := transaction.NewTransaction(
			dbTransactionID,
			dbUserID,
			tt,
			ct,
			amount,
			balanceBefore,
			balanceAfter,
			ts,
			metadata,
		)
		
		if paymentRequestID.Valid {
			t.SetPaymentRequestID(paymentRequestID.String)
		}
		
		transactions = append(transactions, t)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate transactions: %w", err)
	}
	
	return transactions, nil
}

// FindByPaymentRequestID PaymentRequest IDでトランザクションを取得
func (r *TransactionRepository) FindByPaymentRequestID(ctx context.Context, paymentRequestID string) (*transaction.Transaction, error) {
	query := `
		SELECT 
			transaction_id, user_id, transaction_type, currency_type,
			amount, balance_before, balance_after, status,
			payment_request_id, metadata, created_at, updated_at
		FROM transactions
		WHERE payment_request_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`
	
	var dbTransactionID, dbUserID, dbTransactionType, dbCurrencyType string
	var amount, balanceBefore, balanceAfter int64
	var dbStatus string
	var paymentRequestIDValue sql.NullString
	var metadataJSON sql.NullString
	var createdAt, updatedAt time.Time
	
	err := r.db.QueryRowContext(ctx, query, paymentRequestID).Scan(
		&dbTransactionID,
		&dbUserID,
		&dbTransactionType,
		&dbCurrencyType,
		&amount,
		&balanceBefore,
		&balanceAfter,
		&dbStatus,
		&paymentRequestIDValue,
		&metadataJSON,
		&createdAt,
		&updatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, transaction.ErrTransactionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find transaction: %w", err)
	}
	
	tt, err := transaction.NewTransactionType(dbTransactionType)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction type: %w", err)
	}
	
	ct, err := currency.NewCurrencyType(dbCurrencyType)
	if err != nil {
		return nil, fmt.Errorf("invalid currency type: %w", err)
	}
	
	ts, err := transaction.NewTransactionStatus(dbStatus)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction status: %w", err)
	}
	
	var metadata map[string]interface{}
	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}
	
	t := transaction.NewTransaction(
		dbTransactionID,
		dbUserID,
		tt,
		ct,
		amount,
		balanceBefore,
		balanceAfter,
		ts,
		metadata,
	)
	
	if paymentRequestIDValue.Valid {
		t.SetPaymentRequestID(paymentRequestIDValue.String)
	}
	
	return t, nil
}
