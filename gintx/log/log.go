// Copyright 2025 ink-yht-code
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package log 提供全局日志功能，支持 ctx 注入 request_id
package log

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	defaultLogger *zap.Logger
	sugarLogger   *zap.SugaredLogger
)

// Config 日志配置
type Config struct {
	Level    string // debug, info, warn, error
	Encoding string // json, console
	Output   string // stdout, stderr, or file path
}

// Init 初始化日志
func Init(cfg Config) error {
	level := getLevel(cfg.Level)
	encoding := cfg.Encoding
	if encoding == "" {
		encoding = "json"
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Development:      false,
		Encoding:         encoding,
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{cfg.Output},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := config.Build()
	if err != nil {
		return err
	}

	defaultLogger = logger
	sugarLogger = logger.Sugar()
	return nil
}

func getLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// L 返回默认 logger
func L() *zap.Logger {
	if defaultLogger == nil {
		defaultLogger = zap.NewNop()
	}
	return defaultLogger
}

// S 返回默认 sugared logger
func S() *zap.SugaredLogger {
	if sugarLogger == nil {
		sugarLogger = zap.NewNop().Sugar()
	}
	return sugarLogger
}

type ctxKey struct{}

// WithContext 将 logger 放入 ctx
func WithContext(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

// Ctx 从 ctx 获取 logger（自动注入 request_id）
func Ctx(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return L()
	}

	if logger, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok {
		return logger
	}

	// 尝试从 ctx 获取 request_id
	if requestID := GetRequestID(ctx); requestID != "" {
		return L().With(zap.String("request_id", requestID))
	}

	return L()
}

// GetRequestID 从 ctx 获取 request_id
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	// 尝试多种常见的 request_id key
	for _, key := range []string{"request_id", "requestId", "x-request-id", "X-Request-Id"} {
		if v := ctx.Value(key); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

// Sync 刷新日志缓冲
func Sync() error {
	if defaultLogger != nil {
		return defaultLogger.Sync()
	}
	return nil
}

// Debug 调试日志
func Debug(msg string, fields ...zap.Field) {
	L().Debug(msg, fields...)
}

// Info 信息日志
func Info(msg string, fields ...zap.Field) {
	L().Info(msg, fields...)
}

// Warn 警告日志
func Warn(msg string, fields ...zap.Field) {
	L().Warn(msg, fields...)
}

// Error 错误日志
func Error(msg string, fields ...zap.Field) {
	L().Error(msg, fields...)
}

// Fatal 致命错误日志
func Fatal(msg string, fields ...zap.Field) {
	L().Fatal(msg, fields...)
}

// DebugCtx 带 ctx 的调试日志
func DebugCtx(ctx context.Context, msg string, fields ...zap.Field) {
	Ctx(ctx).Debug(msg, fields...)
}

// InfoCtx 带 ctx 的信息日志
func InfoCtx(ctx context.Context, msg string, fields ...zap.Field) {
	Ctx(ctx).Info(msg, fields...)
}

// WarnCtx 带 ctx 的警告日志
func WarnCtx(ctx context.Context, msg string, fields ...zap.Field) {
	Ctx(ctx).Warn(msg, fields...)
}

// ErrorCtx 带 ctx 的错误日志
func ErrorCtx(ctx context.Context, msg string, fields ...zap.Field) {
	Ctx(ctx).Error(msg, fields...)
}
