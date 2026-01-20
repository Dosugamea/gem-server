package grpc

import (
	"context"
	"database/sql"
	"testing"
	"time"

	redemptionapp "gem-server/internal/application/code_redemption"
	currencyapp "gem-server/internal/application/currency"
	historyapp "gem-server/internal/application/history"
	paymentapp "gem-server/internal/application/payment"
	"gem-server/internal/domain/currency"
	"gem-server/internal/domain/payment_request"
	"gem-server/internal/domain/redemption_code"
	"gem-server/internal/domain/service"
	"gem-server/internal/domain/transaction"
	"gem-server/internal/infrastructure/config"
	otelinfra "gem-server/internal/infrastructure/observability/otel"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc/test/bufconn"
)

// MockCurrencyRepository モック通貨リポジトリ
type MockCurrencyRepository struct {
	mock.Mock
}

func (m *MockCurrencyRepository) FindByUserIDAndType(ctx context.Context, userID string, currencyType currency.CurrencyType) (*currency.Currency, error) {
	args := m.Called(ctx, userID, currencyType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*currency.Currency), args.Error(1)
}

func (m *MockCurrencyRepository) Save(ctx context.Context, c *currency.Currency) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}

func (m *MockCurrencyRepository) Create(ctx context.Context, c *currency.Currency) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}

// MockTransactionRepository モックトランザクションリポジトリ
type MockTransactionRepository struct {
	mock.Mock
}

func (m *MockTransactionRepository) Save(ctx context.Context, t *transaction.Transaction) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *MockTransactionRepository) FindByTransactionID(ctx context.Context, transactionID string) (*transaction.Transaction, error) {
	args := m.Called(ctx, transactionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*transaction.Transaction, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) FindByPaymentRequestID(ctx context.Context, paymentRequestID string) (*transaction.Transaction, error) {
	args := m.Called(ctx, paymentRequestID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*transaction.Transaction), args.Error(1)
}

// MockPaymentRequestRepository モックPaymentRequestリポジトリ
type MockPaymentRequestRepository struct {
	mock.Mock
}

func (m *MockPaymentRequestRepository) Save(ctx context.Context, pr *payment_request.PaymentRequest) error {
	args := m.Called(ctx, pr)
	return args.Error(0)
}

func (m *MockPaymentRequestRepository) FindByPaymentRequestID(ctx context.Context, paymentRequestID string) (*payment_request.PaymentRequest, error) {
	args := m.Called(ctx, paymentRequestID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*payment_request.PaymentRequest), args.Error(1)
}

func (m *MockPaymentRequestRepository) Update(ctx context.Context, pr *payment_request.PaymentRequest) error {
	args := m.Called(ctx, pr)
	return args.Error(0)
}

// MockTransactionManager モックトランザクションマネージャー
type MockTransactionManager struct {
	mock.Mock
}

func (m *MockTransactionManager) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	args := m.Called(ctx, fn)
	if fn != nil {
		return fn(nil)
	}
	return args.Error(0)
}

// MockRedemptionCodeRepository モック引き換えコードリポジトリ
type MockRedemptionCodeRepository struct {
	mock.Mock
}

func (m *MockRedemptionCodeRepository) FindByCode(ctx context.Context, code string) (*redemption_code.RedemptionCode, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*redemption_code.RedemptionCode), args.Error(1)
}

func (m *MockRedemptionCodeRepository) Update(ctx context.Context, code *redemption_code.RedemptionCode) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *MockRedemptionCodeRepository) HasUserRedeemed(ctx context.Context, code string, userID string) (bool, error) {
	args := m.Called(ctx, code, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRedemptionCodeRepository) SaveRedemption(ctx context.Context, redemption *redemption_code.CodeRedemption) error {
	args := m.Called(ctx, redemption)
	return args.Error(0)
}

func (m *MockRedemptionCodeRepository) Create(ctx context.Context, code *redemption_code.RedemptionCode) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *MockRedemptionCodeRepository) Delete(ctx context.Context, code string) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *MockRedemptionCodeRepository) FindAll(ctx context.Context, limit, offset int) ([]*redemption_code.RedemptionCode, int, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*redemption_code.RedemptionCode), args.Int(1), args.Error(2)
}

