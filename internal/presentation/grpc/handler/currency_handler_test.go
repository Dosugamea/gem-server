package handler

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	redemptionapp "gem-server/internal/application/code_redemption"
	currencyapp "gem-server/internal/application/currency"
	historyapp "gem-server/internal/application/history"
	paymentapp "gem-server/internal/application/payment"
	"gem-server/internal/domain/currency"
	"gem-server/internal/domain/payment_request"
	"gem-server/internal/domain/redemption_code"
	"gem-server/internal/domain/service"
	"gem-server/internal/domain/transaction"
	otelinfra "gem-server/internal/infrastructure/observability/otel"
	"gem-server/internal/presentation/grpc/pb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	args := m.Called(ctx, fn)
	if fn != nil {
		return fn(nil)
	}
	return args.Error(0)
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

func (m *MockRedemptionCodeRepository) Create(ctx context.Context, code *redemption_code.RedemptionCode) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *MockRedemptionCodeRepository) Delete(ctx context.Context, code string) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *MockRedemptionCodeRepository) FindAll(ctx context.Context, limit, offset int) ([]*redemption_code.RedemptionCode, int, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*redemption_code.RedemptionCode), args.Int(1), args.Error(2)
}

// setupTestHandler テスト用のハンドラーをセットアップ
func setupTestHandler(t *testing.T) (*CurrencyHandler, *MockCurrencyRepository, *MockTransactionRepository, *MockPaymentRequestRepository, *MockTransactionManager, *MockRedemptionCodeRepository) {
	t.Helper()

	mockCurrencyRepo := new(MockCurrencyRepository)
	mockTransactionRepo := new(MockTransactionRepository)
	mockPaymentRequestRepo := new(MockPaymentRequestRepository)
	mockTxManager := new(MockTransactionManager)
	mockRedemptionCodeRepo := new(MockRedemptionCodeRepository)

	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)
	metrics, _ := otelinfra.NewMetrics("test")
	currencyService := service.NewCurrencyService(mockCurrencyRepo)

	currencyAppService := currencyapp.NewCurrencyApplicationService(
		mockCurrencyRepo,
		mockTransactionRepo,
		mockTxManager,
		currencyService,
		logger,
		metrics,
	)
	paymentAppService := paymentapp.NewPaymentApplicationService(
		mockCurrencyRepo,
		mockTransactionRepo,
		mockPaymentRequestRepo,
		mockTxManager,
		logger,
		metrics,
	)
	redemptionAppService := redemptionapp.NewCodeRedemptionApplicationService(
		mockCurrencyRepo,
		mockTransactionRepo,
		mockRedemptionCodeRepo,
		mockTxManager,
		logger,
		metrics,
	)
	historyAppService := historyapp.NewHistoryApplicationService(
		mockTransactionRepo,
		logger,
		metrics,
	)

	handler := NewCurrencyHandler(
		currencyAppService,
		paymentAppService,
		redemptionAppService,
		historyAppService,
	)

	return handler, mockCurrencyRepo, mockTransactionRepo, mockPaymentRequestRepo, mockTxManager, mockRedemptionCodeRepo
}

func TestCurrencyHandler_GetBalance(t *testing.T) {
	tests := []struct {
		name           string
		req            *pb.GetBalanceRequest
		setupMock      func(*MockCurrencyRepository, *MockTransactionRepository, *MockTransactionManager)
		expectedStatus codes.Code
		checkResponse  func(*testing.T, *pb.GetBalanceResponse)
	}{
		{
			name: "正常系: 残高取得成功",
			req: &pb.GetBalanceRequest{
				UserId: "user123",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				paidCurrency := currency.NewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				freeCurrency := currency.NewCurrency("user123", currency.CurrencyTypeFree, 500, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil)
			},
			expectedStatus: codes.OK,
			checkResponse: func(t *testing.T, resp *pb.GetBalanceResponse) {
				assert.Equal(t, "user123", resp.UserId)
				assert.Equal(t, "1000", resp.Balances["paid"])
				assert.Equal(t, "500", resp.Balances["free"])
			},
		},
		{
			name: "異常系: user_idが空",
			req: &pb.GetBalanceRequest{
				UserId: "",
			},
			setupMock:      func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {},
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "正常系: 通貨が見つからない場合は残高0を返す",
			req: &pb.GetBalanceRequest{
				UserId: "user123",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(nil, currency.ErrCurrencyNotFound)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(nil, currency.ErrCurrencyNotFound)
			},
			expectedStatus: codes.OK,
			checkResponse: func(t *testing.T, resp *pb.GetBalanceResponse) {
				assert.Equal(t, "user123", resp.UserId)
				assert.Equal(t, "0", resp.Balances["paid"])
				assert.Equal(t, "0", resp.Balances["free"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockCurrencyRepo, mockTransactionRepo, _, mockTxManager, _ := setupTestHandler(t)

			tt.setupMock(mockCurrencyRepo, mockTransactionRepo, mockTxManager)

			ctx := context.Background()
			resp, err := handler.GetBalance(ctx, tt.req)

			if tt.expectedStatus == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, resp)
				if tt.checkResponse != nil {
					tt.checkResponse(t, resp)
				}
			} else {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.expectedStatus, st.Code())
			}

			mockCurrencyRepo.AssertExpectations(t)
		})
	}
}

