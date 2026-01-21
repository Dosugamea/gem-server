package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"gem-server/internal/domain/currency"
)

// CurrencyRepository MySQL実装のCurrencyRepository
type CurrencyRepository struct {
	db     *DB
	tracer trace.Tracer
}

// NewCurrencyRepository 新しいCurrencyRepositoryを作成
func NewCurrencyRepository(db *DB) *CurrencyRepository {
	return &CurrencyRepository{
		db:     db,
		tracer: otel.Tracer("currency-repository"),
	}
}

// FindByUserIDAndType ユーザーIDと通貨タイプで通貨を取得
func (r *CurrencyRepository) FindByUserIDAndType(ctx context.Context, userID string, currencyType currency.CurrencyType) (*currency.Currency, error) {
	ctx, span := r.tracer.Start(ctx, "CurrencyRepository.FindByUserIDAndType")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.user_id", userID),
		attribute.String("db.currency_type", currencyType.String()),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "currency_balances"),
	)

	query := `
		SELECT user_id, currency_type, balance, version
		FROM currency_balances
		WHERE user_id = ? AND currency_type = ?
	`

	var dbUserID string
	var dbCurrencyType string
	var balance int64
	var version int

	err := r.db.QueryRowContext(ctx, query, userID, currencyType.String()).Scan(
		&dbUserID,
		&dbCurrencyType,
		&balance,
		&version,
	)

	if err == sql.ErrNoRows {
		span.SetStatus(otelcodes.Ok, "currency not found")
		return nil, currency.ErrCurrencyNotFound
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, fmt.Errorf("failed to find currency: %w", err)
	}

	span.SetAttributes(
		attribute.Int64("db.balance", balance),
		attribute.Int("db.version", version),
	)
	span.SetStatus(otelcodes.Ok, "currency found")

	ct, err := currency.NewCurrencyType(dbCurrencyType)
	if err != nil {
		return nil, fmt.Errorf("invalid currency type: %w", err)
	}

	c, err := currency.NewCurrency(dbUserID, ct, balance, version)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct currency entity: %w", err)
	}

	return c, nil
}

// Save 通貨を保存（更新、楽観的ロック対応）
func (r *CurrencyRepository) Save(ctx context.Context, c *currency.Currency) error {
	ctx, span := r.tracer.Start(ctx, "CurrencyRepository.Save")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.user_id", c.UserID()),
		attribute.String("db.currency_type", c.CurrencyType().String()),
		attribute.Int64("db.balance", c.Balance()),
		attribute.Int("db.version", c.Version()),
		attribute.String("db.operation", "UPDATE"),
		attribute.String("db.table", "currency_balances"),
	)

	query := `
		UPDATE currency_balances
		SET balance = ?, version = version + 1, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND currency_type = ? AND version = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		c.Balance(),
		c.UserID(),
		c.CurrencyType().String(),
		c.Version(),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return fmt.Errorf("failed to save currency: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		err := fmt.Errorf("optimistic lock failed: version mismatch or currency not found")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return err
	}

	span.SetAttributes(attribute.Int64("db.rows_affected", rowsAffected))
	span.SetStatus(otelcodes.Ok, "currency saved")
	return nil
}

// Create 新しい通貨を作成
func (r *CurrencyRepository) Create(ctx context.Context, c *currency.Currency) error {
	ctx, span := r.tracer.Start(ctx, "CurrencyRepository.Create")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.user_id", c.UserID()),
		attribute.String("db.currency_type", c.CurrencyType().String()),
		attribute.Int64("db.balance", c.Balance()),
		attribute.Int("db.version", c.Version()),
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.table", "currency_balances"),
	)

	// ユーザーが存在するか確認（存在しない場合は作成）
	if err := r.ensureUserExists(ctx, c.UserID()); err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return fmt.Errorf("failed to ensure user exists: %w", err)
	}

	query := `
		INSERT INTO currency_balances (user_id, currency_type, balance, version)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			balance = VALUES(balance),
			version = VALUES(version),
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := r.db.ExecContext(ctx, query,
		c.UserID(),
		c.CurrencyType().String(),
		c.Balance(),
		c.Version(),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return fmt.Errorf("failed to create currency: %w", err)
	}

	span.SetStatus(otelcodes.Ok, "currency created")
	return nil
}

// ensureUserExists ユーザーが存在することを確認（存在しない場合は作成）
func (r *CurrencyRepository) ensureUserExists(ctx context.Context, userID string) error {
	query := `
		INSERT INTO users (user_id)
		VALUES (?)
		ON DUPLICATE KEY UPDATE updated_at = CURRENT_TIMESTAMP
	`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to ensure user exists: %w", err)
	}

	return nil
}
