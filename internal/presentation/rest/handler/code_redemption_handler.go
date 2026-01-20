package handler

import (
	"net/http"
	"strconv"
	"time"

	redemptionapp "gem-server/internal/application/code_redemption"

	"github.com/labstack/echo/v4"
)

// RedeemCodeRequest コード引き換えリクエスト
// @Description コード引き換えリクエスト
type RedeemCodeRequest struct {
	Code string `json:"code" example:"REDEEM123"`
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
	// トークンからuser_idを取得
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "user_id not found in token")
	}

	var reqBody struct {
		Code string `json:"code"`
	}

	if err := c.Bind(&reqBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	req := &redemptionapp.RedeemCodeRequest{
		Code:   reqBody.Code,
		UserID: userID,
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

// CreateCodeRequest 引き換えコード作成リクエスト
// @Description 引き換えコード作成リクエスト
type CreateCodeRequest struct {
	Code         string                 `json:"code" example:"PROMO2024"`
	CodeType     string                 `json:"code_type" example:"promotion" enums:"promotion,gift,event"`
	CurrencyType string                 `json:"currency_type" example:"free" enums:"paid,free"`
	Amount       string                 `json:"amount" example:"1000"`
	MaxUses      int                    `json:"max_uses" example:"100"`
	ValidFrom    string                 `json:"valid_from" example:"2024-01-01T00:00:00Z"`
	ValidUntil   string                 `json:"valid_until" example:"2024-12-31T23:59:59Z"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// CreateCodeResponse 引き換えコード作成レスポンス
// @Description 引き換えコード作成レスポンス
type CreateCodeResponse struct {
	Code         string                 `json:"code" example:"PROMO2024"`
	CodeType     string                 `json:"code_type" example:"promotion"`
	CurrencyType string                 `json:"currency_type" example:"free"`
	Amount       string                 `json:"amount" example:"1000"`
	MaxUses      int                    `json:"max_uses" example:"100"`
	CurrentUses  int                    `json:"current_uses" example:"0"`
	ValidFrom    string                 `json:"valid_from" example:"2024-01-01T00:00:00Z"`
	ValidUntil   string                 `json:"valid_until" example:"2024-12-31T23:59:59Z"`
	Status       string                 `json:"status" example:"active"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    string                 `json:"created_at" example:"2024-01-01T00:00:00Z"`
}

// DeleteCodeResponse 引き換えコード削除レスポンス
// @Description 引き換えコード削除レスポンス
type DeleteCodeResponse struct {
	Code      string `json:"code" example:"PROMO2024"`
	DeletedAt string `json:"deleted_at" example:"2024-01-01T00:00:00Z"`
}

// GetCodeResponse 引き換えコード取得レスポンス
// @Description 引き換えコード取得レスポンス
type GetCodeResponse struct {
	Code         string                 `json:"code" example:"PROMO2024"`
	CodeType     string                 `json:"code_type" example:"promotion"`
	CurrencyType string                 `json:"currency_type" example:"free"`
	Amount       string                 `json:"amount" example:"1000"`
	MaxUses      int                    `json:"max_uses" example:"100"`
	CurrentUses  int                    `json:"current_uses" example:"0"`
	ValidFrom    string                 `json:"valid_from" example:"2024-01-01T00:00:00Z"`
	ValidUntil   string                 `json:"valid_until" example:"2024-12-31T23:59:59Z"`
	Status       string                 `json:"status" example:"active"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    string                 `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt    string                 `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// ListCodesResponse 引き換えコード一覧取得レスポンス
// @Description 引き換えコード一覧取得レスポンス
type ListCodesResponse struct {
	Codes  []CodeItem `json:"codes"`
	Total  int        `json:"total" example:"100"`
	Limit  int        `json:"limit" example:"50"`
	Offset int        `json:"offset" example:"0"`
}

// CodeItem 引き換えコードアイテム
// @Description 引き換えコードアイテム
type CodeItem struct {
	Code         string                 `json:"code" example:"PROMO2024"`
	CodeType     string                 `json:"code_type" example:"promotion"`
	CurrencyType string                 `json:"currency_type" example:"free"`
	Amount       string                 `json:"amount" example:"1000"`
	MaxUses      int                    `json:"max_uses" example:"100"`
	CurrentUses  int                    `json:"current_uses" example:"0"`
	ValidFrom    string                 `json:"valid_from" example:"2024-01-01T00:00:00Z"`
	ValidUntil   string                 `json:"valid_until" example:"2024-12-31T23:59:59Z"`
	Status       string                 `json:"status" example:"active"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    string                 `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt    string                 `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// CreateCode 引き換えコード作成ハンドラー（管理API用）
// @Summary 引き換えコードを作成（管理API）
// @Description 新しい引き換えコードを作成します
// @Tags admin
// @Accept json
// @Produce json
// @Param X-API-Key header string true "APIキー"
// @Param request body CreateCodeRequest true "引き換えコード作成リクエスト"
// @Success 201 {object} CreateCodeResponse "引き換えコード作成成功"
// @Failure 400 {object} ErrorResponse "不正なリクエスト"
// @Failure 401 {object} ErrorResponse "認証エラー"
// @Failure 409 {object} ErrorResponse "コードが既に存在"
// @Router /admin/codes [post]
func (h *CodeRedemptionHandler) CreateCode(c echo.Context) error {
	var reqBody CreateCodeRequest

	if err := c.Bind(&reqBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// コードが空の場合のバリデーション
	if reqBody.Code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "code is required")
	}

	// 日付のパース
	validFrom, err := time.Parse(time.RFC3339, reqBody.ValidFrom)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid valid_from format")
	}

	validUntil, err := time.Parse(time.RFC3339, reqBody.ValidUntil)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid valid_until format")
	}

	// 金額をint64に変換
	amount, err := strconv.ParseInt(reqBody.Amount, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid amount format")
	}

	req := &redemptionapp.CreateCodeRequest{
		Code:         reqBody.Code,
		CodeType:     reqBody.CodeType,
		CurrencyType: reqBody.CurrencyType,
		Amount:       amount,
		MaxUses:      reqBody.MaxUses,
		ValidFrom:    validFrom,
		ValidUntil:   validUntil,
		Metadata:     reqBody.Metadata,
	}

	resp, err := h.redemptionService.CreateCode(c.Request().Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, CreateCodeResponse{
		Code:         resp.Code,
		CodeType:     resp.CodeType,
		CurrencyType: resp.CurrencyType,
		Amount:       strconv.FormatInt(resp.Amount, 10),
		MaxUses:      resp.MaxUses,
		CurrentUses:  resp.CurrentUses,
		ValidFrom:    resp.ValidFrom.Format(time.RFC3339),
		ValidUntil:   resp.ValidUntil.Format(time.RFC3339),
		Status:       resp.Status,
		Metadata:     resp.Metadata,
		CreatedAt:    resp.CreatedAt.Format(time.RFC3339),
	})
}

