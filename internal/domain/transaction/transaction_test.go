package transaction

import (
	"gem-server/internal/domain/currency"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustNewTransaction(
	transactionID string,
	userID string,
	transactionType TransactionType,
	currencyType currency.CurrencyType,
	amount int64,
	balanceBefore int64,
	balanceAfter int64,
	status TransactionStatus,
	metadata map[string]interface{},
) *Transaction {
	tx, err := NewTransaction(
		transactionID, userID, transactionType, currencyType, amount, balanceBefore, balanceAfter, status, metadata,
	)
	if err != nil {
		panic(err)
	}
	return tx
}

func mustNewTransactionWithRequester(
	transactionID string,
	userID string,
	transactionType TransactionType,
	currencyType currency.CurrencyType,
	amount int64,
	balanceBefore int64,
	balanceAfter int64,
	status TransactionStatus,
	requester *string,
	metadata map[string]interface{},
) *Transaction {
	tx, err := NewTransactionWithRequester(
		transactionID, userID, transactionType, currencyType, amount, balanceBefore, balanceAfter, status, requester, metadata,
	)
	if err != nil {
		panic(err)
	}
	return tx
}

func TestNewTransaction(t *testing.T) {
	metadata := map[string]interface{}{
		"reason": "test",
	}

	tests := []struct {
		name            string
		transactionID   string
		userID          string
		transactionType TransactionType
		currencyType    currency.CurrencyType
		amount          int64
		balanceBefore   int64
		balanceAfter    int64
		status          TransactionStatus
		metadata        map[string]interface{}
		want            *Transaction
		wantErr         error
	}{
		{
			name:            "正常系: 付与トランザクション",
			transactionID:   "tx123",
			userID:          "user123",
			transactionType: TransactionTypeGrant,
			currencyType:    currency.CurrencyTypePaid,
			amount:          1000,
			balanceBefore:   0,
			balanceAfter:    1000,
			status:          TransactionStatusCompleted,
			metadata:        metadata,
			want: &Transaction{
				transactionID:    "tx123",
				userID:           "user123",
				transactionType:  TransactionTypeGrant,
				currencyType:     currency.CurrencyTypePaid,
				amount:           1000,
				balanceBefore:    0,
				balanceAfter:     1000,
				status:           TransactionStatusCompleted,
				paymentRequestID: nil,
				metadata:         metadata,
			},
			wantErr: nil,
		},
		{
			name:            "異常系: 無効なTransactionID",
			transactionID:   "tx/123",
			userID:          "user123",
			transactionType: TransactionTypeGrant,
			currencyType:    currency.CurrencyTypePaid,
			amount:          1000,
			balanceBefore:   0,
			balanceAfter:    1000,
			status:          TransactionStatusCompleted,
			metadata:        metadata,
			want:            nil,
			wantErr:         ErrInvalidTransactionID,
		},
		{
			name:            "異常系: 無効なUserID",
			transactionID:   "tx123",
			userID:          "user/123",
			transactionType: TransactionTypeGrant,
			currencyType:    currency.CurrencyTypePaid,
			amount:          1000,
			balanceBefore:   0,
			balanceAfter:    1000,
			status:          TransactionStatusCompleted,
			metadata:        metadata,
			want:            nil,
			wantErr:         ErrInvalidUserID,
		},
		{
			name:            "異常系: 無効な金額 (0)",
			transactionID:   "tx123",
			userID:          "user123",
			transactionType: TransactionTypeGrant,
			currencyType:    currency.CurrencyTypePaid,
			amount:          0,
			balanceBefore:   0,
			balanceAfter:    0,
			status:          TransactionStatusCompleted,
			metadata:        metadata,
			want:            nil,
			wantErr:         ErrInvalidAmount,
		},
		{
			name:            "異常系: 残高範囲外 (過大)",
			transactionID:   "tx123",
			userID:          "user123",
			transactionType: TransactionTypeGrant,
			currencyType:    currency.CurrencyTypePaid,
			amount:          1000,
			balanceBefore:   MaxAmount + 1,
			balanceAfter:    MaxAmount + 1001, // 単純化のためここも大きく
			status:          TransactionStatusCompleted,
			metadata:        metadata,
			want:            nil,
			wantErr:         ErrBalanceOutOfRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTransaction(
				tt.transactionID,
				tt.userID,
				tt.transactionType,
				tt.currencyType,
				tt.amount,
				tt.balanceBefore,
				tt.balanceAfter,
				tt.status,
				tt.metadata,
			)
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want.TransactionID(), got.TransactionID())
				assert.Equal(t, tt.want.UserID(), got.UserID())
				assert.Equal(t, tt.want.TransactionType(), got.TransactionType())
				assert.Equal(t, tt.want.CurrencyType(), got.CurrencyType())
				assert.Equal(t, tt.want.Amount(), got.Amount())
				assert.Equal(t, tt.want.BalanceBefore(), got.BalanceBefore())
				assert.Equal(t, tt.want.BalanceAfter(), got.BalanceAfter())
				assert.Equal(t, tt.want.Status(), got.Status())
				assert.Nil(t, got.PaymentRequestID())
			}
		})
	}
}

