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

// GetTransactionHistory トランザクション履歴取得ハンドラー
// GET /api/v1/users/{user_id}/transactions
func (h *HistoryHandler) GetTransactionHistory(c echo.Context) error {
	userID := c.Param("user_id")
	if userID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "user_id is required")
	}

	// パスパラメータのuser_idとトークンのuser_idが一致するか確認
	tokenUserID, ok := c.Get("user_id").(string)
	if !ok || tokenUserID != userID {
		return echo.NewHTTPError(http.StatusForbidden, "user_id mismatch")
	}

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
	transactions := make([]map[string]interface{}, len(resp.Transactions))
	for i, txn := range resp.Transactions {
		transactions[i] = map[string]interface{}{
			"transaction_id":   txn.TransactionID(),
			"transaction_type": txn.TransactionType().String(),
			"currency_type":    txn.CurrencyType().String(),
			"amount":           strconv.FormatInt(txn.Amount(), 10),
			"balance_before":   strconv.FormatInt(txn.BalanceBefore(), 10),
			"balance_after":    strconv.FormatInt(txn.BalanceAfter(), 10),
			"status":           txn.Status().String(),
			"created_at":       txn.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"transactions": transactions,
		"total":        resp.Total,
		"limit":        resp.Limit,
		"offset":       resp.Offset,
	})
}
