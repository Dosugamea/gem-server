package middleware

import (
	"time"

	otelinfra "gem-server/internal/infrastructure/observability/otel"

	"github.com/labstack/echo/v4"
)

// MetricsMiddleware メトリクス記録ミドルウェア
func MetricsMiddleware(metrics *otelinfra.Metrics) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// リクエスト数を記録
			metrics.RecordRequest(c.Request().Context(), c.Request().Method, c.Path())

			// 次のハンドラーを実行
			err := next(c)

			// レスポンス時間を記録（秒単位）
			duration := time.Since(start).Seconds()
			metrics.RecordResponseTime(c.Request().Context(), c.Request().Method, c.Path(), duration)

			// エラーが発生した場合はエラー数を記録
			if err != nil {
				statusCode := c.Response().Status
				// 4xx, 5xxエラーの場合のみ記録
				if statusCode >= 400 {
					errorType := "client_error"
					if statusCode >= 500 {
						errorType = "server_error"
					}
					metrics.RecordError(c.Request().Context(), errorType)
				}
			}

			return err
		}
	}
}
