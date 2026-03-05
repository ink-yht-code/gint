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
//
// This file is derived from ginx (https://github.com/ecodeclub/ginx)
// Original Copyright by ecodeclub and contributors
// Modifications: Support for dual-token injection

package cookie

import (
	"github.com/ink-yht-code/gint/gint/gctx"
	"github.com/ink-yht-code/gint/gint/session"
)

var _ session.TokenCarrier = (*Carrier)(nil)

// Carrier 基于 Cookie 的 Token 载体
type Carrier struct {
	cookieName string // Cookie 名称
	domain     string // Cookie 域名
	path       string // Cookie 路径
	maxAge     int    // Cookie 最大存活时间（秒）
	secure     bool   // 是否只在 HTTPS 下传输
	httpOnly   bool   // 是否禁止 JavaScript 访问
}

// Option Cookie 配置选项
type Option func(*Carrier)

// WithDomain 设置 Cookie 域名
func WithDomain(domain string) Option {
	return func(c *Carrier) {
		c.domain = domain
	}
}

// WithPath 设置 Cookie 路径
func WithPath(path string) Option {
	return func(c *Carrier) {
		c.path = path
	}
}

// WithMaxAge 设置 Cookie 最大存活时间
func WithMaxAge(maxAge int) Option {
	return func(c *Carrier) {
		c.maxAge = maxAge
	}
}

// WithSecure 设置是否只在 HTTPS 下传输
func WithSecure(secure bool) Option {
	return func(c *Carrier) {
		c.secure = secure
	}
}

// WithHttpOnly 设置是否禁止 JavaScript 访问
func WithHttpOnly(httpOnly bool) Option {
	return func(c *Carrier) {
		c.httpOnly = httpOnly
	}
}

// NewCarrier 创建 Cookie Token 载体
// cookieName: Cookie 名称，默认为 "gint_token"
func NewCarrier(cookieName string, opts ...Option) *Carrier {
	if cookieName == "" {
		cookieName = "gint_token"
	}

	c := &Carrier{
		cookieName: cookieName,
		path:       "/",
		maxAge:     86400, // 默认 24 小时
		httpOnly:   true,  // 默认启用 HttpOnly
		secure:     false, // 默认不强制 HTTPS
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Inject 将 Token 注入到 Cookie 中
func (c *Carrier) Inject(ctx *gctx.Context, token string) {
	ctx.SetCookie(
		c.cookieName,
		token,
		c.maxAge,
		c.path,
		c.domain,
		c.secure,
		c.httpOnly,
	)
}

// Extract 从 Cookie 中提取 Token
func (c *Carrier) Extract(ctx *gctx.Context) string {
	return ctx.Cookie(c.cookieName).StringOr("")
}

// Clear 清除 Cookie
func (c *Carrier) Clear(ctx *gctx.Context) {
	ctx.SetCookie(
		c.cookieName,
		"",
		-1, // 设置为负数表示删除
		c.path,
		c.domain,
		c.secure,
		c.httpOnly,
	)
}
