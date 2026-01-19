package middleware

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	otelinfra "gem-server/internal/infrastructure/observability/otel"

	"gem-server/internal/domain/currency"
	"gem-server/internal/domain/payment_request"
	"gem-server/internal/domain/redemption_code"
	"gem-server/internal/domain/transaction"
)

// ErrorResponse エラーレスポンス
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// ErrorHandlerMiddleware エラーハンドリングミドルウェア
func ErrorHandlerMiddleware(logger *otelinfra.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			if err == nil {
				return nil
			}

			// エラーハンドリング
			return handleError(c, err, logger)
		}
	}
}

// handleError エラーを処理して適切なHTTPレスポンスを返す
func handleError(c echo.Context, err error, logger *otelinfra.Logger) error {
	ctx := c.Request().Context()

	// ドメインエラーの判定と処理
	if errors.Is(err, currency.ErrInsufficientBalance) {
		logger.Warn(ctx, "Insufficient balance", map[string]interface{}{
			"error": err.Error(),
		})
		return c.JSON(http.StatusConflict, ErrorResponse{
			Error:   "insufficient_balance",
			Message: err.Error(),
		})
	}

	if errors.Is(err, currency.ErrInvalidAmount) {
		logger.Warn(ctx, "Invalid amount", map[string]interface{}{
			"error": err.Error(),
		})
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_amount",
			Message: err.Error(),
		})
	}

	if errors.Is(err, currency.ErrCurrencyNotFound) {
		logger.Warn(ctx, "Currency not found", map[string]interface{}{
			"error": err.Error(),
		})
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "currency_not_found",
			Message: err.Error(),
		})
	}

	if errors.Is(err, transaction.ErrTransactionNotFound) {
		logger.Warn(ctx, "Transaction not found", map[string]interface{}{
			"error": err.Error(),
		})
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "transaction_not_found",
			Message: err.Error(),
		})
	}

	if errors.Is(err, payment_request.ErrPaymentRequestNotFound) {
		logger.Warn(ctx, "Payment request not found", map[string]interface{}{
			"error": err.Error(),
		})
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "payment_request_not_found",
			Message: err.Error(),
		})
	}

	if errors.Is(err, payment_request.ErrPaymentRequestAlreadyProcessed) {
		logger.Warn(ctx, "Payment request already processed", map[string]interface{}{
			"error": err.Error(),
		})
		return c.JSON(http.StatusConflict, ErrorResponse{
			Error:   "payment_request_already_processed",
			Message: err.Error(),
		})
	}

	if errors.Is(err, redemption_code.ErrCodeNotFound) {
		logger.Warn(ctx, "Code not found", map[string]interface{}{
			"error": err.Error(),
		})
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "code_not_found",
			Message: err.Error(),
		})
	}

	if errors.Is(err, redemption_code.ErrCodeNotRedeemable) {
		logger.Warn(ctx, "Code not redeemable", map[string]interface{}{
			"error": err.Error(),
		})
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "code_not_redeemable",
			Message: err.Error(),
		})
	}

	if errors.Is(err, redemption_code.ErrCodeAlreadyUsed) {
		logger.Warn(ctx, "Code already used", map[string]interface{}{
			"error": err.Error(),
		})
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "code_already_used",
			Message: err.Error(),
		})
	}

	if errors.Is(err, redemption_code.ErrUserAlreadyRedeemed) {
		logger.Warn(ctx, "User already redeemed", map[string]interface{}{
			"error": err.Error(),
		})
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "user_already_redeemed",
			Message: err.Error(),
		})
	}

	// EchoのHTTPエラー
	var httpErr *echo.HTTPError
	if errors.As(err, &httpErr) {
		logger.Warn(ctx, "HTTP error", map[string]interface{}{
			"status_code": httpErr.Code,
			"message":     httpErr.Message,
		})
		message := ""
		if msg, ok := httpErr.Message.(string); ok {
			message = msg
		} else {
			message = http.StatusText(httpErr.Code)
		}
		return c.JSON(httpErr.Code, ErrorResponse{
			Error:   http.StatusText(httpErr.Code),
			Message: message,
		})
	}

	// 予期しないエラー
	logger.Error(ctx, "Internal server error", err, map[string]interface{}{
		"path": c.Request().URL.Path,
	})
	return c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error:   "internal_server_error",
		Message: "An unexpected error occurred",
	})
}
