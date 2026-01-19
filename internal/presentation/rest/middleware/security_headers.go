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

			// コンテンツセキュリティポリシー
			// Swagger関連のパスでは外部CDNを許可
			path := c.Request().URL.Path
			var csp string
			if isSwaggerPath(path) {
				// Swagger UI用: unpkg.comとcdn.jsdelivr.netを許可
				csp = "default-src 'self'; script-src 'self' 'unsafe-inline' https://unpkg.com https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline' https://unpkg.com https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data: https:;"
			} else {
				// 通常のAPI用: より厳格な設定
				csp = "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'"
			}
			c.Response().Header().Set("Content-Security-Policy", csp)

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

// isSwaggerPath Swagger関連のパスかどうかを判定
func isSwaggerPath(path string) bool {
	return path == "/swagger" || path == "/redoc" || path == "/openapi.yaml"
}
