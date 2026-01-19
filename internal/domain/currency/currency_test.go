package currency

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCurrency(t *testing.T) {
	tests := []struct {
		name         string
		userID       string
		currencyType CurrencyType
		balance      int64
		version      int
		want         *Currency
	}{
		{
			name:         "正常系: 有償通貨の作成",
			userID:       "user123",
			currencyType: CurrencyTypePaid,
			balance:      1000,
			version:      1,
			want: &Currency{
				userID:       "user123",
				currencyType: CurrencyTypePaid,
				balance:      1000,
				version:      1,
			},
		},
		{
			name:         "正常系: 無償通貨の作成",
			userID:       "user456",
			currencyType: CurrencyTypeFree,
			balance:      500,
			version:      0,
			want: &Currency{
				userID:       "user456",
				currencyType: CurrencyTypeFree,
				balance:      500,
				version:      0,
			},
		},
		{
			name:         "正常系: マイナス残高の作成",
			userID:       "user789",
			currencyType: CurrencyTypePaid,
			balance:      -100,
			version:      1,
			want: &Currency{
				userID:       "user789",
				currencyType: CurrencyTypePaid,
				balance:      -100,
				version:      1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewCurrency(tt.userID, tt.currencyType, tt.balance, tt.version)
			assert.Equal(t, tt.want.userID, got.UserID())
			assert.Equal(t, tt.want.currencyType, got.CurrencyType())
			assert.Equal(t, tt.want.balance, got.Balance())
			assert.Equal(t, tt.want.version, got.Version())
		})
	}
}

func TestCurrency_Grant(t *testing.T) {
	tests := []struct {
		name        string
		currency    *Currency
		amount      int64
		wantBalance int64
		wantVersion int
		wantError   error
	}{
		{
			name:        "正常系: 通貨を付与",
			currency:    NewCurrency("user123", CurrencyTypePaid, 1000, 1),
			amount:      500,
			wantBalance: 1500,
			wantVersion: 2,
			wantError:   nil,
		},
		{
			name:        "正常系: ゼロ残高から付与",
			currency:    NewCurrency("user123", CurrencyTypePaid, 0, 1),
			amount:      1000,
			wantBalance: 1000,
			wantVersion: 2,
			wantError:   nil,
		},
		{
			name:        "正常系: マイナス残高から付与",
			currency:    NewCurrency("user123", CurrencyTypePaid, -100, 1),
			amount:      200,
			wantBalance: 100,
			wantVersion: 2,
			wantError:   nil,
		},
		{
			name:        "異常系: 無効な金額（0）",
			currency:    NewCurrency("user123", CurrencyTypePaid, 1000, 1),
			amount:      0,
			wantBalance: 1000,
			wantVersion: 1,
			wantError:   ErrInvalidAmount,
		},
		{
			name:        "異常系: 無効な金額（マイナス）",
			currency:    NewCurrency("user123", CurrencyTypePaid, 1000, 1),
			amount:      -100,
			wantBalance: 1000,
			wantVersion: 1,
			wantError:   ErrInvalidAmount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.currency.Grant(tt.amount)
			if tt.wantError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantError, err)
				assert.Equal(t, tt.wantBalance, tt.currency.Balance())
				assert.Equal(t, tt.wantVersion, tt.currency.Version())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantBalance, tt.currency.Balance())
				assert.Equal(t, tt.wantVersion, tt.currency.Version())
			}
		})
	}
}

