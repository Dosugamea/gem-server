package interceptor

import (
	"context"
	"testing"

	"gem-server/internal/infrastructure/config"
	otelinfra "gem-server/internal/infrastructure/observability/otel"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAPIKeyInterceptor(t *testing.T) {
	tests := []struct {
		name          string
		apiKey        string
		config        *config.AdminAPIConfig
		expectedCode  codes.Code
		expectedError string
	}{
		{
			name:   "正常系: 有効なAPIキー",
			apiKey: "test-api-key",
			config: &config.AdminAPIConfig{
				Enabled: true,
				APIKey:  "test-api-key",
			},
			expectedCode: codes.OK,
		},
		{
			name:   "異常系: APIキーが空",
			apiKey: "",
			config: &config.AdminAPIConfig{
				Enabled: true,
				APIKey:  "test-api-key",
			},
			expectedCode:  codes.Unauthenticated,
			expectedError: "missing X-API-Key metadata", // メタデータは存在するが、x-api-keyが空の場合
		},
		{
			name:   "異常系: 無効なAPIキー",
			apiKey: "invalid-key",
			config: &config.AdminAPIConfig{
				Enabled: true,
				APIKey:  "test-api-key",
			},
			expectedCode:  codes.Unauthenticated,
			expectedError: "invalid API key",
		},
		{
			name:   "異常系: 管理APIが無効化されている",
			apiKey: "test-api-key",
			config: &config.AdminAPIConfig{
				Enabled: false,
				APIKey:  "test-api-key",
			},
			expectedCode:  codes.PermissionDenied,
			expectedError: "admin API is disabled",
		},
		{
			name:   "異常系: メタデータが存在しない",
			apiKey: "",
			config: &config.AdminAPIConfig{
				Enabled: true,
				APIKey:  "test-api-key",
			},
			expectedCode:  codes.Unauthenticated,
			expectedError: "missing metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracer := noop.NewTracerProvider().Tracer("test")
			logger := otelinfra.NewLogger(tracer)

			interceptor := APIKeyInterceptor(tt.config, logger)

			ctx := context.Background()
			if tt.name != "異常系: メタデータが存在しない" {
				md := metadata.New(map[string]string{})
				if tt.apiKey != "" {
					md.Set("x-api-key", tt.apiKey)
				}
				ctx = metadata.NewIncomingContext(ctx, md)
			}

			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				return "success", nil
			}

			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.Test/TestMethod",
			}

			resp, err := interceptor(ctx, "test-request", info, handler)

			if tt.expectedCode == codes.OK {
				assert.NoError(t, err)
				assert.Equal(t, "success", resp)
			} else {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.expectedCode, st.Code())
				if tt.expectedError != "" {
					assert.Contains(t, st.Message(), tt.expectedError)
				}
			}
		})
	}
}
