package code_redemption

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"

	"gem-server/internal/domain/currency"
	"gem-server/internal/domain/redemption_code"
	"gem-server/internal/domain/transaction"
	otelinfra "gem-server/internal/infrastructure/observability/otel"
)

// MockCurrencyRepository モック通貨リポジトリ
type MockCurrencyRepository struct {
	mock.Mock
}

func (m *MockCurrencyRepository) FindByUserIDAndType(ctx context.Context, userID string, currencyType currency.CurrencyType) (*currency.Currency, error) {
	args := m.Called(ctx, userID, currencyType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*currency.Currency), args.Error(1)
}

func (m *MockCurrencyRepository) Save(ctx context.Context, c *currency.Currency) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}

func (m *MockCurrencyRepository) Create(ctx context.Context, c *currency.Currency) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}

// MockTransactionRepository モックトランザクションリポジトリ
type MockTransactionRepository struct {
	mock.Mock
}

func (m *MockTransactionRepository) Save(ctx context.Context, t *transaction.Transaction) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *MockTransactionRepository) FindByTransactionID(ctx context.Context, transactionID string) (*transaction.Transaction, error) {
	args := m.Called(ctx, transactionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*transaction.Transaction, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) FindByPaymentRequestID(ctx context.Context, paymentRequestID string) (*transaction.Transaction, error) {
	args := m.Called(ctx, paymentRequestID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*transaction.Transaction), args.Error(1)
}

// MockRedemptionCodeRepository モック引き換えコードリポジトリ
type MockRedemptionCodeRepository struct {
	mock.Mock
}

func (m *MockRedemptionCodeRepository) FindByCode(ctx context.Context, code string) (*redemption_code.RedemptionCode, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*redemption_code.RedemptionCode), args.Error(1)
}

func (m *MockRedemptionCodeRepository) Update(ctx context.Context, code *redemption_code.RedemptionCode) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *MockRedemptionCodeRepository) HasUserRedeemed(ctx context.Context, code string, userID string) (bool, error) {
	args := m.Called(ctx, code, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRedemptionCodeRepository) SaveRedemption(ctx context.Context, redemption *redemption_code.CodeRedemption) error {
	args := m.Called(ctx, redemption)
	return args.Error(0)
}

// MockTransactionManager モックトランザクションマネージャー
type MockTransactionManager struct {
	mock.Mock
}

func (m *MockTransactionManager) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	args := m.Called(ctx, fn)
	if fn != nil {
		return fn(nil)
	}
	return args.Error(0)
}

