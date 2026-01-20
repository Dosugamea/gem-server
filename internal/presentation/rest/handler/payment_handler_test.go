package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	paymentapp "gem-server/internal/application/payment"
	otelinfra "gem-server/internal/infrastructure/observability/otel"
	restmiddleware "gem-server/internal/presentation/rest/middleware"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestPaymentHandler_ProcessPayment(t *testing.T) {
	tests := []struct {
		name           string
		tokenUserID    string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name:           "異常系: user_idがトークンにない",
			tokenUserID:    "",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "異常系: 無効なリクエストボディ",
			tokenUserID:    "user123",
			requestBody:    nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "異常系: 無効な金額フォーマット",
			tokenUserID: "user123",
			requestBody: map[string]interface{}{
				"payment_request_id": "pr123",
				"method_name":        "test",
				"amount":             "invalid",
				"currency":           "JPY",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			mockCurrencyRepo := new(MockCurrencyRepository)
			mockTransactionRepo := new(MockTransactionRepository)
			mockPaymentRequestRepo := new(MockPaymentRequestRepository)
			mockTxManager := new(MockTransactionManager)
			tracer := noop.NewTracerProvider().Tracer("test")
			logger := otelinfra.NewLogger(tracer)
			metrics, _ := otelinfra.NewMetrics("test")

			// エラーハンドリングミドルウェアを設定
			e.Use(restmiddleware.ErrorHandlerMiddleware(logger))

			appService := paymentapp.NewPaymentApplicationService(
				mockCurrencyRepo,
				mockTransactionRepo,
				mockPaymentRequestRepo,
				mockTxManager,
				logger,
				metrics,
			)

			handler := NewPaymentHandler(appService)

			var body []byte
			if tt.requestBody != nil {
				body, _ = json.Marshal(tt.requestBody)
			}
			req := httptest.NewRequest(http.MethodPost, "/payment/process", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			if tt.tokenUserID != "" {
				c.Set("user_id", tt.tokenUserID)
			}

			// ミドルウェアを手動で実行
			middlewareFunc := restmiddleware.ErrorHandlerMiddleware(logger)
			handlerFunc := middlewareFunc(func(c echo.Context) error {
				return handler.ProcessPayment(c)
			})
			err := handlerFunc(c)
			if err != nil {
				e.HTTPErrorHandler(err, c)
			}
			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}
