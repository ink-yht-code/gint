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
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ink-yht-code/gint/gint/internal/jwt"
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
