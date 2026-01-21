package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"gem-server/internal/domain/currency"
	"gem-server/internal/domain/redemption_code"
)

// RedemptionCodeRepository MySQL実装のRedemptionCodeRepository
type RedemptionCodeRepository struct {
	db     *DB
	tracer trace.Tracer
}

// NewRedemptionCodeRepository 新しいRedemptionCodeRepositoryを作成
func NewRedemptionCodeRepository(db *DB) *RedemptionCodeRepository {
	return &RedemptionCodeRepository{
		db:     db,
		tracer: otel.Tracer("redemption-code-repository"),
	}
}

// FindByCode コードで引き換えコードを取得
func (r *RedemptionCodeRepository) FindByCode(ctx context.Context, code string) (*redemption_code.RedemptionCode, error) {
	ctx, span := r.tracer.Start(ctx, "RedemptionCodeRepository.FindByCode")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.code", code),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "redemption_codes"),
	)

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
		span.SetStatus(otelcodes.Ok, "redemption code not found")
		return nil, redemption_code.ErrCodeNotFound
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, fmt.Errorf("failed to find redemption code: %w", err)
	}

	span.SetAttributes(
		attribute.String("db.code_type", dbCodeType),
		attribute.String("db.currency_type", dbCurrencyType),
		attribute.Int64("db.amount", amount),
		attribute.String("db.status", dbStatus),
	)
	span.SetStatus(otelcodes.Ok, "redemption code found")

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

	rc, err := redemption_code.NewRedemptionCode(
		dbCode,
		ct,
		currencyType,
		amount,
		maxUses,
		validFrom,
		validUntil,
		metadata,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create redemption code entity: %w", err)
	}

	// current_usesとstatusを設定
	rc.SetCurrentUses(currentUses)
	rc.SetStatus(status)

	return rc, nil
}

// Update 引き換えコードを更新
func (r *RedemptionCodeRepository) Update(ctx context.Context, code *redemption_code.RedemptionCode) error {
	ctx, span := r.tracer.Start(ctx, "RedemptionCodeRepository.Update")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.code", code.Code()),
		attribute.Int("db.current_uses", code.CurrentUses()),
		attribute.String("db.status", code.Status().String()),
		attribute.String("db.operation", "UPDATE"),
		attribute.String("db.table", "redemption_codes"),
	)

	query := `
		UPDATE redemption_codes
		SET 
			current_uses = ?,
			status = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE code = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		code.CurrentUses(),
		code.Status().String(),
		code.Code(),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return fmt.Errorf("failed to update redemption code: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	span.SetAttributes(attribute.Int64("db.rows_affected", rowsAffected))
	span.SetStatus(otelcodes.Ok, "redemption code updated")
	return nil
}

// HasUserRedeemed ユーザーが既にこのコードを引き換え済みかチェック
func (r *RedemptionCodeRepository) HasUserRedeemed(ctx context.Context, code string, userID string) (bool, error) {
	ctx, span := r.tracer.Start(ctx, "RedemptionCodeRepository.HasUserRedeemed")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.code", code),
		attribute.String("db.user_id", userID),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "code_redemptions"),
	)

	query := `
		SELECT COUNT(*) 
		FROM code_redemptions
		WHERE code = ? AND user_id = ?
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, code, userID).Scan(&count)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return false, fmt.Errorf("failed to check redemption: %w", err)
	}

	span.SetAttributes(attribute.Int("db.count", count))
	span.SetStatus(otelcodes.Ok, fmt.Sprintf("user redeemed: %v", count > 0))
	return count > 0, nil
}

// SaveRedemption 引き換え履歴を保存
func (r *RedemptionCodeRepository) SaveRedemption(ctx context.Context, redemption *redemption_code.CodeRedemption) error {
	ctx, span := r.tracer.Start(ctx, "RedemptionCodeRepository.SaveRedemption")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.redemption_id", redemption.RedemptionID()),
		attribute.String("db.code", redemption.Code()),
		attribute.String("db.user_id", redemption.UserID()),
		attribute.String("db.transaction_id", redemption.TransactionID()),
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.table", "code_redemptions"),
	)

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
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return fmt.Errorf("failed to save redemption: %w", err)
	}

	span.SetStatus(otelcodes.Ok, "redemption saved")
	return nil
}