// DeleteCode 引き換えコード削除ハンドラー（管理API用）
// @Summary 引き換えコードを削除（管理API）
// @Description 引き換えコードを削除します（使用済みコードは削除不可）
// @Tags admin
// @Accept json
// @Produce json
// @Param code path string true "引き換えコード" example(PROMO2024)
// @Param X-API-Key header string true "APIキー"
// @Success 200 {object} DeleteCodeResponse "引き換えコード削除成功"
// @Failure 400 {object} ErrorResponse "不正なリクエスト"
// @Failure 401 {object} ErrorResponse "認証エラー"
// @Failure 404 {object} ErrorResponse "コードが見つからない"
// @Failure 409 {object} ErrorResponse "コードが使用済みのため削除不可"
// @Router /admin/codes/{code} [delete]
func (h *CodeRedemptionHandler) DeleteCode(c echo.Context) error {
	code := c.Param("code")
	if code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "code is required")
	}

	req := &redemptionapp.DeleteCodeRequest{
		Code: code,
	}

	resp, err := h.redemptionService.DeleteCode(c.Request().Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, DeleteCodeResponse{
		Code:      resp.Code,
		DeletedAt: resp.DeletedAt.Format(time.RFC3339),
	})
}

// GetCode 引き換えコード取得ハンドラー（管理API用）
// @Summary 引き換えコードを取得（管理API）
// @Description 指定された引き換えコードの詳細を取得します
// @Tags admin
// @Accept json
// @Produce json
// @Param code path string true "引き換えコード" example(PROMO2024)
// @Param X-API-Key header string true "APIキー"
// @Success 200 {object} GetCodeResponse "引き換えコード取得成功"
// @Failure 400 {object} ErrorResponse "不正なリクエスト"
// @Failure 401 {object} ErrorResponse "認証エラー"
// @Failure 404 {object} ErrorResponse "コードが見つからない"
// @Router /admin/codes/{code} [get]
func (h *CodeRedemptionHandler) GetCode(c echo.Context) error {
	code := c.Param("code")
	if code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "code is required")
	}

	req := &redemptionapp.GetCodeRequest{
		Code: code,
	}

	resp, err := h.redemptionService.GetCode(c.Request().Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, GetCodeResponse{
		Code:         resp.Code,
		CodeType:     resp.CodeType,
		CurrencyType: resp.CurrencyType,
		Amount:       strconv.FormatInt(resp.Amount, 10),
		MaxUses:      resp.MaxUses,
		CurrentUses:  resp.CurrentUses,
		ValidFrom:    resp.ValidFrom.Format(time.RFC3339),
		ValidUntil:   resp.ValidUntil.Format(time.RFC3339),
		Status:       resp.Status,
		Metadata:     resp.Metadata,
		CreatedAt:    resp.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    resp.UpdatedAt.Format(time.RFC3339),
	})
}