func TestCodeRedemptionApplicationService_Redeem(t *testing.T) {
	tests := []struct {
		name       string
		req        *RedeemCodeRequest
		setupMocks func(*MockCurrencyRepository, *MockTransactionRepository, *MockRedemptionCodeRepository, *MockTransactionManager)
		wantError  bool
		checkFunc  func(*testing.T, *RedeemCodeResponse, error)
	}{
		{
			name: "正常系: コードを引き換え（既存通貨に付与）",
			req: &RedeemCodeRequest{
				Code:   "TESTCODE123",
				UserID: "user123",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtm *MockTransactionManager) {
				// コードを取得
				codeType, _ := redemption_code.NewCodeType("promotion")
				code := redemption_code.NewRedemptionCode(
					"TESTCODE123",
					codeType,
					currency.CurrencyTypePaid,
					1000,
					1,                             // maxUses
					time.Now().Add(-24*time.Hour), // validFrom
					time.Now().Add(24*time.Hour),  // validUntil
					map[string]interface{}{},
				)
				mrcr.On("FindByCode", mock.Anything, "TESTCODE123").Return(code, nil)
				// ユーザーはまだ引き換えていない
				mrcr.On("HasUserRedeemed", mock.Anything, "TESTCODE123", "user123").Return(false, nil)
				// コードを更新
				mrcr.On("Update", mock.Anything, mock.AnythingOfType("*redemption_code.RedemptionCode")).Return(nil)
				// 既存の通貨に付与
				existingCurrency := currency.NewCurrency("user123", currency.CurrencyTypePaid, 500, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(existingCurrency, nil)
				mcr.On("Save", mock.Anything, mock.MatchedBy(func(c *currency.Currency) bool {
					return c.Balance() == 1500 && c.Version() == 2
				})).Return(nil)
				// トランザクション履歴を記録
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil)
				// 引き換え履歴を記録
				mrcr.On("SaveRedemption", mock.Anything, mock.AnythingOfType("*redemption_code.CodeRedemption")).Return(nil)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(nil)
			},
			wantError: false,
			checkFunc: func(t *testing.T, resp *RedeemCodeResponse, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.RedemptionID)
				assert.NotEmpty(t, resp.TransactionID)
				assert.Equal(t, "TESTCODE123", resp.Code)
				assert.Equal(t, "paid", resp.CurrencyType)
				assert.Equal(t, int64(1000), resp.Amount)
				assert.Equal(t, int64(1500), resp.BalanceAfter)
				assert.Equal(t, "completed", resp.Status)
			},
		},
		{
			name: "正常系: コードを引き換え（新規通貨を作成）",
			req: &RedeemCodeRequest{
				Code:   "TESTCODE123",
				UserID: "user123",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtm *MockTransactionManager) {
				codeType, _ := redemption_code.NewCodeType("promotion")
				code := redemption_code.NewRedemptionCode(
					"TESTCODE123",
					codeType,
					currency.CurrencyTypeFree,
					500,
					1,                             // maxUses
					time.Now().Add(-24*time.Hour), // validFrom
					time.Now().Add(24*time.Hour),  // validUntil
					map[string]interface{}{},
				)
				mrcr.On("FindByCode", mock.Anything, "TESTCODE123").Return(code, nil)
				mrcr.On("HasUserRedeemed", mock.Anything, "TESTCODE123", "user123").Return(false, nil)
				mrcr.On("Update", mock.Anything, mock.AnythingOfType("*redemption_code.RedemptionCode")).Return(nil)
				// 通貨が存在しない
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(nil, currency.ErrCurrencyNotFound)
				mcr.On("Create", mock.Anything, mock.MatchedBy(func(c *currency.Currency) bool {
					return c.Balance() == 0 && c.Version() == 0
				})).Return(nil)
				mcr.On("Save", mock.Anything, mock.MatchedBy(func(c *currency.Currency) bool {
					return c.Balance() == 500 && c.Version() == 1
				})).Return(nil)
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil)
				mrcr.On("SaveRedemption", mock.Anything, mock.AnythingOfType("*redemption_code.CodeRedemption")).Return(nil)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(nil)
			},
			wantError: false,
			checkFunc: func(t *testing.T, resp *RedeemCodeResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, "free", resp.CurrencyType)
				assert.Equal(t, int64(500), resp.Amount)
				assert.Equal(t, int64(500), resp.BalanceAfter)
			},
		},
		{
			name: "異常系: コードが見つからない",
			req: &RedeemCodeRequest{
				Code:   "INVALID",
				UserID: "user123",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtm *MockTransactionManager) {
				mrcr.On("FindByCode", mock.Anything, "INVALID").Return(nil, redemption_code.ErrCodeNotFound)
			},
			wantError: true,
			checkFunc: func(t *testing.T, resp *RedeemCodeResponse, err error) {
				assert.Error(t, err)
				assert.Equal(t, redemption_code.ErrCodeNotFound, err)
			},
		},
		{
			name: "異常系: コードが無効",
			req: &RedeemCodeRequest{
				Code:   "TESTCODE123",
				UserID: "user123",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtm *MockTransactionManager) {
				// 無効なコード（期限切れ）
				codeType, _ := redemption_code.NewCodeType("promotion")
				code := redemption_code.NewRedemptionCode(
					"TESTCODE123",
					codeType,
					currency.CurrencyTypePaid,
					1000,
					1,                             // maxUses
					time.Now().Add(-48*time.Hour), // validFrom
					time.Now().Add(-24*time.Hour), // validUntil - 期限切れ
					map[string]interface{}{},
				)
				mrcr.On("FindByCode", mock.Anything, "TESTCODE123").Return(code, nil)
			},
			wantError: true,
			checkFunc: func(t *testing.T, resp *RedeemCodeResponse, err error) {
				assert.Error(t, err)
				assert.Equal(t, redemption_code.ErrCodeNotRedeemable, err)
			},
		},
		{
			name: "異常系: ユーザーが既に引き換え済み",
			req: &RedeemCodeRequest{
				Code:   "TESTCODE123",
				UserID: "user123",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtm *MockTransactionManager) {
				codeType, _ := redemption_code.NewCodeType("promotion")
				code := redemption_code.NewRedemptionCode(
					"TESTCODE123",
					codeType,
					currency.CurrencyTypePaid,
					1000,
					1,                             // maxUses
					time.Now().Add(-24*time.Hour), // validFrom
					time.Now().Add(24*time.Hour),  // validUntil
					map[string]interface{}{},
				)
				mrcr.On("FindByCode", mock.Anything, "TESTCODE123").Return(code, nil)
				mrcr.On("HasUserRedeemed", mock.Anything, "TESTCODE123", "user123").Return(true, nil)
			},
			wantError: true,
			checkFunc: func(t *testing.T, resp *RedeemCodeResponse, err error) {
				assert.Error(t, err)
				assert.Equal(t, redemption_code.ErrUserAlreadyRedeemed, err)
			},
		},
		{
			name: "異常系: コードの使用回数上限に達した",
			req: &RedeemCodeRequest{
				Code:   "TESTCODE123",
				UserID: "user123",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtm *MockTransactionManager) {
				codeType, _ := redemption_code.NewCodeType("promotion")
				code := redemption_code.NewRedemptionCode(
					"TESTCODE123",
					codeType,
					currency.CurrencyTypePaid,
					1000,
					1,                             // maxUses
					time.Now().Add(-24*time.Hour), // validFrom
					time.Now().Add(24*time.Hour),  // validUntil
					map[string]interface{}{},
				)
				// 使用回数を上限まで増やす
				_ = code.Redeem()
				mrcr.On("FindByCode", mock.Anything, "TESTCODE123").Return(code, nil)
				mrcr.On("HasUserRedeemed", mock.Anything, "TESTCODE123", "user123").Return(false, nil)
			},
			wantError: true,
			checkFunc: func(t *testing.T, resp *RedeemCodeResponse, err error) {
				assert.Error(t, err)
				assert.Equal(t, redemption_code.ErrCodeNotRedeemable, err)
			},
		},
		{
			name: "異常系: FindByCode でデータベースエラー",
			req: &RedeemCodeRequest{
				Code:   "TESTCODE123",
				UserID: "user123",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtm *MockTransactionManager) {
				mrcr.On("FindByCode", mock.Anything, "TESTCODE123").Return(nil, assert.AnError)
			},
			wantError: true,
			checkFunc: func(t *testing.T, resp *RedeemCodeResponse, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to find code")
			},
		},
		{
			name: "異常系: HasUserRedeemed でエラー",
			req: &RedeemCodeRequest{
				Code:   "TESTCODE123",
				UserID: "user123",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtm *MockTransactionManager) {
				codeType, _ := redemption_code.NewCodeType("promotion")
				code := redemption_code.NewRedemptionCode(
					"TESTCODE123",
					codeType,
					currency.CurrencyTypePaid,
					1000,
					1,                             // maxUses
					time.Now().Add(-24*time.Hour), // validFrom
					time.Now().Add(24*time.Hour),  // validUntil
					map[string]interface{}{},
				)
				mrcr.On("FindByCode", mock.Anything, "TESTCODE123").Return(code, nil)
				mrcr.On("HasUserRedeemed", mock.Anything, "TESTCODE123", "user123").Return(false, assert.AnError)
			},
			wantError: true,
			checkFunc: func(t *testing.T, resp *RedeemCodeResponse, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to check redemption status")
			},
		},
		{
			name: "異常系: コード更新でエラー",
			req: &RedeemCodeRequest{
				Code:   "TESTCODE123",
				UserID: "user123",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtm *MockTransactionManager) {
				codeType, _ := redemption_code.NewCodeType("promotion")
				code := redemption_code.NewRedemptionCode(
					"TESTCODE123",
					codeType,
					currency.CurrencyTypePaid,
					1000,
					1,                             // maxUses
					time.Now().Add(-24*time.Hour), // validFrom
					time.Now().Add(24*time.Hour),  // validUntil
					map[string]interface{}{},
				)
				mrcr.On("FindByCode", mock.Anything, "TESTCODE123").Return(code, nil)
				mrcr.On("HasUserRedeemed", mock.Anything, "TESTCODE123", "user123").Return(false, nil)
				mrcr.On("Update", mock.Anything, mock.AnythingOfType("*redemption_code.RedemptionCode")).Return(assert.AnError)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(assert.AnError)
			},
			wantError: true,
		},
		{
			name: "異常系: 通貨作成でエラー",
			req: &RedeemCodeRequest{
				Code:   "TESTCODE123",
				UserID: "user123",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtm *MockTransactionManager) {
				codeType, _ := redemption_code.NewCodeType("promotion")
				code := redemption_code.NewRedemptionCode(
					"TESTCODE123",
					codeType,
					currency.CurrencyTypeFree,
					500,
					1,                             // maxUses
					time.Now().Add(-24*time.Hour), // validFrom
					time.Now().Add(24*time.Hour),  // validUntil
					map[string]interface{}{},
				)
				mrcr.On("FindByCode", mock.Anything, "TESTCODE123").Return(code, nil)
				mrcr.On("HasUserRedeemed", mock.Anything, "TESTCODE123", "user123").Return(false, nil)
				mrcr.On("Update", mock.Anything, mock.AnythingOfType("*redemption_code.RedemptionCode")).Return(nil)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(nil, currency.ErrCurrencyNotFound)
				mcr.On("Create", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(assert.AnError)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(assert.AnError)
			},
			wantError: true,
		},
		{
			name: "異常系: 通貨保存でエラー（リトライ後も失敗）",
			req: &RedeemCodeRequest{
				Code:   "TESTCODE123",
				UserID: "user123",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtm *MockTransactionManager) {
				codeType, _ := redemption_code.NewCodeType("promotion")
				code := redemption_code.NewRedemptionCode(
					"TESTCODE123",
					codeType,
					currency.CurrencyTypePaid,
					1000,
					1,                             // maxUses
					time.Now().Add(-24*time.Hour), // validFrom
					time.Now().Add(24*time.Hour),  // validUntil
					map[string]interface{}{},
				)
				mrcr.On("FindByCode", mock.Anything, "TESTCODE123").Return(code, nil)
				mrcr.On("HasUserRedeemed", mock.Anything, "TESTCODE123", "user123").Return(false, nil)
				mrcr.On("Update", mock.Anything, mock.AnythingOfType("*redemption_code.RedemptionCode")).Return(nil)
				existingCurrency := currency.NewCurrency("user123", currency.CurrencyTypePaid, 500, 1)
				// リトライをシミュレート（3回失敗）
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(existingCurrency, nil).Times(3)
				mcr.On("Save", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(assert.AnError).Times(3)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(assert.AnError)
			},
			wantError: true,
		},
		{
			name: "異常系: トランザクション保存でエラー",
			req: &RedeemCodeRequest{
				Code:   "TESTCODE123",
				UserID: "user123",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtm *MockTransactionManager) {
				codeType, _ := redemption_code.NewCodeType("promotion")
				code := redemption_code.NewRedemptionCode(
					"TESTCODE123",
					codeType,
					currency.CurrencyTypePaid,
					1000,
					1,                             // maxUses
					time.Now().Add(-24*time.Hour), // validFrom
					time.Now().Add(24*time.Hour),  // validUntil
					map[string]interface{}{},
				)
				mrcr.On("FindByCode", mock.Anything, "TESTCODE123").Return(code, nil)
				mrcr.On("HasUserRedeemed", mock.Anything, "TESTCODE123", "user123").Return(false, nil)
				mrcr.On("Update", mock.Anything, mock.AnythingOfType("*redemption_code.RedemptionCode")).Return(nil)
				existingCurrency := currency.NewCurrency("user123", currency.CurrencyTypePaid, 500, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(existingCurrency, nil)
				mcr.On("Save", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(nil)
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(assert.AnError)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(assert.AnError)
			},
			wantError: true,
		},
		{
			name: "異常系: 引き換え履歴保存でエラー",
			req: &RedeemCodeRequest{
				Code:   "TESTCODE123",
				UserID: "user123",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtm *MockTransactionManager) {
				codeType, _ := redemption_code.NewCodeType("promotion")
				code := redemption_code.NewRedemptionCode(
					"TESTCODE123",
					codeType,
					currency.CurrencyTypePaid,
					1000,
					1,                             // maxUses
					time.Now().Add(-24*time.Hour), // validFrom
					time.Now().Add(24*time.Hour),  // validUntil
					map[string]interface{}{},
				)
				mrcr.On("FindByCode", mock.Anything, "TESTCODE123").Return(code, nil)
				mrcr.On("HasUserRedeemed", mock.Anything, "TESTCODE123", "user123").Return(false, nil)
				mrcr.On("Update", mock.Anything, mock.AnythingOfType("*redemption_code.RedemptionCode")).Return(nil)
				existingCurrency := currency.NewCurrency("user123", currency.CurrencyTypePaid, 500, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(existingCurrency, nil)
				mcr.On("Save", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(nil)
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil)
				mrcr.On("SaveRedemption", mock.Anything, mock.AnythingOfType("*redemption_code.CodeRedemption")).Return(assert.AnError)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(assert.AnError)
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCurrencyRepo := new(MockCurrencyRepository)
			mockTransactionRepo := new(MockTransactionRepository)
			mockRedemptionCodeRepo := new(MockRedemptionCodeRepository)
			mockTxManager := new(MockTransactionManager)

			tt.setupMocks(mockCurrencyRepo, mockTransactionRepo, mockRedemptionCodeRepo, mockTxManager)

			tracer := otel.Tracer("test")
			logger := otelinfra.NewLogger(tracer)
			metrics, err := otelinfra.NewMetrics("test")
			require.NoError(t, err)

			svc := NewCodeRedemptionApplicationService(
				mockCurrencyRepo,
				mockTransactionRepo,
				mockRedemptionCodeRepo,
				mockTxManager,
				logger,
				metrics,
			)

			ctx := context.Background()
			got, err := svc.Redeem(ctx, tt.req)

			if tt.wantError {
				assert.Error(t, err)
				if tt.checkFunc != nil {
					tt.checkFunc(t, got, err)
				}
			} else {
				if tt.checkFunc != nil {
					tt.checkFunc(t, got, err)
				} else {
					require.NoError(t, err)
					assert.NotNil(t, got)
				}
			}
		})
	}
}
