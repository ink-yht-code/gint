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
// Modifications: Added RenewToken method for dual-token support, thread-safe global provider

package session

import (
	"context"
	"sync/atomic"

	"github.com/ink-yht-code/gint/gctx"
	"github.com/ink-yht-code/gint/internal/jwt"
)

const (
	// CtxSessionKey 在 Context 中存储 Session 的 key
	CtxSessionKey = "gint:session"
)

// Session 会话接口
// 混合了 JWT 的设计，轻量数据存储在 JWT 中，完整数据存储在 Redis 中
type Session interface {
	// Set 设置会话数据
	Set(ctx context.Context, key string, val any) error

	// Get 获取会话数据
	Get(ctx context.Context, key string) (any, error)

	// Del 删除会话数据
	Del(ctx context.Context, key string) error

	// Destroy 销毁整个会话
	Destroy(ctx context.Context) error

	// Claims 获取 JWT 中的声明数据
	Claims() *jwt.Claims

	// Refresh 刷新会话过期时间
	Refresh(ctx context.Context) error
}

// Provider 会话提供者接口
// 负责创建、获取、销毁会话
type Provider interface {
	// NewSession 创建新会话
	// userId: 用户 ID（string 类型）
	// jwtData: 存储在 JWT 中的数据
	// sessData: 存储在 Session 中的数据
	NewSession(ctx *gctx.Context, userId string, jwtData map[string]string, sessData map[string]any) (Session, error)

	// Get 获取会话
	// 会自动验证会话的有效性
	Get(ctx *gctx.Context) (Session, error)

	// Destroy 销毁会话
	Destroy(ctx *gctx.Context) error

	// RenewToken 刷新 Token
	// 在 Token 即将过期时调用
	RenewToken(ctx *gctx.Context) error
}

// TokenCarrier Token 载体接口
// 定义了如何在请求中携带和提取 Token
type TokenCarrier interface {
	// Inject 将 Token 注入到响应中
	Inject(ctx *gctx.Context, token string)

	// Extract 从请求中提取 Token
	Extract(ctx *gctx.Context) string

	// Clear 清除 Token
	Clear(ctx *gctx.Context)
}

var defaultProvider atomic.Value // 存储 Provider，并发安全

// SetDefaultProvider 设置默认的 Session Provider
// 注意：应该在程序启动时调用一次，不要在运行时频繁调用
func SetDefaultProvider(provider Provider) {
	defaultProvider.Store(provider)
}

// getDefaultProvider 获取默认 Provider
func getDefaultProvider() Provider {
	p := defaultProvider.Load()
	if p == nil {
		panic("session provider 未初始化，请先调用 SetDefaultProvider")
	}
	return p.(Provider)
}

// Get 使用默认 Provider 获取 Session
func Get(ctx *gctx.Context) (Session, error) {
	return getDefaultProvider().Get(ctx)
}

// NewSession 使用默认 Provider 创建 Session
func NewSession(ctx *gctx.Context, userId string, jwtData map[string]string, sessData map[string]any) (Session, error) {
	return getDefaultProvider().NewSession(ctx, userId, jwtData, sessData)
}
