---
name: タスク08 - 実装の詳細
overview: 依存性注入、エラーハンドリング、コード引き換え、トランザクション管理の実装詳細を定義します
---

# タスク08: 実装の詳細

## 11.1 依存性注入とDIコンテナ

```go
// infrastructure/di/container.go
package di

import (
    "github.com/google/wire"
)

//go:generate wire

func InitializeApplication() (*Application, error) {
    wire.Build(
        // Infrastructure
        NewMySQLRepository,
        NewRedisCache,
        NewTracerProvider,
        NewMeterProvider,
        
        // Application
        NewCurrencyApplicationService,
        NewPaymentApplicationService,
        NewHistoryApplicationService,
        
        // Presentation
        NewCurrencyHandler,
        NewPaymentHandler,
        NewHistoryHandler,
        NewEchoServer,
        
        // Application struct
        NewApplication,
    )
    return &Application{}, nil
}
```

## 11.2 エラーハンドリングとバリデーション

```go
// domain/currency/errors.go
package currency

import "errors"

var (
    ErrInsufficientBalance = errors.New("insufficient balance")
    ErrInvalidAmount       = errors.New("invalid amount")
    ErrDuplicateTransaction = errors.New("duplicate transaction")
)

// application/currency/service.go

// 指定通貨タイプでの消費
func (s *CurrencyApplicationService) Consume(ctx context.Context, req *ConsumeRequest) (*ConsumeResponse, error) {
    // バリデーション
    if err := s.validateConsumeRequest(req); err != nil {
        return nil, fmt.Errorf("validation error: %w", err)
    }
    
    // 優先順位制御が有効な場合は優先順位付き消費を使用
    if req.UsePriority || req.CurrencyType == "auto" {
        return s.ConsumeWithPriority(ctx, req)
    }
    
    // ドメインロジック実行
    currency, err := s.currencyRepo.FindByUserIDAndType(ctx, req.UserID, req.CurrencyType)
    if err != nil {
        return nil, fmt.Errorf("failed to find currency: %w", err)
    }
    
    if err := currency.Consume(req.Amount); err != nil {
        return nil, fmt.Errorf("failed to consume: %w", err)
    }
    
    // トランザクション記録
    // ...
}

// 優先順位付き消費（無料通貨優先）
func (s *CurrencyApplicationService) ConsumeWithPriority(ctx context.Context, req *ConsumeRequest) (*ConsumeResponse, error) {
    // バリデーション
    if err := s.validateConsumeRequest(req); err != nil {
        return nil, fmt.Errorf("validation error: %w", err)
    }
    
    totalAmount := req.Amount // int64
    var consumptionDetails []ConsumptionDetail
    
    // 1. 無料通貨の残高を確認
    freeCurrency, err := s.currencyRepo.FindByUserIDAndType(ctx, req.UserID, "free")
    if err != nil {
        return nil, fmt.Errorf("failed to find free currency: %w", err)
    }
    
    freeBalance := freeCurrency.Balance() // int64
    remainingAmount := totalAmount
    
    // 2. 無料通貨で支払える分を消費
    if freeBalance >= remainingAmount {
        // 無料通貨だけで足りる場合
        freeAmount := remainingAmount
        if err := freeCurrency.Consume(freeAmount); err != nil {
            return nil, fmt.Errorf("failed to consume free currency: %w", err)
        }
        consumptionDetails = append(consumptionDetails, ConsumptionDetail{
            CurrencyType: "free",
            Amount: freeAmount,
            BalanceBefore: freeBalance,
            BalanceAfter: freeBalance - freeAmount,
        })
        remainingAmount = 0
    } else {
        // 無料通貨を全て消費
        if freeBalance > 0 {
            if err := freeCurrency.Consume(freeBalance); err != nil {
                return nil, fmt.Errorf("failed to consume free currency: %w", err)
            }
            consumptionDetails = append(consumptionDetails, ConsumptionDetail{
                CurrencyType: "free",
                Amount: freeBalance,
                BalanceBefore: freeBalance,
                BalanceAfter: 0,
            })
            remainingAmount = remainingAmount - freeBalance
        }
    }
    
    // 3. 不足分があれば有料通貨から消費
    if remainingAmount > 0 {
        paidCurrency, err := s.currencyRepo.FindByUserIDAndType(ctx, req.UserID, "paid")
        if err != nil {
            return nil, fmt.Errorf("failed to find paid currency: %w", err)
        }
        
        paidBalance := paidCurrency.Balance() // int64
        if paidBalance < remainingAmount {
            return nil, fmt.Errorf("insufficient balance: need %d, have %d (free: %d, paid: %d)", 
                totalAmount, freeBalance+paidBalance, freeBalance, paidBalance)
        }
        
        if err := paidCurrency.Consume(remainingAmount); err != nil {
            return nil, fmt.Errorf("failed to consume paid currency: %w", err)
        }
        
        consumptionDetails = append(consumptionDetails, ConsumptionDetail{
            CurrencyType: "paid",
            Amount: remainingAmount,
            BalanceBefore: paidBalance,
            BalanceAfter: paidBalance - remainingAmount,
        })
    }
    
    // 4. 各通貨タイプごとにトランザクション履歴を記録
    // ...
    
    return &ConsumeResponse{
        TransactionID: generateTransactionID(),
        ConsumptionDetails: consumptionDetails,
        TotalConsumed: totalAmount,
        Status: "completed",
    }, nil
}
```