// Create 引き換えコードを作成
func (r *RedemptionCodeRepository) Create(ctx context.Context, code *redemption_code.RedemptionCode) error {
	ctx, span := r.tracer.Start(ctx, "RedemptionCodeRepository.Create")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.code", code.Code()),
		attribute.String("db.code_type", code.CodeType().String()),
		attribute.String("db.currency_type", code.CurrencyType().String()),
		attribute.Int64("db.amount", code.Amount()),
		attribute.Int("db.max_uses", code.MaxUses()),
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.table", "redemption_codes"),
	)

	// メタデータをJSONに変換
	var metadataJSON sql.NullString
	if code.Metadata() != nil && len(code.Metadata()) > 0 {
		metadataBytes, err := json.Marshal(code.Metadata())
		if err != nil {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, err.Error())
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = sql.NullString{
			String: string(metadataBytes),
			Valid:  true,
		}
	}

	query := `
		INSERT INTO redemption_codes (
			code, code_type, currency_type, amount,
			max_uses, current_uses, valid_from, valid_until,
			status, metadata, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		code.Code(),
		code.CodeType().String(),
		code.CurrencyType().String(),
		code.Amount(),
		code.MaxUses(),
		code.CurrentUses(),
		code.ValidFrom(),
		code.ValidUntil(),
		code.Status().String(),
		metadataJSON,
		code.CreatedAt(),
		code.UpdatedAt(),
	)

	if err != nil {
		// MySQLの重複キーエラーをチェック
		if isDuplicateKeyError(err) {
			span.SetStatus(otelcodes.Error, "code already exists")
			return redemption_code.ErrCodeAlreadyExists
		}
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return fmt.Errorf("failed to create redemption code: %w", err)
	}

	span.SetStatus(otelcodes.Ok, "redemption code created")
	return nil
}

// Delete 引き換えコードを削除
func (r *RedemptionCodeRepository) Delete(ctx context.Context, code string) error {
	ctx, span := r.tracer.Start(ctx, "RedemptionCodeRepository.Delete")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.code", code),
		attribute.String("db.operation", "DELETE"),
		attribute.String("db.table", "redemption_codes"),
	)

	// まず、コードが使用されているかチェック
	checkQuery := `
		SELECT COUNT(*) 
		FROM code_redemptions
		WHERE code = ?
	`
	var count int
	err := r.db.QueryRowContext(ctx, checkQuery, code).Scan(&count)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return fmt.Errorf("failed to check code usage: %w", err)
	}

	if count > 0 {
		span.SetStatus(otelcodes.Error, "code has been used and cannot be deleted")
		return redemption_code.ErrCodeCannotBeDeleted
	}

	// コードを削除
	query := `DELETE FROM redemption_codes WHERE code = ?`

	result, err := r.db.ExecContext(ctx, query, code)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return fmt.Errorf("failed to delete redemption code: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		span.SetStatus(otelcodes.Ok, "redemption code not found")
		return redemption_code.ErrCodeNotFound
	}

	span.SetAttributes(attribute.Int64("db.rows_affected", rowsAffected))
	span.SetStatus(otelcodes.Ok, "redemption code deleted")
	return nil
}

// FindAll 引き換えコードの一覧を取得（ページネーション対応）
func (r *RedemptionCodeRepository) FindAll(ctx context.Context, limit, offset int) ([]*redemption_code.RedemptionCode, int, error) {
	ctx, span := r.tracer.Start(ctx, "RedemptionCodeRepository.FindAll")
	defer span.End()

	span.SetAttributes(
		attribute.Int("db.limit", limit),
		attribute.Int("db.offset", offset),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "redemption_codes"),
	)

	// 総件数を取得
	countQuery := `SELECT COUNT(*) FROM redemption_codes`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery).Scan(&total)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, 0, fmt.Errorf("failed to count redemption codes: %w", err)
	}

	// 一覧を取得
	query := `
		SELECT 
			code, code_type, currency_type, amount,
			max_uses, current_uses, valid_from, valid_until,
			status, metadata, created_at, updated_at
		FROM redemption_codes
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, 0, fmt.Errorf("failed to query redemption codes: %w", err)
	}
	defer rows.Close()

	codes := []*redemption_code.RedemptionCode{}
	for rows.Next() {
		var dbCode, dbCodeType, dbCurrencyType, dbStatus string
		var amount int64
		var maxUses, currentUses int
		var validFrom, validUntil time.Time
		var metadataJSON sql.NullString
		var createdAt, updatedAt time.Time

		err := rows.Scan(
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
		if err != nil {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, err.Error())
			return nil, 0, fmt.Errorf("failed to scan redemption code: %w", err)
		}

		ct, err := redemption_code.NewCodeType(dbCodeType)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid code type: %w", err)
		}

		currencyType, err := currency.NewCurrencyType(dbCurrencyType)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid currency type: %w", err)
		}

		status, err := redemption_code.NewCodeStatus(dbStatus)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid code status: %w", err)
		}

		var metadata map[string]interface{}
		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &metadata); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		rc, err := redemption_code.NewRedemptionCode(
			dbCode,
			ct,
			currencyType,
			amount,
			maxUses,
			validFrom,
			validUntil,
			metadata,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to create redemption code entity: %w", err)
		}

		rc.SetCurrentUses(currentUses)
		rc.SetStatus(status)

		codes = append(codes, rc)
	}

	if err := rows.Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, 0, fmt.Errorf("failed to iterate redemption codes: %w", err)
	}

	span.SetAttributes(
		attribute.Int("db.total", total),
		attribute.Int("db.count", len(codes)),
	)
	span.SetStatus(otelcodes.Ok, fmt.Sprintf("found %d redemption codes", len(codes)))
	return codes, total, nil
}

// isDuplicateKeyError MySQLの重複キーエラーかどうかをチェック
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// MySQLの重複キーエラーコード1062をチェック
	return strings.Contains(errStr, "1062") || strings.Contains(errStr, "Duplicate entry")
}
