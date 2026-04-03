// Copyright 2025 ink-yht-code
//
// Proprietary License
//
// IMPORTANT: This software is NOT open source.
// You may NOT use, copy, modify, merge, publish, distribute, sublicense,
// or sell copies of this file, in whole or in part, without prior written
// permission from the copyright holder.
//
// This software is provided "AS IS", without warranty of any kind.

package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Logger *zap.Logger
	sugar  *zap.SugaredLogger

	// 数据库日志相关
	enableDBLog    bool
	minDBLogLevel  zapcore.Level
	dbLogWriter    DBLogWriter
	dbLogWriterMux sync.RWMutex
)

func init() {
	// 避免调用方未 Init 时出现 nil panic
	Logger = zap.NewNop()
	sugar = Logger.Sugar()
}

// DBLogWriter 数据库日志写入器接口
type DBLogWriter interface {
	WriteLog(level, module, message string, fields map[string]any, traceID, userID string)
}

// Init 初始化 zap 日志
func Init(cfg Config) error {
	// 1. 设置日志级别
	level := parseLevel(cfg.Level)

	// 2. 设置编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 3. 构建 cores
	core := zapcore.NewTee(buildCores(cfg, encoderConfig, level)...)

	// 4. 创建 logger，添加调用者信息，跳过一层（封装层）
	Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	sugar = Logger.Sugar()

	// 5. 替换全局 logger
	zap.ReplaceGlobals(Logger)

	// 6. 初始化数据库日志配置
	enableDBLog = cfg.EnableDB
	minDBLogLevel = parseLevel(cfg.DBLevel)

	Info("日志系统初始化完成",
		zap.Bool("enable_db_log", enableDBLog),
		zap.String("db_log_level", minDBLogLevel.String()),
		zap.String("level", level.String()),
		zap.String("format", cfg.Format),
		zap.String("output", cfg.Output),
	)

	return nil
}

// buildCores 创建多输出 cores
// 控制台始终使用 text 彩色格式，文件按配置格式
func buildCores(cfg Config, encoderConfig zapcore.EncoderConfig, level zapcore.LevelEnabler) []zapcore.Core {
	var cores []zapcore.Core

	// 控制台编码器：text + 彩色
	consoleCfg := encoderConfig
	consoleCfg.EncodeLevel = encodeLevelColor
	consoleEncoder := zapcore.NewConsoleEncoder(consoleCfg)

	// 文件编码器：按配置
	fileCfg := encoderConfig
	if cfg.Format == "json" {
		fileCfg.EncodeLevel = zapcore.LowercaseLevelEncoder
	} else {
		fileCfg.EncodeLevel = encodeLevelColor
	}

	var fileEncoder zapcore.Encoder
	if cfg.Format == "json" {
		fileEncoder = zapcore.NewJSONEncoder(fileCfg)
	} else {
		fileEncoder = zapcore.NewConsoleEncoder(fileCfg)
	}

	switch cfg.Output {
	case "file":
		cores = append(cores, zapcore.NewCore(fileEncoder, createFileSyncer(cfg), level))
	case "both":
		cores = append(cores,
			zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level),
			zapcore.NewCore(fileEncoder, createFileSyncer(cfg), level),
		)
	default: // stdout
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level))
	}

	return cores
}

// encodeLevelColor 为不同级别添加 ANSI 颜色
func encodeLevelColor(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	var color string
	switch level {
	case zapcore.DebugLevel:
		color = "\x1b[36m" // 青色
	case zapcore.InfoLevel:
		color = "\x1b[32m" // 绿色
	case zapcore.WarnLevel:
		color = "\x1b[33m" // 黄色
	case zapcore.ErrorLevel:
		color = "\x1b[31m" // 红色
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		color = "\x1b[35m" // 洋红
	default:
		color = ""
	}

	levelText := level.String()
	if color != "" {
		enc.AppendString(color + levelText + "\x1b[0m")
		return
	}
	enc.AppendString(levelText)
}

// parseLevel 解析日志级别
func parseLevel(level string) zapcore.Level {
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

// createFileSyncer 创建文件同步器（支持日志轮转）
func createFileSyncer(cfg Config) zapcore.WriteSyncer {
	// 确保日志目录存在
	if dir := filepath.Dir(cfg.Filename); dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}

	// 使用自定义日志轮转器
	rotator := &fileRotator{
		filename:   cfg.Filename,
		maxSize:    int64(cfg.MaxSize) * 1024 * 1024, // MB -> Bytes
		maxBackups: cfg.MaxBackups,
		maxAge:     cfg.MaxAge,
		compress:   cfg.Compress,
	}
	rotator.init()
	return zapcore.AddSync(rotator)
}

// fileRotator 自定义日志文件轮转器
type fileRotator struct {
	filename   string
	maxSize    int64
	maxBackups int
	maxAge     int
	compress   bool

	mu       sync.Mutex
	file     *os.File
	size     int64
	baseName string
	ext      string
}

func (r *fileRotator) init() {
	r.baseName, r.ext = splitExt(r.filename)
	r.openNew()
	go r.cleanup()
}

