package currency

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCurrencyType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    CurrencyType
		wantErr bool
	}{
		{
			name:    "正常系: paid",
			input:   "paid",
			want:    CurrencyTypePaid,
			wantErr: false,
		},
		{
			name:    "正常系: free",
			input:   "free",
			want:    CurrencyTypeFree,
			wantErr: false,
		},
		{
			name:    "異常系: 無効な値",
			input:   "invalid",
			want:    "",
			wantErr: true,
		},
		{
			name:    "異常系: 空文字列",
			input:   "",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCurrencyType(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCurrencyType_String(t *testing.T) {
	tests := []struct {
		name string
		ct   CurrencyType
		want string
	}{
		{
			name: "正常系: paid",
			ct:   CurrencyTypePaid,
			want: "paid",
		},
		{
			name: "正常系: free",
			ct:   CurrencyTypeFree,
			want: "free",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ct.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCurrencyType_IsPaid(t *testing.T) {
	tests := []struct {
		name string
		ct   CurrencyType
		want bool
	}{
		{
			name: "正常系: paid",
			ct:   CurrencyTypePaid,
			want: true,
		},
		{
			name: "正常系: free",
			ct:   CurrencyTypeFree,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ct.IsPaid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCurrencyType_IsFree(t *testing.T) {
	tests := []struct {
		name string
		ct   CurrencyType
		want bool
	}{
		{
			name: "正常系: free",
			ct:   CurrencyTypeFree,
			want: true,
		},
		{
			name: "正常系: paid",
			ct:   CurrencyTypePaid,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ct.IsFree()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCurrencyType_Valid(t *testing.T) {
	tests := []struct {
		name string
		ct   CurrencyType
		want bool
	}{
		{
			name: "正常系: paid",
			ct:   CurrencyTypePaid,
			want: true,
		},
		{
			name: "正常系: free",
			ct:   CurrencyTypeFree,
			want: true,
		},
		{
			name: "異常系: 無効な値",
			ct:   CurrencyType("invalid"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ct.Valid()
			assert.Equal(t, tt.want, got)
		})
	}
}
