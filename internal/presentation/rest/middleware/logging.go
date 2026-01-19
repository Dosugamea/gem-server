package middleware

import (
	"time"

	otelinfra "gem-server/internal/infrastructure/observability/otel"
	"github.com/labstack/echo/v4"
)

// LoggingMiddleware ログミドルウェア
func LoggingMiddleware(logger *otelinfra.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// リクエスト情報をログに記録
			logger.Info(c.Request().Context(), "HTTP request started", map[string]interface{}{
				"method":      c.Request().Method,
				"path":        c.Request().URL.Path,
				"remote_addr": c.Request().RemoteAddr,
				"user_agent":  c.Request().UserAgent(),
			})

			// 次のハンドラーを実行
			err := next(c)

			// レスポンス情報をログに記録
			duration := time.Since(start)
			fields := map[string]interface{}{
				"method":      c.Request().Method,
				"path":        c.Request().URL.Path,
				"status_code": c.Response().Status,
				"duration_ms": duration.Milliseconds(),
			}

			if err != nil {
				logger.Error(c.Request().Context(), "HTTP request failed", err, fields)
			} else {
				logger.Info(c.Request().Context(), "HTTP request completed", fields)
			}

			return err
		}
	}
}