func (r *fileRotator) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file == nil {
		if err := r.openNew(); err != nil {
			return 0, err
		}
	}

	// 检查是否需要轮转
	if r.size+int64(len(p)) > r.maxSize && r.maxSize > 0 {
		if err := r.rotate(); err != nil {
			return 0, err
		}
	}

	n, err = r.file.Write(p)
	r.size += int64(n)
	return n, err
}

func (r *fileRotator) Sync() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.file != nil {
		return r.file.Sync()
	}
	return nil
}

func (r *fileRotator) openNew() error {
	file, err := os.OpenFile(r.filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}
	r.file = file
	r.size = info.Size()
	return nil
}

func (r *fileRotator) rotate() error {
	if r.file != nil {
		r.file.Close()
	}

	// 重命名当前文件
	timestamp := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("%s-%s%s", r.baseName, timestamp, r.ext)
	if err := os.Rename(r.filename, backupName); err != nil && !os.IsNotExist(err) {
		return err
	}

	// 可选压缩
	if r.compress {
		go r.compressFile(backupName)
	}

	return r.openNew()
}

func (r *fileRotator) compressFile(filename string) {
	// 简单起见，暂不实现压缩
}

func (r *fileRotator) cleanup() {
	if r.maxAge <= 0 && r.maxBackups <= 0 {
		return
	}

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		r.doCleanup()
	}
}

func (r *fileRotator) doCleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	dir := filepath.Dir(r.filename)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var backups []string
	prefix := r.baseName + "-"
	for _, entry := range entries {
		name := entry.Name()
		if len(name) > len(prefix) && name[:len(prefix)] == prefix {
			backups = append(backups, filepath.Join(dir, name))
		}
	}

	// 按时间排序（旧文件在前）
	// 简单实现：根据文件名排序

	now := time.Now()
	for i, path := range backups {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		// 按时间删除
		if r.maxAge > 0 {
			if now.Sub(info.ModTime()) > time.Duration(r.maxAge)*24*time.Hour {
				os.Remove(path)
				continue
			}
		}

		// 按数量删除
		if r.maxBackups > 0 && i < len(backups)-r.maxBackups {
			os.Remove(path)
		}
	}
}

// splitExt 分割文件名和扩展名
func splitExt(path string) (base, ext string) {
	ext = filepath.Ext(path)
	base = path[:len(path)-len(ext)]
	return
}

// Sync 刷新日志缓冲区
func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}

// SetDBLogWriter 设置数据库日志写入器
func SetDBLogWriter(writer DBLogWriter) {
	dbLogWriterMux.Lock()
	defer dbLogWriterMux.Unlock()
	dbLogWriter = writer
}

// shouldWriteToDB 判断是否需要将日志写入数据库
func shouldWriteToDB(level zapcore.Level) bool {
	if !enableDBLog {
		return false
	}
	dbLogWriterMux.RLock()
	defer dbLogWriterMux.RUnlock()
	return dbLogWriter != nil && level >= minDBLogLevel
}

