package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	currencyapp "gem-server/internal/application/currency"
	"gem-server/internal/domain/currency"
	"gem-server/internal/domain/service"
	otelinfra "gem-server/internal/infrastructure/observability/otel"
	restmiddleware "gem-server/internal/presentation/rest/middleware"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestCurrencyHandler_GetBalance(t *testing.T) {
	tests := []struct {
		name           string
		tokenUserID    string
		setupMock      func(*MockCurrencyRepository)
		expectedStatus int
	}{
		{
			name:        "正常系: 残高取得成功",
			tokenUserID: "user123",
			setupMock: func(mcr *MockCurrencyRepository) {
				paidCurrency := currency.MustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				freeCurrency := currency.MustNewCurrency("user123", currency.CurrencyTypeFree, 500, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "異常系: user_idがトークンにない",
			tokenUserID:    "",
			setupMock:      func(mcr *MockCurrencyRepository) {},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			mockCurrencyRepo := new(MockCurrencyRepository)
			mockTransactionRepo := new(MockTransactionRepository)
			mockTxManager := new(MockTransactionManager)
			tracer := noop.NewTracerProvider().Tracer("test")
			logger := otelinfra.NewLogger(tracer)
			metrics, _ := otelinfra.NewMetrics("test")
			currencyService := service.NewCurrencyService(mockCurrencyRepo)

			// エラーハンドリングミドルウェアを設定
			e.Use(restmiddleware.ErrorHandlerMiddleware(logger))

			tt.setupMock(mockCurrencyRepo)

			appService := currencyapp.NewCurrencyApplicationService(
				mockCurrencyRepo,
				mockTransactionRepo,
				mockTxManager,
				currencyService,
				logger,
				metrics,
			)

			handler := NewCurrencyHandler(appService)
			// ルーティングを設定（ユーザーAPI）
			e.GET("/me/balance", handler.GetBalance)

			req := httptest.NewRequest(http.MethodGet, "/me/balance", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			if tt.tokenUserID != "" {
				c.Set("user_id", tt.tokenUserID)
			}

			// ミドルウェアを手動で実行
			middlewareFunc := restmiddleware.ErrorHandlerMiddleware(logger)
			handlerFunc := middlewareFunc(func(c echo.Context) error {
				return handler.GetBalance(c)
			})
			err := handlerFunc(c)
			if err != nil {
				e.HTTPErrorHandler(err, c)
			}
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "user123", response["user_id"])
			}
		})
	}
}

func TestCurrencyHandler_GetBalanceAdmin(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		setupMock      func(*MockCurrencyRepository)
		expectedStatus int
	}{
		{
			name:   "正常系: 残高取得成功",
			userID: "user123",
			setupMock: func(mcr *MockCurrencyRepository) {
				paidCurrency := currency.MustNewCurrency("user123", currency.CurrencyTypePaid, 1000, 1)
				freeCurrency := currency.MustNewCurrency("user123", currency.CurrencyTypeFree, 500, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypeFree).Return(freeCurrency, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "異常系: user_idが空",
			userID:         "",
			setupMock:      func(mcr *MockCurrencyRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			mockCurrencyRepo := new(MockCurrencyRepository)
			mockTransactionRepo := new(MockTransactionRepository)
			mockTxManager := new(MockTransactionManager)
			tracer := noop.NewTracerProvider().Tracer("test")
			logger := otelinfra.NewLogger(tracer)
			metrics, _ := otelinfra.NewMetrics("test")
			currencyService := service.NewCurrencyService(mockCurrencyRepo)

			// エラーハンドリングミドルウェアを設定
			e.Use(restmiddleware.ErrorHandlerMiddleware(logger))

			tt.setupMock(mockCurrencyRepo)

			appService := currencyapp.NewCurrencyApplicationService(
				mockCurrencyRepo,
				mockTransactionRepo,
				mockTxManager,
				currencyService,
				logger,
				metrics,
			)

			handler := NewCurrencyHandler(appService)
			// ルーティングを設定（管理API）
			e.GET("/admin/users/:user_id/balance", handler.GetBalanceAdmin)

			req := httptest.NewRequest(http.MethodGet, "/admin/users/"+tt.userID+"/balance", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("user_id")
			c.SetParamValues(tt.userID)

			// ミドルウェアを手動で実行
			middlewareFunc := restmiddleware.ErrorHandlerMiddleware(logger)
			handlerFunc := middlewareFunc(func(c echo.Context) error {
				return handler.GetBalanceAdmin(c)
			})
			err := handlerFunc(c)
			if err != nil {
				e.HTTPErrorHandler(err, c)
			}
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "user123", response["user_id"])
			}
		})
	}
}