func TestCurrency_Consume(t *testing.T) {
	tests := []struct {
		name        string
		currency    *Currency
		amount      int64
		wantBalance int64
		wantVersion int
		wantError   error
	}{
		{
			name:        "正常系: 通貨を消費",
			currency:    NewCurrency("user123", CurrencyTypePaid, 1000, 1),
			amount:      300,
			wantBalance: 700,
			wantVersion: 2,
			wantError:   nil,
		},
		{
			name:        "正常系: 残高全額を消費",
			currency:    NewCurrency("user123", CurrencyTypePaid, 1000, 1),
			amount:      1000,
			wantBalance: 0,
			wantVersion: 2,
			wantError:   nil,
		},
		{
			name:        "異常系: 残高不足",
			currency:    NewCurrency("user123", CurrencyTypePaid, 1000, 1),
			amount:      1500,
			wantBalance: 1000,
			wantVersion: 1,
			wantError:   ErrInsufficientBalance,
		},
		{
			name:        "異常系: 無効な金額（0）",
			currency:    NewCurrency("user123", CurrencyTypePaid, 1000, 1),
			amount:      0,
			wantBalance: 1000,
			wantVersion: 1,
			wantError:   ErrInvalidAmount,
		},
		{
			name:        "異常系: 無効な金額（マイナス）",
			currency:    NewCurrency("user123", CurrencyTypePaid, 1000, 1),
			amount:      -100,
			wantBalance: 1000,
			wantVersion: 1,
			wantError:   ErrInvalidAmount,
		},
		{
			name:        "異常系: ゼロ残高から消費",
			currency:    NewCurrency("user123", CurrencyTypePaid, 0, 1),
			amount:      100,
			wantBalance: 0,
			wantVersion: 1,
			wantError:   ErrInsufficientBalance,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.currency.Consume(tt.amount)
			if tt.wantError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantError, err)
				assert.Equal(t, tt.wantBalance, tt.currency.Balance())
				assert.Equal(t, tt.wantVersion, tt.currency.Version())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantBalance, tt.currency.Balance())
				assert.Equal(t, tt.wantVersion, tt.currency.Version())
			}
		})
	}
}

func TestCurrency_ConsumeAllowNegative(t *testing.T) {
	tests := []struct {
		name        string
		currency    *Currency
		amount      int64
		wantBalance int64
		wantVersion int
		wantError   error
	}{
		{
			name:        "正常系: 通貨を消費（残高あり）",
			currency:    NewCurrency("user123", CurrencyTypePaid, 1000, 1),
			amount:      300,
			wantBalance: 700,
			wantVersion: 2,
			wantError:   nil,
		},
		{
			name:        "正常系: マイナス残高を許可して消費",
			currency:    NewCurrency("user123", CurrencyTypePaid, 1000, 1),
			amount:      1500,
			wantBalance: -500,
			wantVersion: 2,
			wantError:   nil,
		},
		{
			name:        "正常系: ゼロ残高からマイナスへ",
			currency:    NewCurrency("user123", CurrencyTypePaid, 0, 1),
			amount:      500,
			wantBalance: -500,
			wantVersion: 2,
			wantError:   nil,
		},
		{
			name:        "正常系: 既にマイナス残高からさらに消費",
			currency:    NewCurrency("user123", CurrencyTypePaid, -100, 1),
			amount:      200,
			wantBalance: -300,
			wantVersion: 2,
			wantError:   nil,
		},
		{
			name:        "異常系: 無効な金額（0）",
			currency:    NewCurrency("user123", CurrencyTypePaid, 1000, 1),
			amount:      0,
			wantBalance: 1000,
			wantVersion: 1,
			wantError:   ErrInvalidAmount,
		},
		{
			name:        "異常系: 無効な金額（マイナス）",
			currency:    NewCurrency("user123", CurrencyTypePaid, 1000, 1),
			amount:      -100,
			wantBalance: 1000,
			wantVersion: 1,
			wantError:   ErrInvalidAmount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.currency.ConsumeAllowNegative(tt.amount)
			if tt.wantError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantError, err)
				assert.Equal(t, tt.wantBalance, tt.currency.Balance())
				assert.Equal(t, tt.wantVersion, tt.currency.Version())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantBalance, tt.currency.Balance())
				assert.Equal(t, tt.wantVersion, tt.currency.Version())
			}
		})
	}
}

func TestCurrency_IncrementVersion(t *testing.T) {
	tests := []struct {
		name        string
		currency    *Currency
		wantVersion int
	}{
		{
			name:        "正常系: バージョンをインクリメント",
			currency:    NewCurrency("user123", CurrencyTypePaid, 1000, 1),
			wantVersion: 2,
		},
		{
			name:        "正常系: ゼロからインクリメント",
			currency:    NewCurrency("user123", CurrencyTypePaid, 1000, 0),
			wantVersion: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.currency.IncrementVersion()
			assert.Equal(t, tt.wantVersion, tt.currency.Version())
		})
	}
}