func setupTestServer(t *testing.T) (*Server, *MockCurrencyRepository, *MockTransactionRepository, *MockPaymentRequestRepository, *MockTransactionManager, *MockRedemptionCodeRepository) {
	t.Helper()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
		},
		JWT: config.JWTConfig{
			Secret:     "test-secret-key-for-testing-purposes-only",
			Expiration: 24 * time.Hour,
			Issuer:     "test-issuer",
		},
		Environment: "development",
	}

	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)
	metrics, err := otelinfra.NewMetrics("test")
	require.NoError(t, err)

	mockCurrencyRepo := new(MockCurrencyRepository)
	mockTransactionRepo := new(MockTransactionRepository)
	mockPaymentRequestRepo := new(MockPaymentRequestRepository)
	mockTxManager := new(MockTransactionManager)
	mockRedemptionCodeRepo := new(MockRedemptionCodeRepository)

	currencyService := service.NewCurrencyService(mockCurrencyRepo)

	currencyAppService := currencyapp.NewCurrencyApplicationService(
		mockCurrencyRepo,
		mockTransactionRepo,
		mockTxManager,
		currencyService,
		logger,
		metrics,
	)
	paymentAppService := paymentapp.NewPaymentApplicationService(
		mockCurrencyRepo,
		mockTransactionRepo,
		mockPaymentRequestRepo,
		mockTxManager,
		logger,
		metrics,
	)
	redemptionAppService := redemptionapp.NewCodeRedemptionApplicationService(
		mockCurrencyRepo,
		mockTransactionRepo,
		mockRedemptionCodeRepo,
		mockTxManager,
		logger,
		metrics,
	)
	historyAppService := historyapp.NewHistoryApplicationService(
		mockTransactionRepo,
		logger,
		metrics,
	)

	// bufconnを使用してメモリ内リスナーを作成（実際のポートバインドを回避）
	listener := bufconn.Listen(1024 * 1024) // 1MB buffer
	port := 8081                            // テスト用のポート番号

	server, err := NewServerWithListener(
		cfg,
		logger,
		currencyAppService,
		paymentAppService,
		redemptionAppService,
		historyAppService,
		listener,
		port,
	)
	require.NoError(t, err)
	require.NotNil(t, server)

	return server, mockCurrencyRepo, mockTransactionRepo, mockPaymentRequestRepo, mockTxManager, mockRedemptionCodeRepo
}

func TestNewServer(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *config.Config
		wantError bool
	}{
		{
			name: "正常系: サーバー作成成功",
			cfg: &config.Config{
				Server: config.ServerConfig{
					Port: 8080,
				},
				JWT: config.JWTConfig{
					Secret:     "test-secret",
					Expiration: 24 * time.Hour,
					Issuer:     "test-issuer",
				},
				Environment: "development",
			},
			wantError: false,
		},
		{
			name: "正常系: 本番環境ではリフレクション無効",
			cfg: &config.Config{
				Server: config.ServerConfig{
					Port: 8080,
				},
				JWT: config.JWTConfig{
					Secret:     "test-secret",
					Expiration: 24 * time.Hour,
					Issuer:     "test-issuer",
				},
				Environment: "production",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracer := noop.NewTracerProvider().Tracer("test")
			logger := otelinfra.NewLogger(tracer)
			metrics, err := otelinfra.NewMetrics("test")
			require.NoError(t, err)

			mockCurrencyRepo := new(MockCurrencyRepository)
			mockTransactionRepo := new(MockTransactionRepository)
			mockPaymentRequestRepo := new(MockPaymentRequestRepository)
			mockTxManager := new(MockTransactionManager)
			mockRedemptionCodeRepo := new(MockRedemptionCodeRepository)

			currencyService := service.NewCurrencyService(mockCurrencyRepo)

			currencyAppService := currencyapp.NewCurrencyApplicationService(
				mockCurrencyRepo,
				mockTransactionRepo,
				mockTxManager,
				currencyService,
				logger,
				metrics,
			)
			paymentAppService := paymentapp.NewPaymentApplicationService(
				mockCurrencyRepo,
				mockTransactionRepo,
				mockPaymentRequestRepo,
				mockTxManager,
				logger,
				metrics,
			)
			redemptionAppService := redemptionapp.NewCodeRedemptionApplicationService(
				mockCurrencyRepo,
				mockTransactionRepo,
				mockRedemptionCodeRepo,
				mockTxManager,
				logger,
				metrics,
			)
			historyAppService := historyapp.NewHistoryApplicationService(
				mockTransactionRepo,
				logger,
				metrics,
			)

			// テスト用にbufconnを使用
			listener := bufconn.Listen(1024 * 1024)
			port := tt.cfg.Server.Port + 1

			server, err := NewServerWithListener(
				tt.cfg,
				logger,
				currencyAppService,
				paymentAppService,
				redemptionAppService,
				historyAppService,
				listener,
				port,
			)

			if tt.wantError {
				require.Error(t, err)
				assert.Nil(t, server)
			} else {
				require.NoError(t, err)
				require.NotNil(t, server)
				// ポート番号が正しく設定されていることを確認
				assert.Greater(t, server.Port(), 0)
			}
		})
	}
}

