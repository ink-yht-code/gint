// 版权所有 2025 ink-yht-code
//
// 专有许可
//
// 重要说明：本软件并非开源软件。
// 未经版权持有人事先书面许可，
// 不得使用、复制、修改、合并、发布、分发、再许可，
// 也不得全部或部分出售本文件的副本。
//
// 本软件按“现状”提供，不附带任何形式的担保。

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

	enableDBLog    bool
	minDBLogLevel  zapcore.Level
	dbLogWriter    DBLogWriter
	dbLogWriterMux sync.RWMutex
)

func init() {
	Logger = zap.NewNop()
	sugar = Logger.Sugar()
}

// DBLogWriter 定义数据库日志写入器。
type DBLogWriter interface {
	WriteLog(level, module, message string, fields map[string]any, traceID, userID string)
}

// Init 初始化全局 zap 日志器。
func Init(cfg Config) error {
	level := parseLevel(cfg.Level)

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

	core := zapcore.NewTee(buildCores(cfg, encoderConfig, level)...)

	Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	sugar = Logger.Sugar()

	zap.ReplaceGlobals(Logger)

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

// buildCores 创建日志输出目标，控制台始终使用带颜色的文本格式。
func buildCores(cfg Config, encoderConfig zapcore.EncoderConfig, level zapcore.LevelEnabler) []zapcore.Core {
	var cores []zapcore.Core

	consoleCfg := encoderConfig
	consoleCfg.EncodeLevel = encodeLevelColor
	consoleEncoder := zapcore.NewConsoleEncoder(consoleCfg)

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
	default:
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level))
	}

	return cores
}

// encodeLevelColor 使用 ANSI 转义码为控制台日志级别着色。
func encodeLevelColor(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	var color string
	switch level {
	case zapcore.DebugLevel:
		color = "\x1b[36m"
	case zapcore.InfoLevel:
		color = "\x1b[32m"
	case zapcore.WarnLevel:
		color = "\x1b[33m"
	case zapcore.ErrorLevel:
		color = "\x1b[31m"
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		color = "\x1b[35m"
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

// parseLevel 将字符串日志级别转换为 zapcore.Level。
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

// createFileSyncer 创建支持轮转的文件写入器。
func createFileSyncer(cfg Config) zapcore.WriteSyncer {
	if dir := filepath.Dir(cfg.Filename); dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}

	rotator := &fileRotator{
		filename:   cfg.Filename,
		maxSize:    int64(cfg.MaxSize) * 1024 * 1024,
		maxBackups: cfg.MaxBackups,
		maxAge:     cfg.MaxAge,
		compress:   cfg.Compress,
	}
	rotator.init()
	return zapcore.AddSync(rotator)
}

// fileRotator 是一个简化版的日志轮转写入器。
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
	_ = r.openNew()
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
		_ = file.Close()
		return err
	}
	r.file = file
	r.size = info.Size()
	return nil
}

func (r *fileRotator) rotate() error {
	if r.file != nil {
		_ = r.file.Close()
	}

	timestamp := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("%s-%s%s", r.baseName, timestamp, r.ext)
	if err := os.Rename(r.filename, backupName); err != nil && !os.IsNotExist(err) {
		return err
	}

	if r.compress {
		go r.compressFile(backupName)
	}

	return r.openNew()
}

func (r *fileRotator) compressFile(filename string) {
	_ = filename
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

	now := time.Now()
	for i, path := range backups {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		if r.maxAge > 0 && now.Sub(info.ModTime()) > time.Duration(r.maxAge)*24*time.Hour {
			_ = os.Remove(path)
			continue
		}

		if r.maxBackups > 0 && i < len(backups)-r.maxBackups {
			_ = os.Remove(path)
		}
	}
}

// splitExt 将文件路径拆分为主文件名和扩展名。
func splitExt(path string) (base, ext string) {
	ext = filepath.Ext(path)
	base = path[:len(path)-len(ext)]
	return
}

// Sync 刷新缓冲中的日志。
func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}

// SetDBLogWriter 注册数据库日志写入器。
func SetDBLogWriter(writer DBLogWriter) {
	dbLogWriterMux.Lock()
	defer dbLogWriterMux.Unlock()
	dbLogWriter = writer
}

// shouldWriteToDB 判断当前日志是否需要额外写入数据库。
func shouldWriteToDB(level zapcore.Level) bool {
	if !enableDBLog {
		return false
	}
	dbLogWriterMux.RLock()
	defer dbLogWriterMux.RUnlock()
	return dbLogWriter != nil && level >= minDBLogLevel
}

// fieldsToMap 将 zap 字段转换为普通 map。
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

	delete(result, "time")
	delete(result, "level")
	delete(result, "msg")

	return result
}

