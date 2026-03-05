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

// Package httpx 提供 HTTP server 初始化和中间件
package httpx

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ink-yht-code/gint/gintx/log"
	"go.uber.org/zap"
)

// Config HTTP 配置
type Config struct {
	Enabled bool
	Addr    string
}

// Server HTTP 服务器
type Server struct {
	Engine *gin.Engine
	Server *http.Server
}

// NewServer 创建 HTTP 服务器
func NewServer(cfg Config) *Server {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(Recovery(), RequestID(), Logger())

	return &Server{
		Engine: engine,
		Server: &http.Server{
			Addr:    cfg.Addr,
			Handler: engine,
		},
	}
}

// Run 启动服务器
func (s *Server) Run() error {
	return s.Server.ListenAndServe()
}

// Shutdown 关闭服务器
func (s *Server) Shutdown(ctx context.Context) error {
	return s.Server.Shutdown(ctx)
}

// RequestID 请求 ID 中间件
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-Id")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-Id", requestID)

		// 注入到 ctx
		ctx := context.WithValue(c.Request.Context(), "request_id", requestID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// Logger 日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		log.Ctx(c.Request.Context()).Info("HTTP request",
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", latency),
		)
	}
}

// Recovery 恢复中间件
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, err any) {
		log.Ctx(c.Request.Context()).Error("Panic recovered",
			zap.Any("error", err),
		)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "internal error",
		})
	})
}
