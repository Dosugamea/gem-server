package middleware

import (
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// TracingMiddleware OpenTelemetryトレーシングミドルウェア
func TracingMiddleware() echo.MiddlewareFunc {
	tracer := otel.Tracer("gem-server")

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()

			// トレースコンテキストの伝播
			propagator := otel.GetTextMapPropagator()
			ctx = propagator.Extract(ctx, propagation.HeaderCarrier(c.Request().Header))

			// スパンの開始
			spanName := c.Request().Method + " " + c.Path()
			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
			)
			defer span.End()

			// スパンに属性を設定
			span.SetAttributes(
				attribute.String("http.method", c.Request().Method),
				attribute.String("http.url", c.Request().URL.String()),
				attribute.String("http.route", c.Path()),
				attribute.String("http.user_agent", c.Request().UserAgent()),
			)

			// コンテキストをリクエストに設定
			c.SetRequest(c.Request().WithContext(ctx))

			// 次のハンドラーを実行
			err := next(c)

			// レスポンス情報を記録
			statusCode := c.Response().Status
			span.SetAttributes(
				attribute.Int("http.status_code", statusCode),
			)

			if err != nil {
				span.RecordError(err)
			}

			return err
		}
	}
}
