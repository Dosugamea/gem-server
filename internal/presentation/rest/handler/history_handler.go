package handler

import (
	"net/http"
	"strconv"

	historyapp "gem-server/internal/application/history"

	"github.com/labstack/echo/v4"
)

// HistoryHandler 履歴関連ハンドラー
type HistoryHandler struct {
	historyService *historyapp.HistoryApplicationService
}

// NewHistoryHandler 新しいHistoryHandlerを作成
func NewHistoryHandler(historyService *historyapp.HistoryApplicationService) *HistoryHandler {
	return &HistoryHandler{
		historyService: historyService,
	}
}

// GetTransactionHistory トランザクション履歴取得ハンドラー（ユーザーAPI用）
// @Summary トランザクション履歴を取得
// @Description 自分のトランザクション履歴を取得します。ページネーションとフィルタリングに対応しています
// @Tags history
// @Accept json
// @Produce json
// @Security Bearer
// @Param limit query int false "取得件数（デフォルト: 50, 最大: 100)" default(50) example(50)
// @Param offset query int false "オフセット（デフォルト: 0)" default(0) example(0)
// @Param currency_type query string false "通貨タイプでフィルタ（paid/free）" example(paid)
// @Param transaction_type query string false "トランザクションタイプでフィルタ（grant/consume/payment/redemption）" example(consume)
// @Success 200 {object} TransactionHistoryResponse "履歴取得成功"
// @Failure 400 {object} ErrorResponse "不正なリクエスト"
// @Failure 401 {object} ErrorResponse "認証エラー"
// @Router /me/transactions [get]
func (h *HistoryHandler) GetTransactionHistory(c echo.Context) error {
	// トークンからuser_idを取得
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "user_id not found in token")
	}

	return h.getTransactionHistoryInternal(c, userID)
}

// GetTransactionHistoryAdmin トランザクション履歴取得ハンドラー（管理API用）
// @Summary トランザクション履歴を取得（管理API）
// @Description 指定されたユーザーのトランザクション履歴を取得します。ページネーションとフィルタリングに対応しています
// @Tags admin
// @Accept json
// @Produce json
// @Param user_id path string true "ユーザーID" example(user123)
// @Param X-API-Key header string true "APIキー"
// @Param limit query int false "取得件数（デフォルト: 50, 最大: 100)" default(50) example(50)
// @Param offset query int false "オフセット（デフォルト: 0)" default(0) example(0)
// @Param currency_type query string false "通貨タイプでフィルタ（paid/free）" example(paid)
// @Param transaction_type query string false "トランザクションタイプでフィルタ（grant/consume/payment/redemption）" example(consume)
// @Success 200 {object} TransactionHistoryResponse "履歴取得成功"
// @Failure 400 {object} ErrorResponse "不正なリクエスト"
// @Failure 401 {object} ErrorResponse "認証エラー"
// @Router /admin/users/{user_id}/transactions [get]
func (h *HistoryHandler) GetTransactionHistoryAdmin(c echo.Context) error {
	userID := c.Param("user_id")
	if userID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "user_id is required")
	}

	return h.getTransactionHistoryInternal(c, userID)
}

// getTransactionHistoryInternal トランザクション履歴取得の内部実装
func (h *HistoryHandler) getTransactionHistoryInternal(c echo.Context, userID string) error {

	// クエリパラメータを取得
	limit := 50 // デフォルト値
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
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

	currencyType := c.QueryParam("currency_type")
	transactionType := c.QueryParam("transaction_type")

	req := &historyapp.GetTransactionHistoryRequest{
		UserID:          userID,
		Limit:           limit,
		Offset:          offset,
		CurrencyType:    currencyType,
		TransactionType: transactionType,
	}

	resp, err := h.historyService.GetTransactionHistory(c.Request().Context(), req)
	if err != nil {
		return err
	}

	// トランザクションをレスポンス形式に変換
	transactions := make([]TransactionItem, len(resp.Transactions))
	for i, txn := range resp.Transactions {
		transactions[i] = TransactionItem{
			TransactionID:   txn.TransactionID(),
			TransactionType: txn.TransactionType().String(),
			CurrencyType:    txn.CurrencyType().String(),
			Amount:          strconv.FormatInt(txn.Amount(), 10),
			BalanceBefore:   strconv.FormatInt(txn.BalanceBefore(), 10),
			BalanceAfter:    strconv.FormatInt(txn.BalanceAfter(), 10),
			Status:          txn.Status().String(),
			CreatedAt:       txn.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return c.JSON(http.StatusOK, TransactionHistoryResponse{
		Transactions: transactions,
		Total:        resp.Total,
		Limit:        resp.Limit,
		Offset:       resp.Offset,
	})
}
