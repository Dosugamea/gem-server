package payment

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"

	"gem-server/internal/domain/currency"
	"gem-server/internal/domain/payment_request"
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

// MockPaymentRequestRepository モックPaymentRequestリポジトリ
type MockPaymentRequestRepository struct {
	mock.Mock
}

func (m *MockPaymentRequestRepository) Save(ctx context.Context, pr *payment_request.PaymentRequest) error {
	args := m.Called(ctx, pr)
	return args.Error(0)
}

func (m *MockPaymentRequestRepository) FindByPaymentRequestID(ctx context.Context, paymentRequestID string) (*payment_request.PaymentRequest, error) {
	args := m.Called(ctx, paymentRequestID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*payment_request.PaymentRequest), args.Error(1)
}

func (m *MockPaymentRequestRepository) Update(ctx context.Context, pr *payment_request.PaymentRequest) error {
	args := m.Called(ctx, pr)
	return args.Error(0)
}

// MockTransactionManager モックトランザクションマネージャー
type MockTransactionManager struct {
	mock.Mock
}

func (m *MockTransactionManager) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	args := m.Called(mock.Anything, mock.Anything)
	if fn != nil {
		return fn(nil)
	}
	return args.Error(0)
}

func TestPaymentApplicationService_ProcessPayment(t *testing.T) {
	tests := []struct {
		name       string
		req        *ProcessPaymentRequest
		setupMocks func(*MockCurrencyRepository, *MockTransactionRepository, *MockPaymentRequestRepository, *MockTransactionManager)
		wantError  bool
		checkFunc  func(*testing.T, *ProcessPaymentResponse, error)
	}{
		{
			name: "正常系: 新規決済（無料通貨のみで支払い）",
			req: &ProcessPaymentRequest{
				PaymentRequestID: "pr123",
				UserID:           "user123",
				Amount:           500,
				Currency:         "JPY",
				MethodName:       "test",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mprr *MockPaymentRequestRepository, mtm *MockTransactionManager) {
				// PaymentRequestが見つからない
				mprr.On("FindByPaymentRequestID", mock.Anything, "pr123").Return(nil, payment_request.ErrPaymentRequestNotFound)
				// 無料通貨が存在
				freeCurrency := currency.MustNewCurrency("user123", currency.CurrencyTypeFree, 1000, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil)
				mcr.On("Save", mock.Anything, mock.MatchedBy(func(c *currency.Currency) bool {
					return c.Balance() == 500 && c.Version() == 2
				})).Return(nil)
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil)
				mprr.On("Save", mock.Anything, mock.MatchedBy(func(pr *payment_request.PaymentRequest) bool {
					return pr.IsCompleted()
				})).Return(nil)
				mtm.On("WithTransaction", mock.Anything, mock.Anything).Return(nil)
			},
			wantError: false,
			checkFunc: func(t *testing.T, resp *ProcessPaymentResponse, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.TransactionID)
				assert.Equal(t, "pr123", resp.PaymentRequestID)
				assert.Equal(t, int64(500), resp.TotalConsumed)
				assert.Equal(t, "completed", resp.Status)
				assert.Len(t, resp.ConsumptionDetails, 1)
				assert.Equal(t, "free", resp.ConsumptionDetails[0].CurrencyType)
			},
		},
		{
			name: "正常系: 既に完了済みの決済（冪等性保証）",
			req: &ProcessPaymentRequest{
				PaymentRequestID: "pr123",
				UserID:           "user123",
				Amount:           500,
				Currency:         "JPY",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mprr *MockPaymentRequestRepository, mtm *MockTransactionManager) {
				// 既に完了済みのPaymentRequest
				completedPR := payment_request.MustNewPaymentRequest("pr123", "user123", 500, "JPY", currency.CurrencyTypePaid)
				completedPR.Complete()
				mprr.On("FindByPaymentRequestID", mock.Anything, "pr123").Return(completedPR, nil)
				// 既存のトランザクション
				existingTxn := transaction.MustNewTransaction(
					"txn123",
					"user123",
					transaction.TransactionTypeConsume,
					currency.CurrencyTypePaid,
					500,
					1000,
					500,
					transaction.TransactionStatusCompleted,
					map[string]interface{}{"payment_request_id": "pr123"},
				)
				mtr.On("FindByPaymentRequestID", mock.Anything, "pr123").Return(existingTxn, nil)
			},
			wantError: false,
			checkFunc: func(t *testing.T, resp *ProcessPaymentResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, "txn123", resp.TransactionID)
				assert.Equal(t, "pr123", resp.PaymentRequestID)
				assert.Equal(t, "completed", resp.Status)
			},
		},
		{
			name: "異常系: 無効な金額",
			req: &ProcessPaymentRequest{
				PaymentRequestID: "pr123",
				UserID:           "user123",
				Amount:           0,
				Currency:         "JPY",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mprr *MockPaymentRequestRepository, mtm *MockTransactionManager) {
				mprr.On("FindByPaymentRequestID", mock.Anything, "pr123").Return(nil, payment_request.ErrPaymentRequestNotFound)
			},
			wantError: true,
		},
		{
			name: "正常系: 無料+有償通貨で支払う",
			req: &ProcessPaymentRequest{
				PaymentRequestID: "pr123",
				UserID:           "user123",
				Amount:           1500,
				Currency:         "JPY",
				MethodName:       "test",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mprr *MockPaymentRequestRepository, mtm *MockTransactionManager) {
				mprr.On("FindByPaymentRequestID", mock.Anything, "pr123").Return(nil, payment_request.ErrPaymentRequestNotFound)
				// 無料通貨と有償通貨が存在
				freeCurrency := currency.MustNewCurrency("user123", currency.CurrencyTypeFree, 1000, 1)
				paidCurrency := currency.MustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil)
				// 無料通貨を全て消費
				mcr.On("Save", mock.Anything, mock.MatchedBy(func(c *currency.Currency) bool {
					return c.CurrencyType() == currency.CurrencyTypeFree && c.Balance() == 0 && c.Version() == 2
				})).Return(nil).Once()
				// 有償通貨を500消費
				mcr.On("Save", mock.Anything, mock.MatchedBy(func(c *currency.Currency) bool {
					return c.CurrencyType() == currency.CurrencyTypePaid && c.Balance() == 500 && c.Version() == 2
				})).Return(nil).Once()
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil).Twice()
				mprr.On("Save", mock.Anything, mock.MatchedBy(func(pr *payment_request.PaymentRequest) bool {
					return pr.IsCompleted()
				})).Return(nil)
				mtm.On("WithTransaction", mock.Anything, mock.Anything).Return(nil)
			},
			wantError: false,
			checkFunc: func(t *testing.T, resp *ProcessPaymentResponse, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.TransactionID)
				assert.Equal(t, "pr123", resp.PaymentRequestID)
				assert.Equal(t, int64(1500), resp.TotalConsumed)
				assert.Equal(t, "completed", resp.Status)
				assert.Len(t, resp.ConsumptionDetails, 2)
			},
		},
		{
			name: "異常系: 既に処理中のPaymentRequest",
			req: &ProcessPaymentRequest{
				PaymentRequestID: "pr123",
				UserID:           "user123",
				Amount:           500,
				Currency:         "JPY",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mprr *MockPaymentRequestRepository, mtm *MockTransactionManager) {
				// 既に失敗状態のPaymentRequest
				failedPR := payment_request.MustNewPaymentRequest("pr123", "user123", 500, "JPY", currency.CurrencyTypePaid)
				failedPR.Fail()
				mprr.On("FindByPaymentRequestID", mock.Anything, "pr123").Return(failedPR, nil)
			},
			wantError: true,
			checkFunc: func(t *testing.T, resp *ProcessPaymentResponse, err error) {
				assert.Error(t, err)
				assert.Equal(t, payment_request.ErrPaymentRequestAlreadyProcessed, err)
			},
		},
		{
			name: "異常系: 残高不足",
			req: &ProcessPaymentRequest{
				PaymentRequestID: "pr123",
				UserID:           "user123",
				Amount:           2000,
				Currency:         "JPY",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mprr *MockPaymentRequestRepository, mtm *MockTransactionManager) {
				mprr.On("FindByPaymentRequestID", mock.Anything, "pr123").Return(nil, payment_request.ErrPaymentRequestNotFound)
				// 無料通貨が存在しない
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(nil, currency.ErrCurrencyNotFound)
				// 有料通貨の残高が不足
				paidCurrency := currency.MustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil)
				// PaymentRequestを失敗状態にする
				mprr.On("Update", mock.Anything, mock.MatchedBy(func(pr *payment_request.PaymentRequest) bool {
					return !pr.IsCompleted()
				})).Return(nil)
				mtm.On("WithTransaction", mock.Anything, mock.Anything).Return(currency.ErrInsufficientBalance)
			},
			wantError: true,
			checkFunc: func(t *testing.T, resp *ProcessPaymentResponse, err error) {
				assert.Error(t, err)
				assert.Equal(t, currency.ErrInsufficientBalance, err)
			},
		},
		{
			name: "異常系: FindByPaymentRequestID でエラー（ErrPaymentRequestNotFound以外）",
			req: &ProcessPaymentRequest{
				PaymentRequestID: "pr123",
				UserID:           "user123",
				Amount:           500,
				Currency:         "JPY",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mprr *MockPaymentRequestRepository, mtm *MockTransactionManager) {
				mprr.On("FindByPaymentRequestID", mock.Anything, "pr123").Return(nil, assert.AnError)
			},
			wantError: true,
			checkFunc: func(t *testing.T, resp *ProcessPaymentResponse, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to find payment request")
			},
		},
		{
			name: "異常系: 既存のトランザクション取得でエラー（完了済みの場合）",
			req: &ProcessPaymentRequest{
				PaymentRequestID: "pr123",
				UserID:           "user123",
				Amount:           500,
				Currency:         "JPY",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mprr *MockPaymentRequestRepository, mtm *MockTransactionManager) {
				completedPR := payment_request.MustNewPaymentRequest("pr123", "user123", 500, "JPY", currency.CurrencyTypePaid)
				completedPR.Complete()
				mprr.On("FindByPaymentRequestID", mock.Anything, "pr123").Return(completedPR, nil)
				mtr.On("FindByPaymentRequestID", mock.Anything, "pr123").Return(nil, assert.AnError)
			},
			wantError: true,
			checkFunc: func(t *testing.T, resp *ProcessPaymentResponse, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to find transaction")
			},
		},
		{
			name: "異常系: 無料通貨取得でエラー（ErrCurrencyNotFound以外）",
			req: &ProcessPaymentRequest{
				PaymentRequestID: "pr123",
				UserID:           "user123",
				Amount:           500,
				Currency:         "JPY",
				MethodName:       "test",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mprr *MockPaymentRequestRepository, mtm *MockTransactionManager) {
				mprr.On("FindByPaymentRequestID", mock.Anything, "pr123").Return(nil, payment_request.ErrPaymentRequestNotFound)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(nil, assert.AnError)
				mtm.On("WithTransaction", mock.Anything, mock.Anything).Return(assert.AnError)
			},
			wantError: true,
		},
		{
			name: "異常系: 無料通貨保存でエラー（リトライ後も失敗）",
			req: &ProcessPaymentRequest{
				PaymentRequestID: "pr123",
				UserID:           "user123",
				Amount:           500,
				Currency:         "JPY",
				MethodName:       "test",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mprr *MockPaymentRequestRepository, mtm *MockTransactionManager) {
				mprr.On("FindByPaymentRequestID", mock.Anything, "pr123").Return(nil, payment_request.ErrPaymentRequestNotFound)
				freeCurrency := currency.MustNewCurrency("user123", currency.CurrencyTypeFree, 1000, 1)
				// リトライをシミュレート（3回失敗）
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil).Times(3)
				mcr.On("Save", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(assert.AnError).Times(3)
				mtm.On("WithTransaction", mock.Anything, mock.Anything).Return(assert.AnError)
			},
			wantError: true,
		},
		{
			name: "異常系: 有償通貨取得でエラー",
			req: &ProcessPaymentRequest{
				PaymentRequestID: "pr123",
				UserID:           "user123",
				Amount:           1500,
				Currency:         "JPY",
				MethodName:       "test",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mprr *MockPaymentRequestRepository, mtm *MockTransactionManager) {
				mprr.On("FindByPaymentRequestID", mock.Anything, "pr123").Return(nil, payment_request.ErrPaymentRequestNotFound)
				freeCurrency := currency.MustNewCurrency("user123", currency.CurrencyTypeFree, 1000, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil).Once()
				mcr.On("Save", mock.Anything, mock.MatchedBy(func(c *currency.Currency) bool {
					return c.CurrencyType() == currency.CurrencyTypeFree && c.Balance() == 0
				})).Return(nil).Once()
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil).Once()
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(nil, assert.AnError).Once()
				mtm.On("WithTransaction", mock.Anything, mock.Anything).Return(assert.AnError)
			},
			wantError: true,
		},
		{
			name: "異常系: 有償通貨保存でエラー（リトライ後も失敗）",
			req: &ProcessPaymentRequest{
				PaymentRequestID: "pr123",
				UserID:           "user123",
				Amount:           1500,
				Currency:         "JPY",
				MethodName:       "test",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mprr *MockPaymentRequestRepository, mtm *MockTransactionManager) {
				mprr.On("FindByPaymentRequestID", mock.Anything, "pr123").Return(nil, payment_request.ErrPaymentRequestNotFound)
				freeCurrency := currency.MustNewCurrency("user123", currency.CurrencyTypeFree, 1000, 1)
				paidCurrency := currency.MustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil).Once()
				mcr.On("Save", mock.Anything, mock.MatchedBy(func(c *currency.Currency) bool {
					return c.CurrencyType() == currency.CurrencyTypeFree && c.Balance() == 0
				})).Return(nil).Once()
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil).Once()
				// 有償通貨のリトライをシミュレート（3回失敗）
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil).Times(3)
				mcr.On("Save", mock.Anything, mock.MatchedBy(func(c *currency.Currency) bool {
					return c.CurrencyType() == currency.CurrencyTypePaid
				})).Return(assert.AnError).Times(3)
				mprr.On("Update", mock.Anything, mock.AnythingOfType("*payment_request.PaymentRequest")).Return(nil).Times(3)
				mtm.On("WithTransaction", mock.Anything, mock.Anything).Return(assert.AnError)
			},
			wantError: true,
		},
		{
			name: "異常系: トランザクション保存でエラー（無料通貨）",
			req: &ProcessPaymentRequest{
				PaymentRequestID: "pr123",
				UserID:           "user123",
				Amount:           500,
				Currency:         "JPY",
				MethodName:       "test",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mprr *MockPaymentRequestRepository, mtm *MockTransactionManager) {
				mprr.On("FindByPaymentRequestID", mock.Anything, "pr123").Return(nil, payment_request.ErrPaymentRequestNotFound)
				freeCurrency := currency.MustNewCurrency("user123", currency.CurrencyTypeFree, 1000, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil)
				mcr.On("Save", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(nil)
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(assert.AnError)
				mtm.On("WithTransaction", mock.Anything, mock.Anything).Return(assert.AnError)
			},
			wantError: true,
		},
		{
			name: "異常系: PaymentRequest保存でエラー",
			req: &ProcessPaymentRequest{
				PaymentRequestID: "pr123",
				UserID:           "user123",
				Amount:           500,
				Currency:         "JPY",
				MethodName:       "test",
			},
			setupMocks: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mprr *MockPaymentRequestRepository, mtm *MockTransactionManager) {
				mprr.On("FindByPaymentRequestID", mock.Anything, "pr123").Return(nil, payment_request.ErrPaymentRequestNotFound)
				freeCurrency := currency.MustNewCurrency("user123", currency.CurrencyTypeFree, 1000, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil)
				mcr.On("Save", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(nil)
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil)
				mprr.On("Save", mock.Anything, mock.AnythingOfType("*payment_request.PaymentRequest")).Return(assert.AnError)
				mtm.On("WithTransaction", mock.Anything, mock.Anything).Return(assert.AnError)
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCurrencyRepo := new(MockCurrencyRepository)
			mockTransactionRepo := new(MockTransactionRepository)
			mockPaymentRequestRepo := new(MockPaymentRequestRepository)
			mockTxManager := new(MockTransactionManager)

			tt.setupMocks(mockCurrencyRepo, mockTransactionRepo, mockPaymentRequestRepo, mockTxManager)

			tracer := otel.Tracer("test")
			logger := otelinfra.NewLogger(tracer)
			metrics, err := otelinfra.NewMetrics("test")
			require.NoError(t, err)

			svc := NewPaymentApplicationService(
				mockCurrencyRepo,
				mockTransactionRepo,
				mockPaymentRequestRepo,
				mockTxManager,
				logger,
				metrics,
			)

			ctx := context.Background()
			got, err := svc.ProcessPayment(ctx, tt.req)

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
