package middleware

import (
	"github.com/labstack/echo/v4"
)

// SecurityHeadersMiddleware セキュリティヘッダーを設定するミドルウェア
func SecurityHeadersMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// XSS保護
			c.Response().Header().Set("X-XSS-Protection", "1; mode=block")

			// クリックジャッキング保護
			c.Response().Header().Set("X-Frame-Options", "DENY")

			// MIMEタイプスニッフィング保護
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")

			// コンテンツセキュリティポリシー（必要に応じて調整）
			c.Response().Header().Set("Content-Security-Policy",
				"default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")

			// Strict-Transport-Security（HTTPS使用時）
			if c.Scheme() == "https" {
				c.Response().Header().Set("Strict-Transport-Security",
					"max-age=31536000; includeSubDomains")
			}

			// Referrer-Policy
			c.Response().Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			return next(c)
		}
	}
}
