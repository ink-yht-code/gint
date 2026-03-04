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

package memory

import (
	"errors"
	"sync"
	"time"

	"github.com/ink-yht-code/gint/gctx"
	"github.com/ink-yht-code/gint/internal/jwt"
	"github.com/ink-yht-code/gint/session"

	"github.com/google/uuid"
)

// Provider 内存 Session Provider
// 注意：仅用于开发测试，生产环境请使用 Redis
type Provider struct {
	jwtManager jwt.Manager
	expiration time.Duration
	carrier    session.TokenCarrier
	sessions   map[string]*Session // sessionID -> Session
	mu         sync.RWMutex
}

// NewProvider 创建内存 Session Provider
// jwtKey: JWT 签名密钥
// accessExpire: Access Token 过期时间（建议 15 分钟 - 2 小时）
// refreshExpire: Refresh Token 过期时间（建议 7 天 - 30 天）
// carrier: Token 载体（Header 或 Cookie）
func NewProvider(jwtKey string, accessExpire, refreshExpire time.Duration, carrier session.TokenCarrier) *Provider {
	p := &Provider{
		jwtManager: jwt.NewManager(jwt.NewOptions(jwtKey, accessExpire, refreshExpire)),
		expiration: refreshExpire, // Session 过期时间使用 Refresh Token 的过期时间
		carrier:    carrier,
		sessions:   make(map[string]*Session),
	}

	// 启动定期清理过期 Session 的协程
	go p.cleanExpiredSessions()

	return p
}

// NewSession 创建新的 Session
func (p *Provider) NewSession(ctx *gctx.Context, userId string, jwtData map[string]string, sessData map[string]any) (session.Session, error) {

	// 生成 Session ID
	sessionId := uuid.New().String()

	// 生成 JWT Claims
	claims := jwt.Claims{
		UserId: userId,
		SSID:   sessionId,
		Data:   jwtData,
	}

	// 生成 Token 对（Access Token + Refresh Token）
	tokenPair, err := p.jwtManager.GenerateTokenPair(claims)
	if err != nil {
		return nil, err
	}

	// 创建 Session
	sess := &Session{
		id:         sessionId,
		claims:     &claims,
		data:       sessData,
		expireTime: time.Now().Add(p.expiration),
	}

	// 存储到内存
	p.mu.Lock()
	p.sessions[sessionId] = sess
	p.mu.Unlock()

	// 注入 Access Token
	p.carrier.Inject(ctx, tokenPair.AccessToken)

	// 注入 Refresh Token（通过自定义 Header）
	ctx.Context.Header("X-Refresh-Token", tokenPair.RefreshToken)

	return sess, nil
}

// Get 获取已存在的 Session
func (p *Provider) Get(ctx *gctx.Context) (session.Session, error) {
	// 提取 Token
	token := p.carrier.Extract(ctx)
	if token == "" {
		return nil, errors.New("token not found")
	}

	// 验证 Token
	claims, err := p.jwtManager.VerifyToken(token)
	if err != nil {
		return nil, err
	}

	// 从内存中获取 Session（使用读锁）
	p.mu.RLock()
	sess, ok := p.sessions[claims.SSID]
	p.mu.RUnlock()

	if !ok {
		return nil, ErrSessionNotFound
	}

	// 检查是否过期并续期（只锁 Session，不锁 Provider）
	sess.mu.Lock()
	now := time.Now()
	if now.After(sess.expireTime) {
		sess.mu.Unlock()
		// 删除过期的 Session
		p.mu.Lock()
		delete(p.sessions, claims.SSID)
		p.mu.Unlock()
		return nil, ErrSessionExpired
	}

	// 自动续期
	sess.expireTime = now.Add(p.expiration)
	sess.mu.Unlock()

	return sess, nil
}

// Destroy 销毁 Session
func (p *Provider) Destroy(ctx *gctx.Context) error {
	// 提取 Token
	token := p.carrier.Extract(ctx)
	if token == "" {
		return errors.New("token not found")
	}

	// 验证 Token
	claims, err := p.jwtManager.VerifyToken(token)
	if err != nil {
		return err
	}

	// 从内存中删除 Session
	p.mu.Lock()
	delete(p.sessions, claims.SSID)
	p.mu.Unlock()

	// 清除 Token
	p.carrier.Clear(ctx)

	return nil
}

// RenewToken 刷新 Token（使用 Refresh Token 获取新的 Access Token）
func (p *Provider) RenewToken(ctx *gctx.Context) error {
	// 从请求中提取 Refresh Token
	refreshToken := ctx.GetHeader("X-Refresh-Token")
	if refreshToken == "" {
		return errors.New("未找到 Refresh Token")
	}

	// 验证 Refresh Token
	claims, err := p.jwtManager.VerifyRefreshToken(refreshToken)
	if err != nil {
		return err
	}

	// 从内存中获取 Session
	p.mu.RLock()
	sess, ok := p.sessions[claims.SSID]
	p.mu.RUnlock()

	if !ok {
		return errors.New("会话不存在")
	}

	// 检查是否过期
	if time.Now().After(sess.expireTime) {
		return errors.New("会话已过期")
	}

	// 生成新的 Token 对
	tokenPair, err := p.jwtManager.GenerateTokenPair(*claims)
	if err != nil {
		return err
	}

	// 注入新的 Access Token
	p.carrier.Inject(ctx, tokenPair.AccessToken)

	// 注入新的 Refresh Token
	ctx.Context.Header("X-Refresh-Token", tokenPair.RefreshToken)

	// 刷新过期时间
	sess.mu.Lock()
	sess.expireTime = time.Now().Add(p.expiration)
	sess.mu.Unlock()

	return nil
}

// cleanExpiredSessions 定期清理过期的 Session
func (p *Provider) cleanExpiredSessions() {
	ticker := time.NewTicker(time.Minute * 5) // 每 5 分钟清理一次
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()

		// 先收集过期的 Session ID（避免在持有锁时检查每个 Session）
		expiredIDs := make([]string, 0)

		p.mu.RLock()
		for id, sess := range p.sessions {
			// 不加锁快速检查（过期时间只会延后，不会提前）
			if now.After(sess.expireTime) {
				expiredIDs = append(expiredIDs, id)
			}
		}
		p.mu.RUnlock()

		// 删除过期的 Session
		if len(expiredIDs) > 0 {
			p.mu.Lock()
			for _, id := range expiredIDs {
				// 再次检查，因为可能在这期间被续期了
				if sess, ok := p.sessions[id]; ok {
					if now.After(sess.expireTime) {
						delete(p.sessions, id)
					}
				}
			}
			p.mu.Unlock()
		}
	}
}
