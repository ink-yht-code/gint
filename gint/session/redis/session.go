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
// Modifications: Enhanced with dual-token support

package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ink-yht-code/gint/gint/internal/jwt"
	"github.com/ink-yht-code/gint/gint/session"
)

var _ session.Session = (*Session)(nil)

// Session Redis 会话实现
type Session struct {
	client     redis.Cmdable
	key        string        // Redis key
	claims     *jwt.Claims   // JWT 声明
	expiration time.Duration // 过期时间
}

// Set 设置会话数据
func (s *Session) Set(ctx context.Context, key string, val any) error {
	// 将值序列化为 JSON
	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("序列化数据失败: %w", err)
	}

	// 存储到 Redis
	err = s.client.HSet(ctx, s.key, key, data).Err()
	if err != nil {
		return fmt.Errorf("存储数据失败: %w", err)
	}

	// 更新过期时间
	return s.client.Expire(ctx, s.key, s.expiration).Err()
}

// Get 获取会话数据
func (s *Session) Get(ctx context.Context, key string) (any, error) {
	data, err := s.client.HGet(ctx, s.key, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("键 %s 不存在", key)
		}
		return nil, fmt.Errorf("获取数据失败: %w", err)
	}

	var result any
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		// 如果反序列化失败，直接返回字符串
		return data, nil
	}

	return result, nil
}

// Del 删除会话数据
func (s *Session) Del(ctx context.Context, key string) error {
	return s.client.HDel(ctx, s.key, key).Err()
}

// Destroy 销毁会话
func (s *Session) Destroy(ctx context.Context) error {
	return s.client.Del(ctx, s.key).Err()
}

// Claims 获取 JWT 声明
func (s *Session) Claims() *jwt.Claims {
	return s.claims
}

// Refresh 刷新会话过期时间
func (s *Session) Refresh(ctx context.Context) error {
	return s.client.Expire(ctx, s.key, s.expiration).Err()
}

// init 初始化会话数据
func (s *Session) init(ctx context.Context, data map[string]any) error {
	if len(data) == 0 {
		return nil
	}

	// 使用 Pipeline 批量设置
	pipe := s.client.Pipeline()

	for key, val := range data {
		jsonData, err := json.Marshal(val)
		if err != nil {
			return fmt.Errorf("序列化数据失败: %w", err)
		}
		pipe.HSet(ctx, s.key, key, jsonData)
	}

	// 设置过期时间
	pipe.Expire(ctx, s.key, s.expiration)

	// 执行 Pipeline
	_, err := pipe.Exec(ctx)
	return err
}

// newSession 创建新的 Redis Session
func newSession(ssid string, expiration time.Duration, client redis.Cmdable, claims *jwt.Claims) *Session {
	return &Session{
		client:     client,
		key:        sessionKey(ssid),
		claims:     claims,
		expiration: expiration,
	}
}

// sessionKey 生成 Session 的 Redis key
func sessionKey(ssid string) string {
	return "gint:session:" + ssid
}
