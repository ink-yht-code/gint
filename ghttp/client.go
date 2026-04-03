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

package ghttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/ink-yht-code/gint/gctx"
	"github.com/ink-yht-code/gint/logger"
	"go.uber.org/zap"
)

// Client HTTP 客户端
type Client struct {
	client    *http.Client
	timeout   time.Duration
	maxRetry  int
	retryWait time.Duration
	trace     bool // 是否自动传递链路追踪
}

// Option 客户端配置选项
type Option func(*Client)

// NewClient 创建 HTTP 客户端
func NewClient(opts ...Option) *Client {
	c := &Client{
		client:    &http.Client{},
		timeout:   30 * time.Second,
		maxRetry:  0,
		retryWait: 100 * time.Millisecond,
		trace:     true,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithTimeout 设置超时时间
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.timeout = d
		c.client.Timeout = d
	}
}

// WithRetry 设置重试次数
func WithRetry(maxRetry int, wait ...time.Duration) Option {
	return func(c *Client) {
		c.maxRetry = maxRetry
		if len(wait) > 0 {
			c.retryWait = wait[0]
		}
	}
}

// WithTransport 设置自定义 Transport
func WithTransport(transport http.RoundTripper) Option {
	return func(c *Client) {
		c.client.Transport = transport
	}
}

// WithTrace 设置是否自动传递链路追踪
func WithTrace(enable bool) Option {
	return func(c *Client) {
		c.trace = enable
	}
}

// --- 请求方法 ---

// Request 请求配置
type Request struct {
	method  string
	url     string
	headers map[string]string
	body    any
	ctx     context.Context
	gctx    *gctx.Context
}

// NewRequest 创建请求
func (c *Client) NewRequest(method, url string) *Request {
	return &Request{
		method:  method,
		url:     url,
		headers: make(map[string]string),
	}
}

// WithContext 设置上下文
func (r *Request) WithContext(ctx context.Context) *Request {
	r.ctx = ctx
	return r
}

// WithGctx 设置 gctx.Context（自动传递链路追踪）
func (r *Request) WithGctx(gctx *gctx.Context) *Request {
	r.gctx = gctx
	r.ctx = gctx.Request.Context()
	return r
}

// WithHeader 设置请求头
func (r *Request) WithHeader(key, value string) *Request {
	r.headers[key] = value
	return r
}

// WithHeaders 设置多个请求头
func (r *Request) WithHeaders(headers map[string]string) *Request {
	for k, v := range headers {
		r.headers[k] = v
	}
	return r
}

// WithBody 设置请求体
func (r *Request) WithBody(body any) *Request {
	r.body = body
	return r
}

// WithJSON 设置 JSON 请求体
func (r *Request) WithJSON(body any) *Request {
	r.body = body
	r.headers["Content-Type"] = "application/json"
	return r
}

// Do 执行请求
func (c *Client) Do(r *Request) (*http.Response, error) {
	ctx := r.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	// 构建可重放的请求体（用于重试）
	var bodyBytes []byte
	if r.body != nil {
		switch v := r.body.(type) {
		case []byte:
			bodyBytes = v
		case string:
			bodyBytes = []byte(v)
		case io.Reader:
			data, err := io.ReadAll(v)
			if err != nil {
				return nil, err
			}
			bodyBytes = data
		default:
			data, err := json.Marshal(r.body)
			if err != nil {
				return nil, err
			}
			bodyBytes = data
		}
	}

	// 执行请求（带重试）
	var resp *http.Response
	var err error
	for i := 0; i <= c.maxRetry; i++ {
		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		// 每次重试都重新构造 request，避免 body 被消费后无法重试
		req, reqErr := http.NewRequestWithContext(ctx, r.method, r.url, bodyReader)
		if reqErr != nil {
			return nil, reqErr
		}

		// 设置请求头
		for k, v := range r.headers {
			req.Header.Set(k, v)
		}

		// 自动传递链路追踪
		if c.trace && r.gctx != nil {
			injectTrace(r.gctx, req)
		}

		resp, err = c.client.Do(req)
		if err == nil && resp.StatusCode < 500 {
			break
		}

		// 重试逻辑
		if i < c.maxRetry {
			logger.Debug("HTTP 请求重试",
				zap.String("url", r.url),
				zap.Int("attempt", i+1),
				zap.Error(err))
			time.Sleep(c.retryWait)
		}
	}

	return resp, err
}

