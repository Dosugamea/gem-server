package mysql

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"

	"gem-server/internal/domain/currency"
)

func TestCurrencyRepository_FindByUserIDAndType(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &CurrencyRepository{
		db:     &DB{DB: db},
		tracer: otel.Tracer("test"),
	}

	tests := []struct {
		name         string
		userID       string
		currencyType currency.CurrencyType
		setupMock    func()
		want         *currency.Currency
		wantError    bool
		errorType    error
	}{
		{
			name:         "正常系: 通貨が見つかる",
			userID:       "user123",
			currencyType: currency.CurrencyTypePaid,
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"user_id", "currency_type", "balance", "version"}).
					AddRow("user123", "paid", 1000, 1)
				mock.ExpectQuery(`SELECT user_id, currency_type, balance, version`).
					WithArgs("user123", "paid").
					WillReturnRows(rows)
			},
			want:      currency.MustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1),
			wantError: false,
		},
		{
			name:         "異常系: 通貨が見つからない",
			userID:       "user123",
			currencyType: currency.CurrencyTypePaid,
			setupMock: func() {
				mock.ExpectQuery(`SELECT user_id, currency_type, balance, version`).
					WithArgs("user123", "paid").
					WillReturnError(sql.ErrNoRows)
			},
			want:      nil,
			wantError: true,
			errorType: currency.ErrCurrencyNotFound,
		},
		{
			name:         "異常系: DBエラー",
			userID:       "user123",
			currencyType: currency.CurrencyTypePaid,
			setupMock: func() {
				mock.ExpectQuery(`SELECT user_id, currency_type, balance, version`).
					WithArgs("user123", "paid").
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
			got, err := repo.FindByUserIDAndType(ctx, tt.userID, tt.currencyType)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.Equal(t, tt.errorType, err)
				}
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tt.want.UserID(), got.UserID())
				assert.Equal(t, tt.want.CurrencyType(), got.CurrencyType())
				assert.Equal(t, tt.want.Balance(), got.Balance())
				assert.Equal(t, tt.want.Version(), got.Version())
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCurrencyRepository_Save(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &CurrencyRepository{
		db:     &DB{DB: db},
		tracer: otel.Tracer("test"),
	}

	tests := []struct {
		name      string
		currency  *currency.Currency
		setupMock func()
		wantError bool
	}{
		{
			name:     "正常系: 通貨を保存",
			currency: currency.MustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1),
			setupMock: func() {
				mock.ExpectExec(`UPDATE currency_balances`).
					WithArgs(int64(1000), "user123", "paid", 1).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantError: false,
		},
		{
			name:     "異常系: 楽観的ロック失敗（行が更新されない）",
			currency: currency.MustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1),
			setupMock: func() {
				mock.ExpectExec(`UPDATE currency_balances`).
					WithArgs(int64(1000), "user123", "paid", 1).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantError: true,
		},
		{
			name:     "異常系: DBエラー",
			currency: currency.MustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1),
			setupMock: func() {
				mock.ExpectExec(`UPDATE currency_balances`).
					WithArgs(int64(1000), "user123", "paid", 1).
					WillReturnError(sql.ErrConnDone)
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			ctx := context.Background()
			err := repo.Save(ctx, tt.currency)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCurrencyRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &CurrencyRepository{
		db:     &DB{DB: db},
		tracer: otel.Tracer("test"),
	}

	tests := []struct {
		name      string
		currency  *currency.Currency
		setupMock func()
		wantError bool
	}{
		{
			name:     "正常系: 新規通貨を作成",
			currency: currency.MustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 0),
			setupMock: func() {
				// ensureUserExistsのモック
				mock.ExpectExec(`INSERT INTO users`).
					WithArgs("user123").
					WillReturnResult(sqlmock.NewResult(1, 1))
				// Createのモック
				mock.ExpectExec(`INSERT INTO currency_balances`).
					WithArgs("user123", "paid", int64(1000), 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantError: false,
		},
		{
			name:     "異常系: ユーザー作成エラー",
			currency: currency.MustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 0),
			setupMock: func() {
				mock.ExpectExec(`INSERT INTO users`).
					WithArgs("user123").
					WillReturnError(sql.ErrConnDone)
			},
			wantError: true,
		},
		{
			name:     "異常系: 通貨作成エラー",
			currency: currency.MustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 0),
			setupMock: func() {
				mock.ExpectExec(`INSERT INTO users`).
					WithArgs("user123").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(`INSERT INTO currency_balances`).
					WithArgs("user123", "paid", int64(1000), 0).
					WillReturnError(sql.ErrConnDone)
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			ctx := context.Background()
			err := repo.Create(ctx, tt.currency)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}


