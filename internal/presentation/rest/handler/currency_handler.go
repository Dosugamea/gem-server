package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	currencyapp "gem-server/internal/application/currency"
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

// GetBalance 残高取得ハンドラー
// GET /api/v1/users/{user_id}/balance
func (h *CurrencyHandler) GetBalance(c echo.Context) error {
	userID := c.Param("user_id")
	if userID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "user_id is required")
	}

	// パスパラメータのuser_idとトークンのuser_idが一致するか確認（認証ミドルウェアで設定）
	tokenUserID, ok := c.Get("user_id").(string)
	if !ok || tokenUserID != userID {
		return echo.NewHTTPError(http.StatusForbidden, "user_id mismatch")
	}

	req := &currencyapp.GetBalanceRequest{
		UserID: userID,
	}

	resp, err := h.currencyService.GetBalance(c.Request().Context(), req)
	if err != nil {
		return err
	}

	// レスポンスを文字列形式に変換
	balances := make(map[string]string)
	balances["paid"] = strconv.FormatInt(resp.Balances["paid"], 10)
	balances["free"] = strconv.FormatInt(resp.Balances["free"], 10)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user_id":  resp.UserID,
		"balances": balances,
	})
}

// GrantCurrency 通貨付与ハンドラー
// POST /api/v1/users/{user_id}/grant
func (h *CurrencyHandler) GrantCurrency(c echo.Context) error {
	userID := c.Param("user_id")
	if userID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "user_id is required")
	}

	// パスパラメータのuser_idとトークンのuser_idが一致するか確認
	tokenUserID, ok := c.Get("user_id").(string)
	if !ok || tokenUserID != userID {
		return echo.NewHTTPError(http.StatusForbidden, "user_id mismatch")
	}

	var reqBody struct {
		CurrencyType string                 `json:"currency_type"`
		Amount       string                 `json:"amount"`
		Reason       string                 `json:"reason"`
		Metadata     map[string]interface{} `json:"metadata"`
	}

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
		Metadata:     reqBody.Metadata,
	}

	resp, err := h.currencyService.Grant(c.Request().Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"transaction_id": resp.TransactionID,
		"balance_after": strconv.FormatInt(resp.BalanceAfter, 10),
		"status":         resp.Status,
	})
}

// ConsumeCurrency 通貨消費ハンドラー
// POST /api/v1/users/{user_id}/consume
func (h *CurrencyHandler) ConsumeCurrency(c echo.Context) error {
	userID := c.Param("user_id")
	if userID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "user_id is required")
	}

	// パスパラメータのuser_idとトークンのuser_idが一致するか確認
	tokenUserID, ok := c.Get("user_id").(string)
	if !ok || tokenUserID != userID {
		return echo.NewHTTPError(http.StatusForbidden, "user_id mismatch")
	}

	var reqBody struct {
		CurrencyType string                 `json:"currency_type"`
		Amount       string                 `json:"amount"`
		ItemID       string                 `json:"item_id"`
		UsePriority  bool                   `json:"use_priority"`
		Metadata     map[string]interface{} `json:"metadata"`
	}

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
	response := map[string]interface{}{
		"transaction_id": resp.TransactionID,
		"status":         resp.Status,
	}

	if len(resp.ConsumptionDetails) > 0 {
		// 優先順位制御使用時
		details := make([]map[string]string, len(resp.ConsumptionDetails))
		for i, detail := range resp.ConsumptionDetails {
			details[i] = map[string]string{
				"currency_type":  detail.CurrencyType,
				"amount":          strconv.FormatInt(detail.Amount, 10),
				"balance_before":  strconv.FormatInt(detail.BalanceBefore, 10),
				"balance_after":   strconv.FormatInt(detail.BalanceAfter, 10),
			}
		}
		response["consumption_details"] = details
		response["total_consumed"] = strconv.FormatInt(resp.TotalConsumed, 10)
	} else {
		// 単一通貨タイプ消費時
		response["balance_after"] = strconv.FormatInt(resp.BalanceAfter, 10)
	}

	return c.JSON(http.StatusOK, response)
}