// --- 便捷方法 ---

// Get 发送 GET 请求
func (c *Client) Get(ctx context.Context, url string, result ...any) (*http.Response, error) {
	req := c.NewRequest(http.MethodGet, url)
	if gctx, ok := ctx.(*gctx.Context); ok {
		req.WithGctx(gctx)
	} else {
		req.WithContext(ctx)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	if len(result) > 0 {
		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(result[0]); err != nil {
			return resp, err
		}
	}
	return resp, nil
}

// Post 发送 POST 请求
func (c *Client) Post(ctx context.Context, url string, body any, result ...any) (*http.Response, error) {
	req := c.NewRequest(http.MethodPost, url).WithJSON(body)
	if gctx, ok := ctx.(*gctx.Context); ok {
		req.WithGctx(gctx)
	} else {
		req.WithContext(ctx)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	if len(result) > 0 {
		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(result[0]); err != nil {
			return resp, err
		}
	}
	return resp, nil
}

// Put 发送 PUT 请求
func (c *Client) Put(ctx context.Context, url string, body any, result ...any) (*http.Response, error) {
	req := c.NewRequest(http.MethodPut, url).WithJSON(body)
	if gctx, ok := ctx.(*gctx.Context); ok {
		req.WithGctx(gctx)
	} else {
		req.WithContext(ctx)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	if len(result) > 0 {
		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(result[0]); err != nil {
			return resp, err
		}
	}
	return resp, nil
}

// Delete 发送 DELETE 请求
func (c *Client) Delete(ctx context.Context, url string, result ...any) (*http.Response, error) {
	req := c.NewRequest(http.MethodDelete, url)
	if gctx, ok := ctx.(*gctx.Context); ok {
		req.WithGctx(gctx)
	} else {
		req.WithContext(ctx)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	if len(result) > 0 {
		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(result[0]); err != nil {
			return resp, err
		}
	}
	return resp, nil
}

// --- 辅助函数 ---

// injectTrace 注入链路追踪信息
func injectTrace(ctx *gctx.Context, req *http.Request) {
	// 传递 traceID
	if traceID := ctx.TraceId(); traceID != "" {
		req.Header.Set("X-Trace-ID", traceID)
	}

	// 传递 UserContext
	uc := ctx.UserContext()
	if uc == nil || uc.IsEmpty() {
		return
	}

	req.Header.Set("X-User-ID", uc.UserId)
	if uc.Role != "" {
		req.Header.Set("X-User-Role", uc.Role)
	}
	if uc.TenantId != "" {
		req.Header.Set("X-Tenant-ID", uc.TenantId)
	}
	if uc.Username != "" {
		req.Header.Set("X-Username", uc.Username)
	}
	if uc.Email != "" {
		req.Header.Set("X-Email", uc.Email)
	}
}

// ParseResponse 解析响应
func ParseResponse[T any](resp *http.Response, err error) (*T, error) {
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.New(string(body))
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// --- 默认客户端 ---

var defaultClient = NewClient()

// Get 使用默认客户端发送 GET 请求
func Get(ctx context.Context, url string, result ...any) (*http.Response, error) {
	return defaultClient.Get(ctx, url, result...)
}

// Post 使用默认客户端发送 POST 请求
func Post(ctx context.Context, url string, body any, result ...any) (*http.Response, error) {
	return defaultClient.Post(ctx, url, body, result...)
}

// Put 使用默认客户端发送 PUT 请求
func Put(ctx context.Context, url string, body any, result ...any) (*http.Response, error) {
	return defaultClient.Put(ctx, url, body, result...)
}

// Delete 使用默认客户端发送 DELETE 请求
func Delete(ctx context.Context, url string, result ...any) (*http.Response, error) {
	return defaultClient.Delete(ctx, url, result...)
}
