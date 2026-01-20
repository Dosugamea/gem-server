package code_redemption

import (
	"time"

	"gem-server/internal/domain/redemption_code"
)

// RedeemCodeRequest コード引き換えリクエスト
type RedeemCodeRequest struct {
	Code   string
	UserID string
}

// RedeemCodeResponse コード引き換えレスポンス
type RedeemCodeResponse struct {
	RedemptionID  string
	TransactionID string
	Code          string
	CurrencyType  string
	Amount        int64
	BalanceAfter  int64
	Status        string
}

// CreateCodeRequest 引き換えコード作成リクエスト
type CreateCodeRequest struct {
	Code         string
	CodeType     string
	CurrencyType string
	Amount       int64
	MaxUses      int
	ValidFrom    time.Time
	ValidUntil   time.Time
	Metadata     map[string]interface{}
}

// CreateCodeResponse 引き換えコード作成レスポンス
type CreateCodeResponse struct {
	Code         string
	CodeType     string
	CurrencyType string
	Amount       int64
	MaxUses      int
	CurrentUses  int
	ValidFrom    time.Time
	ValidUntil   time.Time
	Status       string
	Metadata     map[string]interface{}
	CreatedAt    time.Time
}

// DeleteCodeRequest 引き換えコード削除リクエスト
type DeleteCodeRequest struct {
	Code string
}

// DeleteCodeResponse 引き換えコード削除レスポンス
type DeleteCodeResponse struct {
	Code      string
	DeletedAt time.Time
}

// GetCodeRequest 引き換えコード取得リクエスト
type GetCodeRequest struct {
	Code string
}

// GetCodeResponse 引き換えコード取得レスポンス
type GetCodeResponse struct {
	Code         string
	CodeType     string
	CurrencyType string
	Amount       int64
	MaxUses      int
	CurrentUses  int
	ValidFrom    time.Time
	ValidUntil   time.Time
	Status       string
	Metadata     map[string]interface{}
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ListCodesRequest 引き換えコード一覧取得リクエスト
type ListCodesRequest struct {
	Limit    int
	Offset   int
	Status   string // optional: "active", "expired", "disabled"
	CodeType string // optional: "promotion", "gift", "event"
}

// ListCodesResponse 引き換えコード一覧取得レスポンス
type ListCodesResponse struct {
	Codes  []*redemption_code.RedemptionCode
	Total  int
	Limit  int
	Offset int
}
