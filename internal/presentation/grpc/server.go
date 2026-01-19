package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	redemptionapp "gem-server/internal/application/code_redemption"
	currencyapp "gem-server/internal/application/currency"
	historyapp "gem-server/internal/application/history"
	paymentapp "gem-server/internal/application/payment"
	"gem-server/internal/infrastructure/config"
	otelinfra "gem-server/internal/infrastructure/observability/otel"
	"gem-server/internal/presentation/grpc/handler"
	"gem-server/internal/presentation/grpc/interceptor"
	"gem-server/internal/presentation/grpc/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// Server gRPCサーバー
type Server struct {
	server   *grpc.Server
	listener net.Listener
	port     int
}

// NewServer 新しいgRPCサーバーを作成
func NewServer(
	cfg *config.Config,
	logger *otelinfra.Logger,
	currencyService *currencyapp.CurrencyApplicationService,
	paymentService *paymentapp.PaymentApplicationService,
	redemptionService *redemptionapp.CodeRedemptionApplicationService,
	historyService *historyapp.HistoryApplicationService,
) (*Server, error) {
	port := cfg.Server.Port + 1 // REST APIのポート+1を使用
	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", address, err)
	}

	return NewServerWithListener(cfg, logger, currencyService, paymentService, redemptionService, historyService, listener, port)
}

// NewServerWithListener リスナーを指定してgRPCサーバーを作成（テスト用）
func NewServerWithListener(
	cfg *config.Config,
	logger *otelinfra.Logger,
	currencyService *currencyapp.CurrencyApplicationService,
	paymentService *paymentapp.PaymentApplicationService,
	redemptionService *redemptionapp.CodeRedemptionApplicationService,
	historyService *historyapp.HistoryApplicationService,
	listener net.Listener,
	port int,
) (*Server, error) {
	// インターセプターを設定
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(interceptor.AuthInterceptor(&cfg.JWT, logger)),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     15 * time.Second,
			MaxConnectionAge:      30 * time.Second,
			MaxConnectionAgeGrace: 5 * time.Second,
			Time:                  5 * time.Second,
			Timeout:               1 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	}

	// gRPCサーバーを作成
	grpcServer := grpc.NewServer(opts...)

	// ハンドラーを登録
	currencyHandler := handler.NewCurrencyHandler(
		currencyService,
		paymentService,
		redemptionService,
		historyService,
	)
	pb.RegisterCurrencyServiceServer(grpcServer, currencyHandler)

	// リフレクションを有効化（開発環境用）
	if cfg.Environment == "development" {
		reflection.Register(grpcServer)
	}

	return &Server{
		server:   grpcServer,
		listener: listener,
		port:     port,
	}, nil
}

// Start サーバーを起動
func (s *Server) Start() error {
	log.Printf("gRPC server starting on port %d", s.port)
	if err := s.server.Serve(s.listener); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}
	return nil
}

// Stop サーバーを停止
func (s *Server) Stop(ctx context.Context) error {
	log.Println("Stopping gRPC server...")

	// グレースフルシャットダウン
	stopped := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(stopped)
	}()

	// タイムアウトを設定
	select {
	case <-stopped:
		log.Println("gRPC server stopped")
		return nil
	case <-ctx.Done():
		// タイムアウトした場合は強制停止
		log.Println("gRPC server shutdown timeout, forcing stop...")
		s.server.Stop()
		return ctx.Err()
	}
}

// Port サーバーのポート番号を返す
func (s *Server) Port() int {
	return s.port
}
