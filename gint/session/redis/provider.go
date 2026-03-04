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
// Modifications: Implemented dual-token mechanism (Access + Refresh Token)

package redis

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/ink-yht-code/gint/gctx"
	"github.com/ink-yht-code/gint/internal/jwt"
	"github.com/ink-yht-code/gint/session"
)

var _ session.Provider = (*Provider)(nil)

// Provider Redis Session 提供者
type Provider struct {
	client       redis.Cmdable
	jwtManager   jwt.Manager
	tokenCarrier session.TokenCarrier
	expiration   time.Duration
}

// NewProvider 创建 Redis Session 提供者
// client: Redis 客户端
// jwtKey: JWT 签名密钥
// accessExpire: Access Token 过期时间（建议 15 分钟 - 2 小时）
// refreshExpire: Refresh Token 过期时间（建议 7 天 - 30 天）
// tokenCarrier: Token 载体（如何传输 Token）
func NewProvider(client redis.Cmdable, jwtKey string, accessExpire, refreshExpire time.Duration, tokenCarrier session.TokenCarrier) *Provider {
	return &Provider{
		client:       client,
		jwtManager:   jwt.NewManager(jwt.NewOptions(jwtKey, accessExpire, refreshExpire)),
		tokenCarrier: tokenCarrier,
		expiration:   refreshExpire, // Session 过期时间使用 Refresh Token 的过期时间
	}
}

// NewSession 创建新会话
func (p *Provider) NewSession(ctx *gctx.Context, userId string, jwtData map[string]string, sessData map[string]any) (session.Session, error) {
	// 生成 Session ID
	ssid := uuid.New().String()

	// 创建 JWT Claims
	claims := jwt.Claims{
		UserId: userId,
		SSID:   ssid,
		Data:   jwtData,
	}

	// 生成 Token 对（Access Token + Refresh Token）
	tokenPair, err := p.jwtManager.GenerateTokenPair(claims)
	if err != nil {
		return nil, fmt.Errorf("生成 Token 失败: %w", err)
	}

	// 将 Access Token 注入到响应中
	p.tokenCarrier.Inject(ctx, tokenPair.AccessToken)

	// 将 Refresh Token 也注入到响应中（通过自定义 Header）
	ctx.Context.Header("X-Refresh-Token", tokenPair.RefreshToken)

	// 创建 Session
	sess := newSession(ssid, p.expiration, p.client, &claims)

	// 初始化 Session 数据
	if sessData == nil {
		sessData = make(map[string]any)
	}
	sessData["user_id"] = userId
	sessData["created_at"] = time.Now().Unix()

	if err := sess.init(ctx, sessData); err != nil {
		return nil, fmt.Errorf("初始化会话失败: %w", err)
	}

	return sess, nil
}

// Get 获取会话
func (p *Provider) Get(ctx *gctx.Context) (session.Session, error) {
	// 先尝试从上下文中获取
	if val, exists := ctx.Get(session.CtxSessionKey); exists {
		if sess, ok := val.(session.Session); ok {
			return sess, nil
		}
	}

	// 从请求中提取 Token
	token := p.tokenCarrier.Extract(ctx)
	if token == "" {
		return nil, fmt.Errorf("未找到 Token")
	}

	// 验证 Token
	claims, err := p.jwtManager.VerifyToken(token)
	if err != nil {
		return nil, fmt.Errorf("验证 Token 失败: %w", err)
	}

	// 创建 Session
	sess := newSession(claims.SSID, p.expiration, p.client, claims)

	// 验证 Session 是否存在
	exists, err := p.client.Exists(ctx, sessionKey(claims.SSID)).Result()
	if err != nil {
		return nil, fmt.Errorf("检查会话失败: %w", err)
	}
	if exists == 0 {
		return nil, fmt.Errorf("会话不存在或已过期")
	}

	// 将 Session 存储到上下文中
	ctx.Set(session.CtxSessionKey, sess)

	return sess, nil
}

// Destroy 销毁会话
func (p *Provider) Destroy(ctx *gctx.Context) error {
	// 获取会话
	sess, err := p.Get(ctx)
	if err != nil {
		return err
	}

	// 清除 Token
	p.tokenCarrier.Clear(ctx)

	// 销毁 Session
	return sess.Destroy(ctx)
}

// RenewToken 刷新 Token（使用 Refresh Token 获取新的 Access Token）
func (p *Provider) RenewToken(ctx *gctx.Context) error {
	// 从请求中提取 Refresh Token
	refreshToken := ctx.GetHeader("X-Refresh-Token")
	if refreshToken == "" {
		return fmt.Errorf("未找到 Refresh Token")
	}

	// 验证 Refresh Token
	claims, err := p.jwtManager.VerifyRefreshToken(refreshToken)
	if err != nil {
		return fmt.Errorf("验证 Refresh Token 失败: %w", err)
	}

	// 验证 Session 是否存在
	exists, err := p.client.Exists(ctx, sessionKey(claims.SSID)).Result()
	if err != nil {
		return fmt.Errorf("检查会话失败: %w", err)
	}
	if exists == 0 {
		return fmt.Errorf("会话不存在或已过期")
	}

	// 生成新的 Token 对
	tokenPair, err := p.jwtManager.GenerateTokenPair(*claims)
	if err != nil {
		return fmt.Errorf("生成新 Token 失败: %w", err)
	}

	// 注入新的 Access Token
	p.tokenCarrier.Inject(ctx, tokenPair.AccessToken)

	// 注入新的 Refresh Token
	ctx.Context.Header("X-Refresh-Token", tokenPair.RefreshToken)

	// 刷新 Redis 中的过期时间
	return p.client.Expire(ctx, sessionKey(claims.SSID), p.expiration).Err()
}
