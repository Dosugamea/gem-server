package redemption_code

import "errors"

var (
	// ErrCodeNotFound 引き換えコードが見つからないエラー
	ErrCodeNotFound = errors.New("code not found")
	// ErrCodeExpired 引き換えコードが期限切れエラー
	ErrCodeExpired = errors.New("code expired")
	// ErrCodeAlreadyUsed 引き換えコードが既に使用済みエラー
	ErrCodeAlreadyUsed = errors.New("code already used")
	// ErrCodeDisabled 引き換えコードが無効化されているエラー
	ErrCodeDisabled = errors.New("code disabled")
	// ErrCodeMaxUsesReached 引き換えコードの使用上限に達しているエラー
	ErrCodeMaxUsesReached = errors.New("code max uses reached")
	// ErrUserAlreadyRedeemed ユーザーが既にこのコードを引き換え済みエラー
	ErrUserAlreadyRedeemed = errors.New("user already redeemed")
	// ErrCodeNotRedeemable 引き換え不可能なコードエラー
	ErrCodeNotRedeemable = errors.New("code not redeemable")
)
