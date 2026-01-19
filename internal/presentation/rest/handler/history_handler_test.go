package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	historyapp "gem-server/internal/application/history"
	"gem-server/internal/domain/currency"
	"gem-server/internal/domain/transaction"
	otelinfra "gem-server/internal/infrastructure/observability/otel"
	restmiddleware "gem-server/internal/presentation/rest/middleware"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestHistoryHandler_GetTransactionHistory(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		tokenUserID    string
		queryParams    map[string]string
		setupMock      func(*MockTransactionRepository)
		expectedStatus int
		validateResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:        "正常系: 履歴取得成功",
			userID:      "user123",
			tokenUserID: "user123",
			queryParams: map[string]string{},
			setupMock: func(mtr *MockTransactionRepository) {
				txns := []*transaction.Transaction{
					transaction.NewTransaction(
						"txn1",
						"user123",
						transaction.TransactionTypeGrant,
						currency.CurrencyTypePaid,
						1000,
						0,
						1000,
						transaction.TransactionStatusCompleted,
						map[string]interface{}{},
					),
					transaction.NewTransaction(
						"txn2",
						"user123",
						transaction.TransactionTypeConsume,
						currency.CurrencyTypePaid,
						200,
						1000,
						800,
						transaction.TransactionStatusCompleted,
						map[string]interface{}{},
					),
				}
				mtr.On("FindByUserID", mock.Anything, "user123", 50, 0).Return(txns, nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response, "transactions")
				assert.Contains(t, response, "total")
				assert.Contains(t, response, "limit")
				assert.Contains(t, response, "offset")
				transactions := response["transactions"].([]interface{})
				assert.Greater(t, len(transactions), 0)
			},
		},
		{
			name:        "正常系: limitとoffsetを指定",
			userID:      "user123",
			tokenUserID: "user123",
			queryParams: map[string]string{
				"limit":  "10",
				"offset": "5",
			},
			setupMock: func(mtr *MockTransactionRepository) {
				txns := []*transaction.Transaction{}
				mtr.On("FindByUserID", mock.Anything, "user123", 10, 5).Return(txns, nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, float64(10), response["limit"])
				assert.Equal(t, float64(5), response["offset"])
			},
		},
		{
			name:        "正常系: currency_typeでフィルタ",
			userID:      "user123",
			tokenUserID: "user123",
			queryParams: map[string]string{
				"currency_type": "paid",
			},
			setupMock: func(mtr *MockTransactionRepository) {
				txns := []*transaction.Transaction{
					transaction.NewTransaction(
						"txn1",
						"user123",
						transaction.TransactionTypeGrant,
						currency.CurrencyTypePaid,
						1000,
						0,
						1000,
						transaction.TransactionStatusCompleted,
						map[string]interface{}{},
					),
				}
				mtr.On("FindByUserID", mock.Anything, "user123", 50, 0).Return(txns, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "正常系: transaction_typeでフィルタ",
			userID:      "user123",
			tokenUserID: "user123",
			queryParams: map[string]string{
				"transaction_type": "grant",
			},
			setupMock: func(mtr *MockTransactionRepository) {
				txns := []*transaction.Transaction{
					transaction.NewTransaction(
						"txn1",
						"user123",
						transaction.TransactionTypeGrant,
						currency.CurrencyTypePaid,
						1000,
						0,
						1000,
						transaction.TransactionStatusCompleted,
						map[string]interface{}{},
					),
				}
				mtr.On("FindByUserID", mock.Anything, "user123", 50, 0).Return(txns, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "異常系: user_idが空",
			userID:         "",
			tokenUserID:    "user123",
			queryParams:    map[string]string{},
			setupMock:      func(mtr *MockTransactionRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "異常系: user_id不一致",
			userID:         "user123",
			tokenUserID:    "user456",
			queryParams:    map[string]string{},
			setupMock:      func(mtr *MockTransactionRepository) {},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:        "異常系: 無効なlimit（負の値）",
			userID:      "user123",
			tokenUserID: "user123",
			queryParams: map[string]string{
				"limit": "-1",
			},
			setupMock:      func(mtr *MockTransactionRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "異常系: 無効なlimit（100を超える）",
			userID:      "user123",
			tokenUserID: "user123",
			queryParams: map[string]string{
				"limit": "101",
			},
			setupMock:      func(mtr *MockTransactionRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "異常系: 無効なlimit（文字列）",
			userID:      "user123",
			tokenUserID: "user123",
			queryParams: map[string]string{
				"limit": "invalid",
			},
			setupMock:      func(mtr *MockTransactionRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "異常系: 無効なoffset（負の値）",
			userID:      "user123",
			tokenUserID: "user123",
			queryParams: map[string]string{
				"offset": "-1",
			},
			setupMock:      func(mtr *MockTransactionRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "異常系: 無効なoffset（文字列）",
			userID:      "user123",
			tokenUserID: "user123",
			queryParams: map[string]string{
				"offset": "invalid",
			},
			setupMock:      func(mtr *MockTransactionRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			mockTransactionRepo := new(MockTransactionRepository)
			tracer := noop.NewTracerProvider().Tracer("test")
			logger := otelinfra.NewLogger(tracer)
			metrics, _ := otelinfra.NewMetrics("test")

			// エラーハンドリングミドルウェアを設定
			e.Use(restmiddleware.ErrorHandlerMiddleware(logger))

			tt.setupMock(mockTransactionRepo)

			appService := historyapp.NewHistoryApplicationService(
				mockTransactionRepo,
				logger,
				metrics,
			)

			handler := NewHistoryHandler(appService)

			// URLにクエリパラメータを追加
			url := "/api/v1/users/" + tt.userID + "/transactions"
			req := httptest.NewRequest(http.MethodGet, url, nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("user_id")
			c.SetParamValues(tt.userID)
			if tt.tokenUserID != "" {
				c.Set("user_id", tt.tokenUserID)
			}

			// ミドルウェアを手動で実行
			middlewareFunc := restmiddleware.ErrorHandlerMiddleware(logger)
			handlerFunc := middlewareFunc(func(c echo.Context) error {
				return handler.GetTransactionHistory(c)
			})
			err := handlerFunc(c)
			if err != nil {
				e.HTTPErrorHandler(err, c)
			}
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.validateResponse != nil {
				tt.validateResponse(t, rec)
			}
		})
	}
}
