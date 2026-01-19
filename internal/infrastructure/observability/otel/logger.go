package otel

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"go.opentelemetry.io/otel/trace"
)

// Logger 構造化ロガー
type Logger struct {
	tracer trace.Tracer
}

// NewLogger 新しいLoggerを作成
func NewLogger(tracer trace.Tracer) *Logger {
	return &Logger{
		tracer: tracer,
	}
}

// LogLevel ログレベル
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
)

// LogEntry ログエントリ
type LogEntry struct {
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	TraceID   string                 `json:"trace_id,omitempty"`
	SpanID    string                 `json:"span_id,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

// Log ログを出力
func (l *Logger) Log(ctx context.Context, level LogLevel, message string, fields map[string]interface{}) {
	entry := LogEntry{
		Level:     string(level),
		Message:   message,
		Fields:    fields,
		Timestamp: fmt.Sprintf("%d", os.Getpid()), // 簡易的なタイムスタンプ
	}

	// トレースIDとSpanIDを取得
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		entry.TraceID = span.SpanContext().TraceID().String()
		entry.SpanID = span.SpanContext().SpanID().String()
	}

	// JSON形式で出力
	jsonData, err := json.Marshal(entry)
	if err != nil {
		log.Printf("failed to marshal log entry: %v", err)
		return
	}

	log.Println(string(jsonData))
}

// Debug Debugレベルのログを出力
func (l *Logger) Debug(ctx context.Context, message string, fields map[string]interface{}) {
	l.Log(ctx, LogLevelDebug, message, fields)
}

// Info Infoレベルのログを出力
func (l *Logger) Info(ctx context.Context, message string, fields map[string]interface{}) {
	l.Log(ctx, LogLevelInfo, message, fields)
}

// Warn Warnレベルのログを出力
func (l *Logger) Warn(ctx context.Context, message string, fields map[string]interface{}) {
	l.Log(ctx, LogLevelWarn, message, fields)
}

// Error Errorレベルのログを出力
func (l *Logger) Error(ctx context.Context, message string, err error, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	l.Log(ctx, LogLevelError, message, fields)
}
