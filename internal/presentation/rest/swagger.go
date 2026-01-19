package rest

import (
	"gem-server/internal/presentation/openapi"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// SetupSwagger Swagger UI / ReDoc統合を設定
func SetupSwagger(e *echo.Echo) {
	// OpenAPI仕様ファイルの配信
	e.GET("/openapi.yaml", func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "application/x-yaml")
		return c.Blob(200, "application/x-yaml", openapi.Spec)
	})

	// Swagger UI用の静的ファイル配信（簡易版）
	// 実際の実装では、swaggo/echo-swaggerなどのライブラリを使用することを推奨
	e.GET("/swagger", func(c echo.Context) error {
		return c.HTML(200, `
<!DOCTYPE html>
<html>
<head>
	<title>API Documentation</title>
	<link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.0.0/swagger-ui.css" />
	<style>
		html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
		*, *:before, *:after { box-sizing: inherit; }
		body { margin:0; background: #fafafa; }
	</style>
</head>
<body>
	<div id="swagger-ui"></div>
	<script src="https://unpkg.com/swagger-ui-dist@5.0.0/swagger-ui-bundle.js"></script>
	<script>
		window.onload = function() {
			SwaggerUIBundle({
				url: "/openapi.yaml",
				dom_id: '#swagger-ui',
				presets: [
					SwaggerUIBundle.presets.apis,
					SwaggerUIBundle.presets.standalone
				],
				layout: "StandaloneLayout"
			});
		};
	</script>
</body>
</html>
		`)
	})

	// ReDoc用のHTML
	e.GET("/redoc", func(c echo.Context) error {
		return c.HTML(200, `
<!DOCTYPE html>
<html>
<head>
	<title>API Documentation - ReDoc</title>
	<meta charset="utf-8"/>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
	<style>
		body { margin: 0; padding: 0; }
	</style>
</head>
<body>
	<redoc spec-url="/openapi.yaml"></redoc>
	<script src="https://cdn.jsdelivr.net/npm/redoc@latest/bundles/redoc.standalone.js"></script>
</body>
</html>
		`)
	})

	// CORS設定（Swagger UI用）
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.OPTIONS},
	}))
}
