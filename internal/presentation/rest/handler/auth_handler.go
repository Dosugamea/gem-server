package handler

import (
	"net/http"

	authapp "gem-server/internal/application/auth"

	"github.com/labstack/echo/v4"
)

// GenerateTokenRequest トークン生成リクエスト
// @Description トークン生成リクエスト
type GenerateTokenRequest struct {
	UserID string `json:"user_id" example:"user123"`
}

// GenerateTokenResponse トークン生成レスポンス
// @Description トークン生成レスポンス
type GenerateTokenResponse struct {
	Token     string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidXNlcjEyMyIsImV4cCI6MTcwMDAwMDAwMH0.signature"`
	ExpiresIn int    `json:"expires_in" example:"3600"`
	TokenType string `json:"token_type" example:"Bearer"`
}

// ErrorResponse エラーレスポンス
// @Description エラーレスポンス
type ErrorResponse struct {
	Error string `json:"error" example:"invalid request body"`
}

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
// @Tags auth
// @Accept json
// @Produce json
// @Param request body GenerateTokenRequest true "トークン生成リクエスト"
// @Success 200 {object} GenerateTokenResponse "トークン生成成功"
// @Failure 400 {object} ErrorResponse "不正なリクエスト"
// @Router /auth/token [post]
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

	return c.JSON(http.StatusOK, GenerateTokenResponse{
		Token:     resp.Token,
		ExpiresIn: int(resp.ExpiresIn),
		TokenType: resp.TokenType,
	})
}
