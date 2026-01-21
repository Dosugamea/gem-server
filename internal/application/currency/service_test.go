package currency

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"

	"gem-server/internal/domain/currency"
	"gem-server/internal/domain/service"
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

// MockTransactionManager モックトランザクションマネージャー
type MockTransactionManager struct {
	mock.Mock
}

func (m *MockTransactionManager) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	args := m.Called(ctx, fn)
	// 実際のトランザクションは使わず、関数を直接実行
	if fn != nil {
		return fn(nil)
	}
	return args.Error(0)
}

func TestCurrencyApplicationService_GetBalance(t *testing.T) {
	tests := []struct {
		name       string
		req        *GetBalanceRequest
		setupMocks func(*MockCurrencyRepository)
		want       *GetBalanceResponse
		wantError  bool
	}{
		{
			name: "正常系: 有償・無償通貨両方存在",
			req: &GetBalanceRequest{
				UserID: "user123",
			},
			setupMocks: func(mcr *MockCurrencyRepository) {
				paidCurrency := mustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				freeCurrency := mustNewCurrency("user123", currency.CurrencyTypeFree, 500, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil)
			},
			want: &GetBalanceResponse{
				UserID: "user123",
				Balances: map[string]int64{
					"paid": 1000,
					"free": 500,
				},
			},
			wantError: false,
		},
		{
			name: "正常系: 有償通貨のみ存在",
			req: &GetBalanceRequest{
				UserID: "user123",
			},
			setupMocks: func(mcr *MockCurrencyRepository) {
				paidCurrency := mustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(nil, currency.ErrCurrencyNotFound)
			},
			want: &GetBalanceResponse{
				UserID: "user123",
				Balances: map[string]int64{
					"paid": 1000,
					"free": 0,
				},
			},
			wantError: false,
		},
		{
			name: "正常系: 無償通貨のみ存在",
			req: &GetBalanceRequest{
				UserID: "user123",
			},
			setupMocks: func(mcr *MockCurrencyRepository) {
				freeCurrency := mustNewCurrency("user123", currency.CurrencyTypeFree, 500, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(nil, currency.ErrCurrencyNotFound)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil)
			},
			want: &GetBalanceResponse{
				UserID: "user123",
				Balances: map[string]int64{
					"paid": 0,
					"free": 500,
				},
			},
			wantError: false,
		},
		{
			name: "正常系: 通貨が存在しない",
			req: &GetBalanceRequest{
				UserID: "user123",
			},
			setupMocks: func(mcr *MockCurrencyRepository) {
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(nil, currency.ErrCurrencyNotFound)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(nil, currency.ErrCurrencyNotFound)
			},
			want: &GetBalanceResponse{
				UserID: "user123",
				Balances: map[string]int64{
					"paid": 0,
					"free": 0,
				},
			},
			wantError: false,
		},
		{
			name: "異常系: 有償通貨取得エラー",
			req: &GetBalanceRequest{
				UserID: "user123",
			},
			setupMocks: func(mcr *MockCurrencyRepository) {
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(nil, errors.New("database error"))
			},
			want:      nil,
			wantError: true,
		},
		{
			name: "異常系: 無償通貨取得エラー",
			req: &GetBalanceRequest{
				UserID: "user123",
			},
			setupMocks: func(mcr *MockCurrencyRepository) {
				paidCurrency := mustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(nil, errors.New("database error"))
			},
			want:      nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCurrencyRepo := new(MockCurrencyRepository)
			mockTransactionRepo := new(MockTransactionRepository)
			mockTxManager := new(MockTransactionManager)

			tt.setupMocks(mockCurrencyRepo)

			// モックロガーとメトリクスを作成（実際の実装を使う）
			tracer := otel.Tracer("test")
			logger := otelinfra.NewLogger(tracer)
			metrics, err := otelinfra.NewMetrics("test")
			require.NoError(t, err)
			currencyService := service.NewCurrencyService(mockCurrencyRepo)

			svc := NewCurrencyApplicationService(
				mockCurrencyRepo,
				mockTransactionRepo,
				mockTxManager,
				currencyService,
				logger,
				metrics,
			)

			ctx := context.Background()
			got, err := svc.GetBalance(ctx, tt.req)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want.UserID, got.UserID)
				assert.Equal(t, tt.want.Balances["paid"], got.Balances["paid"])
				assert.Equal(t, tt.want.Balances["free"], got.Balances["free"])
			}

			mockCurrencyRepo.AssertExpectations(t)
		})
	}
}

