package auth

import (
	"context"
	"fmt"
	"time"

	"gem-server/internal/infrastructure/config"
	otelinfra "gem-server/internal/infrastructure/observability/otel"

	"github.com/golang-jwt/jwt/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// AuthApplicationService 認証アプリケーションサービス
type AuthApplicationService struct {
	jwtConfig *config.JWTConfig
	logger    *otelinfra.Logger
}

// NewAuthApplicationService 新しいAuthApplicationServiceを作成
func NewAuthApplicationService(jwtConfig *config.JWTConfig, logger *otelinfra.Logger) *AuthApplicationService {
	return &AuthApplicationService{
		jwtConfig: jwtConfig,
		logger:    logger,
	}
}

// GenerateToken JWTトークンを生成
func (s *AuthApplicationService) GenerateToken(ctx context.Context, req *GenerateTokenRequest) (*GenerateTokenResponse, error) {
	tracer := otel.Tracer("auth-service")
	ctx, span := tracer.Start(ctx, "AuthApplicationService.GenerateToken")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", req.UserID),
	)

	// ユーザーIDのバリデーション
	if req.UserID == "" {
		err := fmt.Errorf("user_id is required")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.logger.Error(ctx, "User ID is required", err, nil)
		return nil, err
	}

	// トークンの有効期限を計算
	now := time.Now()
	expiresAt := now.Add(s.jwtConfig.Expiration)

	// JWTクレームを作成
	claims := jwt.MapClaims{
		"user_id": req.UserID,
		"iss":     s.jwtConfig.Issuer,
		"iat":     now.Unix(),
		"exp":     expiresAt.Unix(),
	}

	// JWTトークンを生成
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtConfig.Secret))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.logger.Error(ctx, "Failed to generate token", err, map[string]interface{}{
			"user_id": req.UserID,
		})
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	s.logger.Info(ctx, "Token generated successfully", map[string]interface{}{
		"user_id":    req.UserID,
		"expires_at": expiresAt.Unix(),
	})

	return &GenerateTokenResponse{
		Token:     tokenString,
		ExpiresIn: int64(s.jwtConfig.Expiration.Seconds()),
		TokenType: "Bearer",
	}, nil
}
