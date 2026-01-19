package currency

import (
	"fmt"
)

// CurrencyType 通貨タイプを表す値オブジェクト
type CurrencyType string

const (
	CurrencyTypePaid CurrencyType = "paid" // 有償通貨
	CurrencyTypeFree CurrencyType = "free" // 無償通貨
)

// NewCurrencyType 新しいCurrencyTypeを作成
func NewCurrencyType(s string) (CurrencyType, error) {
	switch s {
	case "paid", "free":
		return CurrencyType(s), nil
	default:
		return "", fmt.Errorf("invalid currency type: %s", s)
	}
}

// String 文字列表現を返す
func (ct CurrencyType) String() string {
	return string(ct)
}

// IsPaid 有償通貨かどうかを返す
func (ct CurrencyType) IsPaid() bool {
	return ct == CurrencyTypePaid
}

// IsFree 無償通貨かどうかを返す
func (ct CurrencyType) IsFree() bool {
	return ct == CurrencyTypeFree
}

// Valid 有効な通貨タイプかどうかを返す
func (ct CurrencyType) Valid() bool {
	return ct == CurrencyTypePaid || ct == CurrencyTypeFree
}
