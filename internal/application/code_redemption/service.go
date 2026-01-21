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
)

// CodeRedemptionApplicationService コード引き換えアプリケーションサービス
type CodeRedemptionApplicationService struct {
	currencyRepo       currency.CurrencyRepository
	transactionRepo    transaction.TransactionRepository
	redemptionCodeRepo redemption_code.RedemptionCodeRepository
	txManager          transaction.TransactionManager
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
	txManager transaction.TransactionManager,
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
				c, err = currency.NewCurrency(req.UserID, currencyType, 0, 0)
				if err != nil {
					return fmt.Errorf("failed to create currency entity: %w", err)
				}
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
			txn, err := transaction.NewTransaction(
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
			if err != nil {
				return fmt.Errorf("failed to create transaction entity: %w", err)
			}

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

// CreateCode 引き換えコードを作成
func (s *CodeRedemptionApplicationService) CreateCode(ctx context.Context, req *CreateCodeRequest) (*CreateCodeResponse, error) {
	ctx, span := s.tracer.Start(ctx, "CodeRedemptionApplicationService.CreateCode")
	defer span.End()

	span.SetAttributes(
		attribute.String("code", req.Code),
		attribute.String("code_type", req.CodeType),
		attribute.String("currency_type", req.CurrencyType),
		attribute.Int64("amount", req.Amount),
	)

	s.logger.Info(ctx, "Creating redemption code", map[string]interface{}{
		"code":          req.Code,
		"code_type":     req.CodeType,
		"currency_type": req.CurrencyType,
		"amount":        req.Amount,
		"max_uses":      req.MaxUses,
		"valid_from":    req.ValidFrom,
		"valid_until":   req.ValidUntil,
	})

	// バリデーション
	if req.Code == "" {
		err := fmt.Errorf("code is required")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, err
	}

	if req.ValidUntil.Before(req.ValidFrom) {
		err := fmt.Errorf("valid_until must be after valid_from")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, err
	}

	if req.Amount < 0 {
		err := fmt.Errorf("amount must be non-negative")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, err
	}

	if req.MaxUses < 0 {
		err := fmt.Errorf("max_uses must be non-negative")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, err
	}

	// コードタイプのバリデーション
	codeType, err := redemption_code.NewCodeType(req.CodeType)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, fmt.Errorf("invalid code type: %w", err)
	}

	// 通貨タイプのバリデーション
	currencyType, err := currency.NewCurrencyType(req.CurrencyType)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, fmt.Errorf("invalid currency type: %w", err)
	}

	// ドメインエンティティの作成
	rc, err := redemption_code.NewRedemptionCode(
		req.Code,
		codeType,
		currencyType,
		req.Amount,
		req.MaxUses,
		req.ValidFrom,
		req.ValidUntil,
		req.Metadata,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, fmt.Errorf("failed to create redemption code entity: %w", err)
	}

	// リポジトリに保存
	if err := s.redemptionCodeRepo.Create(ctx, rc); err != nil {
		if err == redemption_code.ErrCodeAlreadyExists {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, err.Error())
			return nil, err
		}
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		s.logger.Error(ctx, "Failed to create redemption code", err, map[string]interface{}{
			"code": req.Code,
		})
		return nil, fmt.Errorf("failed to create redemption code: %w", err)
	}

	s.logger.Info(ctx, "Redemption code created successfully", map[string]interface{}{
		"code": req.Code,
	})

	return &CreateCodeResponse{
		Code:         rc.Code(),
		CodeType:     rc.CodeType().String(),
		CurrencyType: rc.CurrencyType().String(),
		Amount:       rc.Amount(),
		MaxUses:      rc.MaxUses(),
		CurrentUses:  rc.CurrentUses(),
		ValidFrom:    rc.ValidFrom(),
		ValidUntil:   rc.ValidUntil(),
		Status:       rc.Status().String(),
		Metadata:     rc.Metadata(),
		CreatedAt:    rc.CreatedAt(),
	}, nil
}

