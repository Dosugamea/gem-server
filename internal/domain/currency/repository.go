package currency

import (
	"context"
)

// CurrencyRepository 通貨リポジトリインターフェース
type CurrencyRepository interface {
	// FindByUserIDAndType ユーザーIDと通貨タイプで通貨を取得
	FindByUserIDAndType(ctx context.Context, userID string, currencyType CurrencyType) (*Currency, error)
	
	// Save 通貨を保存（更新、楽観的ロック対応）
	Save(ctx context.Context, currency *Currency) error
	
	// Create 新しい通貨を作成
	Create(ctx context.Context, currency *Currency) error
}
