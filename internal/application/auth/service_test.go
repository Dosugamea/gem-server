package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"

	"gem-server/internal/infrastructure/config"
	otelinfra "gem-server/internal/infrastructure/observability/otel"
)

func TestAuthApplicationService_GenerateToken(t *testing.T) {
	tests := []struct {
		name      string
		req       *GenerateTokenRequest
		jwtConfig *config.JWTConfig
		wantError bool
		checkFunc func(*testing.T, *GenerateTokenResponse, error)
	}{
		{
			name: "正常系: トークンを生成",
			req: &GenerateTokenRequest{
				UserID: "user123",
			},
			jwtConfig: &config.JWTConfig{
				Secret:     "test-secret-key",
				Issuer:     "test-issuer",
				Expiration: 24 * time.Hour,
			},
			wantError: false,
			checkFunc: func(t *testing.T, resp *GenerateTokenResponse, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.Token)
				assert.Equal(t, int64(86400), resp.ExpiresIn) // 24時間 = 86400秒
				assert.Equal(t, "Bearer", resp.TokenType)
			},
		},
		{
			name: "異常系: ユーザーIDが空",
			req: &GenerateTokenRequest{
				UserID: "",
			},
			jwtConfig: &config.JWTConfig{
				Secret:     "test-secret-key",
				Issuer:     "test-issuer",
				Expiration: 24 * time.Hour,
			},
			wantError: true,
			checkFunc: func(t *testing.T, resp *GenerateTokenResponse, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "user_id is required")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracer := otel.Tracer("test")
			logger := otelinfra.NewLogger(tracer)

			svc := NewAuthApplicationService(tt.jwtConfig, logger)

			ctx := context.Background()
			got, err := svc.GenerateToken(ctx, tt.req)

			if tt.wantError {
				assert.Error(t, err)
				if tt.checkFunc != nil {
					tt.checkFunc(t, got, err)
				}
			} else {
				if tt.checkFunc != nil {
					tt.checkFunc(t, got, err)
				} else {
					require.NoError(t, err)
					assert.NotNil(t, got)
				}
			}
		})
	}
}
