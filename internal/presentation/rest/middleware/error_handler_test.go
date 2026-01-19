package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"gem-server/internal/domain/currency"
	"gem-server/internal/domain/payment_request"
	"gem-server/internal/domain/redemption_code"
	"gem-server/internal/domain/transaction"
	otelinfra "gem-server/internal/infrastructure/observability/otel"
)

func TestErrorHandlerMiddleware_NoError(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := ErrorHandlerMiddleware(logger)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestErrorHandlerMiddleware_InsufficientBalance(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := ErrorHandlerMiddleware(logger)
	handler := middleware(func(c echo.Context) error {
		return currency.ErrInsufficientBalance
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestErrorHandlerMiddleware_InvalidAmount(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := ErrorHandlerMiddleware(logger)
	handler := middleware(func(c echo.Context) error {
		return currency.ErrInvalidAmount
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestErrorHandlerMiddleware_CurrencyNotFound(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := ErrorHandlerMiddleware(logger)
	handler := middleware(func(c echo.Context) error {
		return currency.ErrCurrencyNotFound
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestErrorHandlerMiddleware_TransactionNotFound(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := ErrorHandlerMiddleware(logger)
	handler := middleware(func(c echo.Context) error {
		return transaction.ErrTransactionNotFound
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestErrorHandlerMiddleware_PaymentRequestNotFound(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := ErrorHandlerMiddleware(logger)
	handler := middleware(func(c echo.Context) error {
		return payment_request.ErrPaymentRequestNotFound
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestErrorHandlerMiddleware_PaymentRequestAlreadyProcessed(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := ErrorHandlerMiddleware(logger)
	handler := middleware(func(c echo.Context) error {
		return payment_request.ErrPaymentRequestAlreadyProcessed
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestErrorHandlerMiddleware_CodeNotFound(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := ErrorHandlerMiddleware(logger)
	handler := middleware(func(c echo.Context) error {
		return redemption_code.ErrCodeNotFound
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestErrorHandlerMiddleware_CodeNotRedeemable(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := ErrorHandlerMiddleware(logger)
	handler := middleware(func(c echo.Context) error {
		return redemption_code.ErrCodeNotRedeemable
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestErrorHandlerMiddleware_CodeAlreadyUsed(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := ErrorHandlerMiddleware(logger)
	handler := middleware(func(c echo.Context) error {
		return redemption_code.ErrCodeAlreadyUsed
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestErrorHandlerMiddleware_UserAlreadyRedeemed(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := ErrorHandlerMiddleware(logger)
	handler := middleware(func(c echo.Context) error {
		return redemption_code.ErrUserAlreadyRedeemed
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestErrorHandlerMiddleware_HTTPError(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := ErrorHandlerMiddleware(logger)
	handler := middleware(func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusBadRequest, "bad request")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestErrorHandlerMiddleware_HTTPErrorWithNonStringMessage(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := ErrorHandlerMiddleware(logger)
	handler := middleware(func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusBadRequest, 123) // 数値型のメッセージ
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestErrorHandlerMiddleware_UnknownError(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := ErrorHandlerMiddleware(logger)
	handler := middleware(func(c echo.Context) error {
		return errors.New("unknown error")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestErrorHandlerMiddleware_WrappedError(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := ErrorHandlerMiddleware(logger)
	handler := middleware(func(c echo.Context) error {
		return errors.Join(currency.ErrInsufficientBalance, errors.New("wrapped error"))
	})

	err := handler(c)
	require.NoError(t, err)
	// errors.Joinでラップされたエラーでも、errors.Isで判定できる
	assert.Equal(t, http.StatusConflict, rec.Code)
}
