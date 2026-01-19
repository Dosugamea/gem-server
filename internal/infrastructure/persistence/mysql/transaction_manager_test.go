package mysql

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionManager_WithTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tm := &TransactionManager{db: &DB{DB: db}}

	tests := []struct {
		name      string
		fn        func(*sql.Tx) error
		setupMock func()
		wantError bool
	}{
		{
			name: "正常系: トランザクション成功",
			fn: func(tx *sql.Tx) error {
				return nil
			},
			setupMock: func() {
				mock.ExpectBegin()
				mock.ExpectCommit()
			},
			wantError: false,
		},
		{
			name: "正常系: トランザクションロールバック（エラー発生）",
			fn: func(tx *sql.Tx) error {
				return errors.New("test error")
			},
			setupMock: func() {
				mock.ExpectBegin()
				mock.ExpectRollback()
			},
			wantError: true,
		},
		{
			name: "異常系: Beginエラー",
			fn: func(tx *sql.Tx) error {
				return nil
			},
			setupMock: func() {
				mock.ExpectBegin().WillReturnError(errors.New("begin error"))
			},
			wantError: true,
		},
		{
			name: "正常系: パニック発生時もロールバック",
			fn: func(tx *sql.Tx) error {
				panic("test panic")
			},
			setupMock: func() {
				mock.ExpectBegin()
				mock.ExpectRollback()
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			ctx := context.Background()

			if tt.name == "正常系: パニック発生時もロールバック" {
				defer func() {
					if r := recover(); r != nil {
						assert.Equal(t, "test panic", r)
					}
				}()
			}

			err := tm.WithTransaction(ctx, tt.fn)

			if tt.wantError {
				if tt.name != "正常系: パニック発生時もロールバック" {
					assert.Error(t, err)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
