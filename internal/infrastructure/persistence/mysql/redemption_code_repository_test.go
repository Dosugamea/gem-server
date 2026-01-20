package mysql

import (
	"context"
	"database/sql"
	"errors"
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

func TestRedemptionCodeRepository_Create(t *testing.T) {
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
		errorType error
	}{
		{
			name: "正常系: コードを作成",
			code: func() *redemption_code.RedemptionCode {
				codeType, _ := redemption_code.NewCodeType("promotion")
				rc := redemption_code.NewRedemptionCode(
					"NEWCODE123",
					codeType,
					currency.CurrencyTypePaid,
					1000,
					100,
					time.Now().Add(-24*time.Hour),
					time.Now().Add(24*time.Hour),
					map[string]interface{}{"campaign_id": "campaign_001"},
				)
				return rc
			}(),
			setupMock: func() {
				mock.ExpectExec(`INSERT INTO redemption_codes`).
					WithArgs(
						"NEWCODE123",
						"promotion",
						"paid",
						int64(1000),
						100,
						0,
						sqlmock.AnyArg(), // valid_from
						sqlmock.AnyArg(), // valid_until
						"active",
						sqlmock.AnyArg(), // metadata JSON
						sqlmock.AnyArg(), // created_at
						sqlmock.AnyArg(), // updated_at
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantError: false,
		},
		{
			name: "正常系: メタデータなしでコードを作成",
			code: func() *redemption_code.RedemptionCode {
				codeType, _ := redemption_code.NewCodeType("gift")
				rc := redemption_code.NewRedemptionCode(
					"GIFTCODE456",
					codeType,
					currency.CurrencyTypeFree,
					500,
					0, // 無制限
					time.Now(),
					time.Now().Add(7*24*time.Hour),
					nil,
				)
				return rc
			}(),
			setupMock: func() {
				mock.ExpectExec(`INSERT INTO redemption_codes`).
					WithArgs(
						"GIFTCODE456",
						"gift",
						"free",
						int64(500),
						0,
						0,
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						"active",
						sql.NullString{Valid: false},
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantError: false,
		},
		{
			name: "異常系: コードが既に存在",
			code: func() *redemption_code.RedemptionCode {
				codeType, _ := redemption_code.NewCodeType("promotion")
				rc := redemption_code.NewRedemptionCode(
					"DUPLICATECODE",
					codeType,
					currency.CurrencyTypePaid,
					1000,
					1,
					time.Now(),
					time.Now().Add(24*time.Hour),
					map[string]interface{}{},
				)
				return rc
			}(),
			setupMock: func() {
				mock.ExpectExec(`INSERT INTO redemption_codes`).
					WillReturnError(errors.New("Duplicate entry 'DUPLICATECODE' for key 'code'"))
			},
			wantError: true,
			errorType: redemption_code.ErrCodeAlreadyExists,
		},
		{
			name: "異常系: DBエラー",
			code: func() *redemption_code.RedemptionCode {
				codeType, _ := redemption_code.NewCodeType("promotion")
				rc := redemption_code.NewRedemptionCode(
					"ERRORCODE",
					codeType,
					currency.CurrencyTypePaid,
					1000,
					1,
					time.Now(),
					time.Now().Add(24*time.Hour),
					map[string]interface{}{},
				)
				return rc
			}(),
			setupMock: func() {
				mock.ExpectExec(`INSERT INTO redemption_codes`).
					WillReturnError(sql.ErrConnDone)
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			ctx := context.Background()
			err := repo.Create(ctx, tt.code)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.Equal(t, tt.errorType, err)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRedemptionCodeRepository_Delete(t *testing.T) {
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
		wantError bool
		errorType error
	}{
		{
			name: "正常系: コードを削除（未使用）",
			code: "UNUSEDCODE",
			setupMock: func() {
				// 使用状況チェック（未使用）
				rows := sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0)
				mock.ExpectQuery(`SELECT COUNT`).
					WithArgs("UNUSEDCODE").
					WillReturnRows(rows)
				// 削除実行
				mock.ExpectExec(`DELETE FROM redemption_codes`).
					WithArgs("UNUSEDCODE").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantError: false,
		},
		{
			name: "異常系: コードが見つからない",
			code: "NOTFOUNDCODE",
			setupMock: func() {
				// 使用状況チェック
				rows := sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0)
				mock.ExpectQuery(`SELECT COUNT`).
					WithArgs("NOTFOUNDCODE").
					WillReturnRows(rows)
				// 削除実行（0件）
				mock.ExpectExec(`DELETE FROM redemption_codes`).
					WithArgs("NOTFOUNDCODE").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantError: true,
			errorType: redemption_code.ErrCodeNotFound,
		},
		{
			name: "異常系: コードが使用済み（削除不可）",
			code: "USEDCODE",
			setupMock: func() {
				// 使用状況チェック（使用済み）
				rows := sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1)
				mock.ExpectQuery(`SELECT COUNT`).
					WithArgs("USEDCODE").
					WillReturnRows(rows)
				// 削除は実行されない
			},
			wantError: true,
			errorType: redemption_code.ErrCodeCannotBeDeleted,
		},
		{
			name: "異常系: 使用状況チェックでDBエラー",
			code: "ERRORCODE",
			setupMock: func() {
				mock.ExpectQuery(`SELECT COUNT`).
					WithArgs("ERRORCODE").
					WillReturnError(sql.ErrConnDone)
			},
			wantError: true,
		},
		{
			name: "異常系: 削除実行でDBエラー",
			code: "DELETEERRORCODE",
			setupMock: func() {
				// 使用状況チェック（未使用）
				rows := sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0)
				mock.ExpectQuery(`SELECT COUNT`).
					WithArgs("DELETEERRORCODE").
					WillReturnRows(rows)
				// 削除実行でエラー
				mock.ExpectExec(`DELETE FROM redemption_codes`).
					WithArgs("DELETEERRORCODE").
					WillReturnError(sql.ErrConnDone)
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			ctx := context.Background()
			err := repo.Delete(ctx, tt.code)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.Equal(t, tt.errorType, err)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRedemptionCodeRepository_FindAll(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &RedemptionCodeRepository{
		db:     &DB{DB: db},
		tracer: otel.Tracer("test"),
	}

	tests := []struct {
		name      string
		limit     int
		offset    int
		setupMock func()
		wantCount int
		wantTotal int
		wantError bool
	}{
		{
			name:   "正常系: コード一覧を取得",
			limit:  10,
			offset: 0,
			setupMock: func() {
				// 総件数取得
				countRows := sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(25)
				mock.ExpectQuery(`SELECT COUNT`).
					WillReturnRows(countRows)
				// 一覧取得
				rows := sqlmock.NewRows([]string{
					"code", "code_type", "currency_type", "amount",
					"max_uses", "current_uses", "valid_from", "valid_until",
					"status", "metadata", "created_at", "updated_at",
				}).
					AddRow("CODE1", "promotion", "paid", 1000, 100, 0, time.Now().Add(-24*time.Hour), time.Now().Add(24*time.Hour), "active", nil, time.Now(), time.Now()).
					AddRow("CODE2", "gift", "free", 500, 0, 5, time.Now().Add(-12*time.Hour), time.Now().Add(12*time.Hour), "active", `{"campaign_id":"campaign_001"}`, time.Now(), time.Now()).
					AddRow("CODE3", "event", "paid", 2000, 50, 10, time.Now().Add(-6*time.Hour), time.Now().Add(6*time.Hour), "active", nil, time.Now(), time.Now())
				mock.ExpectQuery(`SELECT`).
					WithArgs(10, 0).
					WillReturnRows(rows)
			},
			wantCount: 3,
			wantTotal: 25,
			wantError: false,
		},
		{
			name:   "正常系: 空の結果",
			limit:  10,
			offset: 0,
			setupMock: func() {
				// 総件数取得
				countRows := sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0)
				mock.ExpectQuery(`SELECT COUNT`).
					WillReturnRows(countRows)
				// 一覧取得（空）
				rows := sqlmock.NewRows([]string{
					"code", "code_type", "currency_type", "amount",
					"max_uses", "current_uses", "valid_from", "valid_until",
					"status", "metadata", "created_at", "updated_at",
				})
				mock.ExpectQuery(`SELECT`).
					WithArgs(10, 0).
					WillReturnRows(rows)
			},
			wantCount: 0,
			wantTotal: 0,
			wantError: false,
		},
		{
			name:   "正常系: ページネーション（offset指定）",
			limit:  5,
			offset: 10,
			setupMock: func() {
				// 総件数取得
				countRows := sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(20)
				mock.ExpectQuery(`SELECT COUNT`).
					WillReturnRows(countRows)
				// 一覧取得
				rows := sqlmock.NewRows([]string{
					"code", "code_type", "currency_type", "amount",
					"max_uses", "current_uses", "valid_from", "valid_until",
					"status", "metadata", "created_at", "updated_at",
				}).
					AddRow("CODE11", "promotion", "paid", 1000, 100, 0, time.Now(), time.Now().Add(24*time.Hour), "active", nil, time.Now(), time.Now()).
					AddRow("CODE12", "gift", "free", 500, 0, 0, time.Now(), time.Now().Add(24*time.Hour), "active", nil, time.Now(), time.Now())
				mock.ExpectQuery(`SELECT`).
					WithArgs(5, 10).
					WillReturnRows(rows)
			},
			wantCount: 2,
			wantTotal: 20,
			wantError: false,
		},
		{
			name:   "異常系: 総件数取得でDBエラー",
			limit:  10,
			offset: 0,
			setupMock: func() {
				mock.ExpectQuery(`SELECT COUNT`).
					WillReturnError(sql.ErrConnDone)
			},
			wantError: true,
		},
		{
			name:   "異常系: 一覧取得でDBエラー",
			limit:  10,
			offset: 0,
			setupMock: func() {
				// 総件数取得
				countRows := sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(10)
				mock.ExpectQuery(`SELECT COUNT`).
					WillReturnRows(countRows)
				// 一覧取得でエラー
				mock.ExpectQuery(`SELECT`).
					WithArgs(10, 0).
					WillReturnError(sql.ErrConnDone)
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			ctx := context.Background()
			codes, total, err := repo.FindAll(ctx, tt.limit, tt.offset)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, codes)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, codes)
				assert.Equal(t, tt.wantCount, len(codes))
				assert.Equal(t, tt.wantTotal, total)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