// writeDBLog 将日志写入可选的数据库目标。
func writeDBLog(level zapcore.Level, module, msg string, fields []zap.Field, traceID, userID string) {
	if shouldWriteToDB(level) {
		dbLogWriterMux.RLock()
		writer := dbLogWriter
		dbLogWriterMux.RUnlock()
		writer.WriteLog(level.String(), module, msg, fieldsToMap(fields), traceID, userID)
	}
}

func Debug(msg string, fields ...zap.Field) {
	Logger.Debug(msg, fields...)
	writeDBLog(zapcore.DebugLevel, "", msg, fields, "", "")
}

func Info(msg string, fields ...zap.Field) {
	Logger.Info(msg, fields...)
	writeDBLog(zapcore.InfoLevel, "", msg, fields, "", "")
}

func Warn(msg string, fields ...zap.Field) {
	Logger.Warn(msg, fields...)
	writeDBLog(zapcore.WarnLevel, "", msg, fields, "", "")
}

func Error(msg string, fields ...zap.Field) {
	Logger.Error(msg, fields...)
	writeDBLog(zapcore.ErrorLevel, "", msg, fields, "", "")
}

func Fatal(msg string, fields ...zap.Field) {
	writeDBLog(zapcore.FatalLevel, "", msg, fields, "", "")
	Logger.Fatal(msg, fields...)
}

func DebugWithModule(module, msg string, fields ...zap.Field) {
	Logger.Debug(msg, append([]zap.Field{zap.String("module", module)}, fields...)...)
	writeDBLog(zapcore.DebugLevel, module, msg, fields, "", "")
}

func InfoWithModule(module, msg string, fields ...zap.Field) {
	Logger.Info(msg, append([]zap.Field{zap.String("module", module)}, fields...)...)
	writeDBLog(zapcore.InfoLevel, module, msg, fields, "", "")
}

func WarnWithModule(module, msg string, fields ...zap.Field) {
	Logger.Warn(msg, append([]zap.Field{zap.String("module", module)}, fields...)...)
	writeDBLog(zapcore.WarnLevel, module, msg, fields, "", "")
}

func ErrorWithModule(module, msg string, fields ...zap.Field) {
	Logger.Error(msg, append([]zap.Field{zap.String("module", module)}, fields...)...)
	writeDBLog(zapcore.ErrorLevel, module, msg, fields, "", "")
}

func Debugf(template string, args ...any) {
	sugar.Debugf(template, args...)
}

func Infof(template string, args ...any) {
	sugar.Infof(template, args...)
}

func Warnf(template string, args ...any) {
	sugar.Warnf(template, args...)
}

func Errorf(template string, args ...any) {
	sugar.Errorf(template, args...)
}

func Fatalf(template string, args ...any) {
	sugar.Fatalf(template, args...)
}

// RequestLogger 记录请求开始与结束日志，并注入 X-Request-ID。
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
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

		Info("收到HTTP请求",
			zap.String("request_id", rid),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)

		c.Next()

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

		Info("HTTP请求完成",
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

// WithTraceID 创建一个带 traceID 的日志上下文。
func WithTraceID(traceID string) *LogContext {
	return &LogContext{traceID: traceID}
}

// WithUserID 创建一个带 userID 的日志上下文。
func WithUserID(userID string) *LogContext {
	return &LogContext{userID: userID}
}

// WithTraceAndUserID 创建一个同时带 traceID 和 userID 的日志上下文。
func WithTraceAndUserID(traceID, userID string) *LogContext {
	return &LogContext{traceID: traceID, userID: userID}
}

// LogContext 用于在日志中携带 trace 和用户信息。
type LogContext struct {
	traceID string
	userID  string
}

func (lc *LogContext) Debug(msg string, fields ...zap.Field) {
	allFields := lc.appendContextFields(fields)
	Logger.Debug(msg, allFields...)
	writeDBLog(zapcore.DebugLevel, "", msg, fields, lc.traceID, lc.userID)
}

func (lc *LogContext) Info(msg string, fields ...zap.Field) {
	allFields := lc.appendContextFields(fields)
	Logger.Info(msg, allFields...)
	writeDBLog(zapcore.InfoLevel, "", msg, fields, lc.traceID, lc.userID)
}

func (lc *LogContext) Warn(msg string, fields ...zap.Field) {
	allFields := lc.appendContextFields(fields)
	Logger.Warn(msg, allFields...)
	writeDBLog(zapcore.WarnLevel, "", msg, fields, lc.traceID, lc.userID)
}

func (lc *LogContext) Error(msg string, fields ...zap.Field) {
	allFields := lc.appendContextFields(fields)
	Logger.Error(msg, allFields...)
	writeDBLog(zapcore.ErrorLevel, "", msg, fields, lc.traceID, lc.userID)
}

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
