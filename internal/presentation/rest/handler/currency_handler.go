package handler

import (
	"net/http"
	"strconv"

	currencyapp "gem-server/internal/application/currency"
	"github.com/labstack/echo/v4"
)

// BalanceItem 残高アイテム
// @Description 残高アイテム
type BalanceItem struct {
	Paid string `json:"paid" example:"1000"`
	Free string `json:"free" example:"500"`
}

// BalanceResponse 残高レスポンス
// @Description 残高レスポンス
type BalanceResponse struct {
	UserID   string      `json:"user_id" example:"user123"`
	Balances BalanceItem `json:"balances"`
}

// GrantRequest 通貨付与リクエスト
// @Description 通貨付与リクエスト
type GrantRequest struct {
	CurrencyType string                 `json:"currency_type" example:"free" enums:"paid,free"`
	Amount       string                 `json:"amount" example:"100"`
	Reason       string                 `json:"reason" example:"イベント報酬"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// GrantResponse 通貨付与レスポンス
// @Description 通貨付与レスポンス
type GrantResponse struct {
	TransactionID string `json:"transaction_id" example:"txn_123"`
	BalanceAfter  string `json:"balance_after" example:"600"`
	Status        string `json:"status" example:"completed"`
}

// ConsumeRequest 通貨消費リクエスト
// @Description 通貨消費リクエスト
type ConsumeRequest struct {
	CurrencyType string                 `json:"currency_type" example:"paid" enums:"paid,free,auto"`
	Amount       string                 `json:"amount" example:"50"`
	ItemID       string                 `json:"item_id" example:"item001"`
	UsePriority  bool                   `json:"use_priority" example:"false"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ConsumptionDetail 消費詳細
// @Description 消費詳細
type ConsumptionDetail struct {
	CurrencyType  string `json:"currency_type" example:"paid"`
	Amount        string `json:"amount" example:"50"`
	BalanceBefore string `json:"balance_before" example:"1000"`
	BalanceAfter  string `json:"balance_after" example:"950"`
}

// ConsumeResponse 通貨消費レスポンス
// @Description 通貨消費レスポンス
type ConsumeResponse struct {
	TransactionID      string              `json:"transaction_id" example:"txn_456"`
	ConsumptionDetails []ConsumptionDetail `json:"consumption_details,omitempty"`
	BalanceAfter       string              `json:"balance_after,omitempty" example:"950"`
	TotalConsumed      string              `json:"total_consumed,omitempty" example:"50"`
	Status             string              `json:"status" example:"completed"`
}

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
// @Summary 残高を取得
// @Description 指定されたユーザーの通貨残高を取得します
// @Tags currency
// @Accept json
// @Produce json
// @Security Bearer
// @Param user_id path string true "ユーザーID" example(user123)
// @Success 200 {object} BalanceResponse "残高取得成功"
// @Failure 400 {object} ErrorResponse "不正なリクエスト"
// @Failure 403 {object} ErrorResponse "認証エラー"
// @Router /users/{user_id}/balance [get]
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

	return c.JSON(http.StatusOK, BalanceResponse{
		UserID: resp.UserID,
		Balances: BalanceItem{
			Paid: strconv.FormatInt(resp.Balances["paid"], 10),
			Free: strconv.FormatInt(resp.Balances["free"], 10),
		},
	})
}

// GrantCurrency 通貨付与ハンドラー
// @Summary 通貨を付与
// @Description 指定されたユーザーに無償通貨を付与します
// @Tags currency
// @Accept json
// @Produce json
// @Security Bearer
// @Param user_id path string true "ユーザーID" example(user123)
// @Param request body GrantRequest true "通貨付与リクエスト"
// @Success 200 {object} GrantResponse "通貨付与成功"
// @Failure 400 {object} ErrorResponse "不正なリクエスト"
// @Failure 403 {object} ErrorResponse "認証エラー"
// @Router /users/{user_id}/grant [post]
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

	return c.JSON(http.StatusOK, GrantResponse{
		TransactionID: resp.TransactionID,
		BalanceAfter:  strconv.FormatInt(resp.BalanceAfter, 10),
		Status:        resp.Status,
	})
}

// ConsumeCurrency 通貨消費ハンドラー
// @Summary 通貨を消費
// @Description 指定されたユーザーの通貨を消費します。優先順位制御も可能です
// @Tags currency
// @Accept json
// @Produce json
// @Security Bearer
// @Param user_id path string true "ユーザーID" example(user123)
// @Param request body ConsumeRequest true "通貨消費リクエスト"
// @Success 200 {object} ConsumeResponse "通貨消費成功"
// @Failure 400 {object} ErrorResponse "不正なリクエスト"
// @Failure 403 {object} ErrorResponse "認証エラー"
// @Failure 409 {object} ErrorResponse "残高不足"
// @Router /users/{user_id}/consume [post]
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
