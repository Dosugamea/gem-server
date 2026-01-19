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
	"gem-server/internal/domain/redemption_code"
)

func TestRedemptionCodeRepository_FindByCode(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &RedemptionCodeRepository{
		db:     &DB{DB: db},
		tracer: otel.Tracer("test"),
	}

	tests := []struct {
		name      string
		code      string
		setupMock func()
		want      *redemption_code.RedemptionCode
		wantError bool
		errorType error
	}{
		{
			name: "正常系: コードが見つかる",
			code: "TESTCODE123",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{
					"code", "code_type", "currency_type", "amount",
					"max_uses", "current_uses", "valid_from", "valid_until",
					"status", "metadata", "created_at", "updated_at",
				}).
					AddRow("TESTCODE123", "promotion", "paid", 1000, 1, 0, time.Now().Add(-24*time.Hour), time.Now().Add(24*time.Hour), "active", nil, time.Now(), time.Now())
				mock.ExpectQuery(`SELECT`).
					WithArgs("TESTCODE123").
					WillReturnRows(rows)
			},
			wantError: false,
		},
		{
			name: "異常系: コードが見つからない",
			code: "INVALID",
			setupMock: func() {
				mock.ExpectQuery(`SELECT`).
					WithArgs("INVALID").
					WillReturnError(sql.ErrNoRows)
			},
			want:      nil,
			wantError: true,
			errorType: redemption_code.ErrCodeNotFound,
		},
		{
			name: "異常系: DBエラー",
			code: "TESTCODE123",
			setupMock: func() {
				mock.ExpectQuery(`SELECT`).
					WithArgs("TESTCODE123").
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
			got, err := repo.FindByCode(ctx, tt.code)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.Equal(t, tt.errorType, err)
				}
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tt.code, got.Code())
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRedemptionCodeRepository_Update(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &RedemptionCodeRepository{
		db:     &DB{DB: db},
		tracer: otel.Tracer("test"),
	}

	tests := []struct {
		name      string
		code      *redemption_code.RedemptionCode
		setupMock func()
		wantError bool
	}{
		{
			name: "正常系: コードを更新",
			code: func() *redemption_code.RedemptionCode {
				codeType, _ := redemption_code.NewCodeType("promotion")
				status, _ := redemption_code.NewCodeStatus("active")
				rc := redemption_code.NewRedemptionCode(
					"TESTCODE123",
					codeType,
					currency.CurrencyTypePaid,
					1000,
					1,
					time.Now().Add(-24*time.Hour),
					time.Now().Add(24*time.Hour),
					map[string]interface{}{},
				)
				rc.SetCurrentUses(1)
				rc.SetStatus(status)
				return rc
			}(),
			setupMock: func() {
				mock.ExpectExec(`UPDATE redemption_codes`).
					WithArgs(1, "active", "TESTCODE123").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantError: false,
		},
		{
			name: "異常系: DBエラー",
			code: func() *redemption_code.RedemptionCode {
				codeType, _ := redemption_code.NewCodeType("promotion")
				status, _ := redemption_code.NewCodeStatus("active")
				rc := redemption_code.NewRedemptionCode(
					"TESTCODE123",
					codeType,
					currency.CurrencyTypePaid,
					1000,
					1,
					time.Now().Add(-24*time.Hour),
					time.Now().Add(24*time.Hour),
					map[string]interface{}{},
				)
				rc.SetCurrentUses(1)
				rc.SetStatus(status)
				return rc
			}(),
			setupMock: func() {
				mock.ExpectExec(`UPDATE redemption_codes`).
					WillReturnError(sql.ErrConnDone)
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			ctx := context.Background()
			err := repo.Update(ctx, tt.code)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRedemptionCodeRepository_HasUserRedeemed(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &RedemptionCodeRepository{
		db:     &DB{DB: db},
		tracer: otel.Tracer("test"),
	}

	tests := []struct {
		name      string
		code      string
		userID    string
		setupMock func()
		want      bool
		wantError bool
	}{
		{
			name:   "正常系: ユーザーが引き換え済み",
			code:   "TESTCODE123",
			userID: "user123",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1)
				mock.ExpectQuery(`SELECT COUNT`).
					WithArgs("TESTCODE123", "user123").
					WillReturnRows(rows)
			},
			want:      true,
			wantError: false,
		},
		{
			name:   "正常系: ユーザーが未引き換え",
			code:   "TESTCODE123",
			userID: "user123",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0)
				mock.ExpectQuery(`SELECT COUNT`).
					WithArgs("TESTCODE123", "user123").
					WillReturnRows(rows)
			},
			want:      false,
			wantError: false,
		},
		{
			name:   "異常系: DBエラー",
			code:   "TESTCODE123",
			userID: "user123",
			setupMock: func() {
				mock.ExpectQuery(`SELECT COUNT`).
					WithArgs("TESTCODE123", "user123").
					WillReturnError(sql.ErrConnDone)
			},
			want:      false,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			ctx := context.Background()
			got, err := repo.HasUserRedeemed(ctx, tt.code, tt.userID)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRedemptionCodeRepository_SaveRedemption(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &RedemptionCodeRepository{
		db:     &DB{DB: db},
		tracer: otel.Tracer("test"),
	}

	redemption := redemption_code.NewCodeRedemption("red123", "TESTCODE123", "user123", "txn123")

	mock.ExpectExec(`INSERT INTO code_redemptions`).
		WithArgs("red123", "TESTCODE123", "user123", "txn123", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := context.Background()
	err = repo.SaveRedemption(ctx, redemption)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