func TestCurrencyApplicationService_Grant(t *testing.T) {
	tests := []struct {
		name       string
		req        *GrantRequest
		setupMocks func(*MockCurrencyRepository, *MockTransactionRepository, *MockTransactionManager)
		wantError  bool
		checkFunc  func(*testing.T, *GrantResponse, error)
	}{
		{
			name: "正常系: 既存の通貨に付与",
			req: &GrantRequest{
				UserID:       "user123",
				CurrencyType: "paid",
				Amount:       1000,
				Metadata:     map[string]interface{}{"reason": "test"},
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				existingCurrency := mustNewCurrency("user123", currency.CurrencyTypePaid, 500, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(existingCurrency, nil)
				mcr.On("Save", mock.Anything, mock.MatchedBy(func(c *currency.Currency) bool {
					return c.Balance() == 1500 && c.Version() == 2
				})).Return(nil)
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(nil)
			},
			wantError: false,
			checkFunc: func(t *testing.T, resp *GrantResponse, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.TransactionID)
				assert.Equal(t, int64(1500), resp.BalanceAfter)
				assert.Equal(t, "completed", resp.Status)
			},
		},
		{
			name: "正常系: 新規通貨を作成して付与",
			req: &GrantRequest{
				UserID:       "user123",
				CurrencyType: "free",
				Amount:       500,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(nil, currency.ErrCurrencyNotFound)
				mcr.On("Create", mock.Anything, mock.MatchedBy(func(c *currency.Currency) bool {
					return c.Balance() == 0 && c.Version() == 0
				})).Return(nil)
				mcr.On("Save", mock.Anything, mock.MatchedBy(func(c *currency.Currency) bool {
					return c.Balance() == 500 && c.Version() == 1
				})).Return(nil)
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(nil)
			},
			wantError: false,
			checkFunc: func(t *testing.T, resp *GrantResponse, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.TransactionID)
				assert.Equal(t, int64(500), resp.BalanceAfter)
			},
		},
		{
			name: "異常系: 無効な金額",
			req: &GrantRequest{
				UserID:       "user123",
				CurrencyType: "paid",
				Amount:       0,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				// モックは呼ばれない
			},
			wantError: true,
			checkFunc: func(t *testing.T, resp *GrantResponse, err error) {
				assert.Error(t, err)
				assert.Equal(t, currency.ErrInvalidAmount, err)
			},
		},
		{
			name: "異常系: 無効な通貨タイプ",
			req: &GrantRequest{
				UserID:       "user123",
				CurrencyType: "invalid",
				Amount:       1000,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				// モックは呼ばれない
			},
			wantError: true,
		},
		{
			name: "異常系: 通貨取得でエラー（ErrCurrencyNotFound以外）",
			req: &GrantRequest{
				UserID:       "user123",
				CurrencyType: "paid",
				Amount:       1000,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(nil, assert.AnError)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(assert.AnError)
			},
			wantError: true,
		},
		{
			name: "異常系: 通貨作成でエラー",
			req: &GrantRequest{
				UserID:       "user123",
				CurrencyType: "free",
				Amount:       500,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(nil, currency.ErrCurrencyNotFound)
				mcr.On("Create", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(assert.AnError)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(assert.AnError)
			},
			wantError: true,
		},
		{
			name: "異常系: 通貨保存でエラー（リトライ後も失敗）",
			req: &GrantRequest{
				UserID:       "user123",
				CurrencyType: "paid",
				Amount:       1000,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				existingCurrency := mustNewCurrency("user123", currency.CurrencyTypePaid, 500, 1)
				// リトライをシミュレート（3回失敗）
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(existingCurrency, nil).Times(3)
				mcr.On("Save", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(assert.AnError).Times(3)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(assert.AnError)
			},
			wantError: true,
		},
		{
			name: "異常系: トランザクション保存でエラー",
			req: &GrantRequest{
				UserID:       "user123",
				CurrencyType: "paid",
				Amount:       1000,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				existingCurrency := mustNewCurrency("user123", currency.CurrencyTypePaid, 500, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(existingCurrency, nil)
				mcr.On("Save", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(nil)
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(assert.AnError)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(assert.AnError)
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCurrencyRepo := new(MockCurrencyRepository)
			mockTransactionRepo := new(MockTransactionRepository)
			mockTxManager := new(MockTransactionManager)

			tt.setupMocks(mockCurrencyRepo, mockTransactionRepo, mockTxManager)

			tracer := otel.Tracer("test")
			logger := otelinfra.NewLogger(tracer)
			metrics, err := otelinfra.NewMetrics("test")
			require.NoError(t, err)
			currencyService := service.NewCurrencyService(mockCurrencyRepo)

			svc := NewCurrencyApplicationService(
				mockCurrencyRepo,
				mockTransactionRepo,
				mockTxManager,
				currencyService,
				logger,
				metrics,
			)

			ctx := context.Background()
			got, err := svc.Grant(ctx, tt.req)

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

func TestCurrencyApplicationService_Consume(t *testing.T) {
	tests := []struct {
		name       string
		req        *ConsumeRequest
		setupMocks func(*MockCurrencyRepository, *MockTransactionRepository, *MockTransactionManager)
		wantError  bool
		checkFunc  func(*testing.T, *ConsumeResponse, error)
	}{
		{
			name: "正常系: 有償通貨を消費",
			req: &ConsumeRequest{
				UserID:       "user123",
				CurrencyType: "paid",
				Amount:       500,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				existingCurrency := mustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(existingCurrency, nil)
				mcr.On("Save", mock.Anything, mock.MatchedBy(func(c *currency.Currency) bool {
					return c.Balance() == 500 && c.Version() == 2
				})).Return(nil)
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(nil)
			},
			wantError: false,
			checkFunc: func(t *testing.T, resp *ConsumeResponse, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.TransactionID)
				assert.Equal(t, int64(500), resp.BalanceAfter)
				assert.Equal(t, "completed", resp.Status)
			},
		},
		{
			name: "異常系: 残高不足",
			req: &ConsumeRequest{
				UserID:       "user123",
				CurrencyType: "paid",
				Amount:       2000,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				existingCurrency := mustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(existingCurrency, nil)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(currency.ErrInsufficientBalance)
			},
			wantError: true,
			checkFunc: func(t *testing.T, resp *ConsumeResponse, err error) {
				assert.Error(t, err)
				assert.Equal(t, currency.ErrInsufficientBalance, err)
			},
		},
		{
			name: "異常系: 通貨が見つからない",
			req: &ConsumeRequest{
				UserID:       "user123",
				CurrencyType: "paid",
				Amount:       500,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(nil, currency.ErrCurrencyNotFound)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(currency.ErrCurrencyNotFound)
			},
			wantError: true,
		},
		{
			name: "異常系: 通貨取得でエラー（ErrCurrencyNotFound以外）",
			req: &ConsumeRequest{
				UserID:       "user123",
				CurrencyType: "paid",
				Amount:       500,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(nil, assert.AnError)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(assert.AnError)
			},
			wantError: true,
		},
		{
			name: "異常系: 通貨保存でエラー（リトライ後も失敗）",
			req: &ConsumeRequest{
				UserID:       "user123",
				CurrencyType: "paid",
				Amount:       500,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				existingCurrency := mustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				// リトライをシミュレート（3回失敗）
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(existingCurrency, nil).Times(3)
				mcr.On("Save", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(assert.AnError).Times(3)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(assert.AnError)
			},
			wantError: true,
		},
		{
			name: "異常系: トランザクション保存でエラー",
			req: &ConsumeRequest{
				UserID:       "user123",
				CurrencyType: "paid",
				Amount:       500,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				existingCurrency := mustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(existingCurrency, nil)
				mcr.On("Save", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(nil)
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(assert.AnError)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(assert.AnError)
			},
			wantError: true,
		},
		{
			name: "異常系: 無効な通貨タイプ",
			req: &ConsumeRequest{
				UserID:       "user123",
				CurrencyType: "invalid",
				Amount:       500,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				// モックは呼ばれない
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCurrencyRepo := new(MockCurrencyRepository)
			mockTransactionRepo := new(MockTransactionRepository)
			mockTxManager := new(MockTransactionManager)

			tt.setupMocks(mockCurrencyRepo, mockTransactionRepo, mockTxManager)

			tracer := otel.Tracer("test")
			logger := otelinfra.NewLogger(tracer)
			metrics, err := otelinfra.NewMetrics("test")
			require.NoError(t, err)
			currencyService := service.NewCurrencyService(mockCurrencyRepo)

			svc := NewCurrencyApplicationService(
				mockCurrencyRepo,
				mockTransactionRepo,
				mockTxManager,
				currencyService,
				logger,
				metrics,
			)

			ctx := context.Background()
			got, err := svc.Consume(ctx, tt.req)

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

func TestCurrencyApplicationService_ConsumeWithPriority(t *testing.T) {
	tests := []struct {
		name       string
		req        *ConsumeRequest
		setupMocks func(*MockCurrencyRepository, *MockTransactionRepository, *MockTransactionManager)
		wantError  bool
		checkFunc  func(*testing.T, *ConsumeResponse, error)
	}{
		{
			name: "正常系: 無料通貨のみで支払える",
			req: &ConsumeRequest{
				UserID:       "user123",
				CurrencyType: "auto",
				Amount:       500,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				freeCurrency := mustNewCurrency("user123", currency.CurrencyTypeFree, 1000, 1)
				// HasSufficientBalance用のモック
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil).Once()
				// ConsumeWithPriority内でのFindByUserIDAndType用のモック
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil).Once()
				mcr.On("Save", mock.Anything, mock.MatchedBy(func(c *currency.Currency) bool {
					return c.Balance() == 500 && c.Version() == 2
				})).Return(nil)
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(nil)
			},
			wantError: false,
			checkFunc: func(t *testing.T, resp *ConsumeResponse, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.TransactionID)
				assert.Equal(t, int64(500), resp.TotalConsumed)
				assert.Len(t, resp.ConsumptionDetails, 1)
				assert.Equal(t, "free", resp.ConsumptionDetails[0].CurrencyType)
				assert.Equal(t, int64(500), resp.ConsumptionDetails[0].Amount)
				assert.Equal(t, int64(500), resp.ConsumptionDetails[0].BalanceAfter)
				assert.Equal(t, "completed", resp.Status)
			},
		},
		{
			name: "正常系: 無料+有償通貨で支払う",
			req: &ConsumeRequest{
				UserID:       "user123",
				CurrencyType: "auto",
				Amount:       1500,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				freeCurrency := mustNewCurrency("user123", currency.CurrencyTypeFree, 1000, 1)
				paidCurrency := mustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				// HasSufficientBalance用のモック
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil).Once()
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil).Once()
				// ConsumeWithPriority内でのFindByUserIDAndType用のモック
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil).Once()
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil).Once()
				// 無料通貨を全て消費
				mcr.On("Save", mock.Anything, mock.MatchedBy(func(c *currency.Currency) bool {
					return c.CurrencyType() == currency.CurrencyTypeFree && c.Balance() == 0 && c.Version() == 2
				})).Return(nil).Once()
				// 有償通貨を500消費
				mcr.On("Save", mock.Anything, mock.MatchedBy(func(c *currency.Currency) bool {
					return c.CurrencyType() == currency.CurrencyTypePaid && c.Balance() == 500 && c.Version() == 2
				})).Return(nil).Once()
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil).Twice()
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(nil)
			},
			wantError: false,
			checkFunc: func(t *testing.T, resp *ConsumeResponse, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.TransactionID)
				assert.Equal(t, int64(1500), resp.TotalConsumed)
				assert.Len(t, resp.ConsumptionDetails, 2)
				assert.Equal(t, "free", resp.ConsumptionDetails[0].CurrencyType)
				assert.Equal(t, int64(1000), resp.ConsumptionDetails[0].Amount)
				assert.Equal(t, "paid", resp.ConsumptionDetails[1].CurrencyType)
				assert.Equal(t, int64(500), resp.ConsumptionDetails[1].Amount)
				assert.Equal(t, int64(500), resp.ConsumptionDetails[1].BalanceAfter)
				assert.Equal(t, "completed", resp.Status)
			},
		},
		{
			name: "異常系: 残高不足",
			req: &ConsumeRequest{
				UserID:       "user123",
				CurrencyType: "auto",
				Amount:       3000,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				freeCurrency := mustNewCurrency("user123", currency.CurrencyTypeFree, 1000, 1)
				paidCurrency := mustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				// HasSufficientBalance用のモック
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil)
			},
			wantError: true,
			checkFunc: func(t *testing.T, resp *ConsumeResponse, err error) {
				assert.Error(t, err)
				assert.Equal(t, currency.ErrInsufficientBalance, err)
			},
		},
		{
			name: "異常系: 無料通貨が見つからない（有償通貨のみ）",
			req: &ConsumeRequest{
				UserID:       "user123",
				CurrencyType: "auto",
				Amount:       500,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				paidCurrency := mustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				// HasSufficientBalance用のモック
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(nil, currency.ErrCurrencyNotFound).Once()
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil).Once()
				// ConsumeWithPriority内でのFindByUserIDAndType用のモック
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(nil, currency.ErrCurrencyNotFound).Once()
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil).Once()
				mcr.On("Save", mock.Anything, mock.MatchedBy(func(c *currency.Currency) bool {
					return c.Balance() == 500 && c.Version() == 2
				})).Return(nil)
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(nil)
			},
			wantError: false,
			checkFunc: func(t *testing.T, resp *ConsumeResponse, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.TransactionID)
				assert.Equal(t, int64(500), resp.TotalConsumed)
				assert.Len(t, resp.ConsumptionDetails, 1)
				assert.Equal(t, "paid", resp.ConsumptionDetails[0].CurrencyType)
				assert.Equal(t, int64(500), resp.ConsumptionDetails[0].Amount)
				assert.Equal(t, int64(500), resp.ConsumptionDetails[0].BalanceAfter)
				assert.Equal(t, "completed", resp.Status)
			},
		},
		{
			name: "異常系: 無効な金額",
			req: &ConsumeRequest{
				UserID:       "user123",
				CurrencyType: "auto",
				Amount:       0,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				// モックは呼ばれない
			},
			wantError: true,
			checkFunc: func(t *testing.T, resp *ConsumeResponse, err error) {
				assert.Error(t, err)
				assert.Equal(t, currency.ErrInvalidAmount, err)
			},
		},
		{
			name: "異常系: HasSufficientBalance でエラー",
			req: &ConsumeRequest{
				UserID:       "user123",
				CurrencyType: "auto",
				Amount:       500,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(nil, assert.AnError)
			},
			wantError: true,
			checkFunc: func(t *testing.T, resp *ConsumeResponse, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to check balance")
			},
		},
		{
			name: "異常系: 無料通貨取得でエラー（ErrCurrencyNotFound以外）",
			req: &ConsumeRequest{
				UserID:       "user123",
				CurrencyType: "auto",
				Amount:       500,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				paidCurrency := mustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				// HasSufficientBalance用のモック
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(nil, currency.ErrCurrencyNotFound).Once()
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil).Once()
				// ConsumeWithPriority内でのFindByUserIDAndType用のモック
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(nil, assert.AnError).Once()
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(assert.AnError)
			},
			wantError: true,
		},
		{
			name: "異常系: 有償通貨取得でエラー",
			req: &ConsumeRequest{
				UserID:       "user123",
				CurrencyType: "auto",
				Amount:       1500,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				freeCurrency := mustNewCurrency("user123", currency.CurrencyTypeFree, 1000, 1)
				// HasSufficientBalance用のモック
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil).Once()
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(nil, currency.ErrCurrencyNotFound).Once()
				// ConsumeWithPriority内でのFindByUserIDAndType用のモック
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil).Once()
				mcr.On("Save", mock.Anything, mock.MatchedBy(func(c *currency.Currency) bool {
					return c.CurrencyType() == currency.CurrencyTypeFree && c.Balance() == 0
				})).Return(nil).Once()
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil).Once()
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(nil, assert.AnError).Once()
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(assert.AnError)
			},
			wantError: true,
		},
		{
			name: "異常系: トランザクション保存でエラー（無料通貨）",
			req: &ConsumeRequest{
				UserID:       "user123",
				CurrencyType: "auto",
				Amount:       500,
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				freeCurrency := mustNewCurrency("user123", currency.CurrencyTypeFree, 1000, 1)
				// HasSufficientBalance用のモック
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil).Once()
				// ConsumeWithPriority内でのFindByUserIDAndType用のモック
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil).Once()
				mcr.On("Save", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(nil)
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(assert.AnError)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(assert.AnError)
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCurrencyRepo := new(MockCurrencyRepository)
			mockTransactionRepo := new(MockTransactionRepository)
			mockTxManager := new(MockTransactionManager)

			tt.setupMocks(mockCurrencyRepo, mockTransactionRepo, mockTxManager)

			tracer := otel.Tracer("test")
			logger := otelinfra.NewLogger(tracer)
			metrics, err := otelinfra.NewMetrics("test")
			require.NoError(t, err)
			currencyService := service.NewCurrencyService(mockCurrencyRepo)

			svc := NewCurrencyApplicationService(
				mockCurrencyRepo,
				mockTransactionRepo,
				mockTxManager,
				currencyService,
				logger,
				metrics,
			)

			ctx := context.Background()
			got, err := svc.ConsumeWithPriority(ctx, tt.req)

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

func mustNewCurrency(userID string, currencyType currency.CurrencyType, balance int64, version int) *currency.Currency {
	c, err := currency.NewCurrency(userID, currencyType, balance, version)
	if err != nil {
		panic(err)
	}
	return c
}
