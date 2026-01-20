package handler

import (
	"bytes"
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
				"code": "TESTCODE123",
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
				// ユーザーはまだ引き換えていない
				mrcr.On("HasUserRedeemed", mock.Anything, "TESTCODE123", "user123").Return(false, nil)
				// コードを更新
				mrcr.On("Update", mock.Anything, mock.AnythingOfType("*redemption_code.RedemptionCode")).Return(nil)
				// 既存の通貨に付与
				existingCurrency := currency.NewCurrency("user123", currency.CurrencyTypePaid, 500, 1)
				mcr.On("FindByUserIDAndType", mock.Anything, "user123", currency.CurrencyTypePaid).Return(existingCurrency, nil)
				mcr.On("Save", mock.Anything, mock.MatchedBy(func(c *currency.Currency) bool {
					return c.Balance() == 1500 && c.Version() == 2
				})).Return(nil)
				// トランザクション履歴を記録
				mtr.On("Save", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil)
				// 引き換え履歴を記録
				mrcr.On("SaveRedemption", mock.Anything, mock.AnythingOfType("*redemption_code.CodeRedemption")).Return(nil)
				// トランザクション処理
				mtx.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(*sql.Tx) error")).Return(nil)
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
			name:        "異常系: user_idがトークンにない",
			tokenUserID: "",
			requestBody: map[string]interface{}{"code": "TESTCODE123"},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:        "異常系: 無効なリクエストボディ",
			tokenUserID: "user123",
			requestBody: nil,
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
				// リクエストボディが無効な場合、Bindが失敗してBadRequestが返される
				// モックは呼ばれない
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "異常系: コードが見つからない",
			tokenUserID: "user123",
			requestBody: map[string]interface{}{
				"code": "INVALIDCODE",
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
				"code": "TESTCODE123",
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
				"code": "EXPIREDCODE",
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
			} else {
				// nilの場合は無効なJSONを送る
				body = []byte("invalid json")
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

func TestCodeRedemptionHandler_CreateCode(t *testing.T) {
	tests := []struct {
		name             string
		requestBody      map[string]interface{}
		setupMock        func(*MockCurrencyRepository, *MockTransactionRepository, *MockRedemptionCodeRepository, *MockTransactionManager)
		expectedStatus   int
		validateResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "正常系: コード作成成功",
			requestBody: map[string]interface{}{
				"code":          "NEWCODE123",
				"code_type":     "promotion",
				"currency_type": "paid",
				"amount":        "1000",
				"max_uses":      100,
				"valid_from":    time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
				"valid_until":   time.Now().Add(24 * time.Hour).Format(time.RFC3339),
				"metadata":      map[string]interface{}{"campaign_id": "campaign_001"},
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
				mrcr.On("Create", mock.Anything, mock.AnythingOfType("*redemption_code.RedemptionCode")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "NEWCODE123", response["code"])
				assert.Equal(t, "promotion", response["code_type"])
				assert.Equal(t, "paid", response["currency_type"])
				assert.Equal(t, "1000", response["amount"])
				assert.Equal(t, float64(100), response["max_uses"])
				assert.Equal(t, "active", response["status"])
			},
		},
		{
			name:        "異常系: 無効なリクエストボディ",
			requestBody: nil,
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "異常系: コードが空",
			requestBody: map[string]interface{}{
				"code":          "",
				"code_type":     "promotion",
				"currency_type": "paid",
				"amount":        "1000",
				"max_uses":      100,
				"valid_from":    time.Now().Format(time.RFC3339),
				"valid_until":   time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "異常系: 無効な日付フォーマット",
			requestBody: map[string]interface{}{
				"code":          "INVALIDCODE",
				"code_type":     "promotion",
				"currency_type": "paid",
				"amount":        "1000",
				"max_uses":      100,
				"valid_from":    "invalid-date",
				"valid_until":   time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "異常系: コードが既に存在",
			requestBody: map[string]interface{}{
				"code":          "DUPLICATECODE",
				"code_type":     "promotion",
				"currency_type": "paid",
				"amount":        "1000",
				"max_uses":      100,
				"valid_from":    time.Now().Format(time.RFC3339),
				"valid_until":   time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			},
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
				mrcr.On("Create", mock.Anything, mock.AnythingOfType("*redemption_code.RedemptionCode")).Return(redemption_code.ErrCodeAlreadyExists)
			},
			expectedStatus: http.StatusConflict,
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
			} else {
				body = []byte("invalid json")
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/codes", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			middlewareFunc := restmiddleware.ErrorHandlerMiddleware(logger)
			handlerFunc := middlewareFunc(func(c echo.Context) error {
				return handler.CreateCode(c)
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

func TestCodeRedemptionHandler_DeleteCode(t *testing.T) {
	tests := []struct {
		name             string
		code             string
		setupMock        func(*MockCurrencyRepository, *MockTransactionRepository, *MockRedemptionCodeRepository, *MockTransactionManager)
		expectedStatus   int
		validateResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "正常系: コード削除成功",
			code: "DELETECODE123",
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
				code := redemption_code.NewRedemptionCode(
					"DELETECODE123",
					redemption_code.CodeTypePromotion,
					currency.CurrencyTypePaid,
					1000,
					1,
					time.Now().Add(-24*time.Hour),
					time.Now().Add(24*time.Hour),
					map[string]interface{}{},
				)
				mrcr.On("FindByCode", mock.Anything, "DELETECODE123").Return(code, nil)
				mrcr.On("Delete", mock.Anything, "DELETECODE123").Return(nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "DELETECODE123", response["code"])
				assert.NotEmpty(t, response["deleted_at"])
			},
		},
		{
			name: "異常系: コードが空",
			code: "",
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "異常系: コードが見つからない",
			code: "NOTFOUNDCODE",
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
				mrcr.On("FindByCode", mock.Anything, "NOTFOUNDCODE").Return(nil, redemption_code.ErrCodeNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "異常系: コードが使用済み（削除不可）",
			code: "USEDCODE",
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
				code := redemption_code.NewRedemptionCode(
					"USEDCODE",
					redemption_code.CodeTypePromotion,
					currency.CurrencyTypePaid,
					1000,
					1,
					time.Now().Add(-24*time.Hour),
					time.Now().Add(24*time.Hour),
					map[string]interface{}{},
				)
				mrcr.On("FindByCode", mock.Anything, "USEDCODE").Return(code, nil)
				mrcr.On("Delete", mock.Anything, "USEDCODE").Return(redemption_code.ErrCodeCannotBeDeleted)
			},
			expectedStatus: http.StatusConflict,
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

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/codes/"+tt.code, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("code")
			c.SetParamValues(tt.code)

			middlewareFunc := restmiddleware.ErrorHandlerMiddleware(logger)
			handlerFunc := middlewareFunc(func(c echo.Context) error {
				return handler.DeleteCode(c)
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

func TestCodeRedemptionHandler_GetCode(t *testing.T) {
	tests := []struct {
		name             string
		code             string
		setupMock        func(*MockCurrencyRepository, *MockTransactionRepository, *MockRedemptionCodeRepository, *MockTransactionManager)
		expectedStatus   int
		validateResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "正常系: コード取得成功",
			code: "GETCODE123",
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
				code := redemption_code.NewRedemptionCode(
					"GETCODE123",
					redemption_code.CodeTypePromotion,
					currency.CurrencyTypePaid,
					1000,
					100,
					time.Now().Add(-24*time.Hour),
					time.Now().Add(24*time.Hour),
					map[string]interface{}{"campaign_id": "campaign_001"},
				)
				code.SetCurrentUses(5)
				mrcr.On("FindByCode", mock.Anything, "GETCODE123").Return(code, nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "GETCODE123", response["code"])
				assert.Equal(t, "promotion", response["code_type"])
				assert.Equal(t, "paid", response["currency_type"])
				assert.Equal(t, "1000", response["amount"])
				assert.Equal(t, float64(100), response["max_uses"])
				assert.Equal(t, float64(5), response["current_uses"])
			},
		},
		{
			name: "異常系: コードが空",
			code: "",
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "異常系: コードが見つからない",
			code: "NOTFOUNDCODE",
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
				mrcr.On("FindByCode", mock.Anything, "NOTFOUNDCODE").Return(nil, redemption_code.ErrCodeNotFound)
			},
			expectedStatus: http.StatusNotFound,
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

			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/codes/"+tt.code, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("code")
			c.SetParamValues(tt.code)

			middlewareFunc := restmiddleware.ErrorHandlerMiddleware(logger)
			handlerFunc := middlewareFunc(func(c echo.Context) error {
				return handler.GetCode(c)
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

func TestCodeRedemptionHandler_ListCodes(t *testing.T) {
	tests := []struct {
		name             string
		queryParams      string
		setupMock        func(*MockCurrencyRepository, *MockTransactionRepository, *MockRedemptionCodeRepository, *MockTransactionManager)
		expectedStatus   int
		validateResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:        "正常系: コード一覧取得成功",
			queryParams: "limit=10&offset=0",
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
				codes := []*redemption_code.RedemptionCode{
					redemption_code.NewRedemptionCode(
						"CODE1",
						redemption_code.CodeTypePromotion,
						currency.CurrencyTypePaid,
						1000,
						100,
						time.Now().Add(-24*time.Hour),
						time.Now().Add(24*time.Hour),
						map[string]interface{}{},
					),
					redemption_code.NewRedemptionCode(
						"CODE2",
						redemption_code.CodeTypeGift,
						currency.CurrencyTypeFree,
						500,
						0,
						time.Now().Add(-12*time.Hour),
						time.Now().Add(12*time.Hour),
						map[string]interface{}{},
					),
				}
				mrcr.On("FindAll", mock.Anything, 10, 0).Return(codes, 25, nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				codes, ok := response["codes"].([]interface{})
				require.True(t, ok)
				assert.Equal(t, 2, len(codes))
				assert.Equal(t, float64(25), response["total"])
				assert.Equal(t, float64(10), response["limit"])
				assert.Equal(t, float64(0), response["offset"])
			},
		},
		{
			name:        "正常系: フィルタリング（status=active）",
			queryParams: "limit=10&offset=0&status=active",
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
				codes := []*redemption_code.RedemptionCode{
					redemption_code.NewRedemptionCode(
						"ACTIVECODE",
						redemption_code.CodeTypePromotion,
						currency.CurrencyTypePaid,
						1000,
						100,
						time.Now().Add(-24*time.Hour),
						time.Now().Add(24*time.Hour),
						map[string]interface{}{},
					),
					redemption_code.NewRedemptionCode(
						"EXPIREDCODE",
						redemption_code.CodeTypePromotion,
						currency.CurrencyTypePaid,
						1000,
						100,
						time.Now().Add(-48*time.Hour),
						time.Now().Add(-24*time.Hour),
						map[string]interface{}{},
					),
				}
				status, _ := redemption_code.NewCodeStatus("expired")
				codes[1].SetStatus(status)
				mrcr.On("FindAll", mock.Anything, 10, 0).Return(codes, 2, nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				codes, ok := response["codes"].([]interface{})
				require.True(t, ok)
				// activeのみがフィルタリングされる
				assert.Equal(t, 1, len(codes))
			},
		},
		{
			name:        "正常系: フィルタリング（code_type=promotion）",
			queryParams: "limit=10&offset=0&code_type=promotion",
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
				codes := []*redemption_code.RedemptionCode{
					redemption_code.NewRedemptionCode(
						"PROMOCODE",
						redemption_code.CodeTypePromotion,
						currency.CurrencyTypePaid,
						1000,
						100,
						time.Now().Add(-24*time.Hour),
						time.Now().Add(24*time.Hour),
						map[string]interface{}{},
					),
					redemption_code.NewRedemptionCode(
						"GIFTCODE",
						redemption_code.CodeTypeGift,
						currency.CurrencyTypeFree,
						500,
						0,
						time.Now().Add(-12*time.Hour),
						time.Now().Add(12*time.Hour),
						map[string]interface{}{},
					),
				}
				mrcr.On("FindAll", mock.Anything, 10, 0).Return(codes, 2, nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				codes, ok := response["codes"].([]interface{})
				require.True(t, ok)
				// promotionのみがフィルタリングされる
				assert.Equal(t, 1, len(codes))
			},
		},
		{
			name:        "異常系: 無効なlimitパラメータ",
			queryParams: "limit=invalid&offset=0",
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "異常系: 無効なoffsetパラメータ",
			queryParams: "limit=10&offset=invalid",
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "正常系: デフォルト値（limit/offsetなし）",
			queryParams: "",
			setupMock: func(mcr *MockCurrencyRepository, mtr *MockTransactionRepository, mrcr *MockRedemptionCodeRepository, mtx *MockTransactionManager) {
				codes := []*redemption_code.RedemptionCode{}
				mrcr.On("FindAll", mock.Anything, 50, 0).Return(codes, 0, nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, float64(50), response["limit"])
				assert.Equal(t, float64(0), response["offset"])
			},
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

			url := "/api/v1/admin/codes"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			middlewareFunc := restmiddleware.ErrorHandlerMiddleware(logger)
			handlerFunc := middlewareFunc(func(c echo.Context) error {
				return handler.ListCodes(c)
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
