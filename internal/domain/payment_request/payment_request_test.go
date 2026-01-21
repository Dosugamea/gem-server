package payment_request

import (
	"gem-server/internal/domain/currency"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPaymentRequest(t *testing.T) {
	pr, err := NewPaymentRequest(
		"pr123",
		"user123",
		1000,
		"JPY",
		currency.CurrencyTypePaid,
	)
	require.NoError(t, err)

	assert.Equal(t, "pr123", pr.PaymentRequestID())
	assert.Equal(t, "user123", pr.UserID())
	assert.Equal(t, int64(1000), pr.Amount())
	assert.Equal(t, "JPY", pr.Currency())
	assert.Equal(t, currency.CurrencyTypePaid, pr.CurrencyType())
	assert.Equal(t, PaymentRequestStatusPending, pr.Status())
	assert.NotNil(t, pr.PaymentMethodData())
	assert.NotNil(t, pr.Details())
	assert.NotNil(t, pr.Response())
	assert.WithinDuration(t, time.Now(), pr.CreatedAt(), time.Second)
	assert.WithinDuration(t, time.Now(), pr.UpdatedAt(), time.Second)
}

func TestPaymentRequest_SetPaymentMethodData(t *testing.T) {
	pr := MustNewPaymentRequest(
		"pr123",
		"user123",
		1000,
		"JPY",
		currency.CurrencyTypePaid,
	)

	data := map[string]interface{}{
		"method": "test",
	}
	pr.SetPaymentMethodData(data)

	assert.Equal(t, data, pr.PaymentMethodData())
}

func TestPaymentRequest_SetDetails(t *testing.T) {
	pr := MustNewPaymentRequest(
		"pr123",
		"user123",
		1000,
		"JPY",
		currency.CurrencyTypePaid,
	)

	details := map[string]interface{}{
		"item": "test",
	}
	pr.SetDetails(details)

	assert.Equal(t, details, pr.Details())
}

func TestPaymentRequest_SetResponse(t *testing.T) {
	pr := MustNewPaymentRequest(
		"pr123",
		"user123",
		1000,
		"JPY",
		currency.CurrencyTypePaid,
	)

	response := map[string]interface{}{
		"result": "success",
	}
	pr.SetResponse(response)

	assert.Equal(t, response, pr.Response())
}

func TestPaymentRequest_Complete(t *testing.T) {
	pr := MustNewPaymentRequest(
		"pr123",
		"user123",
		1000,
		"JPY",
		currency.CurrencyTypePaid,
	)

	assert.Equal(t, PaymentRequestStatusPending, pr.Status())
	pr.Complete()
	assert.Equal(t, PaymentRequestStatusCompleted, pr.Status())
	assert.True(t, pr.IsCompleted())
}

func TestPaymentRequest_Fail(t *testing.T) {
	pr := MustNewPaymentRequest(
		"pr123",
		"user123",
		1000,
		"JPY",
		currency.CurrencyTypePaid,
	)

	assert.Equal(t, PaymentRequestStatusPending, pr.Status())
	pr.Fail()
	assert.Equal(t, PaymentRequestStatusFailed, pr.Status())
	assert.False(t, pr.IsCompleted())
}

func TestPaymentRequest_Cancel(t *testing.T) {
	pr := MustNewPaymentRequest(
		"pr123",
		"user123",
		1000,
		"JPY",
		currency.CurrencyTypePaid,
	)

	assert.Equal(t, PaymentRequestStatusPending, pr.Status())
	pr.Cancel()
	assert.Equal(t, PaymentRequestStatusCancelled, pr.Status())
	assert.False(t, pr.IsCompleted())
}

func TestPaymentRequest_IsPending(t *testing.T) {
	pr := MustNewPaymentRequest(
		"pr123",
		"user123",
		1000,
		"JPY",
		currency.CurrencyTypePaid,
	)

	assert.True(t, pr.IsPending())
	pr.Complete()
	assert.False(t, pr.IsPending())
}

func TestPaymentRequestStatus_String(t *testing.T) {
	tests := []struct {
		name string
		prs  PaymentRequestStatus
		want string
	}{
		{
			name: "正常系: pending",
			prs:  PaymentRequestStatusPending,
			want: "pending",
		},
		{
			name: "正常系: completed",
			prs:  PaymentRequestStatusCompleted,
			want: "completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.prs.String()
			assert.Equal(t, tt.want, got)
		})
	}
}
