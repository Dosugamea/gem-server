package mysql

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"

	"gem-server/internal/domain/currency"
	"gem-server/internal/domain/payment_request"
)

func TestPaymentRequestRepository_Save(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &PaymentRequestRepository{
		db:     &DB{DB: db},
		tracer: otel.Tracer("test"),
	}

	tests := []struct {
		name           string
		paymentRequest *payment_request.PaymentRequest
		setupMock      func()
		wantError      bool
	}{
		{
			name: "正常系: PaymentRequestを保存",
			paymentRequest: func() *payment_request.PaymentRequest {
				pr := payment_request.NewPaymentRequest("pr123", "user123", 1000, "JPY", currency.CurrencyTypePaid)
				pr.SetPaymentMethodData(map[string]interface{}{"methodName": "test"})
				pr.SetDetails(map[string]interface{}{"key": "value"})
				return pr
			}(),
			setupMock: func() {
				mock.ExpectExec(`INSERT INTO payment_requests`).
					WithArgs(
						"pr123",
						"user123",
						int64(1000),
						"JPY",
						"paid",
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantError: false,
		},
		{
			name: "異常系: DBエラー",
			paymentRequest: payment_request.NewPaymentRequest("pr123", "user123", 1000, "JPY", currency.CurrencyTypePaid),
			setupMock: func() {
				mock.ExpectExec(`INSERT INTO payment_requests`).
					WillReturnError(sql.ErrConnDone)
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			ctx := context.Background()
			err := repo.Save(ctx, tt.paymentRequest)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPaymentRequestRepository_FindByPaymentRequestID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &PaymentRequestRepository{
		db:     &DB{DB: db},
		tracer: otel.Tracer("test"),
	}

	tests := []struct {
		name            string
		paymentRequestID string
		setupMock       func()
		want            *payment_request.PaymentRequest
		wantError       bool
		errorType       error
	}{
		{
			name:            "正常系: PaymentRequestが見つかる（pending）",
			paymentRequestID: "pr123",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{
					"payment_request_id", "user_id", "amount", "currency", "currency_type",
					"status", "payment_method_data", "details", "response",
					"created_at", "updated_at",
				}).
					AddRow("pr123", "user123", 1000, "JPY", "paid", "pending", nil, nil, nil, time.Now(), time.Now())
				mock.ExpectQuery(`SELECT`).
					WithArgs("pr123").
					WillReturnRows(rows)
			},
			wantError: false,
		},
		{
			name:            "正常系: PaymentRequestが見つかる（completed）",
			paymentRequestID: "pr123",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{
					"payment_request_id", "user_id", "amount", "currency", "currency_type",
					"status", "payment_method_data", "details", "response",
					"created_at", "updated_at",
				}).
					AddRow("pr123", "user123", 1000, "JPY", "paid", "completed", nil, nil, nil, time.Now(), time.Now())
				mock.ExpectQuery(`SELECT`).
					WithArgs("pr123").
					WillReturnRows(rows)
			},
			wantError: false,
		},
		{
			name:            "異常系: PaymentRequestが見つからない",
			paymentRequestID: "pr123",
			setupMock: func() {
				mock.ExpectQuery(`SELECT`).
					WithArgs("pr123").
					WillReturnError(sql.ErrNoRows)
			},
			want:      nil,
			wantError: true,
			errorType: payment_request.ErrPaymentRequestNotFound,
		},
		{
			name:            "異常系: DBエラー",
			paymentRequestID: "pr123",
			setupMock: func() {
				mock.ExpectQuery(`SELECT`).
					WithArgs("pr123").
					WillReturnError(sql.ErrConnDone)
			},
			want:      nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			ctx := context.Background()
			got, err := repo.FindByPaymentRequestID(ctx, tt.paymentRequestID)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.Equal(t, tt.errorType, err)
				}
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tt.paymentRequestID, got.PaymentRequestID())
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPaymentRequestRepository_Update(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &PaymentRequestRepository{
		db:     &DB{DB: db},
		tracer: otel.Tracer("test"),
	}

	pr := payment_request.NewPaymentRequest("pr123", "user123", 1000, "JPY", currency.CurrencyTypePaid)
	pr.Complete()

	mock.ExpectExec(`INSERT INTO payment_requests`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := context.Background()
	err = repo.Update(ctx, pr)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
