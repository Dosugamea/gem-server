package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"gem-server/internal/domain/currency"
	"gem-server/internal/domain/payment_request"
)

// PaymentRequestRepository MySQL実装のPaymentRequestRepository
type PaymentRequestRepository struct {
	db *DB
}

// NewPaymentRequestRepository 新しいPaymentRequestRepositoryを作成
func NewPaymentRequestRepository(db *DB) *PaymentRequestRepository {
	return &PaymentRequestRepository{db: db}
}

// Save PaymentRequestを保存
func (r *PaymentRequestRepository) Save(ctx context.Context, pr *payment_request.PaymentRequest) error {
	query := `
		INSERT INTO payment_requests (
			payment_request_id, user_id, amount, currency, currency_type,
			status, payment_method_data, details, response,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			status = VALUES(status),
			payment_method_data = VALUES(payment_method_data),
			details = VALUES(details),
			response = VALUES(response),
			updated_at = VALUES(updated_at)
	`
	
	paymentMethodDataJSON, err := json.Marshal(pr.PaymentMethodData())
	if err != nil {
		return fmt.Errorf("failed to marshal payment_method_data: %w", err)
	}
	
	detailsJSON, err := json.Marshal(pr.Details())
	if err != nil {
		return fmt.Errorf("failed to marshal details: %w", err)
	}
	
	responseJSON, err := json.Marshal(pr.Response())
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}
	
	_, err = r.db.ExecContext(ctx, query,
		pr.PaymentRequestID(),
		pr.UserID(),
		pr.Amount(),
		pr.Currency(),
		pr.CurrencyType().String(),
		pr.Status().String(),
		string(paymentMethodDataJSON),
		string(detailsJSON),
		string(responseJSON),
		pr.CreatedAt(),
		pr.UpdatedAt(),
	)
	
	if err != nil {
		return fmt.Errorf("failed to save payment request: %w", err)
	}
	
	return nil
}

// FindByPaymentRequestID PaymentRequest IDでPaymentRequestを取得
func (r *PaymentRequestRepository) FindByPaymentRequestID(ctx context.Context, paymentRequestID string) (*payment_request.PaymentRequest, error) {
	query := `
		SELECT 
			payment_request_id, user_id, amount, currency, currency_type,
			status, payment_method_data, details, response,
			created_at, updated_at
		FROM payment_requests
		WHERE payment_request_id = ?
	`
	
	var dbPaymentRequestID, dbUserID, dbCurrency, dbCurrencyType, dbStatus string
	var amount int64
	var paymentMethodDataJSON, detailsJSON, responseJSON sql.NullString
	var createdAt, updatedAt time.Time
	
	err := r.db.QueryRowContext(ctx, query, paymentRequestID).Scan(
		&dbPaymentRequestID,
		&dbUserID,
		&amount,
		&dbCurrency,
		&dbCurrencyType,
		&dbStatus,
		&paymentMethodDataJSON,
		&detailsJSON,
		&responseJSON,
		&createdAt,
		&updatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, payment_request.ErrPaymentRequestNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find payment request: %w", err)
	}
	
	ct, err := currency.NewCurrencyType(dbCurrencyType)
	if err != nil {
		return nil, fmt.Errorf("invalid currency type: %w", err)
	}
	
	pr := payment_request.NewPaymentRequest(
		dbPaymentRequestID,
		dbUserID,
		amount,
		dbCurrency,
		ct,
	)
	
	// ステータスを設定
	switch dbStatus {
	case "pending":
		// デフォルトでpendingなので何もしない
	case "completed":
		pr.Complete()
	case "failed":
		pr.Fail()
	case "cancelled":
		pr.Cancel()
	}
	
	// JSONデータを設定
	if paymentMethodDataJSON.Valid && paymentMethodDataJSON.String != "" {
		var paymentMethodData map[string]interface{}
		if err := json.Unmarshal([]byte(paymentMethodDataJSON.String), &paymentMethodData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payment_method_data: %w", err)
		}
		pr.SetPaymentMethodData(paymentMethodData)
	}
	
	if detailsJSON.Valid && detailsJSON.String != "" {
		var details map[string]interface{}
		if err := json.Unmarshal([]byte(detailsJSON.String), &details); err != nil {
			return nil, fmt.Errorf("failed to unmarshal details: %w", err)
		}
		pr.SetDetails(details)
	}
	
	if responseJSON.Valid && responseJSON.String != "" {
		var response map[string]interface{}
		if err := json.Unmarshal([]byte(responseJSON.String), &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}
		pr.SetResponse(response)
	}
	
	return pr, nil
}

// Update PaymentRequestを更新
func (r *PaymentRequestRepository) Update(ctx context.Context, pr *payment_request.PaymentRequest) error {
	return r.Save(ctx, pr)
}
