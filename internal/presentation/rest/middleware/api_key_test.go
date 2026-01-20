package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gem-server/internal/infrastructure/config"
	otelinfra "gem-server/internal/infrastructure/observability/otel"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestAPIKeyMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		apiKey         string
		clientIP       string
		config         *config.AdminAPIConfig
		expectedStatus int
	}{
		{
			name:   "正常系: 有効なAPIキー",
			apiKey: "test-api-key",
			config: &config.AdminAPIConfig{
				Enabled: true,
				APIKey:  "test-api-key",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "異常系: APIキーが空",
			apiKey: "",
			config: &config.AdminAPIConfig{
				Enabled: true,
				APIKey:  "test-api-key",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "異常系: 無効なAPIキー",
			apiKey: "invalid-key",
			config: &config.AdminAPIConfig{
				Enabled: true,
				APIKey:  "test-api-key",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "異常系: 管理APIが無効化されている",
			apiKey: "test-api-key",
			config: &config.AdminAPIConfig{
				Enabled: false,
				APIKey:  "test-api-key",
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:     "正常系: IP制限あり（許可されたIP）",
			apiKey:   "test-api-key",
			clientIP: "127.0.0.1",
			config: &config.AdminAPIConfig{
				Enabled:    true,
				APIKey:     "test-api-key",
				AllowedIPs: []string{"127.0.0.1", "192.0.2.1"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "異常系: IP制限あり（許可されていないIP）",
			apiKey:   "test-api-key",
			clientIP: "192.168.1.1",
			config: &config.AdminAPIConfig{
				Enabled:    true,
				APIKey:     "test-api-key",
				AllowedIPs: []string{"10.0.0.1"},
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			tracer := noop.NewTracerProvider().Tracer("test")
			logger := otelinfra.NewLogger(tracer)

			middlewareFunc := APIKeyMiddleware(tt.config, logger)
			handler := middlewareFunc(func(c echo.Context) error {
				return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}
			if tt.clientIP != "" {
				req.Header.Set("X-Real-IP", tt.clientIP)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler(c)
			if err != nil {
				e.HTTPErrorHandler(err, c)
			}

			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}
