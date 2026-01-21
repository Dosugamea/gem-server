package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gem-server/internal/domain/currency"
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

func TestCurrencyService_GetTotalBalance(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		setupMocks func(*MockCurrencyRepository)
		want       int64
		wantError  bool
	}{
		{
			name:   "正常系: 有償・無償通貨両方存在",
			userID: "user123",
			setupMocks: func(mcr *MockCurrencyRepository) {
				paidCurrency := currency.MustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				freeCurrency := currency.MustNewCurrency("user123", currency.CurrencyTypeFree, 500, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil)
			},
			want:      1500,
			wantError: false,
		},
		{
			name:   "異常系: 有償通貨取得エラー",
			userID: "user123",
			setupMocks: func(mcr *MockCurrencyRepository) {
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(nil, errors.New("database error"))
			},
			want:      0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockCurrencyRepository)
			tt.setupMocks(mockRepo)

			service := NewCurrencyService(mockRepo)
			ctx := context.Background()
			got, err := service.GetTotalBalance(ctx, tt.userID)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCurrencyService_HasSufficientBalance(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		amount     int64
		setupMocks func(*MockCurrencyRepository)
		want       bool
		wantError  bool
	}{
		{
			name:   "正常系: 無料通貨のみで支払える",
			userID: "user123",
			amount: 500,
			setupMocks: func(mcr *MockCurrencyRepository) {
				freeCurrency := currency.MustNewCurrency("user123", currency.CurrencyTypeFree, 1000, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil)
			},
			want:      true,
			wantError: false,
		},
		{
			name:   "正常系: 無料+有償通貨で支払える",
			userID: "user123",
			amount: 1500,
			setupMocks: func(mcr *MockCurrencyRepository) {
				freeCurrency := currency.MustNewCurrency("user123", currency.CurrencyTypeFree, 1000, 1)
				paidCurrency := currency.MustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil)
			},
			want:      true,
			wantError: false,
		},
		{
			name:   "異常系: 残高不足",
			userID: "user123",
			amount: 3000,
			setupMocks: func(mcr *MockCurrencyRepository) {
				freeCurrency := currency.MustNewCurrency("user123", currency.CurrencyTypeFree, 1000, 1)
				paidCurrency := currency.MustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil)
			},
			want:      false,
			wantError: false,
		},
		{
			name:   "異常系: 無料通貨取得エラー",
			userID: "user123",
			amount: 500,
			setupMocks: func(mcr *MockCurrencyRepository) {
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(nil, errors.New("database error"))
			},
			want:      false,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockCurrencyRepository)
			tt.setupMocks(mockRepo)

			service := NewCurrencyService(mockRepo)
			ctx := context.Background()
			got, err := service.HasSufficientBalance(ctx, tt.userID, tt.amount)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