func TestCurrencyHandler_Grant(t *testing.T) {
	tests := []struct {
		name           string
		req            *pb.GrantRequest
		setupMock      func(*MockCurrencyRepository, *MockTransactionRepository, *MockTransactionManager)
		expectedStatus codes.Code
		checkResponse  func(*testing.T, *pb.GrantResponse)
	}{
		{
			name: "正常系: 通貨付与成功（paid）",
			req: &pb.GrantRequest{
				UserId:       "user123",
				CurrencyType: "paid",
				Amount:       "100",
				Reason:       "test reason",
				Metadata:     map[string]string{"key1": "value1"},
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				paidCurrency := currency.NewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil)
				mcr.On("Save", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(nil)
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(nil)
			},
			expectedStatus: codes.OK,
			checkResponse: func(t *testing.T, resp *pb.GrantResponse) {
				assert.NotEmpty(t, resp.TransactionId)
				assert.NotEmpty(t, resp.BalanceAfter)
				assert.Equal(t, "completed", resp.Status)
			},
		},
		{
			name: "正常系: 通貨付与成功（free、metadataなし）",
			req: &pb.GrantRequest{
				UserId:       "user123",
				CurrencyType: "free",
				Amount:       "50",
				Reason:       "bonus",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				freeCurrency := currency.NewCurrency("user123", currency.CurrencyTypeFree, 500, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil)
				mcr.On("Save", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(nil)
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(nil)
			},
			expectedStatus: codes.OK,
			checkResponse: func(t *testing.T, resp *pb.GrantResponse) {
				assert.NotEmpty(t, resp.TransactionId)
				assert.NotEmpty(t, resp.BalanceAfter)
				assert.Equal(t, "completed", resp.Status)
			},
		},
		{
			name: "異常系: user_idが空",
			req: &pb.GrantRequest{
				UserId:       "",
				CurrencyType: "paid",
				Amount:       "100",
			},
			setupMock:      func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {},
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "異常系: currency_typeが空",
			req: &pb.GrantRequest{
				UserId:       "user123",
				CurrencyType: "",
				Amount:       "100",
			},
			setupMock:      func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {},
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "異常系: amountが空",
			req: &pb.GrantRequest{
				UserId:       "user123",
				CurrencyType: "paid",
				Amount:       "",
			},
			setupMock:      func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {},
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "異常系: 無効な金額フォーマット",
			req: &pb.GrantRequest{
				UserId:       "user123",
				CurrencyType: "paid",
				Amount:       "invalid",
			},
			setupMock:      func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {},
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "異常系: 無効な金額（負の値）",
			req: &pb.GrantRequest{
				UserId:       "user123",
				CurrencyType: "paid",
				Amount:       "-100",
			},
			setupMock:      func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {},
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "異常系: 通貨が見つからない",
			req: &pb.GrantRequest{
				UserId:       "user123",
				CurrencyType: "paid",
				Amount:       "100",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(nil, currency.ErrCurrencyNotFound)
				mcr.On("Create", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(nil)
				mcr.On("Save", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(nil)
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(nil)
			},
			expectedStatus: codes.OK,
			checkResponse: func(t *testing.T, resp *pb.GrantResponse) {
				assert.NotEmpty(t, resp.TransactionId)
				assert.Equal(t, "completed", resp.Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockCurrencyRepo, mockTransactionRepo, _, mockTxManager, _ := setupTestHandler(t)

			tt.setupMock(mockCurrencyRepo, mockTransactionRepo, mockTxManager)

			ctx := context.Background()
			resp, err := handler.Grant(ctx, tt.req)

			if tt.expectedStatus == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, resp)
				if tt.checkResponse != nil {
					tt.checkResponse(t, resp)
				}
			} else {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.expectedStatus, st.Code())
			}

			mockCurrencyRepo.AssertExpectations(t)
			mockTransactionRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}

func TestCurrencyHandler_Consume(t *testing.T) {
	tests := []struct {
		name           string
		req            *pb.ConsumeRequest
		setupMock      func(*MockCurrencyRepository, *MockTransactionRepository, *MockTransactionManager)
		expectedStatus codes.Code
		checkResponse  func(*testing.T, *pb.ConsumeResponse)
	}{
		{
			name: "正常系: 単一通貨タイプで消費（paid）",
			req: &pb.ConsumeRequest{
				UserId:       "user123",
				CurrencyType: "paid",
				Amount:       "50",
				ItemId:       "item123",
				UsePriority:  false,
				Metadata:     map[string]string{"key1": "value1"},
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				paidCurrency := currency.NewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil)
				mcr.On("Save", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(nil)
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(nil)
			},
			expectedStatus: codes.OK,
			checkResponse: func(t *testing.T, resp *pb.ConsumeResponse) {
				assert.NotEmpty(t, resp.TransactionId)
				assert.NotEmpty(t, resp.BalanceAfter)
				assert.Equal(t, "completed", resp.Status)
				assert.Empty(t, resp.ConsumptionDetails)
			},
		},
		{
			name: "正常系: 優先順位制御で消費（UsePriority=true）",
			req: &pb.ConsumeRequest{
				UserId:       "user123",
				CurrencyType: "paid",
				Amount:       "100",
				ItemId:       "item123",
				UsePriority:  true,
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				freeCurrency := currency.NewCurrency("user123", currency.CurrencyTypeFree, 500, 1)
				// HasSufficientBalance用（freeのみ、free通貨が500で消費額100なのでpaidは呼ばれない）
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil)
				// ConsumeWithPriority内でのFindByUserIDAndType用（freeのみ）
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil)
				mcr.On("Save", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(nil)
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(nil)
			},
			expectedStatus: codes.OK,
			checkResponse: func(t *testing.T, resp *pb.ConsumeResponse) {
				assert.NotEmpty(t, resp.TransactionId)
				assert.NotEmpty(t, resp.ConsumptionDetails)
				assert.NotEmpty(t, resp.TotalConsumed)
				assert.Equal(t, "completed", resp.Status)
			},
		},
		{
			name: "正常系: auto通貨タイプで消費（優先順位制御）",
			req: &pb.ConsumeRequest{
				UserId:       "user123",
				CurrencyType: "auto",
				Amount:       "100",
				ItemId:       "item123",
				UsePriority:  false,
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				freeCurrency := currency.NewCurrency("user123", currency.CurrencyTypeFree, 500, 1)
				// HasSufficientBalance用（freeのみ、free通貨が500で消費額100なのでpaidは呼ばれない）
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil)
				// ConsumeWithPriority内でのFindByUserIDAndType用（freeのみ）
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil)
				mcr.On("Save", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(nil)
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(nil)
			},
			expectedStatus: codes.OK,
			checkResponse: func(t *testing.T, resp *pb.ConsumeResponse) {
				assert.NotEmpty(t, resp.TransactionId)
				assert.NotEmpty(t, resp.ConsumptionDetails)
				assert.NotEmpty(t, resp.TotalConsumed)
				assert.Equal(t, "completed", resp.Status)
			},
		},
		{
			name: "異常系: user_idが空",
			req: &pb.ConsumeRequest{
				UserId:       "",
				CurrencyType: "paid",
				Amount:       "50",
			},
			setupMock:      func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {},
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "異常系: currency_typeが空",
			req: &pb.ConsumeRequest{
				UserId:       "user123",
				CurrencyType: "",
				Amount:       "50",
			},
			setupMock:      func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {},
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "異常系: amountが空",
			req: &pb.ConsumeRequest{
				UserId:       "user123",
				CurrencyType: "paid",
				Amount:       "",
			},
			setupMock:      func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {},
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "異常系: 無効な金額フォーマット",
			req: &pb.ConsumeRequest{
				UserId:       "user123",
				CurrencyType: "paid",
				Amount:       "invalid",
			},
			setupMock:      func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {},
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "異常系: 残高不足",
			req: &pb.ConsumeRequest{
				UserId:       "user123",
				CurrencyType: "paid",
				Amount:       "2000",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mtm *MockTransactionManager) {
				paidCurrency := currency.NewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(nil)
			},
			expectedStatus: codes.FailedPrecondition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockCurrencyRepo, mockTransactionRepo, _, mockTxManager, _ := setupTestHandler(t)

			tt.setupMock(mockCurrencyRepo, mockTransactionRepo, mockTxManager)

			ctx := context.Background()
			resp, err := handler.Consume(ctx, tt.req)

			if tt.expectedStatus == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, resp)
				if tt.checkResponse != nil {
					tt.checkResponse(t, resp)
				}
			} else {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.expectedStatus, st.Code())
			}

			mockCurrencyRepo.AssertExpectations(t)
			mockTransactionRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}

func TestCurrencyHandler_ProcessPayment(t *testing.T) {
	tests := []struct {
		name           string
		req            *pb.ProcessPaymentRequest
		setupMock      func(*MockCurrencyRepository, *MockTransactionRepository, *MockPaymentRequestRepository, *MockTransactionManager)
		expectedStatus codes.Code
		checkResponse  func(*testing.T, *pb.ProcessPaymentResponse)
	}{
		{
			name: "正常系: 決済処理成功",
			req: &pb.ProcessPaymentRequest{
				PaymentRequestId: "pr123",
				UserId:           "user123",
				Amount:           "100",
				MethodName:       "test_method",
				Details:          map[string]string{"key1": "value1"},
				Currency:         "JPY",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mpr *MockPaymentRequestRepository, mtm *MockTransactionManager) {
				paidCurrency := currency.NewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				freeCurrency := currency.NewCurrency("user123", currency.CurrencyTypeFree, 500, 1)
				pr := payment_request.NewPaymentRequest("pr123", "user123", 100, "JPY", currency.CurrencyTypePaid)
				mpr.On("FindByPaymentRequestID", mock.Anything, "pr123").Return(pr, nil)
				mpr.On("Update", mock.Anything, mock.AnythingOfType("*payment_request.PaymentRequest")).Return(nil).Maybe()
				mpr.On("Save", mock.Anything, mock.AnythingOfType("*payment_request.PaymentRequest")).Return(nil).Maybe()
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil).Maybe()
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil).Maybe()
				mcr.On("Save", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(nil).Maybe()
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil).Maybe()
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(nil)
			},
			expectedStatus: codes.OK,
			checkResponse: func(t *testing.T, resp *pb.ProcessPaymentResponse) {
				assert.NotEmpty(t, resp.TransactionId)
				assert.Equal(t, "pr123", resp.PaymentRequestId)
				assert.NotEmpty(t, resp.ConsumptionDetails)
				assert.NotEmpty(t, resp.TotalConsumed)
				assert.Equal(t, "completed", resp.Status)
			},
		},
		{
			name: "異常系: payment_request_idが空",
			req: &pb.ProcessPaymentRequest{
				PaymentRequestId: "",
				UserId:           "user123",
				Amount:           "100",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mpr *MockPaymentRequestRepository, mtm *MockTransactionManager) {
			},
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "異常系: user_idが空",
			req: &pb.ProcessPaymentRequest{
				PaymentRequestId: "pr123",
				UserId:           "",
				Amount:           "100",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mpr *MockPaymentRequestRepository, mtm *MockTransactionManager) {
			},
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "異常系: amountが空",
			req: &pb.ProcessPaymentRequest{
				PaymentRequestId: "pr123",
				UserId:           "user123",
				Amount:           "",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mpr *MockPaymentRequestRepository, mtm *MockTransactionManager) {
			},
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "異常系: 無効な金額フォーマット",
			req: &pb.ProcessPaymentRequest{
				PaymentRequestId: "pr123",
				UserId:           "user123",
				Amount:           "invalid",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mpr *MockPaymentRequestRepository, mtm *MockTransactionManager) {
			},
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "正常系: PaymentRequestが見つからない場合は新規作成",
			req: &pb.ProcessPaymentRequest{
				PaymentRequestId: "pr123",
				UserId:           "user123",
				Amount:           "100",
				Currency:         "JPY",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mpr *MockPaymentRequestRepository, mtm *MockTransactionManager) {
				paidCurrency := currency.NewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				freeCurrency := currency.NewCurrency("user123", currency.CurrencyTypeFree, 500, 1)
				mpr.On("FindByPaymentRequestID", mock.Anything, "pr123").Return(nil, payment_request.ErrPaymentRequestNotFound).Once()
				mpr.On("Save", mock.Anything, mock.AnythingOfType("*payment_request.PaymentRequest")).Return(nil).Once()
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil).Once()
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil).Maybe()
				mcr.On("Save", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(nil)
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(nil)
			},
			expectedStatus: codes.OK,
			checkResponse: func(t *testing.T, resp *pb.ProcessPaymentResponse) {
				assert.NotEmpty(t, resp.TransactionId)
				assert.Equal(t, "pr123", resp.PaymentRequestId)
				assert.Equal(t, "completed", resp.Status)
			},
		},
		{
			name: "異常系: PaymentRequestが既に処理済み",
			req: &pb.ProcessPaymentRequest{
				PaymentRequestId: "pr123",
				UserId:           "user123",
				Amount:           "100",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mpr *MockPaymentRequestRepository, mtm *MockTransactionManager) {
				pr := payment_request.NewPaymentRequest("pr123", "user123", 100, "JPY", currency.CurrencyTypePaid)
				pr.Complete()
				txn := transaction.NewTransaction(
					"txn1",
					"user123",
					transaction.TransactionTypeConsume,
					currency.CurrencyTypePaid,
					100,
					1000,
					900,
					transaction.TransactionStatusCompleted,
					map[string]interface{}{},
				)
				mpr.On("FindByPaymentRequestID", mock.Anything, "pr123").Return(pr, nil)
				mtr.On("FindByPaymentRequestID", mock.Anything, "pr123").Return(txn, nil)
			},
			expectedStatus: codes.OK, // 既に処理済みの場合は既存の結果を返すのでOK
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockCurrencyRepo, mockTransactionRepo, mockPaymentRequestRepo, mockTxManager, _ := setupTestHandler(t)

			tt.setupMock(mockCurrencyRepo, mockTransactionRepo, mockPaymentRequestRepo, mockTxManager)

			ctx := context.Background()
			resp, err := handler.ProcessPayment(ctx, tt.req)

			if tt.expectedStatus == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, resp)
				if tt.checkResponse != nil {
					tt.checkResponse(t, resp)
				}
			} else {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.expectedStatus, st.Code())
			}

			mockPaymentRequestRepo.AssertExpectations(t)
			mockCurrencyRepo.AssertExpectations(t)
			mockTransactionRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}

