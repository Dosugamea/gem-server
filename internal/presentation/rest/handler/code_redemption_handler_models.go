package handler

// RedeemCodeRequest コード引き換えリクエスト
// @Description コード引き換えリクエスト
type RedeemCodeRequest struct {
	Code string `json:"code" example:"REDEEM123"`
}

// RedeemCodeResponse コード引き換えレスポンス
// @Description コード引き換えレスポンス
type RedeemCodeResponse struct {
	RedemptionID  string `json:"redemption_id" example:"red_123"`
	TransactionID string `json:"transaction_id" example:"txn_456"`
	Code          string `json:"code" example:"REDEEM123"`
	CurrencyType  string `json:"currency_type" example:"free" enums:"paid,free"`
	Amount        string `json:"amount" example:"500"`
	BalanceAfter  string `json:"balance_after" example:"1000"`
	Status        string `json:"status" example:"completed"`
}

// CreateCodeRequest 引き換えコード作成リクエスト
// @Description 引き換えコード作成リクエスト
type CreateCodeRequest struct {
	Code         string                 `json:"code" example:"PROMO2024"`
	CodeType     string                 `json:"code_type" example:"promotion" enums:"promotion,gift,event"`
	CurrencyType string                 `json:"currency_type" example:"free" enums:"paid,free"`
	Amount       string                 `json:"amount" example:"1000"`
	MaxUses      int                    `json:"max_uses" example:"100"`
	ValidFrom    string                 `json:"valid_from" example:"2024-01-01T00:00:00Z"`
	ValidUntil   string                 `json:"valid_until" example:"2024-12-31T23:59:59Z"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// CreateCodeResponse 引き換えコード作成レスポンス
// @Description 引き換えコード作成レスポンス
type CreateCodeResponse struct {
	Code         string                 `json:"code" example:"PROMO2024"`
	CodeType     string                 `json:"code_type" example:"promotion"`
	CurrencyType string                 `json:"currency_type" example:"free"`
	Amount       string                 `json:"amount" example:"1000"`
	MaxUses      int                    `json:"max_uses" example:"100"`
	CurrentUses  int                    `json:"current_uses" example:"0"`
	ValidFrom    string                 `json:"valid_from" example:"2024-01-01T00:00:00Z"`
	ValidUntil   string                 `json:"valid_until" example:"2024-12-31T23:59:59Z"`
	Status       string                 `json:"status" example:"active"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    string                 `json:"created_at" example:"2024-01-01T00:00:00Z"`
}

// DeleteCodeResponse 引き換えコード削除レスポンス
// @Description 引き換えコード削除レスポンス
type DeleteCodeResponse struct {
	Code      string `json:"code" example:"PROMO2024"`
	DeletedAt string `json:"deleted_at" example:"2024-01-01T00:00:00Z"`
}

// GetCodeResponse 引き換えコード取得レスポンス
// @Description 引き換えコード取得レスポンス
type GetCodeResponse struct {
	Code         string                 `json:"code" example:"PROMO2024"`
	CodeType     string                 `json:"code_type" example:"promotion"`
	CurrencyType string                 `json:"currency_type" example:"free"`
	Amount       string                 `json:"amount" example:"1000"`
	MaxUses      int                    `json:"max_uses" example:"100"`
	CurrentUses  int                    `json:"current_uses" example:"0"`
	ValidFrom    string                 `json:"valid_from" example:"2024-01-01T00:00:00Z"`
	ValidUntil   string                 `json:"valid_until" example:"2024-12-31T23:59:59Z"`
	Status       string                 `json:"status" example:"active"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    string                 `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt    string                 `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// ListCodesResponse 引き換えコード一覧取得レスポンス
// @Description 引き換えコード一覧取得レスポンス
type ListCodesResponse struct {
	Codes  []CodeItem `json:"codes"`
	Total  int        `json:"total" example:"100"`
	Limit  int        `json:"limit" example:"50"`
	Offset int        `json:"offset" example:"0"`
}

// CodeItem 引き換えコードアイテム
// @Description 引き換えコードアイテム
type CodeItem struct {
	Code         string                 `json:"code" example:"PROMO2024"`
	CodeType     string                 `json:"code_type" example:"promotion"`
	CurrencyType string                 `json:"currency_type" example:"free"`
	Amount       string                 `json:"amount" example:"1000"`
	MaxUses      int                    `json:"max_uses" example:"100"`
	CurrentUses  int                    `json:"current_uses" example:"0"`
	ValidFrom    string                 `json:"valid_from" example:"2024-01-01T00:00:00Z"`
	ValidUntil   string                 `json:"valid_until" example:"2024-12-31T23:59:59Z"`
	Status       string                 `json:"status" example:"active"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    string                 `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt    string                 `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}
