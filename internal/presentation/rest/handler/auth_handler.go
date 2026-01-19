package handler

import (
	"net/http"

	authapp "gem-server/internal/application/auth"

	"github.com/labstack/echo/v4"
)

// AuthHandler 認証関連ハンドラー
type AuthHandler struct {
	authService *authapp.AuthApplicationService
}

// NewAuthHandler 新しいAuthHandlerを作成
func NewAuthHandler(authService *authapp.AuthApplicationService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// GenerateToken トークン生成ハンドラー
// POST /api/v1/auth/token
func (h *AuthHandler) GenerateToken(c echo.Context) error {
	var reqBody struct {
		UserID string `json:"user_id"`
	}

	if err := c.Bind(&reqBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if reqBody.UserID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "user_id is required")
	}

	req := &authapp.GenerateTokenRequest{
		UserID: reqBody.UserID,
	}

	resp, err := h.authService.GenerateToken(c.Request().Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"token":      resp.Token,
		"expires_in": resp.ExpiresIn,
		"token_type": resp.TokenType,
	})
}
