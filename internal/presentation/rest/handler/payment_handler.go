package handler

import (
	"net/http"
	"strconv"

	paymentapp "gem-server/internal/application/payment"

	"github.com/labstack/echo/v4"
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

	var reqBody ProcessPaymentRequest
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
