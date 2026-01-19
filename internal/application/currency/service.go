package currency

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
	"gem-server/internal/domain/service"
	"gem-server/internal/domain/transaction"
	otelinfra "gem-server/internal/infrastructure/observability/otel"
	"gem-server/internal/infrastructure/persistence/mysql"
)

// CurrencyApplicationService 通貨アプリケーションサービス
type CurrencyApplicationService struct {
	currencyRepo    currency.CurrencyRepository
	transactionRepo transaction.TransactionRepository
	txManager       *mysql.TransactionManager
	currencyService *service.CurrencyService
	logger          *otelinfra.Logger
	metrics         *otelinfra.Metrics
	tracer          trace.Tracer
	maxRetries      int
}

// NewCurrencyApplicationService 新しいCurrencyApplicationServiceを作成
func NewCurrencyApplicationService(
	currencyRepo currency.CurrencyRepository,
	transactionRepo transaction.TransactionRepository,
	txManager *mysql.TransactionManager,
	currencyService *service.CurrencyService,
	logger *otelinfra.Logger,
	metrics *otelinfra.Metrics,
) *CurrencyApplicationService {
	return &CurrencyApplicationService{
		currencyRepo:    currencyRepo,
		transactionRepo: transactionRepo,
		txManager:       txManager,
		currencyService: currencyService,
		logger:          logger,
		metrics:         metrics,
		tracer:          otel.Tracer("currency-service"),
		maxRetries:      3,
	}
}

// GetBalance 残高を取得
func (s *CurrencyApplicationService) GetBalance(ctx context.Context, req *GetBalanceRequest) (*GetBalanceResponse, error) {
	ctx, span := s.tracer.Start(ctx, "CurrencyApplicationService.GetBalance")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", req.UserID),
	)

	s.logger.Info(ctx, "Getting balance", map[string]interface{}{
		"user_id": req.UserID,
	})

	paidCurrency, err := s.currencyRepo.FindByUserIDAndType(ctx, req.UserID, currency.CurrencyTypePaid)
	if err != nil && err != currency.ErrCurrencyNotFound {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		s.logger.Error(ctx, "Failed to find paid currency", err, map[string]interface{}{
			"user_id": req.UserID,
		})
		return nil, fmt.Errorf("failed to find paid currency: %w", err)
	}

	freeCurrency, err := s.currencyRepo.FindByUserIDAndType(ctx, req.UserID, currency.CurrencyTypeFree)
	if err != nil && err != currency.ErrCurrencyNotFound {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		s.logger.Error(ctx, "Failed to find free currency", err, map[string]interface{}{
			"user_id": req.UserID,
		})
		return nil, fmt.Errorf("failed to find free currency: %w", err)
	}

	balances := make(map[string]int64)
	if paidCurrency != nil {
		balances["paid"] = paidCurrency.Balance()
		s.metrics.RecordCurrencyBalance(ctx, req.UserID, "paid", paidCurrency.Balance())
	} else {
		balances["paid"] = 0
	}

	if freeCurrency != nil {
		balances["free"] = freeCurrency.Balance()
		s.metrics.RecordCurrencyBalance(ctx, req.UserID, "free", freeCurrency.Balance())
	} else {
		balances["free"] = 0
	}

	return &GetBalanceResponse{
		UserID:   req.UserID,
		Balances: balances,
	}, nil
}

