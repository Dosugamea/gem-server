package redemption_code

import (
	"errors"
	"gem-server/internal/domain/currency"
	"time"
)

// RedemptionCode 引き換えコードエンティティ
type RedemptionCode struct {
	code         string
	codeType     CodeType
	currencyType currency.CurrencyType
	amount       int64 // 整数値（小数点なし）
	maxUses      int   // 0 = 無制限
	currentUses  int
	validFrom    time.Time
	validUntil   time.Time
	status       CodeStatus
	metadata     map[string]interface{}
	createdAt    time.Time
	updatedAt    time.Time
}

// NewRedemptionCode 新しいRedemptionCodeエンティティを作成
func NewRedemptionCode(
	code string,
	codeType CodeType,
	currencyType currency.CurrencyType,
	amount int64,
	maxUses int,
	validFrom time.Time,
	validUntil time.Time,
	metadata map[string]interface{},
) (*RedemptionCode, error) {
	if code == "" {
		return nil, errors.New("invalid code")
	}
	if amount <= 0 {
		return nil, errors.New("invalid amount")
	}

	now := time.Now()
	return &RedemptionCode{
		code:         code,
		codeType:     codeType,
		currencyType: currencyType,
		amount:       amount,
		maxUses:      maxUses,
		currentUses:  0,
		validFrom:    validFrom,
		validUntil:   validUntil,
		status:       CodeStatusActive,
		metadata:     metadata,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

// Code コードを返す
func (rc *RedemptionCode) Code() string {
	return rc.code
}

// CodeType コードタイプを返す
func (rc *RedemptionCode) CodeType() CodeType {
	return rc.codeType
}

// CurrencyType 通貨タイプを返す
func (rc *RedemptionCode) CurrencyType() currency.CurrencyType {
	return rc.currencyType
}

// Amount 金額を返す
func (rc *RedemptionCode) Amount() int64 {
	return rc.amount
}

// MaxUses 最大使用回数を返す
func (rc *RedemptionCode) MaxUses() int {
	return rc.maxUses
}

// CurrentUses 現在の使用回数を返す
func (rc *RedemptionCode) CurrentUses() int {
	return rc.currentUses
}

// ValidFrom 有効開始日時を返す
func (rc *RedemptionCode) ValidFrom() time.Time {
	return rc.validFrom
}

// ValidUntil 有効期限を返す
func (rc *RedemptionCode) ValidUntil() time.Time {
	return rc.validUntil
}

// Status ステータスを返す
func (rc *RedemptionCode) Status() CodeStatus {
	return rc.status
}

// Metadata メタデータを返す
func (rc *RedemptionCode) Metadata() map[string]interface{} {
	return rc.metadata
}

// CreatedAt 作成日時を返す
func (rc *RedemptionCode) CreatedAt() time.Time {
	return rc.createdAt
}

// UpdatedAt 更新日時を返す
func (rc *RedemptionCode) UpdatedAt() time.Time {
	return rc.updatedAt
}

// IsValid 有効性をチェック（有効期限、使用回数、ステータス）
func (rc *RedemptionCode) IsValid() bool {
	now := time.Now()

	// ステータスチェック
	if !rc.status.IsActive() {
		return false
	}

	// 有効期限チェック
	if now.Before(rc.validFrom) || now.After(rc.validUntil) {
		return false
	}

	// 使用回数チェック
	if rc.maxUses > 0 && rc.currentUses >= rc.maxUses {
		return false
	}

	return true
}

// CanBeRedeemed 引き換え可能かどうかをチェック
func (rc *RedemptionCode) CanBeRedeemed() bool {
	return rc.IsValid()
}

// Redeem 引き換え処理（使用回数を増やす）
func (rc *RedemptionCode) Redeem() error {
	if !rc.CanBeRedeemed() {
		return ErrCodeNotRedeemable
	}
	rc.currentUses++
	rc.updatedAt = time.Now()
	return nil
}

// Disable コードを無効化
func (rc *RedemptionCode) Disable() {
	rc.status = CodeStatusDisabled
	rc.updatedAt = time.Now()
}

// Expire コードを期限切れにする
func (rc *RedemptionCode) Expire() {
	rc.status = CodeStatusExpired
	rc.updatedAt = time.Now()
}

// SetCurrentUses 現在の使用回数を設定（リポジトリから読み込んだ際に使用）
func (rc *RedemptionCode) SetCurrentUses(uses int) {
	rc.currentUses = uses
}

// SetStatus ステータスを設定（リポジトリから読み込んだ際に使用）
func (rc *RedemptionCode) SetStatus(status CodeStatus) {
	rc.status = status
}

// MustNewRedemptionCode テスト用ヘルパー: NewRedemptionCodeを呼び出し、エラーが発生した場合はpanicする
func MustNewRedemptionCode(
	code string,
	codeType CodeType,
	currencyType currency.CurrencyType,
	amount int64,
	maxUses int,
	validFrom time.Time,
	validUntil time.Time,
	metadata map[string]interface{},
) *RedemptionCode {
	rc, err := NewRedemptionCode(code, codeType, currencyType, amount, maxUses, validFrom, validUntil, metadata)
	if err != nil {
		panic(err)
	}
	return rc
}
