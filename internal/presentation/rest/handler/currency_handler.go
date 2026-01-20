package handler

import (
	"net/http"
	"strconv"

	currencyapp "gem-server/internal/application/currency"

	"github.com/labstack/echo/v4"
)

// CurrencyHandler 通貨関連ハンドラー
type CurrencyHandler struct {
	currencyService *currencyapp.CurrencyApplicationService
}

// NewCurrencyHandler 新しいCurrencyHandlerを作成
func NewCurrencyHandler(currencyService *currencyapp.CurrencyApplicationService) *CurrencyHandler {
	return &CurrencyHandler{
		currencyService: currencyService,
	}
}

// GetBalance 残高取得ハンドラー（ユーザーAPI用）
// @Summary 残高を取得
// @Description 自分の通貨残高を取得します
// @Tags currency
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} BalanceResponse "残高取得成功"
// @Failure 400 {object} ErrorResponse "不正なリクエスト"
// @Failure 403 {object} ErrorResponse "認証エラー"
// @Router /me/balance [get]
func (h *CurrencyHandler) GetBalance(c echo.Context) error {
	// トークンからuser_idを取得
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "user_id not found in token")
	}

	req := &currencyapp.GetBalanceRequest{
		UserID: userID,
	}

	resp, err := h.currencyService.GetBalance(c.Request().Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, BalanceResponse{
		UserID: resp.UserID,
		Balances: BalanceItem{
			Paid: strconv.FormatInt(resp.Balances["paid"], 10),
			Free: strconv.FormatInt(resp.Balances["free"], 10),
		},
	})
}

// GetBalanceAdmin 残高取得ハンドラー（管理API用）
// @Summary 残高を取得（管理API）
// @Description 指定されたユーザーの通貨残高を取得します
// @Tags admin
// @Accept json
// @Produce json
// @Param user_id path string true "ユーザーID" example(user123)
// @Param X-API-Key header string true "APIキー"
// @Success 200 {object} BalanceResponse "残高取得成功"
// @Failure 400 {object} ErrorResponse "不正なリクエスト"
// @Failure 401 {object} ErrorResponse "認証エラー"
// @Router /admin/users/{user_id}/balance [get]
func (h *CurrencyHandler) GetBalanceAdmin(c echo.Context) error {
	userID := c.Param("user_id")
	if userID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "user_id is required")
	}

	req := &currencyapp.GetBalanceRequest{
		UserID: userID,
	}

	resp, err := h.currencyService.GetBalance(c.Request().Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, BalanceResponse{
		UserID: resp.UserID,
		Balances: BalanceItem{
			Paid: strconv.FormatInt(resp.Balances["paid"], 10),
			Free: strconv.FormatInt(resp.Balances["free"], 10),
		},
	})
}

// GrantCurrency 通貨付与ハンドラー（管理API用）
// @Summary 通貨を付与（管理API）
// @Description 指定されたユーザーに無償通貨を付与します
// @Tags admin
// @Accept json
// @Produce json
// @Param user_id path string true "ユーザーID" example(user123)
// @Param X-API-Key header string true "APIキー"
// @Param request body GrantRequest true "通貨付与リクエスト"
// @Success 200 {object} GrantResponse "通貨付与成功"
// @Failure 400 {object} ErrorResponse "不正なリクエスト"
// @Failure 401 {object} ErrorResponse "認証エラー"
// @Router /admin/users/{user_id}/grant [post]
func (h *CurrencyHandler) GrantCurrency(c echo.Context) error {
	userID := c.Param("user_id")
	if userID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "user_id is required")
	}

	var reqBody GrantRequest
	if err := c.Bind(&reqBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// 金額をint64に変換
	amount, err := strconv.ParseInt(reqBody.Amount, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid amount format")
	}

	req := &currencyapp.GrantRequest{
		UserID:       userID,
		CurrencyType: reqBody.CurrencyType,
		Amount:       amount,
		Reason:       reqBody.Reason,
		Requester:    reqBody.Requester,
		Metadata:     reqBody.Metadata,
	}

	resp, err := h.currencyService.Grant(c.Request().Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, GrantResponse{
		TransactionID: resp.TransactionID,
		BalanceAfter:  strconv.FormatInt(resp.BalanceAfter, 10),
		Status:        resp.Status,
	})
}

// ConsumeCurrency 通貨消費ハンドラー（管理API用）
// @Summary 通貨を消費（管理API）
// @Description 指定されたユーザーの通貨を消費します。優先順位制御も可能です
// @Tags admin
// @Accept json
// @Produce json
// @Param user_id path string true "ユーザーID" example(user123)
// @Param X-API-Key header string true "APIキー"
// @Param request body ConsumeRequest true "通貨消費リクエスト"
// @Success 200 {object} ConsumeResponse "通貨消費成功"
// @Failure 400 {object} ErrorResponse "不正なリクエスト"
// @Failure 401 {object} ErrorResponse "認証エラー"
// @Failure 409 {object} ErrorResponse "残高不足"
// @Router /admin/users/{user_id}/consume [post]
func (h *CurrencyHandler) ConsumeCurrency(c echo.Context) error {
	userID := c.Param("user_id")
	if userID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "user_id is required")
	}

	var reqBody ConsumeRequest
	if err := c.Bind(&reqBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// 金額をint64に変換
	amount, err := strconv.ParseInt(reqBody.Amount, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid amount format")
	}

	req := &currencyapp.ConsumeRequest{
		UserID:       userID,
		CurrencyType: reqBody.CurrencyType,
		Amount:       amount,
		ItemID:       reqBody.ItemID,
		UsePriority:  reqBody.UsePriority,
		Requester:    reqBody.Requester,
		Metadata:     reqBody.Metadata,
	}

	var resp *currencyapp.ConsumeResponse
	if reqBody.UsePriority || reqBody.CurrencyType == "auto" {
		// 優先順位制御を使用
		resp, err = h.currencyService.ConsumeWithPriority(c.Request().Context(), req)
	} else {
		// 単一通貨タイプで消費
		resp, err = h.currencyService.Consume(c.Request().Context(), req)
	}

	if err != nil {
		return err
	}

	// レスポンスを構築
	consumeResp := ConsumeResponse{
		TransactionID: resp.TransactionID,
		Status:        resp.Status,
	}

	if len(resp.ConsumptionDetails) > 0 {
		// 優先順位制御使用時
		details := make([]ConsumptionDetail, len(resp.ConsumptionDetails))
		for i, detail := range resp.ConsumptionDetails {
			details[i] = ConsumptionDetail{
				CurrencyType:  detail.CurrencyType,
				Amount:        strconv.FormatInt(detail.Amount, 10),
				BalanceBefore: strconv.FormatInt(detail.BalanceBefore, 10),
				BalanceAfter:  strconv.FormatInt(detail.BalanceAfter, 10),
			}
		}
		consumeResp.ConsumptionDetails = details
		consumeResp.TotalConsumed = strconv.FormatInt(resp.TotalConsumed, 10)
	} else {
		// 単一通貨タイプ消費時
		consumeResp.BalanceAfter = strconv.FormatInt(resp.BalanceAfter, 10)
	}

	return c.JSON(http.StatusOK, consumeResp)
}