// ListCodes 引き換えコード一覧取得ハンドラー（管理API用）
// @Summary 引き換えコード一覧を取得（管理API）
// @Description 引き換えコードの一覧を取得します（ページネーション・フィルタリング対応）
// @Tags admin
// @Accept json
// @Produce json
// @Param limit query int false "取得件数" default(50) example(50)
// @Param offset query int false "オフセット" default(0) example(0)
// @Param status query string false "ステータスフィルタ" example(active) enums(active,expired,disabled)
// @Param code_type query string false "コードタイプフィルタ" example(promotion) enums(promotion,gift,event)
// @Param X-API-Key header string true "APIキー"
// @Success 200 {object} ListCodesResponse "引き換えコード一覧取得成功"
// @Failure 400 {object} ErrorResponse "不正なリクエスト"
// @Failure 401 {object} ErrorResponse "認証エラー"
// @Router /admin/codes [get]
func (h *CodeRedemptionHandler) ListCodes(c echo.Context) error {
	// クエリパラメータの取得
	limit := 50 // デフォルト値
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid limit parameter")
		}
	}

	offset := 0 // デフォルト値
	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		var err error
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid offset parameter")
		}
	}

	status := c.QueryParam("status")
	codeType := c.QueryParam("code_type")

	req := &redemptionapp.ListCodesRequest{
		Limit:    limit,
		Offset:   offset,
		Status:   status,
		CodeType: codeType,
	}

	resp, err := h.redemptionService.ListCodes(c.Request().Context(), req)
	if err != nil {
		return err
	}

	// ドメインエンティティをレスポンス形式に変換
	codes := make([]CodeItem, len(resp.Codes))
	for i, code := range resp.Codes {
		codes[i] = CodeItem{
			Code:         code.Code(),
			CodeType:     code.CodeType().String(),
			CurrencyType: code.CurrencyType().String(),
			Amount:       strconv.FormatInt(code.Amount(), 10),
			MaxUses:      code.MaxUses(),
			CurrentUses:  code.CurrentUses(),
			ValidFrom:    code.ValidFrom().Format(time.RFC3339),
			ValidUntil:   code.ValidUntil().Format(time.RFC3339),
			Status:       code.Status().String(),
			Metadata:     code.Metadata(),
			CreatedAt:    code.CreatedAt().Format(time.RFC3339),
			UpdatedAt:    code.UpdatedAt().Format(time.RFC3339),
		}
	}

	return c.JSON(http.StatusOK, ListCodesResponse{
		Codes:  codes,
		Total:  resp.Total,
		Limit:  resp.Limit,
		Offset: resp.Offset,
	})
}
