package rest

import (
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
}

// NewRouter 新しいRouterを作成
func NewRouter(
	cfg *config.Config,
	logger *otelinfra.Logger,
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
	setupMiddleware(e, cfg, logger)

	// ハンドラーの作成
	currencyHandler := handler.NewCurrencyHandler(currencyService)
	paymentHandler := handler.NewPaymentHandler(paymentService)
	redemptionHandler := handler.NewCodeRedemptionHandler(redemptionService)
	historyHandler := handler.NewHistoryHandler(historyService)

	// ルーティングの設定
	setupRoutes(e, cfg, logger, currencyHandler, paymentHandler, redemptionHandler, historyHandler)

	// Swagger UI / ReDoc統合
	SetupSwagger(e)

	return &Router{
		echo:              e,
		currencyHandler:   currencyHandler,
		paymentHandler:    paymentHandler,
		redemptionHandler: redemptionHandler,
		historyHandler:    historyHandler,
	}, nil
}

// setupMiddleware ミドルウェアを設定
func setupMiddleware(e *echo.Echo, cfg *config.Config, logger *otelinfra.Logger) {
	// リカバリーミドルウェア
	e.Use(middleware.Recover())

	// CORS設定
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"}, // 本番環境では適切に設定
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.OPTIONS},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	// リクエストIDの設定
	e.Use(middleware.RequestID())

	// トレーシングミドルウェア
	e.Use(restmiddleware.TracingMiddleware())

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
	currencyHandler *handler.CurrencyHandler,
	paymentHandler *handler.PaymentHandler,
	redemptionHandler *handler.CodeRedemptionHandler,
	historyHandler *handler.HistoryHandler,
) {
	// API v1グループ
	api := e.Group("/api/v1")

	// 認証が必要なエンドポイント
	authGroup := api.Group("", restmiddleware.AuthMiddleware(&cfg.JWT, logger))

	// 通貨関連エンドポイント
	authGroup.GET("/users/:user_id/balance", currencyHandler.GetBalance)
	authGroup.POST("/users/:user_id/grant", currencyHandler.GrantCurrency)
	authGroup.POST("/users/:user_id/consume", currencyHandler.ConsumeCurrency)

	// 決済関連エンドポイント
	authGroup.POST("/payment/process", paymentHandler.ProcessPayment)

	// コード引き換えエンドポイント
	authGroup.POST("/codes/redeem", redemptionHandler.RedeemCode)

	// 履歴関連エンドポイント
	authGroup.GET("/users/:user_id/transactions", historyHandler.GetTransactionHistory)

	// ヘルスチェックエンドポイント（認証不要）
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
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