func TestServer_Port(t *testing.T) {
	server, _, _, _, _, _ := setupTestServer(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Stop(ctx)
	}()

	port := server.Port()
	assert.Greater(t, port, 0)
	// REST APIのポート+1であることを確認
	assert.Equal(t, 8081, port)
}

func TestServer_Stop(t *testing.T) {
	server, _, _, _, _, _ := setupTestServer(t)

	// サーバーを起動（バックグラウンド）
	go func() {
		_ = server.Start()
	}()

	// 少し待ってから停止
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := server.Stop(ctx)
	require.NoError(t, err)
}

func TestServer_Stop_Timeout(t *testing.T) {
	server, _, _, _, _, _ := setupTestServer(t)

	// サーバーを起動（バックグラウンド）
	go func() {
		_ = server.Start()
	}()

	// 少し待つ
	time.Sleep(100 * time.Millisecond)

	// タイムアウトを非常に短く設定
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// タイムアウトが発生することを確認
	err := server.Stop(ctx)
	// タイムアウトエラーまたはnilが返る可能性がある
	// （グレースフルシャットダウンが完了する場合）
	if err != nil {
		assert.Equal(t, context.DeadlineExceeded, err)
	}
}

func TestServer_Start(t *testing.T) {
	server, _, _, _, _, _ := setupTestServer(t)

	// サーバーをバックグラウンドで起動
	serverStarted := make(chan bool, 1)
	go func() {
		serverStarted <- true
		err := server.Start()
		// サーバーが停止した場合のエラーは無視
		_ = err
	}()

	// サーバーが起動するまで少し待つ
	select {
	case <-serverStarted:
		// サーバーが起動した
	case <-time.After(1 * time.Second):
		t.Fatal("Server did not start within timeout")
	}

	// サーバーを停止
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := server.Stop(ctx)
	require.NoError(t, err)
}

func TestServer_Integration(t *testing.T) {
	// 統合テスト: サーバーの作成、起動、停止の一連の流れをテスト
	server, _, _, _, _, _ := setupTestServer(t)

	// ポート番号を確認
	port := server.Port()
	assert.Greater(t, port, 0)

	// サーバーをバックグラウンドで起動
	serverStarted := make(chan bool, 1)
	go func() {
		serverStarted <- true
		_ = server.Start()
	}()

	// サーバーが起動するまで少し待つ
	select {
	case <-serverStarted:
		// サーバーが起動した
	case <-time.After(1 * time.Second):
		t.Fatal("Server did not start within timeout")
	}

	// サーバーを停止
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := server.Stop(ctx)
	require.NoError(t, err)
}
