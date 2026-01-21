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
	"gem-server/internal/domain/transaction"
)

func TestTransactionRepository_Save(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &TransactionRepository{
		db:     &DB{DB: db},
		tracer: otel.Tracer("test"),
	}

	tests := []struct {
		name        string
		transaction *transaction.Transaction
		setupMock   func()
		wantError   bool
	}{
		{
			name: "正常系: トランザクションを保存",
			transaction: mustNewTransaction(
				"txn123",
				"user123",
				transaction.TransactionTypeGrant,
				currency.CurrencyTypePaid,
				1000,
				0,
				1000,
				transaction.TransactionStatusCompleted,
				map[string]interface{}{"test": "data"},
			),
			setupMock: func() {
				mock.ExpectExec(`INSERT INTO transactions`).
					WithArgs(
						"txn123",
						"user123",
						"grant",
						"paid",
						int64(1000),
						int64(0),
						int64(1000),
						"completed",
						sqlmock.AnyArg(), // payment_request_id
						sqlmock.AnyArg(), // requester
						sqlmock.AnyArg(), // metadata
						sqlmock.AnyArg(), // created_at
						sqlmock.AnyArg(), // updated_at
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantError: false,
		},
		{
			name: "正常系: PaymentRequestIDありで保存",
			transaction: func() *transaction.Transaction {
				txn := mustNewTransaction(
					"txn123",
					"user123",
					transaction.TransactionTypeConsume,
					currency.CurrencyTypePaid,
					500,
					1000,
					500,
					transaction.TransactionStatusCompleted,
					nil,
				)
				prID := "pr123"
				txn.SetPaymentRequestID(prID)
				return txn
			}(),
			setupMock: func() {
				mock.ExpectExec(`INSERT INTO transactions`).
					WithArgs(
						"txn123",
						"user123",
						"consume",
						"paid",
						int64(500),
						int64(1000),
						int64(500),
						"completed",
						"pr123",
						sqlmock.AnyArg(), // requester
						sqlmock.AnyArg(), // metadata
						sqlmock.AnyArg(), // created_at
						sqlmock.AnyArg(), // updated_at
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantError: false,
		},
		{
			name: "異常系: DBエラー",
			transaction: mustNewTransaction(
				"txn123",
				"user123",
				transaction.TransactionTypeGrant,
				currency.CurrencyTypePaid,
				1000,
				0,
				1000,
				transaction.TransactionStatusCompleted,
				nil,
			),
			setupMock: func() {
				mock.ExpectExec(`INSERT INTO transactions`).
					WillReturnError(sql.ErrConnDone)
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			ctx := context.Background()
			err := repo.Save(ctx, tt.transaction)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTransactionRepository_FindByTransactionID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &TransactionRepository{
		db:     &DB{DB: db},
		tracer: otel.Tracer("test"),
	}

	tests := []struct {
		name          string
		transactionID string
		setupMock     func()
		want          *transaction.Transaction
		wantError     bool
		errorType     error
	}{
		{
			name:          "正常系: トランザクションが見つかる",
			transactionID: "txn123",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{
					"transaction_id", "user_id", "transaction_type", "currency_type",
					"amount", "balance_before", "balance_after", "status",
					"payment_request_id", "requester", "metadata", "created_at", "updated_at",
				}).
					AddRow("txn123", "user123", "grant", "paid", 1000, 0, 1000, "completed", nil, nil, nil, time.Now(), time.Now())
				mock.ExpectQuery(`SELECT`).
					WithArgs("txn123").
					WillReturnRows(rows)
			},
			wantError: false,
		},
		{
			name:          "異常系: トランザクションが見つからない",
			transactionID: "txn123",
			setupMock: func() {
				mock.ExpectQuery(`SELECT`).
					WithArgs("txn123").
					WillReturnError(sql.ErrNoRows)
			},
			want:      nil,
			wantError: true,
			errorType: transaction.ErrTransactionNotFound,
		},
		{
			name:          "異常系: DBエラー",
			transactionID: "txn123",
			setupMock: func() {
				mock.ExpectQuery(`SELECT`).
					WithArgs("txn123").
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
			got, err := repo.FindByTransactionID(ctx, tt.transactionID)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.Equal(t, tt.errorType, err)
				}
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tt.transactionID, got.TransactionID())
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTransactionRepository_FindByUserID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &TransactionRepository{
		db:     &DB{DB: db},
		tracer: otel.Tracer("test"),
	}

	tests := []struct {
		name      string
		userID    string
		limit     int
		offset    int
		setupMock func()
		wantCount int
		wantError bool
	}{
		{
			name:   "正常系: トランザクション一覧を取得",
			userID: "user123",
			limit:  10,
			offset: 0,
			setupMock: func() {
				rows := sqlmock.NewRows([]string{
					"transaction_id", "user_id", "transaction_type", "currency_type",
					"amount", "balance_before", "balance_after", "status",
					"payment_request_id", "requester", "metadata", "created_at", "updated_at",
				}).
					AddRow("txn1", "user123", "grant", "paid", 1000, 0, 1000, "completed", nil, nil, nil, time.Now(), time.Now()).
					AddRow("txn2", "user123", "consume", "paid", 500, 1000, 500, "completed", nil, nil, nil, time.Now(), time.Now())
				mock.ExpectQuery(`SELECT`).
					WithArgs("user123", 10, 0).
					WillReturnRows(rows)
			},
			wantCount: 2,
			wantError: false,
		},
		{
			name:   "正常系: 空の結果",
			userID: "user123",
			limit:  10,
			offset: 0,
			setupMock: func() {
				rows := sqlmock.NewRows([]string{
					"transaction_id", "user_id", "transaction_type", "currency_type",
					"amount", "balance_before", "balance_after", "status",
					"payment_request_id", "requester", "metadata", "created_at", "updated_at",
				})
				mock.ExpectQuery(`SELECT`).
					WithArgs("user123", 10, 0).
					WillReturnRows(rows)
			},
			wantCount: 0,
			wantError: false,
		},
		{
			name:   "異常系: DBエラー",
			userID: "user123",
			limit:  10,
			offset: 0,
			setupMock: func() {
				mock.ExpectQuery(`SELECT`).
					WithArgs("user123", 10, 0).
					WillReturnError(sql.ErrConnDone)
			},
			wantCount: 0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			ctx := context.Background()
			got, err := repo.FindByUserID(ctx, tt.userID, tt.limit, tt.offset)

			if tt.wantError {
				assert.Error(t, err)
				if got != nil {
					assert.Len(t, got, 0)
				}
			} else {
				require.NoError(t, err)
				if tt.wantCount == 0 {
					// 空のスライスが返される（nilではない）
					if got != nil {
						assert.Len(t, got, 0)
					} else {
						// nilの場合は空のスライスとして扱う
						got = []*transaction.Transaction{}
						assert.Len(t, got, 0)
					}
				} else {
					assert.NotNil(t, got)
					assert.Len(t, got, tt.wantCount)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTransactionRepository_FindByPaymentRequestID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &TransactionRepository{
		db:     &DB{DB: db},
		tracer: otel.Tracer("test"),
	}

	tests := []struct {
		name             string
		paymentRequestID string
		setupMock        func()
		want             *transaction.Transaction
		wantError        bool
		errorType        error
	}{
		{
			name:             "正常系: PaymentRequestIDでトランザクションが見つかる",
			paymentRequestID: "pr123",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{
					"transaction_id", "user_id", "transaction_type", "currency_type",
					"amount", "balance_before", "balance_after", "status",
					"payment_request_id", "requester", "metadata", "created_at", "updated_at",
				}).
					AddRow("txn123", "user123", "consume", "paid", 500, 1000, 500, "completed", "pr123", nil, nil, time.Now(), time.Now())
				mock.ExpectQuery(`SELECT`).
					WithArgs("pr123").
					WillReturnRows(rows)
			},
			wantError: false,
		},
		{
			name:             "異常系: トランザクションが見つからない",
			paymentRequestID: "pr123",
			setupMock: func() {
				mock.ExpectQuery(`SELECT`).
					WithArgs("pr123").
					WillReturnError(sql.ErrNoRows)
			},
			want:      nil,
			wantError: true,
			errorType: transaction.ErrTransactionNotFound,
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
				if got != nil {
					prID := got.PaymentRequestID()
					assert.NotNil(t, prID)
					if prID != nil {
						assert.Equal(t, tt.paymentRequestID, *prID)
					}
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func mustNewTransaction(transactionID, userID string, transactionType transaction.TransactionType, currencyType currency.CurrencyType, amount, balanceBefore, balanceAfter int64, status transaction.TransactionStatus, metadata map[string]interface{}) *transaction.Transaction {
	tx, err := transaction.NewTransaction(transactionID, userID, transactionType, currencyType, amount, balanceBefore, balanceAfter, status, metadata)
	if err != nil {
		panic(err)
	}
	return tx
}
