package redemption_code

import (
	"fmt"
)

// CodeType コードタイプを表す値オブジェクト
type CodeType string

const (
	CodeTypePromotion CodeType = "promotion" // プロモーションコード
	CodeTypeGift     CodeType = "gift"       // ギフトコード
	CodeTypeEvent    CodeType = "event"      // イベントコード
)

// NewCodeType 新しいCodeTypeを作成
func NewCodeType(s string) (CodeType, error) {
	switch s {
	case "promotion", "gift", "event":
		return CodeType(s), nil
	default:
		return "", fmt.Errorf("invalid code type: %s", s)
	}
}

// String 文字列表現を返す
func (ct CodeType) String() string {
	return string(ct)
}

// Valid 有効なコードタイプかどうかを返す
func (ct CodeType) Valid() bool {
	switch ct {
	case CodeTypePromotion, CodeTypeGift, CodeTypeEvent:
		return true
	default:
		return false
	}
}
