package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	currencyapp "gem-server/internal/application/currency"
	redemptionapp "gem-server/internal/application/code_redemption"
	historyapp "gem-server/internal/application/history"
	paymentapp "gem-server/internal/application/payment"
	"gem-server/internal/domain/service"
	"gem-server/internal/infrastructure/config"
	otelinfra "gem-server/internal/infrastructure/observability/otel"
	"gem-server/internal/infrastructure/persistence/mysql"
	"gem-server/internal/presentation/rest"
)

func main() {
	// 設定の読み込み
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// OpenTelemetryの初期化
	tracerShutdown, err := otelinfra.InitTracer(&cfg.OpenTelemetry)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tracerShutdown(ctx); err != nil {
			log.Printf("Failed to shutdown tracer: %v", err)
		}
	}()

	meterShutdown, err := otelinfra.InitMeter(&cfg.OpenTelemetry)
	if err != nil {
		log.Fatalf("Failed to initialize meter: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := meterShutdown(ctx); err != nil {
			log.Printf("Failed to shutdown meter: %v", err)
		}
	}()

	// ロガーとメトリクスの初期化
	tracer := otelinfra.Tracer("gem-server")
	logger := otelinfra.NewLogger(tracer)
	metrics, err := otelinfra.NewMetrics("gem-server")
	if err != nil {
		log.Fatalf("Failed to create metrics: %v", err)
	}

	// データベース接続の初期化
	db, err := mysql.NewDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// リポジトリの初期化
	currencyRepo := mysql.NewCurrencyRepository(db)
	transactionRepo := mysql.NewTransactionRepository(db)
	paymentRequestRepo := mysql.NewPaymentRequestRepository(db)
	redemptionCodeRepo := mysql.NewRedemptionCodeRepository(db)

	// トランザクションマネージャーの初期化
	txManager := mysql.NewTransactionManager(db)

	// ドメインサービスの初期化
	currencyService := service.NewCurrencyService(currencyRepo)

	// アプリケーションサービスの初期化
	currencyAppService := currencyapp.NewCurrencyApplicationService(
		currencyRepo,
		transactionRepo,
		txManager,
		currencyService,
		logger,
		metrics,
	)

	paymentAppService := paymentapp.NewPaymentApplicationService(
		currencyRepo,
		transactionRepo,
		paymentRequestRepo,
		txManager,
		logger,
		metrics,
	)

	redemptionAppService := redemptionapp.NewCodeRedemptionApplicationService(
		currencyRepo,
		transactionRepo,
		redemptionCodeRepo,
		txManager,
		logger,
		metrics,
	)

	historyAppService := historyapp.NewHistoryApplicationService(
		transactionRepo,
		logger,
		metrics,
	)

	// REST APIルーターの初期化
	router, err := rest.NewRouter(
		cfg,
		logger,
		currencyAppService,
		paymentAppService,
		redemptionAppService,
		historyAppService,
	)
	if err != nil {
		log.Fatalf("Failed to create router: %v", err)
	}

	// サーバーアドレスの設定
	address := fmt.Sprintf(":%d", cfg.Server.Port)

	// グレースフルシャットダウンの設定
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// サーバーを別ゴルーチンで起動
	go func() {
		log.Printf("Server starting on %s", address)
		if err := router.Start(address); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// シグナルを待機
	<-quit
	log.Println("Shutting down server...")

	// グレースフルシャットダウン
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := router.Shutdown(); err != nil {
		log.Printf("Error shutting down server: %v", err)
	}
	_ = shutdownCtx // 将来の拡張用に保持

	log.Println("Server stopped")
}
