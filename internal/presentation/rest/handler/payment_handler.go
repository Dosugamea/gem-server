package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	paymentapp "gem-server/internal/application/payment"
)

// PaymentHandler 決済関連ハンドラー
type PaymentHandler struct {
	paymentService *paymentapp.PaymentApplicationService
}

// NewPaymentHandler 新しいPaymentHandlerを作成
func NewPaymentHandler(paymentService *paymentapp.PaymentApplicationService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
	}
}

// ProcessPayment 決済処理ハンドラー
// POST /api/v1/payment/process
func (h *PaymentHandler) ProcessPayment(c echo.Context) error {
	var reqBody struct {
		PaymentRequestID string            `json:"payment_request_id"`
		UserID           string            `json:"user_id"`
		MethodName       string            `json:"method_name"`
		Details          map[string]string `json:"details"`
		Amount           string            `json:"amount"`
		Currency         string            `json:"currency"`
	}

	if err := c.Bind(&reqBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// トークンのuser_idとリクエストのuser_idが一致するか確認
	tokenUserID, ok := c.Get("user_id").(string)
	if !ok || tokenUserID != reqBody.UserID {
		return echo.NewHTTPError(http.StatusForbidden, "user_id mismatch")
	}

	// 金額をint64に変換
	amount, err := strconv.ParseInt(reqBody.Amount, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid amount format")
	}

	req := &paymentapp.ProcessPaymentRequest{
		PaymentRequestID: reqBody.PaymentRequestID,
		UserID:           reqBody.UserID,
		MethodName:       reqBody.MethodName,
		Details:          reqBody.Details,
		Amount:           amount,
		Currency:         reqBody.Currency,
	}

	resp, err := h.paymentService.ProcessPayment(c.Request().Context(), req)
	if err != nil {
		return err
	}

	// レスポンスを構築
	details := make([]map[string]string, len(resp.ConsumptionDetails))
	for i, detail := range resp.ConsumptionDetails {
		details[i] = map[string]string{
			"currency_type":  detail.CurrencyType,
			"amount":          strconv.FormatInt(detail.Amount, 10),
			"balance_before":  strconv.FormatInt(detail.BalanceBefore, 10),
			"balance_after":   strconv.FormatInt(detail.BalanceAfter, 10),
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"transaction_id":      resp.TransactionID,
		"payment_request_id":   resp.PaymentRequestID,
		"consumption_details":  details,
		"total_consumed":       strconv.FormatInt(resp.TotalConsumed, 10),
		"status":               resp.Status,
	})
}
