package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityHeadersMiddleware_SetsAllHeaders(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/test")

	middleware := SecurityHeadersMiddleware()
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// XSS保護ヘッダーの確認
	assert.Equal(t, "1; mode=block", rec.Header().Get("X-XSS-Protection"))

	// クリックジャッキング保護ヘッダーの確認
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))

	// MIMEタイプスニッフィング保護ヘッダーの確認
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))

	// コンテンツセキュリティポリシーの確認
	csp := rec.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "default-src 'self'")
	assert.Contains(t, csp, "script-src 'self' 'unsafe-inline'")
	assert.Contains(t, csp, "style-src 'self' 'unsafe-inline'")

	// Referrer-Policyの確認
	assert.Equal(t, "strict-origin-when-cross-origin", rec.Header().Get("Referrer-Policy"))
}

func TestSecurityHeadersMiddleware_SetsHSTSHeaderForHTTPS(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "https://example.com/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/test")

	middleware := SecurityHeadersMiddleware()
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.NoError(t, err)

	// HTTPSリクエストの場合、Strict-Transport-Securityヘッダーが設定される
	hsts := rec.Header().Get("Strict-Transport-Security")
	assert.Contains(t, hsts, "max-age=31536000")
	assert.Contains(t, hsts, "includeSubDomains")
}

func TestSecurityHeadersMiddleware_NoHSTSHeaderForHTTP(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/test")

	middleware := SecurityHeadersMiddleware()
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.NoError(t, err)

	// HTTPリクエストの場合、Strict-Transport-Securityヘッダーは設定されない
	hsts := rec.Header().Get("Strict-Transport-Security")
	assert.Empty(t, hsts)
}

func TestSecurityHeadersMiddleware_WorksWithDifferentHTTPMethods(t *testing.T) {
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(method, "/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/test")

			middleware := SecurityHeadersMiddleware()
			handler := middleware(func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})

			err := handler(c)
			require.NoError(t, err)

			// すべてのHTTPメソッドでセキュリティヘッダーが設定される
			assert.Equal(t, "1; mode=block", rec.Header().Get("X-XSS-Protection"))
			assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
			assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
		})
	}
}

func TestSecurityHeadersMiddleware_HeadersSetBeforeResponse(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/test")

	middleware := SecurityHeadersMiddleware()
	handler := middleware(func(c echo.Context) error {
		// ハンドラー内でヘッダーが既に設定されていることを確認
		assert.Equal(t, "1; mode=block", c.Response().Header().Get("X-XSS-Protection"))
		assert.Equal(t, "DENY", c.Response().Header().Get("X-Frame-Options"))
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSecurityHeadersMiddleware_ErrorHandling(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/test")

	middleware := SecurityHeadersMiddleware()
	testErr := echo.NewHTTPError(http.StatusInternalServerError, "test error")
	handler := middleware(func(c echo.Context) error {
		return testErr
	})

	err := handler(c)
	assert.Error(t, err)

	// エラーが発生してもセキュリティヘッダーは設定される
	assert.Equal(t, "1; mode=block", rec.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
}

func TestSecurityHeadersMiddleware_SwaggerPath(t *testing.T) {
	swaggerPaths := []string{"/swagger", "/redoc", "/openapi.yaml"}

	for _, path := range swaggerPaths {
		t.Run(path, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath(path)

			middleware := SecurityHeadersMiddleware()
			handler := middleware(func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})

			err := handler(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)

			// Swaggerパスでは外部CDNが許可されるCSPが設定される
			csp := rec.Header().Get("Content-Security-Policy")
			assert.Contains(t, csp, "https://unpkg.com")
			assert.Contains(t, csp, "https://cdn.jsdelivr.net")
			assert.Contains(t, csp, "https://fonts.googleapis.com")
		})
	}
}

func TestSecurityHeadersMiddleware_NonSwaggerPath(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/123/balance", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/users/123/balance")

	middleware := SecurityHeadersMiddleware()
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// 通常のAPIパスでは外部CDNが許可されないCSPが設定される
	csp := rec.Header().Get("Content-Security-Policy")
	assert.NotContains(t, csp, "https://unpkg.com")
	assert.NotContains(t, csp, "https://cdn.jsdelivr.net")
	assert.Contains(t, csp, "default-src 'self'")
	assert.Contains(t, csp, "script-src 'self' 'unsafe-inline'")
	assert.Contains(t, csp, "style-src 'self' 'unsafe-inline'")
}