func TestTransaction_SetPaymentRequestID(t *testing.T) {
	tx := mustNewTransaction(
		"tx123",
		"user123",
		TransactionTypeConsume,
		currency.CurrencyTypePaid,
		1000,
		5000,
		4000,
		TransactionStatusCompleted,
		nil,
	)

	assert.Nil(t, tx.PaymentRequestID())

	paymentRequestID := "pr123"
	tx.SetPaymentRequestID(paymentRequestID)

	assert.NotNil(t, tx.PaymentRequestID())
	assert.Equal(t, paymentRequestID, *tx.PaymentRequestID())
}

func TestTransaction_UpdateStatus(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus TransactionStatus
		newStatus     TransactionStatus
		wantError     bool
	}{
		{
			name:          "正常系: pending -> completed",
			initialStatus: TransactionStatusPending,
			newStatus:     TransactionStatusCompleted,
			wantError:     false,
		},
		{
			name:          "正常系: pending -> failed",
			initialStatus: TransactionStatusPending,
			newStatus:     TransactionStatusFailed,
			wantError:     false,
		},
		{
			name:          "異常系: 無効なステータス",
			initialStatus: TransactionStatusPending,
			newStatus:     TransactionStatus("invalid"),
			wantError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := mustNewTransaction(
				"tx123",
				"user123",
				TransactionTypeConsume,
				currency.CurrencyTypePaid,
				1000,
				5000,
				4000,
				tt.initialStatus,
				nil,
			)

			err := tx.UpdateStatus(tt.newStatus)
			if tt.wantError {
				assert.Error(t, err)
				assert.Equal(t, tt.initialStatus, tx.Status())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.newStatus, tx.Status())
			}
		})
	}
}

func TestTransaction_GetterMethods(t *testing.T) {
	metadata := map[string]interface{}{
		"key": "value",
	}
	tx := mustNewTransaction(
		"tx123",
		"user123",
		TransactionTypeGrant,
		currency.CurrencyTypePaid,
		1000,
		0,
		1000,
		TransactionStatusCompleted,
		metadata,
	)

	assert.Equal(t, "tx123", tx.TransactionID())
	assert.Equal(t, "user123", tx.UserID())
	assert.Equal(t, TransactionTypeGrant, tx.TransactionType())
	assert.Equal(t, currency.CurrencyTypePaid, tx.CurrencyType())
	assert.Equal(t, int64(1000), tx.Amount())
	assert.Equal(t, int64(0), tx.BalanceBefore())
	assert.Equal(t, int64(1000), tx.BalanceAfter())
	assert.Equal(t, TransactionStatusCompleted, tx.Status())
	assert.Equal(t, metadata, tx.Metadata())
	assert.WithinDuration(t, time.Now(), tx.CreatedAt(), time.Second)
	assert.WithinDuration(t, time.Now(), tx.UpdatedAt(), time.Second)
}

func TestNewTransactionWithRequester(t *testing.T) {
	metadata := map[string]interface{}{
		"reason": "test",
	}
	requester := "game-server-01"

	tx := mustNewTransactionWithRequester(
		"tx123",
		"user123",
		TransactionTypeGrant,
		currency.CurrencyTypePaid,
		1000,
		0,
		1000,
		TransactionStatusCompleted,
		&requester,
		metadata,
	)

	assert.Equal(t, "tx123", tx.TransactionID())
	assert.Equal(t, "user123", tx.UserID())
	assert.NotNil(t, tx.Requester())
	assert.Equal(t, requester, *tx.Requester())
}

func TestTransaction_SetRequester(t *testing.T) {
	tx := mustNewTransaction(
		"tx123",
		"user123",
		TransactionTypeConsume,
		currency.CurrencyTypePaid,
		1000,
		5000,
		4000,
		TransactionStatusCompleted,
		nil,
	)

	assert.Nil(t, tx.Requester())

	requester := "game-server-01"
	tx.SetRequester(requester)

	assert.NotNil(t, tx.Requester())
	assert.Equal(t, requester, *tx.Requester())
}