func TestCurrencyHandler_RedeemCode(t *testing.T) {
	tests := []struct {
		name           string
		req            *pb.RedeemCodeRequest
		setupMock      func(*MockCurrencyRepository, *MockTransactionRepository, *MockRedemptionCodeRepository, *MockTransactionManager)
		expectedStatus codes.Code
		checkResponse  func(*testing.T, *pb.RedeemCodeResponse)
	}{
		{
			name: "正常系: コード引き換え成功",
			req: &pb.RedeemCodeRequest{
				Code:   "TESTCODE123",
				UserId: "user123",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtm *MockTransactionManager) {
				now := time.Now()
				code := redemption_code.NewRedemptionCode(
					"TESTCODE123",
					redemption_code.CodeTypePromotion,
					currency.CurrencyTypePaid,
					100,
					1,
					now.Add(-24*time.Hour),
					now.Add(24*time.Hour),
					map[string]interface{}{},
				)
				paidCurrency := currency.NewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				mrcr.On("FindByCode", mock.Anything, "TESTCODE123").Return(code, nil)
				mrcr.On("HasUserRedeemed", mock.Anything, "TESTCODE123", "user123").Return(false, nil)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil)
				mcr.On("Save", mock.Anything, mock.AnythingOfType("*currency.Currency")).Return(nil)
				mrcr.On("Update", mock.Anything, mock.AnythingOfType("*redemption_code.RedemptionCode")).Return(nil)
				mrcr.On("SaveRedemption", mock.Anything, mock.AnythingOfType("*redemption_code.CodeRedemption")).Return(nil)
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil)
				mtm.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(nil)
			},
			expectedStatus: codes.OK,
			checkResponse: func(t *testing.T, resp *pb.RedeemCodeResponse) {
				assert.NotEmpty(t, resp.RedemptionId)
				assert.NotEmpty(t, resp.TransactionId)
				assert.Equal(t, "TESTCODE123", resp.Code)
				assert.Equal(t, "paid", resp.CurrencyType)
				assert.Equal(t, "100", resp.Amount)
				assert.NotEmpty(t, resp.BalanceAfter)
				assert.Equal(t, "completed", resp.Status)
			},
		},
		{
			name: "異常系: codeが空",
			req: &pb.RedeemCodeRequest{
				Code:   "",
				UserId: "user123",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtm *MockTransactionManager) {
			},
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "異常系: user_idが空",
			req: &pb.RedeemCodeRequest{
				Code:   "TESTCODE123",
				UserId: "",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtm *MockTransactionManager) {
			},
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "異常系: コードが見つからない",
			req: &pb.RedeemCodeRequest{
				Code:   "INVALIDCODE",
				UserId: "user123",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtm *MockTransactionManager) {
				mrcr.On("FindByCode", mock.Anything, "INVALIDCODE").Return(nil, redemption_code.ErrCodeNotFound)
			},
			expectedStatus: codes.NotFound,
		},
		{
			name: "異常系: コードが既に使用済み",
			req: &pb.RedeemCodeRequest{
				Code:   "USEDCODE",
				UserId: "user123",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtm *MockTransactionManager) {
				now := time.Now()
				code := redemption_code.NewRedemptionCode(
					"USEDCODE",
					redemption_code.CodeTypePromotion,
					currency.CurrencyTypePaid,
					100,
					1,
					now.Add(-24*time.Hour),
					now.Add(24*time.Hour),
					map[string]interface{}{},
				)
				code.SetCurrentUses(1)
				mrcr.On("FindByCode", mock.Anything, "USEDCODE").Return(code, nil)
			},
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "異常系: ユーザーが既に引き換え済み",
			req: &pb.RedeemCodeRequest{
				Code:   "TESTCODE123",
				UserId: "user123",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtm *MockTransactionManager) {
				now := time.Now()
				code := redemption_code.NewRedemptionCode(
					"TESTCODE123",
					redemption_code.CodeTypePromotion,
					currency.CurrencyTypePaid,
					100,
					10,
					now.Add(-24*time.Hour),
					now.Add(24*time.Hour),
					map[string]interface{}{},
				)
				mrcr.On("FindByCode", mock.Anything, "TESTCODE123").Return(code, nil)
				mrcr.On("HasUserRedeemed", mock.Anything, "TESTCODE123", "user123").Return(true, nil)
			},
			expectedStatus: codes.AlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockCurrencyRepo, mockTransactionRepo, _, mockTxManager, mockRedemptionCodeRepo := setupTestHandler(t)

			tt.setupMock(mockCurrencyRepo, mockTransactionRepo, mockRedemptionCodeRepo, mockTxManager)

			ctx := context.Background()
			resp, err := handler.RedeemCode(ctx, tt.req)

			if tt.expectedStatus == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, resp)
				if tt.checkResponse != nil {
					tt.checkResponse(t, resp)
				}
			} else {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.expectedStatus, st.Code())
			}

			mockRedemptionCodeRepo.AssertExpectations(t)
			mockCurrencyRepo.AssertExpectations(t)
			mockTransactionRepo.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	}
}

