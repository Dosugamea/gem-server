package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	redemptionapp "gem-server/internal/application/code_redemption"
	"gem-server/internal/domain/currency"
	"gem-server/internal/domain/redemption_code"
	otelinfra "gem-server/internal/infrastructure/observability/otel"
	restmiddleware "gem-server/internal/presentation/rest/middleware"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestCodeRedemptionHandler_RedeemCode(t *testing.T) {
	tests := []struct {
		name             string
		tokenUserID      string
		requestBody      map[string]interface{}
		setupMock        func(*MockCurrencyRepository, *MockTransactionRepository, *MockRedemptionCodeRepository, *MockTransactionManager)
		expectedStatus   int
		validateResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:        "正常系: コード引き換え成功",
			tokenUserID: "user123",
			requestBody: map[string]interface{}{
				"code":    "TESTCODE123",
				"user_id": "user123",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
				// コードを取得
				code := redemption_code.NewRedemptionCode(
					"TESTCODE123",
					redemption_code.CodeTypePromotion,
					currency.CurrencyTypePaid,
					1000,
					1,
					time.Now().Add(-24*time.Hour),
					time.Now().Add(24*time.Hour),
					map[string]interface{}{},
				)
				mrcr.On("FindByCode", mock.Anything, "TESTCODE123").Return(code, nil)
				mrcr.On("HasUserRedeemed", mock.Anything, "TESTCODE123", "user123").Return(false, nil)

				// コード更新（WithTransaction内で呼ばれる）
				mrcr.On("Update", mock.Anything, mock.Anything).Return(nil)

				// 通貨取得・作成（WithTransaction内で呼ばれる）
				paidCurrency := currency.NewCurrency("user123", currency.CurrencyTypePaid, 500, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(paidCurrency, nil)
				mcr.On("Save", mock.Anything, mock.Anything).Return(nil)

				// トランザクション保存（WithTransaction内で呼ばれる）
				mtr.On("Save", mock.Anything, mock.Anything).Return(nil)

				// 引き換え履歴保存（WithTransaction内で呼ばれる）
				mrcr.On("SaveRedemption", mock.Anything, mock.Anything).Return(nil)

				// トランザクション処理（最後に設定）
				mtx.On("WithTransaction", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					fn := args.Get(1).(func(*sql.Tx) error)
					_ = fn(nil)
				})
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response["redemption_id"])
				assert.NotEmpty(t, response["transaction_id"])
				assert.Equal(t, "TESTCODE123", response["code"])
				assert.Equal(t, "paid", response["currency_type"])
				assert.Equal(t, "1000", response["amount"])
				assert.Equal(t, "completed", response["status"])
			},
		},
		{
			name:        "異常系: 無効なリクエストボディ",
			tokenUserID: "user123",
			requestBody: nil,
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
			},
			expectedStatus: http.StatusForbidden, // c.Bindが失敗するとreqBody.UserIDが空になり、user_id mismatchが発生
		},
		{
			name:        "異常系: user_id不一致",
			tokenUserID: "user456",
			requestBody: map[string]interface{}{
				"code":    "TESTCODE123",
				"user_id": "user123",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:        "異常系: コードが見つからない",
			tokenUserID: "user123",
			requestBody: map[string]interface{}{
				"code":    "INVALIDCODE",
				"user_id": "user123",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
				mrcr.On("FindByCode", mock.Anything, "INVALIDCODE").Return(nil, redemption_code.ErrCodeNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:        "異常系: ユーザーが既に引き換え済み",
			tokenUserID: "user123",
			requestBody: map[string]interface{}{
				"code":    "TESTCODE123",
				"user_id": "user123",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
				code := redemption_code.NewRedemptionCode(
					"TESTCODE123",
					redemption_code.CodeTypePromotion,
					currency.CurrencyTypePaid,
					1000,
					1,
					time.Now().Add(-24*time.Hour),
					time.Now().Add(24*time.Hour),
					map[string]interface{}{},
				)
				mrcr.On("FindByCode", mock.Anything, "TESTCODE123").Return(code, nil)
				mrcr.On("HasUserRedeemed", mock.Anything, "TESTCODE123", "user123").Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest, // ErrUserAlreadyRedeemedは400を返す
		},
		{
			name:        "異常系: コードが無効",
			tokenUserID: "user123",
			requestBody: map[string]interface{}{
				"code":    "EXPIREDCODE",
				"user_id": "user123",
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
				// 期限切れのコード
				code := redemption_code.NewRedemptionCode(
					"EXPIREDCODE",
					redemption_code.CodeTypePromotion,
					currency.CurrencyTypePaid,
					1000,
					1,
					time.Now().Add(-48*time.Hour),
					time.Now().Add(-24*time.Hour), // 期限切れ
					map[string]interface{}{},
				)
				mrcr.On("FindByCode", mock.Anything, "EXPIREDCODE").Return(code, nil)
				mrcr.On("HasUserRedeemed", mock.Anything, "EXPIREDCODE", "user123").Return(false, nil)
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			mockCurrencyRepo := new(MockCurrencyRepository)
			mockTransactionRepo := new(MockTransactionRepository)
			mockRedemptionCodeRepo := new(MockRedemptionCodeRepository)
			mockTxManager := new(MockTransactionManager)
			tracer := noop.NewTracerProvider().Tracer("test")
			logger := otelinfra.NewLogger(tracer)
			metrics, _ := otelinfra.NewMetrics("test")

			// エラーハンドリングミドルウェアを設定
			e.Use(restmiddleware.ErrorHandlerMiddleware(logger))

			tt.setupMock(mockCurrencyRepo, mockTransactionRepo, mockRedemptionCodeRepo, mockTxManager)

			appService := redemptionapp.NewCodeRedemptionApplicationService(
				mockCurrencyRepo,
				mockTransactionRepo,
				mockRedemptionCodeRepo,
				mockTxManager,
				logger,
				metrics,
			)

			handler := NewCodeRedemptionHandler(appService)

			var body []byte
			if tt.requestBody != nil {
				body, _ = json.Marshal(tt.requestBody)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/codes/redeem", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			if tt.tokenUserID != "" {
				c.Set("user_id", tt.tokenUserID)
			}

			// ミドルウェアを手動で実行
			middlewareFunc := restmiddleware.ErrorHandlerMiddleware(logger)
			handlerFunc := middlewareFunc(func(c echo.Context) error {
				return handler.RedeemCode(c)
			})
			err := handlerFunc(c)
			if err != nil {
				e.HTTPErrorHandler(err, c)
			}
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.validateResponse != nil {
				tt.validateResponse(t, rec)
			}
		})
	}
}
