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
//
// This file is derived from ginx (https://github.com/ecodeclub/ginx)
// Original Copyright by ecodeclub and contributors
// Modifications: Simplified implementation

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

// AccessLog 访问日志结构
type AccessLog struct {
	Method    string `json:"method"`     // HTTP 方法
	Path      string `json:"path"`       // 请求路径
	Query     string `json:"query"`      // 查询参数
	IP        string `json:"ip"`         // 客户端 IP
	UserID    string `json:"user_id"`    // 用户 ID（如果已登录）
	TraceID   string `json:"trace_id"`   // 链路追踪 ID
	ReqBody   string `json:"req_body"`   // 请求体
	RespBody  string `json:"resp_body"`  // 响应体
	Status    int    `json:"status"`     // HTTP 状态码
	Duration  int64  `json:"duration"`   // 处理时间（毫秒）
	Error     string `json:"error"`      // 错误信息
	UserAgent string `json:"user_agent"` // User-Agent
}

// LogFunc 日志处理函数类型
type LogFunc func(log *AccessLog)

// ZapLogFunc 创建使用 zap logger 的日志处理函数
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

		// 根据状态码选择日志级别
		switch {
		case log.Status >= 500:
			logger.Error("http_request", fields...)
		case log.Status >= 400:
			logger.Warn("http_request", fields...)
		default:
			logger.Info("http_request", fields...)
		}
	}
}

// Builder 访问日志中间件构建器
type Builder struct {
	logFunc       LogFunc // 日志处理函数
	logReqBody    bool    // 是否记录请求体
	logRespBody   bool    // 是否记录响应体
	maxBodyLength int     // 最大记录长度
}

// NewBuilder 创建访问日志中间件构建器
func NewBuilder(logFunc LogFunc) *Builder {
	return &Builder{
		logFunc:       logFunc,
		logReqBody:    false,
		logRespBody:   false,
		maxBodyLength: 1024, // 默认最大 1KB
	}
}

// WithReqBody 设置是否记录请求体
func (b *Builder) WithReqBody(log bool) *Builder {
	b.logReqBody = log
	return b
}

// WithRespBody 设置是否记录响应体
func (b *Builder) WithRespBody(log bool) *Builder {
	b.logRespBody = log
	return b
}

// WithMaxBodyLength 设置最大记录长度
func (b *Builder) WithMaxBodyLength(length int) *Builder {
	b.maxBodyLength = length
	return b
}

// Build 构建中间件
func (b *Builder) Build() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		ctx := &gctx.Context{Context: c}

		// 创建日志对象
		log := &AccessLog{
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			Query:     c.Request.URL.RawQuery,
			IP:        c.ClientIP(),
			TraceID:   ctx.TraceId(),
			UserID:    ctx.UserId(),
			UserAgent: c.Request.UserAgent(),
		}

		// 记录请求体
		if b.logReqBody && c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			// 恢复请求体，以便后续处理
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			if len(bodyBytes) > b.maxBodyLength {
				log.ReqBody = string(bodyBytes[:b.maxBodyLength]) + "...(truncated)"
			} else {
				log.ReqBody = string(bodyBytes)
			}
		}

		// 如果需要记录响应体，使用自定义 ResponseWriter
		if b.logRespBody {
			writer := &responseWriter{
				ResponseWriter: c.Writer,
				body:           &bytes.Buffer{},
			}
			c.Writer = writer

			// 执行请求处理
			c.Next()

			// 记录响应体
			respBody := writer.body.String()
			if len(respBody) > b.maxBodyLength {
				log.RespBody = respBody[:b.maxBodyLength] + "...(truncated)"
			} else {
				log.RespBody = respBody
			}
		} else {
			// 执行请求处理
			c.Next()
		}

		// 记录状态码和处理时间
		log.Status = c.Writer.Status()
		log.Duration = time.Since(start).Milliseconds()

		// 记录错误信息
		if len(c.Errors) > 0 {
			log.Error = c.Errors.String()
		}

		// 调用日志处理函数
		b.logFunc(log)
	}
}

// responseWriter 自定义 ResponseWriter，用于捕获响应体
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write 写入响应体
func (w *responseWriter) Write(data []byte) (int, error) {
	// 同时写入到 body 缓冲区和原始 Writer
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}
