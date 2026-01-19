package transaction

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransactionType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    TransactionType
		wantErr bool
	}{
		{
			name:    "正常系: grant",
			input:   "grant",
			want:    TransactionTypeGrant,
			wantErr: false,
		},
		{
			name:    "正常系: consume",
			input:   "consume",
			want:    TransactionTypeConsume,
			wantErr: false,
		},
		{
			name:    "正常系: refund",
			input:   "refund",
			want:    TransactionTypeRefund,
			wantErr: false,
		},
		{
			name:    "正常系: expire",
			input:   "expire",
			want:    TransactionTypeExpire,
			wantErr: false,
		},
		{
			name:    "正常系: compensate",
			input:   "compensate",
			want:    TransactionTypeCompensate,
			wantErr: false,
		},
		{
			name:    "異常系: 無効な値",
			input:   "invalid",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTransactionType(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestTransactionType_String(t *testing.T) {
	tests := []struct {
		name string
		tt   TransactionType
		want string
	}{
		{
			name: "正常系: grant",
			tt:   TransactionTypeGrant,
			want: "grant",
		},
		{
			name: "正常系: consume",
			tt:   TransactionTypeConsume,
			want: "consume",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tt.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTransactionType_Valid(t *testing.T) {
	tests := []struct {
		name string
		tt   TransactionType
		want bool
	}{
		{
			name: "正常系: grant",
			tt:   TransactionTypeGrant,
			want: true,
		},
		{
			name: "正常系: consume",
			tt:   TransactionTypeConsume,
			want: true,
		},
		{
			name: "正常系: refund",
			tt:   TransactionTypeRefund,
			want: true,
		},
		{
			name: "正常系: expire",
			tt:   TransactionTypeExpire,
			want: true,
		},
		{
			name: "正常系: compensate",
			tt:   TransactionTypeCompensate,
			want: true,
		},
		{
			name: "異常系: 無効な値",
			tt:   TransactionType("invalid"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tt.Valid()
			assert.Equal(t, tt.want, got)
		})
	}
}
