package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authapp "gem-server/internal/application/auth"
	"gem-server/internal/infrastructure/config"
	otelinfra "gem-server/internal/infrastructure/observability/otel"
	restmiddleware "gem-server/internal/presentation/rest/middleware"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestAuthHandler_GenerateToken(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name: "正常系: トークン生成成功",
			requestBody: map[string]interface{}{
				"user_id": "user123",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "異常系: 無効なリクエストボディ",
			requestBody:    nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "異常系: user_idが空",
			requestBody: map[string]interface{}{
				"user_id": "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "異常系: user_idが存在しない",
			requestBody: map[string]interface{}{
				"other_field": "value",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			cfg := &config.JWTConfig{
				Secret:     "test-secret",
				Expiration: 86400 * time.Second,
				Issuer:     "test",
			}
			tracer := noop.NewTracerProvider().Tracer("test")
			logger := otelinfra.NewLogger(tracer)

			// エラーハンドリングミドルウェアを設定
			e.Use(restmiddleware.ErrorHandlerMiddleware(logger))

			service := authapp.NewAuthApplicationService(cfg, logger)
			handler := NewAuthHandler(service)

			// ルーティングを設定
			e.POST("/auth/token", handler.GenerateToken)

			var body []byte
			if tt.requestBody != nil {
				body, _ = json.Marshal(tt.requestBody)
			}
			req := httptest.NewRequest(http.MethodPost, "/auth/token", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response["token"])
				assert.Equal(t, float64(86400), response["expires_in"])
				assert.Equal(t, "Bearer", response["token_type"])
			}
		})
	}
}
