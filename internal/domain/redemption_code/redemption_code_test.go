package redemption_code

import (
	"gem-server/internal/domain/currency"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedemptionCode(t *testing.T) {
	now := time.Now()
	validFrom := now.Add(-24 * time.Hour)
	validUntil := now.Add(24 * time.Hour)
	metadata := map[string]interface{}{
		"campaign": "test_campaign",
	}

	tests := []struct {
		name         string
		code         string
		codeType     CodeType
		currencyType currency.CurrencyType
		amount       int64
		maxUses      int
		validFrom    time.Time
		validUntil   time.Time
		metadata     map[string]interface{}
		want         *RedemptionCode
	}{
		{
			name:         "正常系: 引き換えコードの作成",
			code:         "TEST123",
			codeType:     CodeTypePromotion,
			currencyType: currency.CurrencyTypePaid,
			amount:       1000,
			maxUses:      1,
			validFrom:    validFrom,
			validUntil:   validUntil,
			metadata:     metadata,
			want: &RedemptionCode{
				code:         "TEST123",
				codeType:     CodeTypePromotion,
				currencyType: currency.CurrencyTypePaid,
				amount:       1000,
				maxUses:      1,
				currentUses:  0,
				validFrom:    validFrom,
				validUntil:   validUntil,
				status:       CodeStatusActive,
				metadata:     metadata,
			},
		},
		{
			name:         "正常系: 無制限使用コードの作成",
			code:         "UNLIMITED",
			codeType:     CodeTypeGift,
			currencyType: currency.CurrencyTypeFree,
			amount:       500,
			maxUses:      0,
			validFrom:    validFrom,
			validUntil:   validUntil,
			metadata:     nil,
			want: &RedemptionCode{
				code:         "UNLIMITED",
				codeType:     CodeTypeGift,
				currencyType: currency.CurrencyTypeFree,
				amount:       500,
				maxUses:      0,
				currentUses:  0,
				validFrom:    validFrom,
				validUntil:   validUntil,
				status:       CodeStatusActive,
				metadata:     nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewRedemptionCode(
				tt.code,
				tt.codeType,
				tt.currencyType,
				tt.amount,
				tt.maxUses,
				tt.validFrom,
				tt.validUntil,
				tt.metadata,
			)
			assert.Equal(t, tt.want.Code(), got.Code())
			assert.Equal(t, tt.want.CodeType(), got.CodeType())
			assert.Equal(t, tt.want.CurrencyType(), got.CurrencyType())
			assert.Equal(t, tt.want.Amount(), got.Amount())
			assert.Equal(t, tt.want.MaxUses(), got.MaxUses())
			assert.Equal(t, tt.want.CurrentUses(), got.CurrentUses())
			assert.Equal(t, tt.want.Status(), got.Status())
			assert.WithinDuration(t, tt.want.ValidFrom(), got.ValidFrom(), time.Second)
			assert.WithinDuration(t, tt.want.ValidUntil(), got.ValidUntil(), time.Second)
		})
	}
}

func TestRedemptionCode_IsValid(t *testing.T) {
	now := time.Now()
	validFrom := now.Add(-24 * time.Hour)
	validUntil := now.Add(24 * time.Hour)
	pastValidFrom := now.Add(-48 * time.Hour)
	pastValidUntil := now.Add(-24 * time.Hour)
	futureValidFrom := now.Add(24 * time.Hour)
	futureValidUntil := now.Add(48 * time.Hour)

	tests := []struct {
		name  string
		code  *RedemptionCode
		want  bool
		setup func(*RedemptionCode)
	}{
		{
			name: "正常系: 有効なコード（アクティブ、期限内、使用回数未達）",
			code: NewRedemptionCode(
				"TEST123",
				CodeTypePromotion,
				currency.CurrencyTypePaid,
				1000,
				1,
				validFrom,
				validUntil,
				nil,
			),
			want: true,
		},
		{
			name: "正常系: 無制限使用コード",
			code: NewRedemptionCode(
				"UNLIMITED",
				CodeTypeGift,
				currency.CurrencyTypePaid,
				1000,
				0,
				validFrom,
				validUntil,
				nil,
			),
			want: true,
		},
		{
			name: "異常系: 無効化されたコード",
			code: NewRedemptionCode(
				"TEST123",
				CodeTypePromotion,
				currency.CurrencyTypePaid,
				1000,
				1,
				validFrom,
				validUntil,
				nil,
			),
			want: false,
			setup: func(rc *RedemptionCode) {
				rc.Disable()
			},
		},
		{
			name: "異常系: 期限切れコード",
			code: NewRedemptionCode(
				"TEST123",
				CodeTypePromotion,
				currency.CurrencyTypePaid,
				1000,
				1,
				pastValidFrom,
				pastValidUntil,
				nil,
			),
			want: false,
		},
		{
			name: "異常系: 有効期限前のコード",
			code: NewRedemptionCode(
				"TEST123",
				CodeTypePromotion,
				currency.CurrencyTypePaid,
				1000,
				1,
				futureValidFrom,
				futureValidUntil,
				nil,
			),
			want: false,
		},
		{
			name: "異常系: 使用回数上限に達したコード",
			code: NewRedemptionCode(
				"TEST123",
				CodeTypePromotion,
				currency.CurrencyTypePaid,
				1000,
				1,
				validFrom,
				validUntil,
				nil,
			),
			want: false,
			setup: func(rc *RedemptionCode) {
				rc.SetCurrentUses(1)
			},
		},
		{
			name: "異常系: 期限切れステータスのコード",
			code: NewRedemptionCode(
				"TEST123",
				CodeTypePromotion,
				currency.CurrencyTypePaid,
				1000,
				1,
				validFrom,
				validUntil,
				nil,
			),
			want: false,
			setup: func(rc *RedemptionCode) {
				rc.Expire()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(tt.code)
			}
			got := tt.code.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRedemptionCode_CanBeRedeemed(t *testing.T) {
	now := time.Now()
	validFrom := now.Add(-24 * time.Hour)
	validUntil := now.Add(24 * time.Hour)

	tests := []struct {
		name  string
		code  *RedemptionCode
		want  bool
		setup func(*RedemptionCode)
	}{
		{
			name: "正常系: 引き換え可能なコード",
			code: NewRedemptionCode(
				"TEST123",
				CodeTypePromotion,
				currency.CurrencyTypePaid,
				1000,
				1,
				validFrom,
				validUntil,
				nil,
			),
			want: true,
		},
		{
			name: "異常系: 無効化されたコード",
			code: NewRedemptionCode(
				"TEST123",
				CodeTypePromotion,
				currency.CurrencyTypePaid,
				1000,
				1,
				validFrom,
				validUntil,
				nil,
			),
			want: false,
			setup: func(rc *RedemptionCode) {
				rc.Disable()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(tt.code)
			}
			got := tt.code.CanBeRedeemed()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRedemptionCode_Redeem(t *testing.T) {
	now := time.Now()
	validFrom := now.Add(-24 * time.Hour)
	validUntil := now.Add(24 * time.Hour)

	tests := []struct {
		name      string
		code      *RedemptionCode
		wantUses  int
		wantError error
		setup     func(*RedemptionCode)
	}{
		{
			name: "正常系: 引き換え成功",
			code: NewRedemptionCode(
				"TEST123",
				CodeTypePromotion,
				currency.CurrencyTypePaid,
				1000,
				1,
				validFrom,
				validUntil,
				nil,
			),
			wantUses:  1,
			wantError: nil,
		},
		{
			name: "正常系: 複数回引き換え可能なコード",
			code: NewRedemptionCode(
				"MULTI",
				CodeTypeGift,
				currency.CurrencyTypePaid,
				1000,
				5,
				validFrom,
				validUntil,
				nil,
			),
			wantUses:  1,
			wantError: nil,
		},
		{
			name: "異常系: 無効化されたコード",
			code: NewRedemptionCode(
				"TEST123",
				CodeTypePromotion,
				currency.CurrencyTypePaid,
				1000,
				1,
				validFrom,
				validUntil,
				nil,
			),
			wantUses:  0,
			wantError: ErrCodeNotRedeemable,
			setup: func(rc *RedemptionCode) {
				rc.Disable()
			},
		},
		{
			name: "異常系: 使用回数上限に達したコード",
			code: NewRedemptionCode(
				"TEST123",
				CodeTypePromotion,
				currency.CurrencyTypePaid,
				1000,
				1,
				validFrom,
				validUntil,
				nil,
			),
			wantUses:  1,
			wantError: ErrCodeNotRedeemable,
			setup: func(rc *RedemptionCode) {
				rc.SetCurrentUses(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(tt.code)
			}
			err := tt.code.Redeem()
			if tt.wantError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantError, err)
				assert.Equal(t, tt.wantUses, tt.code.CurrentUses())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantUses, tt.code.CurrentUses())
			}
		})
	}
}

func TestRedemptionCode_Disable(t *testing.T) {
	now := time.Now()
	validFrom := now.Add(-24 * time.Hour)
	validUntil := now.Add(24 * time.Hour)

	code := NewRedemptionCode(
		"TEST123",
		CodeTypePromotion,
		currency.CurrencyTypePaid,
		1000,
		1,
		validFrom,
		validUntil,
		nil,
	)

	assert.Equal(t, CodeStatusActive, code.Status())
	code.Disable()
	assert.Equal(t, CodeStatusDisabled, code.Status())
}

func TestRedemptionCode_Expire(t *testing.T) {
	now := time.Now()
	validFrom := now.Add(-24 * time.Hour)
	validUntil := now.Add(24 * time.Hour)

	code := NewRedemptionCode(
		"TEST123",
		CodeTypePromotion,
		currency.CurrencyTypePaid,
		1000,
		1,
		validFrom,
		validUntil,
		nil,
	)

	assert.Equal(t, CodeStatusActive, code.Status())
	code.Expire()
	assert.Equal(t, CodeStatusExpired, code.Status())
}

func TestRedemptionCode_SetCurrentUses(t *testing.T) {
	now := time.Now()
	validFrom := now.Add(-24 * time.Hour)
	validUntil := now.Add(24 * time.Hour)

	code := NewRedemptionCode(
		"TEST123",
		CodeTypePromotion,
		currency.CurrencyTypePaid,
		1000,
		1,
		validFrom,
		validUntil,
		nil,
	)

	assert.Equal(t, 0, code.CurrentUses())
	code.SetCurrentUses(5)
	assert.Equal(t, 5, code.CurrentUses())
}

func TestRedemptionCode_SetStatus(t *testing.T) {
	now := time.Now()
	validFrom := now.Add(-24 * time.Hour)
	validUntil := now.Add(24 * time.Hour)

	code := NewRedemptionCode(
		"TEST123",
		CodeTypePromotion,
		currency.CurrencyTypePaid,
		1000,
		1,
		validFrom,
		validUntil,
		nil,
	)

	assert.Equal(t, CodeStatusActive, code.Status())
	code.SetStatus(CodeStatusExpired)
	assert.Equal(t, CodeStatusExpired, code.Status())
}
