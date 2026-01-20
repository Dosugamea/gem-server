package interceptor

import (
	"context"
	"strings"

	"gem-server/internal/infrastructure/config"
	otelinfra "gem-server/internal/infrastructure/observability/otel"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// APIKeyInterceptor APIキー認証インターセプター
func APIKeyInterceptor(cfg *config.AdminAPIConfig, logger *otelinfra.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// 管理APIが無効化されている場合はエラー
		if !cfg.Enabled {
			logger.Warn(ctx, "Admin API is disabled", nil)
			return nil, status.Error(codes.PermissionDenied, "admin API is disabled")
		}

		// メタデータからAPIキーを取得
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			logger.Warn(ctx, "Missing metadata", nil)
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		// X-API-KeyメタデータからAPIキーを取得
		apiKeys := md.Get("x-api-key")
		if len(apiKeys) == 0 {
			logger.Warn(ctx, "Missing X-API-Key metadata", nil)
			return nil, status.Error(codes.Unauthenticated, "missing X-API-Key metadata")
		}

		apiKey := apiKeys[0]

		// APIキーの検証
		if apiKey != cfg.APIKey {
			logger.Warn(ctx, "Invalid API key", nil)
			return nil, status.Error(codes.Unauthenticated, "invalid API key")
		}

		// IP制限のチェック（設定されている場合）
		if len(cfg.AllowedIPs) > 0 {
			clientIP := getClientIPFromMetadata(md)
			if clientIP != "" && !isIPAllowed(clientIP, cfg.AllowedIPs) {
				logger.Warn(ctx, "IP address not allowed", map[string]interface{}{
					"ip": clientIP,
				})
				return nil, status.Error(codes.PermissionDenied, "IP address not allowed")
			}
		}

		// 次のハンドラーを実行
		return handler(ctx, req)
	}
}

// getClientIPFromMetadata メタデータからクライアントのIPアドレスを取得
func getClientIPFromMetadata(md metadata.MD) string {
	// X-Forwarded-Forメタデータから取得
	forwardedFor := md.Get("x-forwarded-for")
	if len(forwardedFor) > 0 {
		// カンマ区切りの最初のIPを取得
		ips := strings.Split(forwardedFor[0], ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// X-Real-IPメタデータから取得
	realIP := md.Get("x-real-ip")
	if len(realIP) > 0 {
		return realIP[0]
	}

	return ""
}

// isIPAllowed IPアドレスが許可リストに含まれているかチェック
func isIPAllowed(ip string, allowedIPs []string) bool {
	for _, allowedIP := range allowedIPs {
		if ip == allowedIP {
			return true
		}
		// CIDR表記のサポート（簡易版）
		if strings.Contains(allowedIP, "/") {
			// CIDRマッチングの実装は必要に応じて追加
			// ここでは簡易的にプレフィックスマッチのみ
			if strings.HasPrefix(ip, strings.Split(allowedIP, "/")[0]) {
				return true
			}
		}
	}
	return false
}
