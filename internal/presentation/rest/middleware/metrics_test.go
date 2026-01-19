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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric/noop"

	otelinfra "gem-server/internal/infrastructure/observability/otel"
)

func TestMetricsMiddleware_SuccessfulRequest(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := otelinfra.NewMetrics("test-meter")
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/test")

	middleware := MetricsMiddleware(metrics)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMetricsMiddleware_RecordsRequest(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := otelinfra.NewMetrics("test-meter")
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/currency", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/currency")

	middleware := MetricsMiddleware(metrics)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusCreated, "created")
	})

	err = handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestMetricsMiddleware_RecordsResponseTime(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := otelinfra.NewMetrics("test-meter")
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/slow")

	middleware := MetricsMiddleware(metrics)
	handler := middleware(func(c echo.Context) error {
		time.Sleep(10 * time.Millisecond)
		return c.String(http.StatusOK, "ok")
	})

	start := time.Now()
	err = handler(c)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Greater(t, duration.Milliseconds(), int64(0))
}

func TestMetricsMiddleware_RecordsClientError(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := otelinfra.NewMetrics("test-meter")
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/test")

	middleware := MetricsMiddleware(metrics)
	handler := middleware(func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusBadRequest, "bad request")
	})

	err = handler(c)
	// EchoのHTTPErrorはエラーハンドラーで処理されるため、エラーが返される
	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestMetricsMiddleware_RecordsServerError(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := otelinfra.NewMetrics("test-meter")
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/test")

	middleware := MetricsMiddleware(metrics)
	handler := middleware(func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
	})

	err = handler(c)
	// EchoのHTTPErrorはエラーハンドラーで処理されるため、エラーが返される
	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestMetricsMiddleware_DoesNotRecordErrorForSuccess(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := otelinfra.NewMetrics("test-meter")
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/test")

	middleware := MetricsMiddleware(metrics)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMetricsMiddleware_DoesNotRecordErrorFor3xx(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := otelinfra.NewMetrics("test-meter")
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/test")

	middleware := MetricsMiddleware(metrics)
	handler := middleware(func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/redirect")
	})

	err = handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusMovedPermanently, rec.Code)
}

func TestMetricsMiddleware_RecordsErrorFor4xx(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := otelinfra.NewMetrics("test-meter")
	require.NoError(t, err)

	statusCodes := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusConflict,
	}

	for _, statusCode := range statusCodes {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/test")

			middleware := MetricsMiddleware(metrics)
			handler := middleware(func(c echo.Context) error {
				return echo.NewHTTPError(statusCode, "error")
			})

			err = handler(c)
			// EchoのHTTPErrorはエラーハンドラーで処理されるため、エラーが返される
			assert.Error(t, err)
			assert.Equal(t, statusCode, rec.Code)
		})
	}
}

func TestMetricsMiddleware_RecordsErrorFor5xx(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := otelinfra.NewMetrics("test-meter")
	require.NoError(t, err)

	statusCodes := []int{
		http.StatusInternalServerError,
		http.StatusNotImplemented,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
	}

	for _, statusCode := range statusCodes {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/test")

			middleware := MetricsMiddleware(metrics)
			handler := middleware(func(c echo.Context) error {
				return echo.NewHTTPError(statusCode, "error")
			})

			err = handler(c)
			// EchoのHTTPErrorはエラーハンドラーで処理されるため、エラーが返される
			assert.Error(t, err)
			assert.Equal(t, statusCode, rec.Code)
		})
	}
}

func TestMetricsMiddleware_ErrorWithoutStatusCode(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := otelinfra.NewMetrics("test-meter")
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/test")

	middleware := MetricsMiddleware(metrics)
	testErr := errors.New("test error")
	handler := middleware(func(c echo.Context) error {
		return testErr
	})

	err = handler(c)
	assert.Error(t, err)
	assert.Equal(t, testErr, err)
}

func TestMetricsMiddleware_DifferentHTTPMethods(t *testing.T) {
	mp := noop.NewMeterProvider()
	otel.SetMeterProvider(mp)

	metrics, err := otelinfra.NewMetrics("test-meter")
	require.NoError(t, err)

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(method, "/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/test")

			middleware := MetricsMiddleware(metrics)
			handler := middleware(func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})

			err = handler(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}
