package transaction

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransactionStatus(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    TransactionStatus
		wantErr bool
	}{
		{
			name:    "正常系: pending",
			input:   "pending",
			want:    TransactionStatusPending,
			wantErr: false,
		},
		{
			name:    "正常系: completed",
			input:   "completed",
			want:    TransactionStatusCompleted,
			wantErr: false,
		},
		{
			name:    "正常系: failed",
			input:   "failed",
			want:    TransactionStatusFailed,
			wantErr: false,
		},
		{
			name:    "正常系: cancelled",
			input:   "cancelled",
			want:    TransactionStatusCancelled,
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
			got, err := NewTransactionStatus(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestTransactionStatus_String(t *testing.T) {
	tests := []struct {
		name string
		ts   TransactionStatus
		want string
	}{
		{
			name: "正常系: pending",
			ts:   TransactionStatusPending,
			want: "pending",
		},
		{
			name: "正常系: completed",
			ts:   TransactionStatusCompleted,
			want: "completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ts.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTransactionStatus_Valid(t *testing.T) {
	tests := []struct {
		name string
		ts   TransactionStatus
		want bool
	}{
		{
			name: "正常系: pending",
			ts:   TransactionStatusPending,
			want: true,
		},
		{
			name: "正常系: completed",
			ts:   TransactionStatusCompleted,
			want: true,
		},
		{
			name: "正常系: failed",
			ts:   TransactionStatusFailed,
			want: true,
		},
		{
			name: "正常系: cancelled",
			ts:   TransactionStatusCancelled,
			want: true,
		},
		{
			name: "異常系: 無効な値",
			ts:   TransactionStatus("invalid"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ts.Valid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTransactionStatus_IsCompleted(t *testing.T) {
	tests := []struct {
		name string
		ts   TransactionStatus
		want bool
	}{
		{
			name: "正常系: completed",
			ts:   TransactionStatusCompleted,
			want: true,
		},
		{
			name: "正常系: pending",
			ts:   TransactionStatusPending,
			want: false,
		},
		{
			name: "正常系: failed",
			ts:   TransactionStatusFailed,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ts.IsCompleted()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTransactionStatus_IsFailed(t *testing.T) {
	tests := []struct {
		name string
		ts   TransactionStatus
		want bool
	}{
		{
			name: "正常系: failed",
			ts:   TransactionStatusFailed,
			want: true,
		},
		{
			name: "正常系: completed",
			ts:   TransactionStatusCompleted,
			want: false,
		},
		{
			name: "正常系: pending",
			ts:   TransactionStatusPending,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ts.IsFailed()
			assert.Equal(t, tt.want, got)
		})
	}
}
