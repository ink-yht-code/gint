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
//
// 本文件基于 ginx（https://github.com/ecodeclub/ginx）改造而来
// 原始版权归 ecodeclub 及其贡献者所有
// 当前版本为简化实现

package accesslog

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ink-yht-code/gint/gctx"
	"github.com/ink-yht-code/gint/logger"
	"go.uber.org/zap"
)

// AccessLog 表示一次访问日志记录。
type AccessLog struct {
	Method    string `json:"method"`     // HTTP 方法
	Path      string `json:"path"`       // 请求路径
	Query     string `json:"query"`      // 查询参数
	IP        string `json:"ip"`         // 客户端 IP
	UserID    string `json:"user_id"`    // 用户 ID
	TraceID   string `json:"trace_id"`   // 链路追踪 ID
	ReqBody   string `json:"req_body"`   // 请求体
	RespBody  string `json:"resp_body"`  // 响应体
	Status    int    `json:"status"`     // HTTP 状态码
	Duration  int64  `json:"duration"`   // 处理耗时，单位毫秒
	Error     string `json:"error"`      // 错误信息
	UserAgent string `json:"user_agent"` // User-Agent
}

// LogFunc 定义日志处理函数类型。
type LogFunc func(log *AccessLog)

// ZapLogFunc 创建一个基于 zap 的访问日志处理函数。
func ZapLogFunc() LogFunc {
	return func(log *AccessLog) {
		fields := []zap.Field{
			zap.String("method", log.Method),
			zap.String("path", log.Path),
			zap.String("query", log.Query),
			zap.String("ip", log.IP),
			zap.Int("status", log.Status),
			zap.Int64("duration_ms", log.Duration),
		}
		if log.TraceID != "" {
			fields = append(fields, zap.String("trace_id", log.TraceID))
		}
		if log.UserID != "" {
			fields = append(fields, zap.String("user_id", log.UserID))
		}
		if log.UserAgent != "" {
			fields = append(fields, zap.String("user_agent", log.UserAgent))
		}
		if log.ReqBody != "" {
			fields = append(fields, zap.String("req_body", log.ReqBody))
		}
		if log.RespBody != "" {
			fields = append(fields, zap.String("resp_body", log.RespBody))
		}
		if log.Error != "" {
			fields = append(fields, zap.String("error", log.Error))
		}

		switch {
		case log.Status >= 500:
			logger.Error("HTTP访问日志", fields...)
		case log.Status >= 400:
			logger.Warn("HTTP访问日志", fields...)
		default:
			logger.Info("HTTP访问日志", fields...)
		}
	}
}

// Builder 用于构建访问日志中间件。
type Builder struct {
	logFunc       LogFunc
	logReqBody    bool
	logRespBody   bool
	maxBodyLength int
	sampleRate    float64 // 0 表示全量，0.1 表示 10% 采样
	errorOnly     bool    // 只记录 4xx/5xx
}

// NewBuilder 创建访问日志中间件构建器。
func NewBuilder(logFunc LogFunc) *Builder {
	return &Builder{
		logFunc:       logFunc,
		logReqBody:    false,
		logRespBody:   false,
		maxBodyLength: 1024,
		sampleRate:    0,
		errorOnly:     false,
	}
}

// WithReqBody 设置是否记录请求体。
func (b *Builder) WithReqBody(log bool) *Builder {
	b.logReqBody = log
	return b
}

// WithRespBody 设置是否记录响应体。
func (b *Builder) WithRespBody(log bool) *Builder {
	b.logRespBody = log
	return b
}

// WithMaxBodyLength 设置请求体与响应体的最大记录长度。
func (b *Builder) WithMaxBodyLength(length int) *Builder {
	b.maxBodyLength = length
	return b
}

// WithSampleRate 设置采样率（0-1），0 表示全量记录，0.1 表示 10% 采样。
func (b *Builder) WithSampleRate(rate float64) *Builder {
	b.sampleRate = rate
	return b
}

// WithErrorOnly 只记录 4xx/5xx 请求。
func (b *Builder) WithErrorOnly(errorOnly bool) *Builder {
	b.errorOnly = errorOnly
	return b
}

// Build 构建访问日志中间件。
func (b *Builder) Build() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		ctx := &gctx.Context{Context: c}

		log := &AccessLog{
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			Query:     c.Request.URL.RawQuery,
			IP:        c.ClientIP(),
			TraceID:   ctx.TraceId(),
			UserID:    ctx.UserId(),
			UserAgent: c.Request.UserAgent(),
		}

		if b.logReqBody && c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			if len(bodyBytes) > b.maxBodyLength {
				log.ReqBody = string(bodyBytes[:b.maxBodyLength]) + "...(已截断)"
			} else {
				log.ReqBody = string(bodyBytes)
			}
		}

		if b.logRespBody {
			writer := &responseWriter{
				ResponseWriter: c.Writer,
				body:           &bytes.Buffer{},
			}
			c.Writer = writer

			c.Next()

			respBody := writer.body.String()
			if len(respBody) > b.maxBodyLength {
				log.RespBody = respBody[:b.maxBodyLength] + "...(已截断)"
			} else {
				log.RespBody = respBody
			}
		} else {
			c.Next()
		}

		log.Status = c.Writer.Status()
		log.Duration = time.Since(start).Milliseconds()

		if len(c.Errors) > 0 {
			log.Error = c.Errors.String()
		}

		// 只记录错误请求
		if b.errorOnly && log.Status < 400 {
			return
		}

		// 采样：按概率跳过
		if b.sampleRate > 0 && b.sampleRate < 1 {
			// 用 duration 纳秒的低位做伪随机，避免引入 math/rand 依赖
			if float64(time.Since(start).Nanoseconds()%1000)/1000.0 > b.sampleRate {
				return
			}
		}

		b.logFunc(log)
	}
}

// responseWriter 用于捕获响应体内容。
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write 同时写入原始响应和内存缓冲。
func (w *responseWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}
