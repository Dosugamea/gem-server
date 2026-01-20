package middleware

import (
	"net/http"
	"strings"

	"gem-server/internal/infrastructure/config"
	otelinfra "gem-server/internal/infrastructure/observability/otel"

	"github.com/labstack/echo/v4"
)

// APIKeyMiddleware APIキー認証ミドルウェア
func APIKeyMiddleware(cfg *config.AdminAPIConfig, logger *otelinfra.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()

			// 管理APIが無効化されている場合はエラー
			if !cfg.Enabled {
				logger.Warn(ctx, "Admin API is disabled", nil)
				return c.JSON(http.StatusForbidden, ErrorResponse{
					Error:   "forbidden",
					Message: "Admin API is disabled",
				})
			}

			// X-API-KeyヘッダーからAPIキーを取得
			apiKey := c.Request().Header.Get("X-API-Key")
			if apiKey == "" {
				logger.Warn(ctx, "Missing X-API-Key header", nil)
				return c.JSON(http.StatusUnauthorized, ErrorResponse{
					Error:   "unauthorized",
					Message: "Missing X-API-Key header",
				})
			}

			// APIキーの検証
			if apiKey != cfg.APIKey {
				logger.Warn(ctx, "Invalid API key", nil)
				return c.JSON(http.StatusUnauthorized, ErrorResponse{
					Error:   "unauthorized",
					Message: "Invalid API key",
				})
			}

			// IP制限のチェック（設定されている場合）
			if len(cfg.AllowedIPs) > 0 {
				clientIP := getClientIP(c)
				if !isIPAllowed(clientIP, cfg.AllowedIPs) {
					logger.Warn(ctx, "IP address not allowed", map[string]interface{}{
						"ip": clientIP,
					})
					return c.JSON(http.StatusForbidden, ErrorResponse{
						Error:   "forbidden",
						Message: "IP address not allowed",
					})
				}
			}

			// 次のハンドラーを実行
			return next(c)
		}
	}
}

// getClientIP クライアントのIPアドレスを取得
func getClientIP(c echo.Context) string {
	// X-Forwarded-Forヘッダーから取得（プロキシ経由の場合）
	forwardedFor := c.Request().Header.Get("X-Forwarded-For")
	if forwardedFor != "" {
		// カンマ区切りの最初のIPを取得
		ips := strings.Split(forwardedFor, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// X-Real-IPヘッダーから取得
	realIP := c.Request().Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// RemoteAddrから取得
	addr := c.Request().RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// isIPAllowed IPアドレスが許可リストに含まれているかチェック
func isIPAllowed(ip string, allowedIPs []string) bool {
	for _, allowedIP := range allowedIPs {
		if ip == allowedIP {
			return true
		}
		// CIDR表記のサポート（簡易版）
		if strings.Contains(allowedIP, "/") {
			// CIDRマッチングの実装は必要に応じて追加
			// ここでは簡易的にプレフィックスマッチのみ
			if strings.HasPrefix(ip, strings.Split(allowedIP, "/")[0]) {
				return true
			}
		}
	}
	return false
}
