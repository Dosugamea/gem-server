package redemption_code

import (
	"fmt"
)

// CodeStatus コードステータスを表す値オブジェクト
type CodeStatus string

const (
	CodeStatusActive   CodeStatus = "active"   // 有効
	CodeStatusExpired  CodeStatus = "expired"  // 期限切れ
	CodeStatusDisabled CodeStatus = "disabled" // 無効化
)

// NewCodeStatus 新しいCodeStatusを作成
func NewCodeStatus(s string) (CodeStatus, error) {
	switch s {
	case "active", "expired", "disabled":
		return CodeStatus(s), nil
	default:
		return "", fmt.Errorf("invalid code status: %s", s)
	}
}

// String 文字列表現を返す
func (cs CodeStatus) String() string {
	return string(cs)
}

// Valid 有効なコードステータスかどうかを返す
func (cs CodeStatus) Valid() bool {
	switch cs {
	case CodeStatusActive, CodeStatusExpired, CodeStatusDisabled:
		return true
	default:
		return false
	}
}

// IsActive 有効状態かどうかを返す
func (cs CodeStatus) IsActive() bool {
	return cs == CodeStatusActive
}
