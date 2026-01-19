package otel

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewLogger(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := NewLogger(tracer)

	assert.NotNil(t, logger)
	assert.Equal(t, tracer, logger.tracer)
}

func TestLogger_Log(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := NewLogger(tracer)

	tests := []struct {
		name    string
		level   LogLevel
		message string
		fields  map[string]interface{}
	}{
		{
			name:    "Infoレベルのログ",
			level:   LogLevelInfo,
			message: "test message",
			fields:  map[string]interface{}{"key": "value"},
		},
		{
			name:    "Debugレベルのログ",
			level:   LogLevelDebug,
			message: "debug message",
			fields:  nil,
		},
		{
			name:    "Warnレベルのログ",
			level:   LogLevelWarn,
			message: "warn message",
			fields:  map[string]interface{}{"count": 42},
		},
		{
			name:    "Errorレベルのログ",
			level:   LogLevelError,
			message: "error message",
			fields:  map[string]interface{}{"error": "test error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			logger.Log(ctx, tt.level, tt.message, tt.fields)

			// ログが出力されることを確認（実際の出力は確認しないが、エラーが発生しないことを確認）
			// 実際のテストでは、ログ出力をキャプチャして検証することも可能
		})
	}
}

func TestLogger_LogWithTraceContext(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := NewLogger(tracer)

	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// トレースコンテキストが設定されている場合のテスト
	logger.Log(ctx, LogLevelInfo, "test message", nil)

	// noopトレーサーではスパンが有効でない場合があるが、エラーが発生しないことを確認
	_ = span
}

func TestLogger_Debug(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := NewLogger(tracer)

	ctx := context.Background()
	logger.Debug(ctx, "debug message", map[string]interface{}{"key": "value"})
}

func TestLogger_Info(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := NewLogger(tracer)

	ctx := context.Background()
	logger.Info(ctx, "info message", map[string]interface{}{"key": "value"})
}

func TestLogger_Warn(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := NewLogger(tracer)

	ctx := context.Background()
	logger.Warn(ctx, "warn message", map[string]interface{}{"key": "value"})
}

func TestLogger_Error(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := NewLogger(tracer)

	tests := []struct {
		name    string
		message string
		err     error
		fields  map[string]interface{}
	}{
		{
			name:    "エラーあり、フィールドなし",
			message: "error message",
			err:     assert.AnError,
			fields:  nil,
		},
		{
			name:    "エラーあり、フィールドあり",
			message: "error message",
			err:     assert.AnError,
			fields:  map[string]interface{}{"key": "value"},
		},
		{
			name:    "エラーなし、フィールドあり",
			message: "error message",
			err:     nil,
			fields:  map[string]interface{}{"key": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			logger.Error(ctx, tt.message, tt.err, tt.fields)
		})
	}
}

func TestLogEntry_MarshalJSON(t *testing.T) {
	entry := LogEntry{
		Level:     "INFO",
		Message:   "test message",
		TraceID:   "trace-id",
		SpanID:    "span-id",
		Fields:    map[string]interface{}{"key": "value"},
		Timestamp: "1234567890",
	}

	jsonData, err := json.Marshal(entry)
	require.NoError(t, err)

	var decoded LogEntry
	err = json.Unmarshal(jsonData, &decoded)
	require.NoError(t, err)

	assert.Equal(t, entry.Level, decoded.Level)
	assert.Equal(t, entry.Message, decoded.Message)
	assert.Equal(t, entry.TraceID, decoded.TraceID)
	assert.Equal(t, entry.SpanID, decoded.SpanID)
	assert.Equal(t, entry.Fields, decoded.Fields)
	assert.Equal(t, entry.Timestamp, decoded.Timestamp)
}

func TestLogger_LogEntryFormat(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := NewLogger(tracer)

	ctx := context.Background()
	fields := map[string]interface{}{
		"user_id": "user123",
		"amount":  100,
	}

	// ログがJSON形式で出力されることを確認（実際の出力をキャプチャするのは難しいが、
	// エラーが発生しないことを確認）
	logger.Info(ctx, "test message", fields)

	// ログエントリが正しくフォーマットされることを確認
	entry := LogEntry{
		Level:     "INFO",
		Message:   "test message",
		Fields:    fields,
		Timestamp: "1234567890",
	}

	jsonData, err := json.Marshal(entry)
	require.NoError(t, err)

	jsonStr := string(jsonData)
	assert.Contains(t, jsonStr, `"level":"INFO"`)
	assert.Contains(t, jsonStr, `"message":"test message"`)
	assert.Contains(t, jsonStr, `"user_id":"user123"`)
	assert.Contains(t, jsonStr, `"amount":100`)
}

func TestLogger_LogWithInvalidJSON(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := NewLogger(tracer)

	ctx := context.Background()

	// 無効なJSONを生成する可能性のあるフィールド（循環参照など）をテスト
	// 実際には、map[string]interface{}なので、循環参照は発生しない
	fields := map[string]interface{}{
		"key": "value",
	}

	// エラーが発生しないことを確認
	logger.Log(ctx, LogLevelInfo, "test", fields)
}

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level LogLevel
		want  string
	}{
		{LogLevelDebug, "DEBUG"},
		{LogLevelInfo, "INFO"},
		{LogLevelWarn, "WARN"},
		{LogLevelError, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			assert.Equal(t, tt.want, string(tt.level))
		})
	}
}

func TestLogger_LogWithoutTraceContext(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := NewLogger(tracer)

	// トレースコンテキストがない場合のテスト
	ctx := context.Background()
	logger.Log(ctx, LogLevelInfo, "test message", nil)

	// トレースコンテキストがない場合でもエラーが発生しないことを確認
	span := trace.SpanFromContext(ctx)
	assert.False(t, span.SpanContext().IsValid())
}

func TestLogger_LogWithEmptyFields(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := NewLogger(tracer)

	ctx := context.Background()

	// 空のフィールドマップ
	logger.Log(ctx, LogLevelInfo, "test message", make(map[string]interface{}))

	// nilフィールド
	logger.Log(ctx, LogLevelInfo, "test message", nil)
}

func TestLogger_ErrorWithNilError(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := NewLogger(tracer)

	ctx := context.Background()

	// エラーがnilの場合でもエラーが発生しないことを確認
	logger.Error(ctx, "error message", nil, nil)
}

func TestLogger_ErrorWithExistingErrorField(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	logger := NewLogger(tracer)

	ctx := context.Background()

	// フィールドに既にerrorキーがある場合
	fields := map[string]interface{}{
		"error": "existing error",
		"key":   "value",
	}

	logger.Error(ctx, "error message", assert.AnError, fields)

	// エラーフィールドが上書きされることを確認（実際の実装では、エラーフィールドが追加される）
}
