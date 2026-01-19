package interceptor

import (
	"context"
	"testing"

	"gem-server/internal/infrastructure/config"
	otelinfra "gem-server/internal/infrastructure/observability/otel"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthInterceptor_MissingMetadata(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret: "test-secret",
	}
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	interceptor := AuthInterceptor(cfg, logger)

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/currency.CurrencyService/GetBalance",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	_, err := interceptor(ctx, nil, info, handler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "missing metadata")
}

func TestAuthInterceptor_MissingAuthorizationHeader(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret: "test-secret",
	}
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	interceptor := AuthInterceptor(cfg, logger)

	md := metadata.New(map[string]string{})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	info := &grpc.UnaryServerInfo{
		FullMethod: "/currency.CurrencyService/GetBalance",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	_, err := interceptor(ctx, nil, info, handler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "missing authorization header")
}

func TestAuthInterceptor_InvalidAuthorizationHeaderFormat(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret: "test-secret",
	}
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	interceptor := AuthInterceptor(cfg, logger)

	md := metadata.New(map[string]string{
		"authorization": "InvalidFormat token",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	info := &grpc.UnaryServerInfo{
		FullMethod: "/currency.CurrencyService/GetBalance",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	_, err := interceptor(ctx, nil, info, handler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "invalid authorization header format")
}

func TestAuthInterceptor_InvalidToken(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret: "test-secret",
	}
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	interceptor := AuthInterceptor(cfg, logger)

	md := metadata.New(map[string]string{
		"authorization": "Bearer invalid-token",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	info := &grpc.UnaryServerInfo{
		FullMethod: "/currency.CurrencyService/GetBalance",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	_, err := interceptor(ctx, nil, info, handler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "invalid or expired token")
}

func TestAuthInterceptor_ValidToken(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret: "test-secret",
	}
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	// 有効なJWTトークンを生成
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "user123",
	})
	tokenString, err := token.SignedString([]byte(cfg.Secret))
	require.NoError(t, err)

	interceptor := AuthInterceptor(cfg, logger)

	md := metadata.New(map[string]string{
		"authorization": "Bearer " + tokenString,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	info := &grpc.UnaryServerInfo{
		FullMethod: "/currency.CurrencyService/GetBalance",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// ユーザーIDが設定されていることを確認
		userID, ok := ctx.Value("user_id").(string)
		assert.True(t, ok)
		assert.Equal(t, "user123", userID)
		return "success", nil
	}

	result, err := interceptor(ctx, nil, info, handler)
	require.NoError(t, err)
	assert.Equal(t, "success", result)
}

func TestAuthInterceptor_MissingUserIDInClaims(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret: "test-secret",
	}
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	// user_idがないトークンを生成
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"other_claim": "value",
	})
	tokenString, err := token.SignedString([]byte(cfg.Secret))
	require.NoError(t, err)

	interceptor := AuthInterceptor(cfg, logger)

	md := metadata.New(map[string]string{
		"authorization": "Bearer " + tokenString,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	info := &grpc.UnaryServerInfo{
		FullMethod: "/currency.CurrencyService/GetBalance",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	_, err = interceptor(ctx, nil, info, handler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "missing user_id in token")
}

func TestAuthInterceptor_InvalidUserIDType(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret: "test-secret",
	}
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	// user_idが文字列でないトークンを生成
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 123, // 数値型
	})
	tokenString, err := token.SignedString([]byte(cfg.Secret))
	require.NoError(t, err)

	interceptor := AuthInterceptor(cfg, logger)

	md := metadata.New(map[string]string{
		"authorization": "Bearer " + tokenString,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	info := &grpc.UnaryServerInfo{
		FullMethod: "/currency.CurrencyService/GetBalance",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	_, err = interceptor(ctx, nil, info, handler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "missing user_id in token")
}

func TestAuthInterceptor_WrongSecret(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret: "test-secret",
	}
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	// 異なるシークレットでトークンを生成
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "user123",
	})
	tokenString, err := token.SignedString([]byte("wrong-secret"))
	require.NoError(t, err)

	interceptor := AuthInterceptor(cfg, logger)

	md := metadata.New(map[string]string{
		"authorization": "Bearer " + tokenString,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	info := &grpc.UnaryServerInfo{
		FullMethod: "/currency.CurrencyService/GetBalance",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	_, err = interceptor(ctx, nil, info, handler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "invalid or expired token")
}

func TestAuthInterceptor_InvalidTokenClaims(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret: "test-secret",
	}
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	interceptor := AuthInterceptor(cfg, logger)

	md := metadata.New(map[string]string{
		"authorization": "Bearer invalid.token.here",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	info := &grpc.UnaryServerInfo{
		FullMethod: "/currency.CurrencyService/GetBalance",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	_, err := interceptor(ctx, nil, info, handler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuthInterceptor_MultipleAuthorizationHeaders(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret: "test-secret",
	}
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := otelinfra.NewLogger(tracer)

	// 有効なJWTトークンを生成
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "user123",
	})
	tokenString, err := token.SignedString([]byte(cfg.Secret))
	require.NoError(t, err)

	interceptor := AuthInterceptor(cfg, logger)

	// 複数のauthorizationヘッダーがある場合、最初のものを使用
	md := metadata.New(map[string]string{
		"authorization": "Bearer " + tokenString,
	})
	md.Append("authorization", "Bearer invalid-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	info := &grpc.UnaryServerInfo{
		FullMethod: "/currency.CurrencyService/GetBalance",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// ユーザーIDが設定されていることを確認
		userID, ok := ctx.Value("user_id").(string)
		assert.True(t, ok)
		assert.Equal(t, "user123", userID)
		return "success", nil
	}

	result, err := interceptor(ctx, nil, info, handler)
	require.NoError(t, err)
	assert.Equal(t, "success", result)
}
