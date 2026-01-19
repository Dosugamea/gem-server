package interceptor

import (
	"context"
	"strings"

	"gem-server/internal/infrastructure/config"
	otelinfra "gem-server/internal/infrastructure/observability/otel"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthInterceptor JWT認証インターセプター
func AuthInterceptor(cfg *config.JWTConfig, logger *otelinfra.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// メタデータからトークンを取得
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			logger.Warn(ctx, "Missing metadata", nil)
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		// Authorizationヘッダーからトークンを取得
		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			logger.Warn(ctx, "Missing authorization header", nil)
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		authHeader := authHeaders[0]

		// Bearerトークンの形式を確認
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			logger.Warn(ctx, "Invalid authorization header format", nil)
			return nil, status.Error(codes.Unauthenticated, "invalid authorization header format")
		}

		tokenString := parts[1]

		// JWTトークンの検証
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// 署名アルゴリズムの確認
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.Secret), nil
		})

		if err != nil || !token.Valid {
			logger.Warn(ctx, "Invalid token", map[string]interface{}{
				"error": err.Error(),
			})
			return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
		}

		// クレームからユーザーIDを取得
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			logger.Warn(ctx, "Invalid token claims", nil)
			return nil, status.Error(codes.Unauthenticated, "invalid token claims")
		}

		// ユーザーIDをコンテキストに設定
		userID, ok := claims["user_id"].(string)
		if !ok {
			logger.Warn(ctx, "Missing user_id in token claims", nil)
			return nil, status.Error(codes.Unauthenticated, "missing user_id in token")
		}

		// ユーザーIDをコンテキストに設定
		ctx = context.WithValue(ctx, "user_id", userID)

		// 次のハンドラーを実行
		return handler(ctx, req)
	}
}