// fieldsToMap 将 zap.Field 转换为 map[string]any
func fieldsToMap(fields []zap.Field) map[string]any {
	if len(fields) == 0 {
		return make(map[string]any)
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	encoder := zapcore.NewJSONEncoder(encoderConfig)

	entry := zapcore.Entry{
		Level:      zapcore.DebugLevel,
		Time:       time.Now(),
		LoggerName: "",
		Message:    "",
		Caller:     zapcore.EntryCaller{},
		Stack:      "",
	}

	buf, err := encoder.EncodeEntry(entry, fields)
	if err != nil {
		return make(map[string]any)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		return make(map[string]any)
	}

	// 移除默认字段
	delete(result, "time")
	delete(result, "level")
	delete(result, "msg")

	return result
}

// writeDBLog 写入数据库日志
func writeDBLog(level zapcore.Level, module, msg string, fields []zap.Field, traceID, userID string) {
	if shouldWriteToDB(level) {
		dbLogWriterMux.RLock()
		writer := dbLogWriter
		dbLogWriterMux.RUnlock()
		writer.WriteLog(level.String(), module, msg, fieldsToMap(fields), traceID, userID)
	}
}

// --- 日志方法 ---

// Debug 调试日志
func Debug(msg string, fields ...zap.Field) {
	Logger.Debug(msg, fields...)
	writeDBLog(zapcore.DebugLevel, "", msg, fields, "", "")
}

// Info 信息日志
func Info(msg string, fields ...zap.Field) {
	Logger.Info(msg, fields...)
	writeDBLog(zapcore.InfoLevel, "", msg, fields, "", "")
}

// Warn 警告日志
func Warn(msg string, fields ...zap.Field) {
	Logger.Warn(msg, fields...)
	writeDBLog(zapcore.WarnLevel, "", msg, fields, "", "")
}

// Error 错误日志
func Error(msg string, fields ...zap.Field) {
	Logger.Error(msg, fields...)
	writeDBLog(zapcore.ErrorLevel, "", msg, fields, "", "")
}

// Fatal 致命错误日志（会退出程序）
func Fatal(msg string, fields ...zap.Field) {
	writeDBLog(zapcore.FatalLevel, "", msg, fields, "", "")
	Logger.Fatal(msg, fields...)
}

// --- 带模块的日志方法 ---

// DebugWithModule 带模块的调试日志
func DebugWithModule(module, msg string, fields ...zap.Field) {
	Logger.Debug(msg, append([]zap.Field{zap.String("module", module)}, fields...)...)
	writeDBLog(zapcore.DebugLevel, module, msg, fields, "", "")
}

// InfoWithModule 带模块的信息日志
func InfoWithModule(module, msg string, fields ...zap.Field) {
	Logger.Info(msg, append([]zap.Field{zap.String("module", module)}, fields...)...)
	writeDBLog(zapcore.InfoLevel, module, msg, fields, "", "")
}

// WarnWithModule 带模块的警告日志
func WarnWithModule(module, msg string, fields ...zap.Field) {
	Logger.Warn(msg, append([]zap.Field{zap.String("module", module)}, fields...)...)
	writeDBLog(zapcore.WarnLevel, module, msg, fields, "", "")
}

// ErrorWithModule 带模块的错误日志
func ErrorWithModule(module, msg string, fields ...zap.Field) {
	Logger.Error(msg, append([]zap.Field{zap.String("module", module)}, fields...)...)
	writeDBLog(zapcore.ErrorLevel, module, msg, fields, "", "")
}

// --- SugaredLogger 方法（更灵活，性能略低）---

// Debugf 格式化调试日志
func Debugf(template string, args ...any) {
	sugar.Debugf(template, args...)
}

// Infof 格式化信息日志
func Infof(template string, args ...any) {
	sugar.Infof(template, args...)
}

// Warnf 格式化警告日志
func Warnf(template string, args ...any) {
	sugar.Warnf(template, args...)
}

// Errorf 格式化错误日志
func Errorf(template string, args ...any) {
	sugar.Errorf(template, args...)
}

// Fatalf 格式化致命错误日志
func Fatalf(template string, args ...any) {
	sugar.Fatalf(template, args...)
}

// --- Gin 中间件 ---

// RequestLogger 请求日志中间件
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成或获取请求 ID
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = uuid.New().String()
		}
		c.Writer.Header().Set("X-Request-ID", rid)

		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		// 记录请求开始
		Info("http_request_start",
			zap.String("request_id", rid),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)

		c.Next()

		// 记录请求结束
		latency := time.Since(start)
		status := c.Writer.Status()
		size := c.Writer.Size()
		if size < 0 {
			size = 0
		}

		errMsg := ""
		if len(c.Errors) > 0 {
			errMsg = c.Errors.String()
		}

		Info("http_request_end",
			zap.String("request_id", rid),
			zap.Int("status", status),
			zap.Int("response_size", size),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.String("error", errMsg),
		)
	}
}

// WithTraceID 返回带 traceID 的日志上下文
func WithTraceID(traceID string) *LogContext {
	return &LogContext{traceID: traceID}
}

// WithUserID 返回带 userID 的日志上下文
func WithUserID(userID string) *LogContext {
	return &LogContext{userID: userID}
}

// WithTraceAndUserID 返回带 traceID 和 userID 的日志上下文
func WithTraceAndUserID(traceID, userID string) *LogContext {
	return &LogContext{traceID: traceID, userID: userID}
}

// LogContext 日志上下文，携带 traceID 和 userID
type LogContext struct {
	traceID string
	userID  string
}

// Debug 调试日志
func (lc *LogContext) Debug(msg string, fields ...zap.Field) {
	allFields := lc.appendContextFields(fields)
	Logger.Debug(msg, allFields...)
	writeDBLog(zapcore.DebugLevel, "", msg, fields, lc.traceID, lc.userID)
}

// Info 信息日志
func (lc *LogContext) Info(msg string, fields ...zap.Field) {
	allFields := lc.appendContextFields(fields)
	Logger.Info(msg, allFields...)
	writeDBLog(zapcore.InfoLevel, "", msg, fields, lc.traceID, lc.userID)
}

// Warn 警告日志
func (lc *LogContext) Warn(msg string, fields ...zap.Field) {
	allFields := lc.appendContextFields(fields)
	Logger.Warn(msg, allFields...)
	writeDBLog(zapcore.WarnLevel, "", msg, fields, lc.traceID, lc.userID)
}

// Error 错误日志
func (lc *LogContext) Error(msg string, fields ...zap.Field) {
	allFields := lc.appendContextFields(fields)
	Logger.Error(msg, allFields...)
	writeDBLog(zapcore.ErrorLevel, "", msg, fields, lc.traceID, lc.userID)
}

// appendContextFields 添加上下文字段
func (lc *LogContext) appendContextFields(fields []zap.Field) []zap.Field {
	result := make([]zap.Field, 0, len(fields)+2)
	if lc.traceID != "" {
		result = append(result, zap.String("trace_id", lc.traceID))
	}
	if lc.userID != "" {
		result = append(result, zap.String("user_id", lc.userID))
	}
	result = append(result, fields...)
	return result
}
