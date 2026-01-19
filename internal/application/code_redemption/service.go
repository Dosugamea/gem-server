package code_redemption

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"gem-server/internal/domain/currency"
	"gem-server/internal/domain/redemption_code"
	"gem-server/internal/domain/transaction"
	otelinfra "gem-server/internal/infrastructure/observability/otel"
	"gem-server/internal/infrastructure/persistence/mysql"
)

// CodeRedemptionApplicationService コード引き換えアプリケーションサービス
type CodeRedemptionApplicationService struct {
	currencyRepo       currency.CurrencyRepository
	transactionRepo    transaction.TransactionRepository
	redemptionCodeRepo redemption_code.RedemptionCodeRepository
	txManager          *mysql.TransactionManager
	logger             *otelinfra.Logger
	metrics            *otelinfra.Metrics
	tracer             trace.Tracer
	maxRetries         int
}

// NewCodeRedemptionApplicationService 新しいCodeRedemptionApplicationServiceを作成
func NewCodeRedemptionApplicationService(
	currencyRepo currency.CurrencyRepository,
	transactionRepo transaction.TransactionRepository,
	redemptionCodeRepo redemption_code.RedemptionCodeRepository,
	txManager *mysql.TransactionManager,
	logger *otelinfra.Logger,
	metrics *otelinfra.Metrics,
) *CodeRedemptionApplicationService {
	return &CodeRedemptionApplicationService{
		currencyRepo:       currencyRepo,
		transactionRepo:    transactionRepo,
		redemptionCodeRepo: redemptionCodeRepo,
		txManager:          txManager,
		logger:             logger,
		metrics:            metrics,
		tracer:             otel.Tracer("code-redemption-service"),
		maxRetries:         3,
	}
}

// Redeem コードを引き換える
func (s *CodeRedemptionApplicationService) Redeem(ctx context.Context, req *RedeemCodeRequest) (*RedeemCodeResponse, error) {
	ctx, span := s.tracer.Start(ctx, "CodeRedemptionApplicationService.Redeem")
	defer span.End()

	span.SetAttributes(
		attribute.String("code", req.Code),
		attribute.String("user_id", req.UserID),
	)

	s.logger.Info(ctx, "Redeeming code", map[string]interface{}{
		"code":    req.Code,
		"user_id": req.UserID,
	})

	// コードを取得
	code, err := s.redemptionCodeRepo.FindByCode(ctx, req.Code)
	if err != nil {
		if err == redemption_code.ErrCodeNotFound {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, err.Error())
			return nil, err
		}
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, fmt.Errorf("failed to find code: %w", err)
	}

	// コードの有効性チェック
	if !code.IsValid() {
		err := redemption_code.ErrCodeNotRedeemable
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, err
	}

	// ユーザーが既に引き換え済みかチェック
	hasRedeemed, err := s.redemptionCodeRepo.HasUserRedeemed(ctx, req.Code, req.UserID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, fmt.Errorf("failed to check redemption status: %w", err)
	}
	if hasRedeemed {
		err := redemption_code.ErrUserAlreadyRedeemed
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, err
	}

	// 引き換え可能かチェック
	if !code.CanBeRedeemed() {
		err := redemption_code.ErrCodeNotRedeemable
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, err
	}

	// トランザクションIDを生成
	transactionID := s.generateTransactionID()
	redemptionID := s.generateRedemptionID()

	var result *RedeemCodeResponse

	err = s.txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
		// コードを引き換え（使用回数を増やす）
		if err := code.Redeem(); err != nil {
			return err
		}

		// コードを更新
		if err := s.redemptionCodeRepo.Update(ctx, code); err != nil {
			return fmt.Errorf("failed to update code: %w", err)
		}

		// 通貨を付与
		currencyType := code.CurrencyType()
		amount := code.Amount()

		// 楽観的ロックのリトライロジック
		var retryErr error
		for attempt := 0; attempt < s.maxRetries; attempt++ {
			if attempt > 0 {
				backoff := time.Duration(math.Pow(2, float64(attempt-1))) * 10 * time.Millisecond
				time.Sleep(backoff)
			}

			// 通貨を取得
			c, err := s.currencyRepo.FindByUserIDAndType(ctx, req.UserID, currencyType)
			if err != nil && err != currency.ErrCurrencyNotFound {
				return fmt.Errorf("failed to find currency: %w", err)
			}

			var balanceBefore int64
			if c == nil {
				// 通貨が存在しない場合は作成
				c = currency.NewCurrency(req.UserID, currencyType, 0, 0)
				if err := s.currencyRepo.Create(ctx, c); err != nil {
					return fmt.Errorf("failed to create currency: %w", err)
				}
			} else {
				balanceBefore = c.Balance()
			}

			// 通貨を付与
			if err := c.Grant(amount); err != nil {
				return err
			}

			// 保存（楽観的ロック）
			if err := s.currencyRepo.Save(ctx, c); err != nil {
				if attempt < s.maxRetries-1 {
					retryErr = err
					continue
				}
				return fmt.Errorf("failed to save currency after retries: %w", err)
			}

			// トランザクション履歴を記録
			txn := transaction.NewTransaction(
				transactionID,
				req.UserID,
				transaction.TransactionTypeGrant,
				currencyType,
				amount,
				balanceBefore,
				c.Balance(),
				transaction.TransactionStatusCompleted,
				map[string]interface{}{
					"code":          req.Code,
					"redemption_id": redemptionID,
				},
			)

			if err := s.transactionRepo.Save(ctx, txn); err != nil {
				return fmt.Errorf("failed to save transaction: %w", err)
			}

			// 引き換え履歴を記録
			redemption := redemption_code.NewCodeRedemption(
				redemptionID,
				req.Code,
				req.UserID,
				transactionID,
			)

			if err := s.redemptionCodeRepo.SaveRedemption(ctx, redemption); err != nil {
				return fmt.Errorf("failed to save redemption: %w", err)
			}

			// メトリクス記録
			s.metrics.RecordTransaction(ctx, "grant", currencyType.String())
			s.metrics.RecordCurrencyBalance(ctx, req.UserID, currencyType.String(), c.Balance())

			result = &RedeemCodeResponse{
				RedemptionID:  redemptionID,
				TransactionID: transactionID,
				Code:          req.Code,
				CurrencyType:  currencyType.String(),
				Amount:        amount,
				BalanceAfter:  c.Balance(),
				Status:        "completed",
			}

			return nil
		}

		return retryErr
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		s.logger.Error(ctx, "Failed to redeem code", err, map[string]interface{}{
			"code":    req.Code,
			"user_id": req.UserID,
		})
		s.metrics.RecordError(ctx, "code_redemption_failed")
		return nil, err
	}

	s.logger.Info(ctx, "Code redeemed successfully", map[string]interface{}{
		"code":           req.Code,
		"user_id":        req.UserID,
		"redemption_id":  redemptionID,
		"transaction_id": transactionID,
	})

	return result, nil
}

// generateTransactionID トランザクションIDを生成
func (s *CodeRedemptionApplicationService) generateTransactionID() string {
	return fmt.Sprintf("txn_%d", time.Now().UnixNano())
}

// generateRedemptionID 引き換えIDを生成
func (s *CodeRedemptionApplicationService) generateRedemptionID() string {
	return fmt.Sprintf("red_%d", time.Now().UnixNano())
}