// DeleteCode 引き換えコードを削除
func (s *CodeRedemptionApplicationService) DeleteCode(ctx context.Context, req *DeleteCodeRequest) (*DeleteCodeResponse, error) {
	ctx, span := s.tracer.Start(ctx, "CodeRedemptionApplicationService.DeleteCode")
	defer span.End()

	span.SetAttributes(
		attribute.String("code", req.Code),
	)

	s.logger.Info(ctx, "Deleting redemption code", map[string]interface{}{
		"code": req.Code,
	})

	// バリデーション
	if req.Code == "" {
		err := fmt.Errorf("code is required")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, err
	}

	// コードの存在確認
	_, err := s.redemptionCodeRepo.FindByCode(ctx, req.Code)
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

	// 削除実行（リポジトリ内で使用済みチェックも行う）
	if err := s.redemptionCodeRepo.Delete(ctx, req.Code); err != nil {
		if err == redemption_code.ErrCodeNotFound {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, err.Error())
			return nil, err
		}
		if err == redemption_code.ErrCodeCannotBeDeleted {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, err.Error())
			return nil, err
		}
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		s.logger.Error(ctx, "Failed to delete redemption code", err, map[string]interface{}{
			"code": req.Code,
		})
		return nil, fmt.Errorf("failed to delete redemption code: %w", err)
	}

	s.logger.Info(ctx, "Redemption code deleted successfully", map[string]interface{}{
		"code": req.Code,
	})

	return &DeleteCodeResponse{
		Code:      req.Code,
		DeletedAt: time.Now(),
	}, nil
}

// GetCode 引き換えコードを取得
func (s *CodeRedemptionApplicationService) GetCode(ctx context.Context, req *GetCodeRequest) (*GetCodeResponse, error) {
	ctx, span := s.tracer.Start(ctx, "CodeRedemptionApplicationService.GetCode")
	defer span.End()

	span.SetAttributes(
		attribute.String("code", req.Code),
	)

	s.logger.Info(ctx, "Getting redemption code", map[string]interface{}{
		"code": req.Code,
	})

	// バリデーション
	if req.Code == "" {
		err := fmt.Errorf("code is required")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, err
	}

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

	return &GetCodeResponse{
		Code:         code.Code(),
		CodeType:     code.CodeType().String(),
		CurrencyType: code.CurrencyType().String(),
		Amount:       code.Amount(),
		MaxUses:      code.MaxUses(),
		CurrentUses:  code.CurrentUses(),
		ValidFrom:    code.ValidFrom(),
		ValidUntil:   code.ValidUntil(),
		Status:       code.Status().String(),
		Metadata:     code.Metadata(),
		CreatedAt:    code.CreatedAt(),
		UpdatedAt:    code.UpdatedAt(),
	}, nil
}

// ListCodes 引き換えコードの一覧を取得
func (s *CodeRedemptionApplicationService) ListCodes(ctx context.Context, req *ListCodesRequest) (*ListCodesResponse, error) {
	ctx, span := s.tracer.Start(ctx, "CodeRedemptionApplicationService.ListCodes")
	defer span.End()

	span.SetAttributes(
		attribute.Int("limit", req.Limit),
		attribute.Int("offset", req.Offset),
		attribute.String("status", req.Status),
		attribute.String("code_type", req.CodeType),
	)

	s.logger.Info(ctx, "Listing redemption codes", map[string]interface{}{
		"limit":     req.Limit,
		"offset":    req.Offset,
		"status":    req.Status,
		"code_type": req.CodeType,
	})

	// ページネーションパラメータのバリデーション
	if req.Limit <= 0 {
		req.Limit = 50 // デフォルト値
	}
	if req.Limit > 100 {
		req.Limit = 100 // 最大値
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	// 一覧を取得
	codes, total, err := s.redemptionCodeRepo.FindAll(ctx, req.Limit, req.Offset)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		s.logger.Error(ctx, "Failed to list redemption codes", err, nil)
		return nil, fmt.Errorf("failed to list redemption codes: %w", err)
	}

	// フィルタリング（status, code_type）
	filteredCodes := make([]*redemption_code.RedemptionCode, 0, len(codes))
	for _, code := range codes {
		// ステータスフィルタ
		if req.Status != "" {
			if code.Status().String() != req.Status {
				continue
			}
		}

		// コードタイプフィルタ
		if req.CodeType != "" {
			if code.CodeType().String() != req.CodeType {
				continue
			}
		}

		filteredCodes = append(filteredCodes, code)
	}

	s.logger.Info(ctx, "Redemption codes listed successfully", map[string]interface{}{
		"total":  total,
		"count":  len(filteredCodes),
		"limit":  req.Limit,
		"offset": req.Offset,
	})

	return &ListCodesResponse{
		Codes:  filteredCodes,
		Total:  total,
		Limit:  req.Limit,
		Offset: req.Offset,
	}, nil
}
