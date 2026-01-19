package handler

import (
	"context"
	"database/sql"
	"testing"

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
			name: "異常系: payment_request_idが空",
			req: &pb.ProcessPaymentRequest{
				PaymentRequestId: "",
				UserId:           "user123",
				Amount:           "100",
			},
			setupMock:      func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mpr *MockPaymentRequestRepository, mtm *MockTransactionManager) {},
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "異常系: user_idが空",
			req: &pb.ProcessPaymentRequest{
				PaymentRequestId: "pr123",
				UserId:           "",
				Amount:           "100",
			},
			setupMock:      func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mpr *MockPaymentRequestRepository, mtm *MockTransactionManager) {},
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "異常系: amountが空",
			req: &pb.ProcessPaymentRequest{
				PaymentRequestId: "pr123",
				UserId:           "user123",
				Amount:           "",
			},
			setupMock:      func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mpr *MockPaymentRequestRepository, mtm *MockTransactionManager) {},
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "異常系: 無効な金額フォーマット",
			req: &pb.ProcessPaymentRequest{
				PaymentRequestId: "pr123",
				UserId:           "user123",
				Amount:           "invalid",
			},
			setupMock:      func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mpr *MockPaymentRequestRepository, mtm *MockTransactionManager) {},
			expectedStatus: codes.InvalidArgument,
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
			name: "異常系: codeが空",
			req: &pb.RedeemCodeRequest{
				Code:   "",
				UserId: "user123",
			},
			setupMock:      func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtm *MockTransactionManager) {},
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "異常系: user_idが空",
			req: &pb.RedeemCodeRequest{
				Code:   "TESTCODE123",
				UserId: "",
			},
			setupMock:      func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtm *MockTransactionManager) {},
			expectedStatus: codes.InvalidArgument,
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
			name: "異常系: user_idが空",
			req: &pb.GetTransactionHistoryRequest{
				UserId: "",
			},
			setupMock:      func(mtr *MockTransactionRepository) {},
			expectedStatus: codes.InvalidArgument,
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
