package history

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"gem-server/internal/domain/currency"
	"gem-server/internal/domain/transaction"
	otelinfra "gem-server/internal/infrastructure/observability/otel"
)

// HistoryApplicationService 履歴アプリケーションサービス
type HistoryApplicationService struct {
	transactionRepo transaction.TransactionRepository
	logger          *otelinfra.Logger
	metrics         *otelinfra.Metrics
	tracer          trace.Tracer
}

// NewHistoryApplicationService 新しいHistoryApplicationServiceを作成
func NewHistoryApplicationService(
	transactionRepo transaction.TransactionRepository,
	logger *otelinfra.Logger,
	metrics *otelinfra.Metrics,
) *HistoryApplicationService {
	return &HistoryApplicationService{
		transactionRepo: transactionRepo,
		logger:          logger,
		metrics:         metrics,
		tracer:          otel.Tracer("history-service"),
	}
}

// GetTransactionHistory トランザクション履歴を取得
func (s *HistoryApplicationService) GetTransactionHistory(ctx context.Context, req *GetTransactionHistoryRequest) (*GetTransactionHistoryResponse, error) {
	ctx, span := s.tracer.Start(ctx, "HistoryApplicationService.GetTransactionHistory")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", req.UserID),
		attribute.Int("limit", req.Limit),
		attribute.Int("offset", req.Offset),
	)

	s.logger.Info(ctx, "Getting transaction history", map[string]interface{}{
		"user_id":          req.UserID,
		"limit":            req.Limit,
		"offset":           req.Offset,
		"currency_type":    req.CurrencyType,
		"transaction_type": req.TransactionType,
	})

	// バリデーション
	if req.Limit <= 0 {
		req.Limit = 50 // デフォルト値
	}
	if req.Limit > 100 {
		req.Limit = 100 // 最大値
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	// トランザクション履歴を取得
	transactions, err := s.transactionRepo.FindByUserID(ctx, req.UserID, req.Limit, req.Offset)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		s.logger.Error(ctx, "Failed to get transaction history", err, map[string]interface{}{
			"user_id": req.UserID,
		})
		return nil, fmt.Errorf("failed to get transaction history: %w", err)
	}

	// フィルタリング
	filteredTransactions := make([]*transaction.Transaction, 0, len(transactions))
	for _, txn := range transactions {
		// 通貨タイプフィルタ
		if req.CurrencyType != "" {
			currencyType, err := currency.NewCurrencyType(req.CurrencyType)
			if err == nil && txn.CurrencyType() != currencyType {
				continue
			}
		}

		// トランザクションタイプフィルタ
		if req.TransactionType != "" {
			transactionType, err := transaction.NewTransactionType(req.TransactionType)
			if err == nil && txn.TransactionType() != transactionType {
				continue
			}
		}

		filteredTransactions = append(filteredTransactions, txn)
	}

	// メトリクス記録
	s.metrics.RecordRequest(ctx, "GET", "/api/v1/users/{user_id}/transactions")

	return &GetTransactionHistoryResponse{
		Transactions: filteredTransactions,
		Total:        len(filteredTransactions), // 簡易的な実装（実際には総件数を取得する必要がある）
		Limit:        req.Limit,
		Offset:       req.Offset,
	}, nil
}