// Grant 通貨を付与
func (s *CurrencyApplicationService) Grant(ctx context.Context, req *GrantRequest) (*GrantResponse, error) {
	ctx, span := s.tracer.Start(ctx, "CurrencyApplicationService.Grant")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", req.UserID),
		attribute.String("currency_type", req.CurrencyType),
		attribute.Int64("amount", req.Amount),
	)

	s.logger.Info(ctx, "Granting currency", map[string]interface{}{
		"user_id":       req.UserID,
		"currency_type": req.CurrencyType,
		"amount":        req.Amount,
	})

	// バリデーション
	if req.Amount <= 0 {
		err := currency.ErrInvalidAmount
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, err
	}

	currencyType, err := currency.NewCurrencyType(req.CurrencyType)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, err
	}

	// トランザクションIDを生成
	transactionID := s.generateTransactionID()

	var result *GrantResponse
	err = s.txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
		// 楽観的ロックのリトライロジック
		var retryErr error
		for attempt := 0; attempt < s.maxRetries; attempt++ {
			if attempt > 0 {
				// 指数バックオフ
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
			if err := c.Grant(req.Amount); err != nil {
				return err
			}

			// 保存（楽観的ロック）
			if err := s.currencyRepo.Save(ctx, c); err != nil {
				// 楽観的ロックエラーの場合はリトライ
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
				req.Amount,
				balanceBefore,
				c.Balance(),
				transaction.TransactionStatusCompleted,
				req.Metadata,
			)

			if err := s.transactionRepo.Save(ctx, txn); err != nil {
				return fmt.Errorf("failed to save transaction: %w", err)
			}

			// メトリクス記録
			s.metrics.RecordTransaction(ctx, "grant", currencyType.String())
			s.metrics.RecordCurrencyBalance(ctx, req.UserID, currencyType.String(), c.Balance())

			result = &GrantResponse{
				TransactionID: transactionID,
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
		s.logger.Error(ctx, "Failed to grant currency", err, map[string]interface{}{
			"user_id":       req.UserID,
			"currency_type": req.CurrencyType,
			"amount":        req.Amount,
		})
		s.metrics.RecordError(ctx, "grant_failed")
		return nil, err
	}

	s.logger.Info(ctx, "Currency granted successfully", map[string]interface{}{
		"user_id":        req.UserID,
		"transaction_id": transactionID,
		"balance_after":  result.BalanceAfter,
	})

	return result, nil
}

// Consume 通貨を消費（単一通貨タイプ）
func (s *CurrencyApplicationService) Consume(ctx context.Context, req *ConsumeRequest) (*ConsumeResponse, error) {
	ctx, span := s.tracer.Start(ctx, "CurrencyApplicationService.Consume")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", req.UserID),
		attribute.String("currency_type", req.CurrencyType),
		attribute.Int64("amount", req.Amount),
	)

	s.logger.Info(ctx, "Consuming currency", map[string]interface{}{
		"user_id":       req.UserID,
		"currency_type": req.CurrencyType,
		"amount":        req.Amount,
	})

	// バリデーション
	if req.Amount <= 0 {
		err := currency.ErrInvalidAmount
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, err
	}

	if req.CurrencyType == "auto" {
		return s.ConsumeWithPriority(ctx, req)
	}

	currencyType, err := currency.NewCurrencyType(req.CurrencyType)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, err
	}

	// トランザクションIDを生成
	transactionID := s.generateTransactionID()

	var result *ConsumeResponse
	err = s.txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
		// 楽観的ロックのリトライロジック
		var retryErr error
		for attempt := 0; attempt < s.maxRetries; attempt++ {
			if attempt > 0 {
				backoff := time.Duration(math.Pow(2, float64(attempt-1))) * 10 * time.Millisecond
				time.Sleep(backoff)
			}

			// 通貨を取得
			c, err := s.currencyRepo.FindByUserIDAndType(ctx, req.UserID, currencyType)
			if err != nil {
				return fmt.Errorf("failed to find currency: %w", err)
			}

			balanceBefore := c.Balance()

			// 通貨を消費
			if err := c.Consume(req.Amount); err != nil {
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
				transaction.TransactionTypeConsume,
				currencyType,
				req.Amount,
				balanceBefore,
				c.Balance(),
				transaction.TransactionStatusCompleted,
				req.Metadata,
			)

			if err := s.transactionRepo.Save(ctx, txn); err != nil {
				return fmt.Errorf("failed to save transaction: %w", err)
			}

			// メトリクス記録
			s.metrics.RecordTransaction(ctx, "consume", currencyType.String())
			s.metrics.RecordCurrencyBalance(ctx, req.UserID, currencyType.String(), c.Balance())

			result = &ConsumeResponse{
				TransactionID: transactionID,
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
		s.logger.Error(ctx, "Failed to consume currency", err, map[string]interface{}{
			"user_id":       req.UserID,
			"currency_type": req.CurrencyType,
			"amount":        req.Amount,
		})
		s.metrics.RecordError(ctx, "consume_failed")
		return nil, err
	}

	s.logger.Info(ctx, "Currency consumed successfully", map[string]interface{}{
		"user_id":        req.UserID,
		"transaction_id": transactionID,
		"balance_after":  result.BalanceAfter,
	})

	return result, nil
}

