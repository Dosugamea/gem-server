package handler

import (
	"net/http"
	"strconv"

	redemptionapp "gem-server/internal/application/code_redemption"

	"github.com/labstack/echo/v4"
)

// RedeemCodeRequest コード引き換えリクエスト
// @Description コード引き換えリクエスト
type RedeemCodeRequest struct {
	Code   string `json:"code" example:"REDEEM123"`
	UserID string `json:"user_id" example:"user123"`
}

// RedeemCodeResponse コード引き換えレスポンス
// @Description コード引き換えレスポンス
type RedeemCodeResponse struct {
	RedemptionID  string `json:"redemption_id" example:"red_123"`
	TransactionID string `json:"transaction_id" example:"txn_456"`
	Code          string `json:"code" example:"REDEEM123"`
	CurrencyType  string `json:"currency_type" example:"free" enums:"paid,free"`
	Amount        string `json:"amount" example:"500"`
	BalanceAfter  string `json:"balance_after" example:"1000"`
	Status        string `json:"status" example:"completed"`
}

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
// @Summary コードを引き換え
// @Description 引き換えコードを使用して通貨を付与します
// @Tags redemption
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body RedeemCodeRequest true "コード引き換えリクエスト"
// @Success 200 {object} RedeemCodeResponse "コード引き換え成功"
// @Failure 400 {object} ErrorResponse "不正なリクエスト"
// @Failure 403 {object} ErrorResponse "認証エラー"
// @Failure 404 {object} ErrorResponse "コードが見つからない"
// @Router /codes/redeem [post]
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

	return c.JSON(http.StatusOK, RedeemCodeResponse{
		RedemptionID:  resp.RedemptionID,
		TransactionID: resp.TransactionID,
		Code:          resp.Code,
		CurrencyType:  resp.CurrencyType,
		Amount:        strconv.FormatInt(resp.Amount, 10),
		BalanceAfter:  strconv.FormatInt(resp.BalanceAfter, 10),
		Status:        resp.Status,
	})
}
