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

package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ink-yht-code/gint/gctx"
	"github.com/ink-yht-code/gint/jwt"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
)

// Session 内存 Session 实现
type Session struct {
	id         string
	claims     *jwt.Claims
	data       map[string]any
	expireTime time.Time
	mu         sync.RWMutex
}

// Claims 获取 JWT Claims
func (s *Session) Claims() *jwt.Claims {
	return s.claims
}

// Get 获取 Session 数据
func (s *Session) Get(ctx context.Context, key string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 检查是否过期
	if time.Now().After(s.expireTime) {
		return nil, ErrSessionExpired
	}

	val, ok := s.data[key]
	if !ok {
		return nil, errors.New("key not found")
	}

	return val, nil
}

// Set 设置 Session 数据
func (s *Session) Set(ctx context.Context, key string, val any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否过期
	if time.Now().After(s.expireTime) {
		return ErrSessionExpired
	}

	s.data[key] = val
	return nil
}

// Del 删除 Session 数据
func (s *Session) Del(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否过期
	if time.Now().After(s.expireTime) {
		return ErrSessionExpired
	}

	delete(s.data, key)
	return nil
}

// Destroy 销毁 Session（内存实现中只是标记为过期）
func (s *Session) Destroy(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.expireTime = time.Now().Add(-time.Hour)
	return nil
}

// Refresh 刷新 Session 过期时间
func (s *Session) Refresh(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否已过期
	if time.Now().After(s.expireTime) {
		return ErrSessionExpired
	}

	// 延长过期时间（由 Provider 控制具体时长）
	return nil
}

// UserContext 从 Session 数据构建 UserContext
func (s *Session) UserContext(ctx context.Context) (*gctx.UserContext, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	uc := &gctx.UserContext{
		UserId: s.claims.UserId,
	}

	// 从 Session 数据中获取用户详情（使用下划线风格）
	if username, ok := s.data["username"]; ok {
		if v, ok := username.(string); ok {
			uc.Username = v
		}
	}
	if role, ok := s.data["role"]; ok {
		if v, ok := role.(string); ok {
			uc.Role = v
		}
	}
	if tenantId, ok := s.data["tenant_id"]; ok {
		if v, ok := tenantId.(string); ok {
			uc.TenantId = v
		}
	}
	if email, ok := s.data["email"]; ok {
		if v, ok := email.(string); ok {
			uc.Email = v
		}
	}
	if dept, ok := s.data["department"]; ok {
		if v, ok := dept.(string); ok {
			uc.Department = v
		}
	}

	return uc, nil
}
