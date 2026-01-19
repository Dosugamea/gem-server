package rest

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// SetupSwagger Swagger UI / ReDoc統合を設定
func SetupSwagger(e *echo.Echo) {
	// Swagger UI（swaggo/echo-swaggerを使用）
	// swag initで生成されたドキュメントを使用
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// ReDoc用のHTML（swagで生成されたdoc.jsonを使用）
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
	<redoc spec-url="/swagger/doc.json"></redoc>
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
