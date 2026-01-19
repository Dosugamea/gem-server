package history

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"

	"gem-server/internal/domain/currency"
	"gem-server/internal/domain/transaction"
	otelinfra "gem-server/internal/infrastructure/observability/otel"
)

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

func TestHistoryApplicationService_GetTransactionHistory(t *testing.T) {
	tests := []struct {
		name       string
		req        *GetTransactionHistoryRequest
		setupMocks func(*MockTransactionRepository)
		wantError  bool
		checkFunc  func(*testing.T, *GetTransactionHistoryResponse, error)
	}{
		{
			name: "正常系: 履歴を取得",
			req: &GetTransactionHistoryRequest{
				UserID: "user123",
				Limit:  10,
				Offset: 0,
			},
			setupMocks: func(mtr *MockTransactionRepository) {
				transactions := []*transaction.Transaction{
					transaction.NewTransaction(
						"txn1",
						"user123",
						transaction.TransactionTypeGrant,
						currency.CurrencyTypePaid,
						1000,
						0,
						1000,
						transaction.TransactionStatusCompleted,
						nil,
					),
					transaction.NewTransaction(
						"txn2",
						"user123",
						transaction.TransactionTypeConsume,
						currency.CurrencyTypePaid,
						500,
						1000,
						500,
						transaction.TransactionStatusCompleted,
						nil,
					),
				}
				mtr.On("FindByUserID", mock.Anything, "user123", 10, 0).Return(transactions, nil)
			},
			wantError: false,
			checkFunc: func(t *testing.T, resp *GetTransactionHistoryResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Transactions, 2)
				assert.Equal(t, 2, resp.Total)
				assert.Equal(t, 10, resp.Limit)
				assert.Equal(t, 0, resp.Offset)
			},
		},
		{
			name: "正常系: 通貨タイプでフィルタリング",
			req: &GetTransactionHistoryRequest{
				UserID:       "user123",
				Limit:        10,
				Offset:       0,
				CurrencyType: "paid",
			},
			setupMocks: func(mtr *MockTransactionRepository) {
				transactions := []*transaction.Transaction{
					transaction.NewTransaction(
						"txn1",
						"user123",
						transaction.TransactionTypeGrant,
						currency.CurrencyTypePaid,
						1000,
						0,
						1000,
						transaction.TransactionStatusCompleted,
						nil,
					),
					transaction.NewTransaction(
						"txn2",
						"user123",
						transaction.TransactionTypeGrant,
						currency.CurrencyTypeFree,
						500,
						0,
						500,
						transaction.TransactionStatusCompleted,
						nil,
					),
				}
				mtr.On("FindByUserID", mock.Anything, "user123", 10, 0).Return(transactions, nil)
			},
			wantError: false,
			checkFunc: func(t *testing.T, resp *GetTransactionHistoryResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Transactions, 1)
				assert.Equal(t, currency.CurrencyTypePaid, resp.Transactions[0].CurrencyType())
			},
		},
		{
			name: "正常系: トランザクションタイプでフィルタリング",
			req: &GetTransactionHistoryRequest{
				UserID:          "user123",
				Limit:           10,
				Offset:          0,
				TransactionType: "grant",
			},
			setupMocks: func(mtr *MockTransactionRepository) {
				transactions := []*transaction.Transaction{
					transaction.NewTransaction(
						"txn1",
						"user123",
						transaction.TransactionTypeGrant,
						currency.CurrencyTypePaid,
						1000,
						0,
						1000,
						transaction.TransactionStatusCompleted,
						nil,
					),
					transaction.NewTransaction(
						"txn2",
						"user123",
						transaction.TransactionTypeConsume,
						currency.CurrencyTypePaid,
						500,
						1000,
						500,
						transaction.TransactionStatusCompleted,
						nil,
					),
				}
				mtr.On("FindByUserID", mock.Anything, "user123", 10, 0).Return(transactions, nil)
			},
			wantError: false,
			checkFunc: func(t *testing.T, resp *GetTransactionHistoryResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Transactions, 1)
				assert.Equal(t, transaction.TransactionTypeGrant, resp.Transactions[0].TransactionType())
			},
		},
		{
			name: "正常系: デフォルト値の設定",
			req: &GetTransactionHistoryRequest{
				UserID: "user123",
				Limit:  0,  // デフォルト値に設定される
				Offset: -1, // 0に設定される
			},
			setupMocks: func(mtr *MockTransactionRepository) {
				mtr.On("FindByUserID", mock.Anything, "user123", 50, 0).Return([]*transaction.Transaction{}, nil)
			},
			wantError: false,
			checkFunc: func(t *testing.T, resp *GetTransactionHistoryResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, 50, resp.Limit)
				assert.Equal(t, 0, resp.Offset)
			},
		},
		{
			name: "正常系: 最大値の制限",
			req: &GetTransactionHistoryRequest{
				UserID: "user123",
				Limit:  200, // 100に制限される
				Offset: 0,
			},
			setupMocks: func(mtr *MockTransactionRepository) {
				mtr.On("FindByUserID", mock.Anything, "user123", 100, 0).Return([]*transaction.Transaction{}, nil)
			},
			wantError: false,
			checkFunc: func(t *testing.T, resp *GetTransactionHistoryResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, 100, resp.Limit)
			},
		},
		{
			name: "異常系: データベースエラー",
			req: &GetTransactionHistoryRequest{
				UserID: "user123",
				Limit:  10,
				Offset: 0,
			},
			setupMocks: func(mtr *MockTransactionRepository) {
				mtr.On("FindByUserID", mock.Anything, "user123", 10, 0).Return(nil, assert.AnError)
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransactionRepo := new(MockTransactionRepository)

			tt.setupMocks(mockTransactionRepo)

			tracer := otel.Tracer("test")
			logger := otelinfra.NewLogger(tracer)
			metrics, err := otelinfra.NewMetrics("test")
			require.NoError(t, err)

			svc := NewHistoryApplicationService(
				mockTransactionRepo,
				logger,
				metrics,
			)

			ctx := context.Background()
			got, err := svc.GetTransactionHistory(ctx, tt.req)

			if tt.wantError {
				assert.Error(t, err)
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
