package code_redemption

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
