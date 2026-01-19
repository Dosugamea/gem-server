package middleware

import (
	"strings"

	"gem-server/internal/infrastructure/config"
	otelinfra "gem-server/internal/infrastructure/observability/otel"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// AuthMiddleware JWT認証ミドルウェア
func AuthMiddleware(cfg *config.JWTConfig, logger *otelinfra.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()

			// Authorizationヘッダーからトークンを取得
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				logger.Warn(ctx, "Missing authorization header", nil)
				return c.JSON(401, ErrorResponse{
					Error:   "unauthorized",
					Message: "Missing authorization header",
				})
			}

			// Bearerトークンの形式を確認
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				logger.Warn(ctx, "Invalid authorization header format", nil)
				return c.JSON(401, ErrorResponse{
					Error:   "unauthorized",
					Message: "Invalid authorization header format",
				})
			}

			tokenString := parts[1]

			// JWTトークンの検証
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				// 署名アルゴリズムの確認
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(cfg.Secret), nil
			})

			if err != nil || !token.Valid {
				logger.Warn(ctx, "Invalid token", map[string]interface{}{
					"error": err.Error(),
				})
				return c.JSON(401, ErrorResponse{
					Error:   "unauthorized",
					Message: "Invalid or expired token",
				})
			}

			// クレームからユーザーIDを取得
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				logger.Warn(ctx, "Invalid token claims", nil)
				return c.JSON(401, ErrorResponse{
					Error:   "unauthorized",
					Message: "Invalid token claims",
				})
			}

			// ユーザーIDをコンテキストに設定
			userID, ok := claims["user_id"].(string)
			if !ok {
				logger.Warn(ctx, "Missing user_id in token claims", nil)
				return c.JSON(401, ErrorResponse{
					Error:   "unauthorized",
					Message: "Missing user_id in token",
				})
			}

			// ユーザーIDをリクエストコンテキストに設定
			c.Set("user_id", userID)

			// 次のハンドラーを実行
			return next(c)
		}
	}
}
