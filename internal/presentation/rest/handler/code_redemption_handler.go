package handler

import (
	"net/http"
	"strconv"

	redemptionapp "gem-server/internal/application/code_redemption"

	"github.com/labstack/echo/v4"
)

// CodeRedemptionHandler コード引き換え関連ハンドラー
type CodeRedemptionHandler struct {
	redemptionService *redemptionapp.CodeRedemptionApplicationService
}

// NewCodeRedemptionHandler 新しいCodeRedemptionHandlerを作成
func NewCodeRedemptionHandler(redemptionService *redemptionapp.CodeRedemptionApplicationService) *CodeRedemptionHandler {
	return &CodeRedemptionHandler{
		redemptionService: redemptionService,
	}
}

// RedeemCode コード引き換えハンドラー
// POST /api/v1/codes/redeem
func (h *CodeRedemptionHandler) RedeemCode(c echo.Context) error {
	var reqBody struct {
		Code   string `json:"code"`
		UserID string `json:"user_id"`
	}

	if err := c.Bind(&reqBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// トークンのuser_idとリクエストのuser_idが一致するか確認
	tokenUserID, ok := c.Get("user_id").(string)
	if !ok || tokenUserID != reqBody.UserID {
		return echo.NewHTTPError(http.StatusForbidden, "user_id mismatch")
	}

	req := &redemptionapp.RedeemCodeRequest{
		Code:   reqBody.Code,
		UserID: reqBody.UserID,
	}

	resp, err := h.redemptionService.Redeem(c.Request().Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"redemption_id":  resp.RedemptionID,
		"transaction_id": resp.TransactionID,
		"code":           resp.Code,
		"currency_type":  resp.CurrencyType,
		"amount":         strconv.FormatInt(resp.Amount, 10),
		"balance_after":  strconv.FormatInt(resp.BalanceAfter, 10),
		"status":         resp.Status,
	})
}
