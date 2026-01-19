package redemption_code

import (
	"context"
	"time"
)

// CodeRedemption コード引き換え履歴エンティティ
type CodeRedemption struct {
	redemptionID  string
	code          string
	userID        string
	transactionID string
	redeemedAt    time.Time
}

// NewCodeRedemption 新しいCodeRedemptionエンティティを作成
func NewCodeRedemption(redemptionID, code, userID, transactionID string) *CodeRedemption {
	return &CodeRedemption{
		redemptionID:  redemptionID,
		code:          code,
		userID:        userID,
		transactionID: transactionID,
		redeemedAt:    time.Now(),
	}
}

// RedemptionID 引き換えIDを返す
func (cr *CodeRedemption) RedemptionID() string {
	return cr.redemptionID
}

// Code コードを返す
func (cr *CodeRedemption) Code() string {
	return cr.code
}

// UserID ユーザーIDを返す
func (cr *CodeRedemption) UserID() string {
	return cr.userID
}

// TransactionID トランザクションIDを返す
func (cr *CodeRedemption) TransactionID() string {
	return cr.transactionID
}

// RedeemedAt 引き換え日時を返す
func (cr *CodeRedemption) RedeemedAt() time.Time {
	return cr.redeemedAt
}

// RedemptionCodeRepository 引き換えコードリポジトリインターフェース
type RedemptionCodeRepository interface {
	// FindByCode コードで引き換えコードを取得
	FindByCode(ctx context.Context, code string) (*RedemptionCode, error)
	
	// Update 引き換えコードを更新
	Update(ctx context.Context, code *RedemptionCode) error
	
	// HasUserRedeemed ユーザーが既にこのコードを引き換え済みかチェック
	HasUserRedeemed(ctx context.Context, code string, userID string) (bool, error)
	
	// SaveRedemption 引き換え履歴を保存
	SaveRedemption(ctx context.Context, redemption *CodeRedemption) error
}
