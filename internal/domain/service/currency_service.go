package service

import (
	"context"
	"gem-server/internal/domain/currency"
)

// CurrencyService 通貨関連のドメインサービス
type CurrencyService struct {
	currencyRepo currency.CurrencyRepository
}

// NewCurrencyService 新しいCurrencyServiceを作成
func NewCurrencyService(currencyRepo currency.CurrencyRepository) *CurrencyService {
	return &CurrencyService{
		currencyRepo: currencyRepo,
	}
}

// GetTotalBalance ユーザーの全通貨タイプの合計残高を取得
func (s *CurrencyService) GetTotalBalance(ctx context.Context, userID string) (int64, error) {
	paidCurrency, err := s.currencyRepo.FindByUserIDAndType(ctx, userID, currency.CurrencyTypePaid)
	if err != nil {
		return 0, err
	}

	freeCurrency, err := s.currencyRepo.FindByUserIDAndType(ctx, userID, currency.CurrencyTypeFree)
	if err != nil {
		return 0, err
	}

	return paidCurrency.Balance() + freeCurrency.Balance(), nil
}

// HasSufficientBalance 指定された金額の残高があるかチェック（無料通貨優先で計算）
func (s *CurrencyService) HasSufficientBalance(ctx context.Context, userID string, amount int64) (bool, error) {
	freeCurrency, err := s.currencyRepo.FindByUserIDAndType(ctx, userID, currency.CurrencyTypeFree)
	if err != nil {
		return false, err
	}

	freeBalance := freeCurrency.Balance()
	remainingAmount := amount

	// 無料通貨で支払える分を差し引く
	if freeBalance >= remainingAmount {
		return true, nil
	}

	remainingAmount -= freeBalance

	// 不足分を有料通貨で支払えるかチェック
	paidCurrency, err := s.currencyRepo.FindByUserIDAndType(ctx, userID, currency.CurrencyTypePaid)
	if err != nil {
		return false, err
	}

	return paidCurrency.Balance() >= remainingAmount, nil
}