func TestCurrencyHandler_GrantCurrency(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name:           "異常系: user_idが空",
			userID:         "",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "異常系: 無効なリクエストボディ",
			userID:         "user123",
			requestBody:    nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "異常系: 無効な金額フォーマット",
			userID: "user123",
			requestBody: map[string]interface{}{
				"currency_type": "paid",
				"amount":        "invalid",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			mockCurrencyRepo := new(MockCurrencyRepository)
			mockTransactionRepo := new(MockTransactionRepository)
			mockTxManager := new(MockTransactionManager)
			tracer := noop.NewTracerProvider().Tracer("test")
			logger := otelinfra.NewLogger(tracer)
			metrics, _ := otelinfra.NewMetrics("test")
			currencyService := service.NewCurrencyService(mockCurrencyRepo)

			// エラーハンドリングミドルウェアを設定
			e.Use(restmiddleware.ErrorHandlerMiddleware(logger))

			appService := currencyapp.NewCurrencyApplicationService(
				mockCurrencyRepo,
				mockTransactionRepo,
				mockTxManager,
				currencyService,
				logger,
				metrics,
			)

			handler := NewCurrencyHandler(appService)

			var body []byte
			if tt.requestBody != nil {
				body, _ = json.Marshal(tt.requestBody)
			}
			req := httptest.NewRequest(http.MethodPost, "/admin/users/"+tt.userID+"/grant", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("user_id")
			c.SetParamValues(tt.userID)

			// ミドルウェアを手動で実行
			middlewareFunc := restmiddleware.ErrorHandlerMiddleware(logger)
			handlerFunc := middlewareFunc(func(c echo.Context) error {
				return handler.GrantCurrency(c)
			})
			err := handlerFunc(c)
			if err != nil {
				e.HTTPErrorHandler(err, c)
			}
			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func TestCurrencyHandler_ConsumeCurrency(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name:           "異常系: user_idが空",
			userID:         "",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "異常系: 無効な金額フォーマット",
			userID: "user123",
			requestBody: map[string]interface{}{
				"currency_type": "paid",
				"amount":        "invalid",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			mockCurrencyRepo := new(MockCurrencyRepository)
			mockTransactionRepo := new(MockTransactionRepository)
			mockTxManager := new(MockTransactionManager)
			tracer := noop.NewTracerProvider().Tracer("test")
			logger := otelinfra.NewLogger(tracer)
			metrics, _ := otelinfra.NewMetrics("test")
			currencyService := service.NewCurrencyService(mockCurrencyRepo)

			// エラーハンドリングミドルウェアを設定
			e.Use(restmiddleware.ErrorHandlerMiddleware(logger))

			appService := currencyapp.NewCurrencyApplicationService(
				mockCurrencyRepo,
				mockTransactionRepo,
				mockTxManager,
				currencyService,
				logger,
				metrics,
			)

			handler := NewCurrencyHandler(appService)

			var body []byte
			if tt.requestBody != nil {
				body, _ = json.Marshal(tt.requestBody)
			}
			req := httptest.NewRequest(http.MethodPost, "/admin/users/"+tt.userID+"/consume", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("user_id")
			c.SetParamValues(tt.userID)

			// ミドルウェアを手動で実行
			middlewareFunc := restmiddleware.ErrorHandlerMiddleware(logger)
			handlerFunc := middlewareFunc(func(c echo.Context) error {
				return handler.ConsumeCurrency(c)
			})
			err := handlerFunc(c)
			if err != nil {
				e.HTTPErrorHandler(err, c)
			}
			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}
