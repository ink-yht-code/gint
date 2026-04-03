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

package trace

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ink-yht-code/gint/gctx"
)

const (
	// HeaderTraceID HTTP Header 中的 TraceID 字段
	HeaderTraceID = "X-Trace-ID"
	// HeaderUserID HTTP Header 中的 UserID 字段
	HeaderUserID = "X-User-ID"
	// HeaderUserRole HTTP Header 中的用户角色字段
	HeaderUserRole = "X-User-Role"
	// HeaderTenantID HTTP Header 中的租户ID字段
	HeaderTenantID = "X-Tenant-ID"
	// HeaderUsername HTTP Header 中的用户名字段
	HeaderUsername = "X-Username"
	// HeaderEmail HTTP Header 中的邮箱字段
	HeaderEmail = "X-Email"
)

// Middleware 链路追踪中间件
// 自动从请求头提取或生成 traceID，并注入到上下文
// 同时支持从请求头提取完整的 UserContext
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := &gctx.Context{Context: c}

		// 提取或生成 traceID
		traceID := c.GetHeader(HeaderTraceID)
		if traceID == "" {
			traceID = uuid.New().String()
		}
		ctx.SetTraceId(traceID)
		c.Writer.Header().Set(HeaderTraceID, traceID)

		// 提取完整的 UserContext
		userID := c.GetHeader(HeaderUserID)
		if userID != "" {
			uc := &gctx.UserContext{
				UserId:   userID,
				Role:     c.GetHeader(HeaderUserRole),
				TenantId: c.GetHeader(HeaderTenantID),
				Username: c.GetHeader(HeaderUsername),
				Email:    c.GetHeader(HeaderEmail),
			}
			ctx.SetUserContext(uc)
		}

		c.Next()
	}
}

// --- HTTP Client 包装器，自动传递上下文 ---

// Transport 自定义 Transport，自动注入 traceID 和 UserContext
type Transport struct {
	// Base 基础 Transport
	Base http.RoundTripper
	// GetTraceID 获取 traceID 的函数（从 context 获取）
	GetTraceID func(r *http.Request) string
	// GetUserContext 获取 UserContext 的函数（从 context 获取）
	GetUserContext func(r *http.Request) *gctx.UserContext
}

// RoundTrip 实现 http.RoundTripper 接口
func (t *Transport) RoundTrip(r *http.Request) (*http.Response, error) {
	// 注入 traceID
	if t.GetTraceID != nil {
		if traceID := t.GetTraceID(r); traceID != "" {
			r.Header.Set(HeaderTraceID, traceID)
		}
	}

	// 注入 UserContext
	if t.GetUserContext != nil {
		if uc := t.GetUserContext(r); uc != nil && !uc.IsEmpty() {
			r.Header.Set(HeaderUserID, uc.UserId)
			if uc.Role != "" {
				r.Header.Set(HeaderUserRole, uc.Role)
			}
			if uc.TenantId != "" {
				r.Header.Set(HeaderTenantID, uc.TenantId)
			}
			if uc.Username != "" {
				r.Header.Set(HeaderUsername, uc.Username)
			}
			if uc.Email != "" {
				r.Header.Set(HeaderEmail, uc.Email)
			}
		}
	}

	return t.base().RoundTrip(r)
}

func (t *Transport) base() http.RoundTripper {
	if t.Base == nil {
		return http.DefaultTransport
	}
	return t.Base
}

// NewClient 创建带有链路追踪的 HTTP Client
func NewClient() *http.Client {
	return &http.Client{
		Transport: &Transport{},
	}
}

// --- 辅助函数 ---

// InjectContext 将 gin.Context 中的 traceID/userID 注入到 HTTP 请求
// 推荐使用 InjectGctx，它支持完整的 UserContext
// Deprecated: 使用 InjectGctx 替代
func InjectContext(c *gin.Context, req *http.Request) {
	ctx := &gctx.Context{Context: c}
	InjectGctx(ctx, req)
}

// InjectGctx 将 gctx.Context 中的 traceID 和 UserContext 注入到 HTTP 请求
func InjectGctx(ctx *gctx.Context, req *http.Request) {
	if traceID := ctx.TraceId(); traceID != "" {
		req.Header.Set(HeaderTraceID, traceID)
	}
	if uc := ctx.UserContext(); uc != nil && !uc.IsEmpty() {
		req.Header.Set(HeaderUserID, uc.UserId)
		if uc.Role != "" {
			req.Header.Set(HeaderUserRole, uc.Role)
		}
		if uc.TenantId != "" {
			req.Header.Set(HeaderTenantID, uc.TenantId)
		}
		if uc.Username != "" {
			req.Header.Set(HeaderUsername, uc.Username)
		}
		if uc.Email != "" {
			req.Header.Set(HeaderEmail, uc.Email)
		}
	}
}

// NewRequestWithContext 创建带有上下文追踪信息的请求
func NewRequestWithContext(ctx *gctx.Context, method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	InjectGctx(ctx, req)
	return req, nil
}

// DoWithContext 使用 gctx.Context 发送 HTTP 请求，自动传递追踪信息
func DoWithContext(ctx *gctx.Context, client *http.Client, req *http.Request) (*http.Response, error) {
	InjectGctx(ctx, req)
	return client.Do(req)
}
