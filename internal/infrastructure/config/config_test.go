package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func()
		cleanupEnv  func()
		wantError   bool
		checkConfig func(*testing.T, *Config)
	}{
		{
			name: "正常系: デフォルト値で設定を読み込む",
			setupEnv: func() {
				os.Setenv("DB_HOST", "localhost")
				os.Setenv("DB_NAME", "test_db")
				os.Setenv("JWT_SECRET", "test-secret")
			},
			cleanupEnv: func() {
				os.Unsetenv("DB_HOST")
				os.Unsetenv("DB_NAME")
				os.Unsetenv("JWT_SECRET")
			},
			wantError: false,
			checkConfig: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "localhost", cfg.Database.Host)
				assert.Equal(t, "test_db", cfg.Database.Database)
				assert.Equal(t, "test-secret", cfg.JWT.Secret)
				assert.Equal(t, 8080, cfg.Server.Port)
				assert.Equal(t, 3306, cfg.Database.Port)
			},
		},
		{
			name: "正常系: 環境変数から設定を読み込む",
			setupEnv: func() {
				os.Setenv("ENVIRONMENT", "production")
				os.Setenv("SERVER_PORT", "9000")
				os.Setenv("DB_HOST", "db.example.com")
				os.Setenv("DB_PORT", "3307")
				os.Setenv("DB_NAME", "prod_db")
				os.Setenv("JWT_SECRET", "prod-secret")
				os.Setenv("JWT_EXPIRATION", "12h")
			},
			cleanupEnv: func() {
				os.Unsetenv("ENVIRONMENT")
				os.Unsetenv("SERVER_PORT")
				os.Unsetenv("DB_HOST")
				os.Unsetenv("DB_PORT")
				os.Unsetenv("DB_NAME")
				os.Unsetenv("JWT_SECRET")
				os.Unsetenv("JWT_EXPIRATION")
			},
			wantError: false,
			checkConfig: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "production", cfg.Environment)
				assert.Equal(t, 9000, cfg.Server.Port)
				assert.Equal(t, "db.example.com", cfg.Database.Host)
				assert.Equal(t, 3307, cfg.Database.Port)
				assert.Equal(t, "prod_db", cfg.Database.Database)
				assert.Equal(t, "prod-secret", cfg.JWT.Secret)
				assert.Equal(t, 12*time.Hour, cfg.JWT.Expiration)
			},
		},
		{
			name: "正常系: DB_HOSTが未設定でデフォルト値が使われる",
			setupEnv: func() {
				os.Unsetenv("DB_HOST")
				os.Setenv("DB_NAME", "test_db")
				os.Setenv("JWT_SECRET", "test-secret")
			},
			cleanupEnv: func() {
				os.Unsetenv("DB_NAME")
				os.Unsetenv("JWT_SECRET")
			},
			wantError: false,
			checkConfig: func(t *testing.T, cfg *Config) {
				// デフォルト値が使われていることを確認
				assert.Equal(t, "localhost", cfg.Database.Host)
			},
		},
		{
			name: "正常系: DB_NAMEが未設定でデフォルト値が使われる",
			setupEnv: func() {
				os.Setenv("DB_HOST", "localhost")
				os.Unsetenv("DB_NAME")
				os.Setenv("JWT_SECRET", "test-secret")
			},
			cleanupEnv: func() {
				os.Unsetenv("DB_HOST")
				os.Unsetenv("JWT_SECRET")
			},
			wantError: false,
			checkConfig: func(t *testing.T, cfg *Config) {
				// デフォルト値が使われていることを確認
				assert.Equal(t, "gem_db", cfg.Database.Database)
			},
		},
		{
			name: "異常系: JWT_SECRETが空",
			setupEnv: func() {
				os.Setenv("DB_HOST", "localhost")
				os.Setenv("DB_NAME", "test_db")
			},
			cleanupEnv: func() {
				os.Unsetenv("DB_HOST")
				os.Unsetenv("DB_NAME")
			},
			wantError:   true,
			checkConfig: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanupEnv()

			cfg, err := Load()

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, cfg)
				if tt.checkConfig != nil {
					tt.checkConfig(t, cfg)
				}
			}
		})
	}
}

func TestDatabaseConfig_DSN(t *testing.T) {
	cfg := DatabaseConfig{
		User:     "testuser",
		Password: "testpass",
		Host:     "localhost",
		Port:     3306,
		Database: "testdb",
	}

	dsn := cfg.DSN()
	assert.Contains(t, dsn, "testuser")
	assert.Contains(t, dsn, "testpass")
	assert.Contains(t, dsn, "localhost")
	assert.Contains(t, dsn, "3306")
	assert.Contains(t, dsn, "testdb")
}

func TestRedisConfig_Address(t *testing.T) {
	cfg := RedisConfig{
		Host: "redis.example.com",
		Port: 6379,
	}

	address := cfg.Address()
	assert.Equal(t, "redis.example.com:6379", address)
}

func TestGetEnvAsInt(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue int
		want         int
	}{
		{
			name:         "環境変数が設定されている",
			envValue:     "123",
			defaultValue: 0,
			want:         123,
		},
		{
			name:         "環境変数が空",
			envValue:     "",
			defaultValue: 456,
			want:         456,
		},
		{
			name:         "環境変数が無効な値",
			envValue:     "invalid",
			defaultValue: 789,
			want:         789,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("TEST_INT", tt.envValue)
			defer os.Unsetenv("TEST_INT")

			got := getEnvAsInt("TEST_INT", tt.defaultValue)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetEnvAsBool(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue bool
		want         bool
	}{
		{
			name:         "環境変数がtrue",
			envValue:     "true",
			defaultValue: false,
			want:         true,
		},
		{
			name:         "環境変数がfalse",
			envValue:     "false",
			defaultValue: true,
			want:         false,
		},
		{
			name:         "環境変数が空",
			envValue:     "",
			defaultValue: true,
			want:         true,
		},
		{
			name:         "環境変数が無効な値",
			envValue:     "invalid",
			defaultValue: false,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("TEST_BOOL", tt.envValue)
			defer os.Unsetenv("TEST_BOOL")

			got := getEnvAsBool("TEST_BOOL", tt.defaultValue)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetEnvAsDuration(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue time.Duration
		want         time.Duration
	}{
		{
			name:         "環境変数が有効な時間",
			envValue:     "1h",
			defaultValue: time.Minute,
			want:         time.Hour,
		},
		{
			name:         "環境変数が空",
			envValue:     "",
			defaultValue: time.Minute,
			want:         time.Minute,
		},
		{
			name:         "環境変数が無効な値",
			envValue:     "invalid",
			defaultValue: time.Hour,
			want:         time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("TEST_DURATION", tt.envValue)
			defer os.Unsetenv("TEST_DURATION")

			got := getEnvAsDuration("TEST_DURATION", tt.defaultValue)
			assert.Equal(t, tt.want, got)
		})
	}
}