// ConsumeWithPriority 通貨を消費（無料通貨優先）
func (s *CurrencyApplicationService) ConsumeWithPriority(ctx context.Context, req *ConsumeRequest) (*ConsumeResponse, error) {
	ctx, span := s.tracer.Start(ctx, "CurrencyApplicationService.ConsumeWithPriority")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", req.UserID),
		attribute.Int64("amount", req.Amount),
	)

	s.logger.Info(ctx, "Consuming currency with priority", map[string]interface{}{
		"user_id": req.UserID,
		"amount":  req.Amount,
	})

	// バリデーション
	if req.Amount <= 0 {
		err := currency.ErrInvalidAmount
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, err
	}

	// 残高チェック
	hasBalance, err := s.currencyService.HasSufficientBalance(ctx, req.UserID, req.Amount)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, fmt.Errorf("failed to check balance: %w", err)
	}
	if !hasBalance {
		err := currency.ErrInsufficientBalance
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, err
	}

	// トランザクションIDを生成
	transactionID := s.generateTransactionID()

	var result *ConsumeResponse
	var consumptionDetails []ConsumptionDetail
	var totalConsumed int64

	err = s.txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
		remainingAmount := req.Amount

		// 無料通貨から消費
		freeCurrency, err := s.currencyRepo.FindByUserIDAndType(ctx, req.UserID, currency.CurrencyTypeFree)
		if err != nil && err != currency.ErrCurrencyNotFound {
			return fmt.Errorf("failed to find free currency: %w", err)
		}

		if freeCurrency != nil && freeCurrency.Balance() > 0 {
			freeBalanceBefore := freeCurrency.Balance()
			freeConsumeAmount := remainingAmount
			if freeConsumeAmount > freeBalanceBefore {
				freeConsumeAmount = freeBalanceBefore
			}

			// 楽観的ロックのリトライロジック
			var retryErr error
			for attempt := 0; attempt < s.maxRetries; attempt++ {
				if attempt > 0 {
					backoff := time.Duration(math.Pow(2, float64(attempt-1))) * 10 * time.Millisecond
					time.Sleep(backoff)
					// 再取得
					freeCurrency, err = s.currencyRepo.FindByUserIDAndType(ctx, req.UserID, currency.CurrencyTypeFree)
					if err != nil {
						return fmt.Errorf("failed to find free currency: %w", err)
					}
					freeBalanceBefore = freeCurrency.Balance()
					freeConsumeAmount = remainingAmount
					if freeConsumeAmount > freeBalanceBefore {
						freeConsumeAmount = freeBalanceBefore
					}
				}

				if err := freeCurrency.Consume(freeConsumeAmount); err != nil {
					return err
				}

				if err := s.currencyRepo.Save(ctx, freeCurrency); err != nil {
					if attempt < s.maxRetries-1 {
						retryErr = err
						continue
					}
					return fmt.Errorf("failed to save free currency after retries: %w", err)
				}

				consumptionDetails = append(consumptionDetails, ConsumptionDetail{
					CurrencyType:  "free",
					Amount:        freeConsumeAmount,
					BalanceBefore: freeBalanceBefore,
					BalanceAfter:  freeCurrency.Balance(),
				})

				totalConsumed += freeConsumeAmount
				remainingAmount -= freeConsumeAmount

				// トランザクション履歴を記録
				txn := transaction.NewTransaction(
					fmt.Sprintf("%s_free", transactionID),
					req.UserID,
					transaction.TransactionTypeConsume,
					currency.CurrencyTypeFree,
					freeConsumeAmount,
					freeBalanceBefore,
					freeCurrency.Balance(),
					transaction.TransactionStatusCompleted,
					req.Metadata,
				)
				if err := s.transactionRepo.Save(ctx, txn); err != nil {
					return fmt.Errorf("failed to save transaction: %w", err)
				}

				s.metrics.RecordTransaction(ctx, "consume", "free")
				s.metrics.RecordCurrencyBalance(ctx, req.UserID, "free", freeCurrency.Balance())

				break
			}

			if retryErr != nil {
				return retryErr
			}
		}

		// 不足分を有料通貨から消費
		if remainingAmount > 0 {
			paidCurrency, err := s.currencyRepo.FindByUserIDAndType(ctx, req.UserID, currency.CurrencyTypePaid)
			if err != nil {
				return fmt.Errorf("failed to find paid currency: %w", err)
			}

			paidBalanceBefore := paidCurrency.Balance()

			// 楽観的ロックのリトライロジック
			var retryErr error
			for attempt := 0; attempt < s.maxRetries; attempt++ {
				if attempt > 0 {
					backoff := time.Duration(math.Pow(2, float64(attempt-1))) * 10 * time.Millisecond
					time.Sleep(backoff)
					// 再取得
					paidCurrency, err = s.currencyRepo.FindByUserIDAndType(ctx, req.UserID, currency.CurrencyTypePaid)
					if err != nil {
						return fmt.Errorf("failed to find paid currency: %w", err)
					}
					paidBalanceBefore = paidCurrency.Balance()
				}

				if err := paidCurrency.Consume(remainingAmount); err != nil {
					return err
				}

				if err := s.currencyRepo.Save(ctx, paidCurrency); err != nil {
					if attempt < s.maxRetries-1 {
						retryErr = err
						continue
					}
					return fmt.Errorf("failed to save paid currency after retries: %w", err)
				}

				consumptionDetails = append(consumptionDetails, ConsumptionDetail{
					CurrencyType:  "paid",
					Amount:        remainingAmount,
					BalanceBefore: paidBalanceBefore,
					BalanceAfter:  paidCurrency.Balance(),
				})

				totalConsumed += remainingAmount

				// トランザクション履歴を記録
				txn := transaction.NewTransaction(
					fmt.Sprintf("%s_paid", transactionID),
					req.UserID,
					transaction.TransactionTypeConsume,
					currency.CurrencyTypePaid,
					remainingAmount,
					paidBalanceBefore,
					paidCurrency.Balance(),
					transaction.TransactionStatusCompleted,
					req.Metadata,
				)
				if err := s.transactionRepo.Save(ctx, txn); err != nil {
					return fmt.Errorf("failed to save transaction: %w", err)
				}

				s.metrics.RecordTransaction(ctx, "consume", "paid")
				s.metrics.RecordCurrencyBalance(ctx, req.UserID, "paid", paidCurrency.Balance())

				break
			}

			if retryErr != nil {
				return retryErr
			}
		}

		result = &ConsumeResponse{
			TransactionID:      transactionID,
			ConsumptionDetails: consumptionDetails,
			TotalConsumed:      totalConsumed,
			Status:             "completed",
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		s.logger.Error(ctx, "Failed to consume currency with priority", err, map[string]interface{}{
			"user_id": req.UserID,
			"amount":  req.Amount,
		})
		s.metrics.RecordError(ctx, "consume_priority_failed")
		return nil, err
	}

	s.logger.Info(ctx, "Currency consumed with priority successfully", map[string]interface{}{
		"user_id":        req.UserID,
		"transaction_id": transactionID,
		"total_consumed": totalConsumed,
	})

	return result, nil
}

// generateTransactionID トランザクションIDを生成
func (s *CurrencyApplicationService) generateTransactionID() string {
	return fmt.Sprintf("txn_%d", time.Now().UnixNano())
}