func TestCurrencyHandler_GetTransactionHistory(t *testing.T) {
	tests := []struct {
		name           string
		req            *pb.GetTransactionHistoryRequest
		setupMock      func(*MockTransactionRepository)
		expectedStatus codes.Code
		checkResponse  func(*testing.T, *pb.GetTransactionHistoryResponse)
	}{
		{
			name: "正常系: トランザクション履歴取得成功",
			req: &pb.GetTransactionHistoryRequest{
				UserId: "user123",
				Limit:  10,
				Offset: 0,
			},
			setupMock: func(mtr *MockTransactionRepository) {
				txns := []*transaction.Transaction{
					transaction.NewTransaction(
						"txn1",
						"user123",
						transaction.TransactionTypeGrant,
						currency.CurrencyTypePaid,
						100,
						1000,
						1100,
						transaction.TransactionStatusCompleted,
						map[string]interface{}{},
					),
					transaction.NewTransaction(
						"txn2",
						"user123",
						transaction.TransactionTypeConsume,
						currency.CurrencyTypePaid,
						50,
						1100,
						1050,
						transaction.TransactionStatusCompleted,
						map[string]interface{}{},
					),
				}
				mtr.On("FindByUserID", mock.Anything, "user123", 10, 0).Return(txns, nil)
			},
			expectedStatus: codes.OK,
			checkResponse: func(t *testing.T, resp *pb.GetTransactionHistoryResponse) {
				assert.Len(t, resp.Transactions, 2)
				assert.Equal(t, int32(2), resp.Total)
				assert.Equal(t, int32(10), resp.Limit)
				assert.Equal(t, int32(0), resp.Offset)
				assert.Equal(t, "txn1", resp.Transactions[0].TransactionId)
				assert.Equal(t, "grant", resp.Transactions[0].TransactionType)
			},
		},
		{
			name: "正常系: limitが0の場合はデフォルト値50を使用",
			req: &pb.GetTransactionHistoryRequest{
				UserId: "user123",
				Limit:  0,
				Offset: 0,
			},
			setupMock: func(mtr *MockTransactionRepository) {
				txns := []*transaction.Transaction{}
				mtr.On("FindByUserID", mock.Anything, "user123", 50, 0).Return(txns, nil)
			},
			expectedStatus: codes.OK,
			checkResponse: func(t *testing.T, resp *pb.GetTransactionHistoryResponse) {
				assert.Equal(t, int32(50), resp.Limit)
			},
		},
		{
			name: "正常系: limitが100を超える場合は最大値100を使用",
			req: &pb.GetTransactionHistoryRequest{
				UserId: "user123",
				Limit:  200,
				Offset: 0,
			},
			setupMock: func(mtr *MockTransactionRepository) {
				txns := []*transaction.Transaction{}
				mtr.On("FindByUserID", mock.Anything, "user123", 100, 0).Return(txns, nil)
			},
			expectedStatus: codes.OK,
			checkResponse: func(t *testing.T, resp *pb.GetTransactionHistoryResponse) {
				assert.Equal(t, int32(100), resp.Limit)
			},
		},
		{
			name: "正常系: offsetが負の値の場合は0を使用",
			req: &pb.GetTransactionHistoryRequest{
				UserId: "user123",
				Limit:  10,
				Offset: -10,
			},
			setupMock: func(mtr *MockTransactionRepository) {
				txns := []*transaction.Transaction{}
				mtr.On("FindByUserID", mock.Anything, "user123", 10, 0).Return(txns, nil)
			},
			expectedStatus: codes.OK,
			checkResponse: func(t *testing.T, resp *pb.GetTransactionHistoryResponse) {
				assert.Equal(t, int32(0), resp.Offset)
			},
		},
		{
			name: "正常系: currency_typeでフィルタリング",
			req: &pb.GetTransactionHistoryRequest{
				UserId:       "user123",
				Limit:        10,
				Offset:       0,
				CurrencyType: "paid",
			},
			setupMock: func(mtr *MockTransactionRepository) {
				txns := []*transaction.Transaction{}
				mtr.On("FindByUserID", mock.Anything, "user123", 10, 0).Return(txns, nil)
			},
			expectedStatus: codes.OK,
			checkResponse: func(t *testing.T, resp *pb.GetTransactionHistoryResponse) {
				assert.NotNil(t, resp)
			},
		},
		{
			name: "正常系: transaction_typeでフィルタリング",
			req: &pb.GetTransactionHistoryRequest{
				UserId:          "user123",
				Limit:           10,
				Offset:          0,
				TransactionType: "grant",
			},
			setupMock: func(mtr *MockTransactionRepository) {
				txns := []*transaction.Transaction{}
				mtr.On("FindByUserID", mock.Anything, "user123", 10, 0).Return(txns, nil)
			},
			expectedStatus: codes.OK,
			checkResponse: func(t *testing.T, resp *pb.GetTransactionHistoryResponse) {
				assert.NotNil(t, resp)
			},
		},
		{
			name: "異常系: user_idが空",
			req: &pb.GetTransactionHistoryRequest{
				UserId: "",
			},
			setupMock:      func(mtr *MockTransactionRepository) {},
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "異常系: トランザクション取得エラー",
			req: &pb.GetTransactionHistoryRequest{
				UserId: "user123",
				Limit:  10,
				Offset: 0,
			},
			setupMock: func(mtr *MockTransactionRepository) {
				mtr.On("FindByUserID", mock.Anything, "user123", 10, 0).Return(nil, transaction.ErrTransactionNotFound)
			},
			expectedStatus: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, _, mockTransactionRepo, _, _, _ := setupTestHandler(t)

			tt.setupMock(mockTransactionRepo)

			ctx := context.Background()
			resp, err := handler.GetTransactionHistory(ctx, tt.req)

			if tt.expectedStatus == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, resp)
				if tt.checkResponse != nil {
					tt.checkResponse(t, resp)
				}
			} else {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.expectedStatus, st.Code())
			}

			mockTransactionRepo.AssertExpectations(t)
		})
	}
}

