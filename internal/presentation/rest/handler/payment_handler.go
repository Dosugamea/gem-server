package handler

import (
	"net/http"
	"strconv"

	paymentapp "gem-server/internal/application/payment"

	"github.com/labstack/echo/v4"
)

// ProcessPaymentRequest 決済処理リクエスト
// @Description 決済処理リクエスト
type ProcessPaymentRequest struct {
	PaymentRequestID string            `json:"payment_request_id" example:"req_123"`
	MethodName       string            `json:"method_name" example:"credit_card"`
	Details          map[string]string `json:"details"`
	Amount           string            `json:"amount" example:"1000"`
	Currency         string            `json:"currency" example:"JPY"`
}

// ProcessPaymentResponse 決済処理レスポンス
// @Description 決済処理レスポンス
type ProcessPaymentResponse struct {
	TransactionID      string              `json:"transaction_id" example:"txn_789"`
	PaymentRequestID   string              `json:"payment_request_id" example:"req_123"`
	ConsumptionDetails []ConsumptionDetail `json:"consumption_details"`
	TotalConsumed      string              `json:"total_consumed" example:"1000"`
	Status             string              `json:"status" example:"completed"`
}

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
// @Summary 決済を処理
// @Description 決済リクエストを処理し、通貨を消費します
// @Tags payment
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body ProcessPaymentRequest true "決済処理リクエスト"
// @Success 200 {object} ProcessPaymentResponse "決済処理成功"
// @Failure 400 {object} ErrorResponse "不正なリクエスト"
// @Failure 403 {object} ErrorResponse "認証エラー"
// @Failure 409 {object} ErrorResponse "残高不足"
// @Router /payment/process [post]
func (h *PaymentHandler) ProcessPayment(c echo.Context) error {
	// トークンからuser_idを取得
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "user_id not found in token")
	}

	var reqBody struct {
		PaymentRequestID string            `json:"payment_request_id"`
		MethodName       string            `json:"method_name"`
		Details          map[string]string `json:"details"`
		Amount           string            `json:"amount"`
		Currency         string            `json:"currency"`
	}

	if err := c.Bind(&reqBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// 金額をint64に変換
	amount, err := strconv.ParseInt(reqBody.Amount, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid amount format")
	}

	req := &paymentapp.ProcessPaymentRequest{
		PaymentRequestID: reqBody.PaymentRequestID,
		UserID:           userID,
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
	details := make([]ConsumptionDetail, len(resp.ConsumptionDetails))
	for i, detail := range resp.ConsumptionDetails {
		details[i] = ConsumptionDetail{
			CurrencyType:  detail.CurrencyType,
			Amount:        strconv.FormatInt(detail.Amount, 10),
			BalanceBefore: strconv.FormatInt(detail.BalanceBefore, 10),
			BalanceAfter:  strconv.FormatInt(detail.BalanceAfter, 10),
		}
	}

	return c.JSON(http.StatusOK, ProcessPaymentResponse{
		TransactionID:      resp.TransactionID,
		PaymentRequestID:   resp.PaymentRequestID,
		ConsumptionDetails: details,
		TotalConsumed:      strconv.FormatInt(resp.TotalConsumed, 10),
		Status:             resp.Status,
	})
}
