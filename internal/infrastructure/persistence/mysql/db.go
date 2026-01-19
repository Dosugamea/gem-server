package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gem-server/internal/infrastructure/config"

	_ "github.com/go-sql-driver/mysql"
)

// DB データベース接続とトランザクション管理を提供
type DB struct {
	*sql.DB
}

// NewDB 新しいデータベース接続を作成
func NewDB(cfg *config.DatabaseConfig) (*DB, error) {
	dsn := cfg.DSN()

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 接続プールの設定
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// 接続テスト
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{DB: db}, nil
}

// Close データベース接続を閉じる
func (db *DB) Close() error {
	return db.DB.Close()
}

// HealthCheck データベースのヘルスチェックを実行
func (db *DB) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.PingContext(ctx)
}