func TestCurrencyHandler_handleError(t *testing.T) {
	handler, _, _, _, _, _ := setupTestHandler(t)

	tests := []struct {
		name         string
		err          error
		expectedCode codes.Code
		expectedMsg  string
	}{
		{
			name:         "currency.ErrInsufficientBalance -> FailedPrecondition",
			err:          currency.ErrInsufficientBalance,
			expectedCode: codes.FailedPrecondition,
		},
		{
			name:         "currency.ErrInvalidAmount -> InvalidArgument",
			err:          currency.ErrInvalidAmount,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "currency.ErrCurrencyNotFound -> NotFound",
			err:          currency.ErrCurrencyNotFound,
			expectedCode: codes.NotFound,
		},
		{
			name:         "transaction.ErrTransactionNotFound -> NotFound",
			err:          transaction.ErrTransactionNotFound,
			expectedCode: codes.NotFound,
		},
		{
			name:         "payment_request.ErrPaymentRequestNotFound -> NotFound",
			err:          payment_request.ErrPaymentRequestNotFound,
			expectedCode: codes.NotFound,
		},
		{
			name:         "payment_request.ErrPaymentRequestAlreadyProcessed -> FailedPrecondition",
			err:          payment_request.ErrPaymentRequestAlreadyProcessed,
			expectedCode: codes.FailedPrecondition,
		},
		{
			name:         "redemption_code.ErrCodeNotFound -> NotFound",
			err:          redemption_code.ErrCodeNotFound,
			expectedCode: codes.NotFound,
		},
		{
			name:         "redemption_code.ErrCodeNotRedeemable -> InvalidArgument",
			err:          redemption_code.ErrCodeNotRedeemable,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "redemption_code.ErrCodeAlreadyUsed -> AlreadyExists",
			err:          redemption_code.ErrCodeAlreadyUsed,
			expectedCode: codes.AlreadyExists,
		},
		{
			name:         "redemption_code.ErrUserAlreadyRedeemed -> AlreadyExists",
			err:          redemption_code.ErrUserAlreadyRedeemed,
			expectedCode: codes.AlreadyExists,
		},
		{
			name:         "gRPCステータスエラーはそのまま返す",
			err:          status.Error(codes.Unauthenticated, "unauthorized"),
			expectedCode: codes.Unauthenticated,
		},
		{
			name:         "予期しないエラー -> Internal",
			err:          errors.New("unexpected error"),
			expectedCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.handleError(tt.err)

			require.Error(t, result)
			st, ok := status.FromError(result)
			require.True(t, ok)
			assert.Equal(t, tt.expectedCode, st.Code())
		})
	}
}
