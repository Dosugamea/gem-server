package rest

import (
	"fmt"

	authapp "gem-server/internal/application/auth"
	redemptionapp "gem-server/internal/application/code_redemption"
	currencyapp "gem-server/internal/application/currency"
	historyapp "gem-server/internal/application/history"
	paymentapp "gem-server/internal/application/payment"
	"gem-server/internal/infrastructure/config"
	otelinfra "gem-server/internal/infrastructure/observability/otel"
	"gem-server/internal/presentation/rest/handler"
	restmiddleware "gem-server/internal/presentation/rest/middleware"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Router REST APIルーター
type Router struct {
	echo              *echo.Echo
	currencyHandler   *handler.CurrencyHandler
	paymentHandler    *handler.PaymentHandler
	redemptionHandler *handler.CodeRedemptionHandler
	historyHandler    *handler.HistoryHandler
	authHandler       *handler.AuthHandler
}

// NewRouter 新しいRouterを作成
func NewRouter(
	cfg *config.Config,
	logger *otelinfra.Logger,
	metrics *otelinfra.Metrics,
	authService *authapp.AuthApplicationService,
	currencyService *currencyapp.CurrencyApplicationService,
	paymentService *paymentapp.PaymentApplicationService,
	redemptionService *redemptionapp.CodeRedemptionApplicationService,
	historyService *historyapp.HistoryApplicationService,
) (*Router, error) {
	e := echo.New()

	// Echoのデフォルトエラーハンドラーを無効化（カスタムエラーハンドラーを使用）
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		// エラーハンドリングミドルウェアで処理される
	}

	// ミドルウェアの設定
	setupMiddleware(e, cfg, logger, metrics)

	// ハンドラーの作成
	authHandler := handler.NewAuthHandler(authService)
	currencyHandler := handler.NewCurrencyHandler(currencyService)
	paymentHandler := handler.NewPaymentHandler(paymentService)
	redemptionHandler := handler.NewCodeRedemptionHandler(redemptionService)
	historyHandler := handler.NewHistoryHandler(historyService)

	// ルーティングの設定
	setupRoutes(e, cfg, logger, authHandler, currencyHandler, paymentHandler, redemptionHandler, historyHandler)

	// Swagger UI / ReDoc統合
	SetupSwagger(e)

	return &Router{
		echo:              e,
		currencyHandler:   currencyHandler,
		paymentHandler:    paymentHandler,
		redemptionHandler: redemptionHandler,
		historyHandler:    historyHandler,
		authHandler:       authHandler,
	}, nil
}

// setupMiddleware ミドルウェアを設定
func setupMiddleware(e *echo.Echo, cfg *config.Config, logger *otelinfra.Logger, metrics *otelinfra.Metrics) {
	// リカバリーミドルウェア
	e.Use(middleware.Recover())

	// CORS設定
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"}, // 本番環境では適切に設定
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.OPTIONS},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	// セキュリティヘッダーの設定
	e.Use(restmiddleware.SecurityHeadersMiddleware())

	// リクエストIDの設定
	e.Use(middleware.RequestID())

	// トレーシングミドルウェア
	e.Use(restmiddleware.TracingMiddleware())

	// メトリクスミドルウェア
	e.Use(restmiddleware.MetricsMiddleware(metrics))

	// ログミドルウェア
	e.Use(restmiddleware.LoggingMiddleware(logger))

	// エラーハンドリングミドルウェア
	e.Use(restmiddleware.ErrorHandlerMiddleware(logger))
}

// setupRoutes ルーティングを設定
func setupRoutes(
	e *echo.Echo,
	cfg *config.Config,
	logger *otelinfra.Logger,
	authHandler *handler.AuthHandler,
	currencyHandler *handler.CurrencyHandler,
	paymentHandler *handler.PaymentHandler,
	redemptionHandler *handler.CodeRedemptionHandler,
	historyHandler *handler.HistoryHandler,
) {
	// Payment Handler関連の静的ファイル配信
	setupPaymentHandlerRoutes(e)

	// API v1グループ
	api := e.Group("/api/v1")

	// ユーザーAPI（JWT認証）
	userAPI := api.Group("", restmiddleware.AuthMiddleware(&cfg.JWT, logger))
	userAPI.GET("/me/balance", currencyHandler.GetBalance)
	userAPI.GET("/me/transactions", historyHandler.GetTransactionHistory)
	userAPI.POST("/payment/process", paymentHandler.ProcessPayment)
	userAPI.POST("/codes/redeem", redemptionHandler.RedeemCode)

	// 管理API（APIキー認証）
	adminAPI := api.Group("/admin", restmiddleware.APIKeyMiddleware(&cfg.AdminAPI, logger))
	adminAPI.POST("/users/:user_id/issue_token", authHandler.GenerateToken)
	adminAPI.POST("/users/:user_id/grant", currencyHandler.GrantCurrency)
	adminAPI.POST("/users/:user_id/consume", currencyHandler.ConsumeCurrency)
	adminAPI.GET("/users/:user_id/balance", currencyHandler.GetBalanceAdmin)
	adminAPI.GET("/users/:user_id/transactions", historyHandler.GetTransactionHistoryAdmin)
	
	// 引き換えコード管理API
	adminAPI.POST("/codes", redemptionHandler.CreateCode)
	adminAPI.DELETE("/codes/:code", redemptionHandler.DeleteCode)
	adminAPI.GET("/codes/:code", redemptionHandler.GetCode)
	adminAPI.GET("/codes", redemptionHandler.ListCodes)

	// ヘルスチェックエンドポイント（認証不要）
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})
}

// setupPaymentHandlerRoutes Payment Handler関連のルーティングを設定
func setupPaymentHandlerRoutes(e *echo.Echo) {
	// 静的ファイルの配信（publicディレクトリ）
	e.Static("/pay", "public/pay")

	// Payment Method Manifestへのリンクを設定
	e.GET("/pay", func(c echo.Context) error {
		// リクエストからベースURLを動的に生成
		scheme := c.Scheme()
		host := c.Request().Host
		manifestURL := fmt.Sprintf("%s://%s/pay/payment-manifest.json", scheme, host)

		// Payment Method ManifestへのリンクをHTTPヘッダーに設定
		c.Response().Header().Set("Link", fmt.Sprintf(`<%s>; rel="payment-method-manifest"`, manifestURL))
		// 決済アプリウィンドウのHTMLを返す
		return c.File("public/pay/index.html")
	})

	// Payment Method Manifestの配信
	e.GET("/pay/payment-manifest.json", func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "application/manifest+json")
		return c.File("public/pay/payment-manifest.json")
	})

	// Web App Manifestの配信
	e.GET("/pay/manifest.json", func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "application/manifest+json")
		return c.File("public/pay/manifest.json")
	})

	// Service Workerの配信（適切なContent-Typeを設定）
	e.GET("/pay/sw-payment-handler.js", func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "application/javascript")
		c.Response().Header().Set("Service-Worker-Allowed", "/pay/")
		return c.File("public/pay/sw-payment-handler.js")
	})
}

// Start サーバーを起動
func (r *Router) Start(address string) error {
	return r.echo.Start(address)
}

// Shutdown サーバーをシャットダウン
func (r *Router) Shutdown() error {
	return r.echo.Close()
}
