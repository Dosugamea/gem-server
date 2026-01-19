package payment

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
	"gem-server/internal/domain/payment_request"
	"gem-server/internal/domain/transaction"
	otelinfra "gem-server/internal/infrastructure/observability/otel"
)

// PaymentApplicationService 決済アプリケーションサービス
type PaymentApplicationService struct {
	currencyRepo       currency.CurrencyRepository
	transactionRepo    transaction.TransactionRepository
	paymentRequestRepo payment_request.PaymentRequestRepository
	txManager          transaction.TransactionManager
	logger             *otelinfra.Logger
	metrics            *otelinfra.Metrics
	tracer             trace.Tracer
	maxRetries         int
}

// NewPaymentApplicationService 新しいPaymentApplicationServiceを作成
func NewPaymentApplicationService(
	currencyRepo currency.CurrencyRepository,
	transactionRepo transaction.TransactionRepository,
	paymentRequestRepo payment_request.PaymentRequestRepository,
	txManager transaction.TransactionManager,
	logger *otelinfra.Logger,
	metrics *otelinfra.Metrics,
) *PaymentApplicationService {
	return &PaymentApplicationService{
		currencyRepo:       currencyRepo,
		transactionRepo:    transactionRepo,
		paymentRequestRepo: paymentRequestRepo,
		txManager:          txManager,
		logger:             logger,
		metrics:            metrics,
		tracer:             otel.Tracer("payment-service"),
		maxRetries:         3,
	}
}

// ProcessPayment 決済を処理（無料通貨優先で消費）
func (s *PaymentApplicationService) ProcessPayment(ctx context.Context, req *ProcessPaymentRequest) (*ProcessPaymentResponse, error) {
	ctx, span := s.tracer.Start(ctx, "PaymentApplicationService.ProcessPayment")
	defer span.End()

	span.SetAttributes(
		attribute.String("payment_request_id", req.PaymentRequestID),
		attribute.String("user_id", req.UserID),
		attribute.Int64("amount", req.Amount),
	)

	s.logger.Info(ctx, "Processing payment", map[string]interface{}{
		"payment_request_id": req.PaymentRequestID,
		"user_id":            req.UserID,
		"amount":             req.Amount,
	})

	// バリデーション
	if req.Amount <= 0 {
		err := fmt.Errorf("invalid amount: %d", req.Amount)
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, err
	}

	// 既存のPaymentRequestを確認（冪等性保証）
	existingPR, err := s.paymentRequestRepo.FindByPaymentRequestID(ctx, req.PaymentRequestID)
	if err != nil && err != payment_request.ErrPaymentRequestNotFound {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, fmt.Errorf("failed to find payment request: %w", err)
	}

	// 既に処理済みの場合は、既存の結果を返す
	if existingPR != nil && existingPR.IsCompleted() {
		// 既存のトランザクションを取得
		txn, err := s.transactionRepo.FindByPaymentRequestID(ctx, req.PaymentRequestID)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, err.Error())
			return nil, fmt.Errorf("failed to find transaction: %w", err)
		}

		// 消費詳細を再構築（簡易的な実装）
		consumptionDetails := []ConsumptionDetail{
			{
				CurrencyType:  txn.CurrencyType().String(),
				Amount:        txn.Amount(),
				BalanceBefore: txn.BalanceBefore(),
				BalanceAfter:  txn.BalanceAfter(),
			},
		}

		return &ProcessPaymentResponse{
			TransactionID:      txn.TransactionID(),
			PaymentRequestID:   req.PaymentRequestID,
			ConsumptionDetails: consumptionDetails,
			TotalConsumed:      txn.Amount(),
			Status:             "completed",
		}, nil
	}

	// 既に処理中または失敗の場合はエラー
	if existingPR != nil && !existingPR.IsPending() {
		err := payment_request.ErrPaymentRequestAlreadyProcessed
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, err
	}

	// PaymentRequestを作成または更新
	var pr *payment_request.PaymentRequest
	if existingPR == nil {
		pr = payment_request.NewPaymentRequest(
			req.PaymentRequestID,
			req.UserID,
			req.Amount,
			req.Currency,
			currency.CurrencyTypePaid, // デフォルトは有料通貨
		)
		pr.SetPaymentMethodData(map[string]interface{}{
			"methodName": req.MethodName,
		})
		details := make(map[string]interface{})
		for k, v := range req.Details {
			details[k] = v
		}
		pr.SetDetails(details)
	} else {
		pr = existingPR
	}

	// トランザクションIDを生成
	transactionID := s.generateTransactionID()

	var result *ProcessPaymentResponse
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
					map[string]interface{}{
						"payment_request_id": req.PaymentRequestID,
					},
				)
				txn.SetPaymentRequestID(req.PaymentRequestID)
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

			// 残高チェック
			if paidCurrency.Balance() < remainingAmount {
				err := currency.ErrInsufficientBalance
				pr.Fail()
				_ = s.paymentRequestRepo.Update(ctx, pr)
				return err
			}

			// 楽観的ロックのリトライロジック
			var retryErr error
			for attempt := 0; attempt < s.maxRetries; attempt++ {
				if attempt > 0 {
					backoff := time.Duration(math.Pow(2, float64(attempt-1))) * 10 * time.Millisecond
					time.Sleep(backoff)
					paidCurrency, err = s.currencyRepo.FindByUserIDAndType(ctx, req.UserID, currency.CurrencyTypePaid)
					if err != nil {
						return fmt.Errorf("failed to find paid currency: %w", err)
					}
					paidBalanceBefore = paidCurrency.Balance()
					if paidCurrency.Balance() < remainingAmount {
						err := currency.ErrInsufficientBalance
						pr.Fail()
						_ = s.paymentRequestRepo.Update(ctx, pr)
						return err
					}
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
					map[string]interface{}{
						"payment_request_id": req.PaymentRequestID,
					},
				)
				txn.SetPaymentRequestID(req.PaymentRequestID)
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

		// PaymentRequestを完了状態にする
		pr.Complete()
		pr.SetResponse(map[string]interface{}{
			"transaction_id": transactionID,
			"status":         "completed",
		})

		if err := s.paymentRequestRepo.Save(ctx, pr); err != nil {
			return fmt.Errorf("failed to save payment request: %w", err)
		}

		result = &ProcessPaymentResponse{
			TransactionID:      transactionID,
			PaymentRequestID:   req.PaymentRequestID,
			ConsumptionDetails: consumptionDetails,
			TotalConsumed:      totalConsumed,
			Status:             "completed",
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		s.logger.Error(ctx, "Failed to process payment", err, map[string]interface{}{
			"payment_request_id": req.PaymentRequestID,
			"user_id":            req.UserID,
		})
		s.metrics.RecordError(ctx, "payment_failed")
		return nil, err
	}

	s.logger.Info(ctx, "Payment processed successfully", map[string]interface{}{
		"payment_request_id": req.PaymentRequestID,
		"transaction_id":     transactionID,
		"total_consumed":     totalConsumed,
	})

	return result, nil
}

// generateTransactionID トランザクションIDを生成
func (s *PaymentApplicationService) generateTransactionID() string {
	return fmt.Sprintf("txn_%d", time.Now().UnixNano())
}
