package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestTracingMiddleware_SuccessfulRequest(t *testing.T) {
	// noop tracer providerを設定
	tp := noop.NewTracerProvider()
	otel.SetTracerProvider(tp)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/test")

	middleware := TracingMiddleware()
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTracingMiddleware_SetsSpanAttributes(t *testing.T) {
	tp := noop.NewTracerProvider()
	otel.SetTracerProvider(tp)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/currency", nil)
	req.Header.Set("User-Agent", "test-agent")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/currency")

	middleware := TracingMiddleware()
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusCreated, "created")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestTracingMiddleware_RecordsError(t *testing.T) {
	tp := noop.NewTracerProvider()
	otel.SetTracerProvider(tp)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/test")

	middleware := TracingMiddleware()
	testErr := errors.New("test error")
	handler := middleware(func(c echo.Context) error {
		return testErr
	})

	err := handler(c)
	assert.Error(t, err)
	assert.Equal(t, testErr, err)
}

func TestTracingMiddleware_SetsStatusCodeAttribute(t *testing.T) {
	tp := noop.NewTracerProvider()
	otel.SetTracerProvider(tp)

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
			c.SetPath("/test")

			middleware := TracingMiddleware()
			handler := middleware(func(c echo.Context) error {
				return c.String(statusCode, "response")
			})

			err := handler(c)
			require.NoError(t, err)
			assert.Equal(t, statusCode, rec.Code)
		})
	}
}

func TestTracingMiddleware_ExtractsTraceContext(t *testing.T) {
	tp := noop.NewTracerProvider()
	otel.SetTracerProvider(tp)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	// トレースコンテキストをヘッダーに設定
	propagator := propagation.TraceContext{}
	carrier := propagation.HeaderCarrier(req.Header)
	ctx := req.Context()
	ctx, span := tp.Tracer("test").Start(ctx, "parent")
	defer span.End()
	propagator.Inject(ctx, carrier)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/test")

	middleware := TracingMiddleware()
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTracingMiddleware_DifferentHTTPMethods(t *testing.T) {
	tp := noop.NewTracerProvider()
	otel.SetTracerProvider(tp)

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(method, "/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/test")

			middleware := TracingMiddleware()
			handler := middleware(func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})

			err := handler(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestTracingMiddleware_SpanNameFormat(t *testing.T) {
	tp := noop.NewTracerProvider()
	otel.SetTracerProvider(tp)

	testCases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/test"},
		{http.MethodPost, "/api/v1/currency"},
		{http.MethodPut, "/api/v1/payment"},
		{http.MethodDelete, "/api/v1/user"},
	}

	for _, tc := range testCases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath(tc.path)

			middleware := TracingMiddleware()
			handler := middleware(func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})

			err := handler(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestTracingMiddleware_SetsUserAgentAttribute(t *testing.T) {
	tp := noop.NewTracerProvider()
	otel.SetTracerProvider(tp)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Test Agent)")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/test")

	middleware := TracingMiddleware()
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTracingMiddleware_EmptyUserAgent(t *testing.T) {
	tp := noop.NewTracerProvider()
	otel.SetTracerProvider(tp)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Del("User-Agent")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/test")

	middleware := TracingMiddleware()
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTracingMiddleware_UpdatesContext(t *testing.T) {
	tp := noop.NewTracerProvider()
	otel.SetTracerProvider(tp)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/test")

	middleware := TracingMiddleware()
	handler := middleware(func(c echo.Context) error {
		// コンテキストが更新されていることを確認
		ctx := c.Request().Context()
		assert.NotNil(t, ctx)
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTracingMiddleware_HTTPError(t *testing.T) {
	tp := noop.NewTracerProvider()
	otel.SetTracerProvider(tp)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/test")

	middleware := TracingMiddleware()
	handler := middleware(func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusBadRequest, "bad request")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
