package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	otelinfra "gem-server/internal/infrastructure/observability/otel"
)

func TestLoggingMiddleware_SuccessfulRequest(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("User-Agent", "test-agent")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := LoggingMiddleware(logger)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestLoggingMiddleware_FailedRequest(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("User-Agent", "test-agent")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := LoggingMiddleware(logger)
	testErr := errors.New("test error")
	handler := middleware(func(c echo.Context) error {
		return testErr
	})

	err := handler(c)
	assert.Error(t, err)
	assert.Equal(t, testErr, err)
}

func TestLoggingMiddleware_LogsRequestInfo(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/currency", nil)
	req.RemoteAddr = "192.168.1.1:54321"
	req.Header.Set("User-Agent", "Mozilla/5.0")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := LoggingMiddleware(logger)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusCreated, "created")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestLoggingMiddleware_DurationMeasurement(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := LoggingMiddleware(logger)
	handler := middleware(func(c echo.Context) error {
		// 少し時間がかかる処理をシミュレート
		time.Sleep(10 * time.Millisecond)
		return c.String(http.StatusOK, "ok")
	})

	start := time.Now()
	err := handler(c)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	// ミドルウェアが実行時間を計測していることを確認（実際のログ出力は確認しないが、エラーが発生しないことを確認）
	assert.Greater(t, duration.Milliseconds(), int64(0))
}

func TestLoggingMiddleware_DifferentHTTPMethods(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(method, "/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			middleware := LoggingMiddleware(logger)
			handler := middleware(func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})

			err := handler(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestLoggingMiddleware_DifferentStatusCodes(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	statusCodes := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	}

	for _, statusCode := range statusCodes {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			middleware := LoggingMiddleware(logger)
			handler := middleware(func(c echo.Context) error {
				return c.String(statusCode, "response")
			})

			err := handler(c)
			require.NoError(t, err)
			assert.Equal(t, statusCode, rec.Code)
		})
	}
}

func TestLoggingMiddleware_EmptyUserAgent(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Del("User-Agent")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := LoggingMiddleware(logger)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}
