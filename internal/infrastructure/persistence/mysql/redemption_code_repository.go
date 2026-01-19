package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"gem-server/internal/domain/currency"
	"gem-server/internal/domain/redemption_code"
)

// RedemptionCodeRepository MySQL実装のRedemptionCodeRepository
type RedemptionCodeRepository struct {
	db *DB
}

// NewRedemptionCodeRepository 新しいRedemptionCodeRepositoryを作成
func NewRedemptionCodeRepository(db *DB) *RedemptionCodeRepository {
	return &RedemptionCodeRepository{db: db}
}

// FindByCode コードで引き換えコードを取得
func (r *RedemptionCodeRepository) FindByCode(ctx context.Context, code string) (*redemption_code.RedemptionCode, error) {
	query := `
		SELECT 
			code, code_type, currency_type, amount,
			max_uses, current_uses, valid_from, valid_until,
			status, metadata, created_at, updated_at
		FROM redemption_codes
		WHERE code = ?
	`
	
	var dbCode, dbCodeType, dbCurrencyType, dbStatus string
	var amount int64
	var maxUses, currentUses int
	var validFrom, validUntil time.Time
	var metadataJSON sql.NullString
	var createdAt, updatedAt time.Time
	
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&dbCode,
		&dbCodeType,
		&dbCurrencyType,
		&amount,
		&maxUses,
		&currentUses,
		&validFrom,
		&validUntil,
		&dbStatus,
		&metadataJSON,
		&createdAt,
		&updatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, redemption_code.ErrCodeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find redemption code: %w", err)
	}
	
	ct, err := redemption_code.NewCodeType(dbCodeType)
	if err != nil {
		return nil, fmt.Errorf("invalid code type: %w", err)
	}
	
	currencyType, err := currency.NewCurrencyType(dbCurrencyType)
	if err != nil {
		return nil, fmt.Errorf("invalid currency type: %w", err)
	}
	
	status, err := redemption_code.NewCodeStatus(dbStatus)
	if err != nil {
		return nil, fmt.Errorf("invalid code status: %w", err)
	}
	
	var metadata map[string]interface{}
	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}
	
	rc := redemption_code.NewRedemptionCode(
		dbCode,
		ct,
		currencyType,
		amount,
		maxUses,
		validFrom,
		validUntil,
		metadata,
	)
	
	// current_usesとstatusを設定
	rc.SetCurrentUses(currentUses)
	rc.SetStatus(status)
	
	return rc, nil
}

// Update 引き換えコードを更新
func (r *RedemptionCodeRepository) Update(ctx context.Context, code *redemption_code.RedemptionCode) error {
	query := `
		UPDATE redemption_codes
		SET 
			current_uses = ?,
			status = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE code = ?
	`
	
	_, err := r.db.ExecContext(ctx, query,
		code.CurrentUses(),
		code.Status().String(),
		code.Code(),
	)
	
	if err != nil {
		return fmt.Errorf("failed to update redemption code: %w", err)
	}
	
	return nil
}

// HasUserRedeemed ユーザーが既にこのコードを引き換え済みかチェック
func (r *RedemptionCodeRepository) HasUserRedeemed(ctx context.Context, code string, userID string) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM code_redemptions
		WHERE code = ? AND user_id = ?
	`
	
	var count int
	err := r.db.QueryRowContext(ctx, query, code, userID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check redemption: %w", err)
	}
	
	return count > 0, nil
}

// SaveRedemption 引き換え履歴を保存
func (r *RedemptionCodeRepository) SaveRedemption(ctx context.Context, redemption *redemption_code.CodeRedemption) error {
	query := `
		INSERT INTO code_redemptions (
			redemption_id, code, user_id, transaction_id, redeemed_at
		) VALUES (?, ?, ?, ?, ?)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		redemption.RedemptionID(),
		redemption.Code(),
		redemption.UserID(),
		redemption.TransactionID(),
		redemption.RedeemedAt(),
	)
	
	if err != nil {
		return fmt.Errorf("failed to save redemption: %w", err)
	}
	
	return nil
}
