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
// @Summary 認証トークンを生成
// @Description ユーザーIDを元にJWT認証トークンを生成します
// @Tags admin
// @Accept json
// @Produce json
// @Param user_id path string true "ユーザーID"
// @Success 200 {object} GenerateTokenResponse "トークン生成成功"
// @Failure 400 {object} ErrorResponse "不正なリクエスト"
// @Router /admin/users/{user_id}/issue_token [post]
func (h *AuthHandler) GenerateToken(c echo.Context) error {
	userID := c.Param("user_id")
	if userID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "user_id is required")
	}

	req := &authapp.GenerateTokenRequest{
		UserID: userID,
	}

	resp, err := h.authService.GenerateToken(c.Request().Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, GenerateTokenResponse{
		Token:     resp.Token,
		ExpiresIn: int(resp.ExpiresIn),
		TokenType: resp.TokenType,
	})
}