## 11.3 コード引き換えの実装

```go
// application/code_redemption/service.go
package code_redemption

import (
    "context"
    "fmt"
    "github.com/shopspring/decimal"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
)

type CodeRedemptionApplicationService struct {
    codeRepo        domain.RedemptionCodeRepository
    currencyRepo    domain.CurrencyRepository
    transactionRepo domain.TransactionRepository
    txManager      infrastructure.TransactionManager
}

func (s *CodeRedemptionApplicationService) Redeem(ctx context.Context, req *RedeemCodeRequest) (*RedeemCodeResponse, error) {
    tracer := otel.Tracer("code-redemption-service")
    ctx, span := tracer.Start(ctx, "CodeRedemptionApplicationService.Redeem")
    defer span.End()
    
    span.SetAttributes(
        attribute.String("code", req.Code),
        attribute.String("user_id", req.UserID),
    )
    
    // コードの取得と検証
    code, err := s.codeRepo.FindByCode(ctx, req.Code)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, fmt.Errorf("code not found: %w", err)
    }
    
    // コードの有効性チェック
    if !code.CanBeRedeemed() {
        span.SetStatus(codes.Error, "code not redeemable")
        return nil, domain.ErrCodeNotRedeemable
    }
    
    // ユーザーが既にこのコードを引き換え済みかチェック
    alreadyRedeemed, err := s.codeRepo.HasUserRedeemed(ctx, req.Code, req.UserID)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, fmt.Errorf("failed to check redemption: %w", err)
    }
    if alreadyRedeemed {
        span.SetStatus(codes.Error, "user already redeemed")
        return nil, domain.ErrUserAlreadyRedeemed
    }
    
    // トランザクション内で処理
    var result *RedeemCodeResponse
    err = s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
        // コードの使用回数を更新
        if err := code.Redeem(); err != nil {
            return err
        }
        if err := s.codeRepo.Update(ctx, code); err != nil {
            return fmt.Errorf("failed to update code: %w", err)
        }
        
        // 通貨残高を取得
        currency, err := s.currencyRepo.FindByUserIDAndType(ctx, req.UserID, code.CurrencyType())
        if err != nil {
            return fmt.Errorf("failed to find currency: %w", err)
        }
        
        balanceBefore := currency.Balance()
        
        // 通貨を付与
        if err := currency.Grant(code.Amount()); err != nil {
            return err
        }
        
        // 通貨残高を更新
        if err := s.currencyRepo.Save(ctx, currency); err != nil {
            return fmt.Errorf("failed to save currency: %w", err)
        }
        
        // トランザクション履歴を作成
        transactionID := generateTransactionID()
        transaction := domain.NewTransaction(
            transactionID,
            req.UserID,
            domain.TransactionTypeGrant,
            code.CurrencyType(),
            code.Amount(),
            balanceBefore,
            currency.Balance(),
            domain.TransactionStatusCompleted,
            map[string]interface{}{
                "code": req.Code,
                "code_type": code.CodeType().String(),
            },
        )
        
        if err := s.transactionRepo.Save(ctx, transaction); err != nil {
            return fmt.Errorf("failed to save transaction: %w", err)
        }
        
        // 引き換え履歴を作成
        redemptionID := generateRedemptionID()
        redemption := domain.NewCodeRedemption(
            redemptionID,
            req.Code,
            req.UserID,
            transactionID,
        )
        
        if err := s.codeRepo.SaveRedemption(ctx, redemption); err != nil {
            return fmt.Errorf("failed to save redemption: %w", err)
        }
        
        result = &RedeemCodeResponse{
            RedemptionID: redemptionID,
            TransactionID: transactionID,
            Code: req.Code,
            CurrencyType: code.CurrencyType().String(),
            Amount: code.Amount().String(),
            BalanceAfter: currency.Balance().String(),
            Status: "completed",
        }
        
        return nil
    })
    
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, err
    }
    
    span.SetAttributes(
        attribute.String("redemption_id", result.RedemptionID),
        attribute.String("transaction_id", result.TransactionID),
    )
    
    return result, nil
}
```

## 11.4 トランザクション管理

```go
// infrastructure/persistence/mysql/repository.go
type MySQLRepository struct {
    db *sql.DB
}

func (r *MySQLRepository) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    
    defer func() {
        if p := recover(); p != nil {
            tx.Rollback()
            panic(p)
        } else if err != nil {
            tx.Rollback()
        } else {
            err = tx.Commit()
        }
    }()
    
    ctx = context.WithValue(ctx, "tx", tx)
    return fn(ctx)
}
```
