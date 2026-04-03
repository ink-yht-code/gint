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

	"github.com/ink-yht-code/gint/gctx"
	"github.com/ink-yht-code/gint/jwt"
	"github.com/ink-yht-code/gint/session"
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

// UserContext 从 Session 数据构建 UserContext
func (s *Session) UserContext(ctx context.Context) (*gctx.UserContext, error) {
	uc := &gctx.UserContext{
		UserId: s.claims.UserId,
	}

	// 从 Redis 中批量获取用户详情字段（使用下划线风格）
	fields := []string{"username", "role", "tenant_id", "email", "department"}
	results, err := s.client.HMGet(ctx, s.key, fields...).Result()
	if err != nil {
		return uc, nil // 即使获取失败也返回基本的 UserId
	}

	for i, field := range fields {
		if i >= len(results) || results[i] == nil {
			continue
		}
		if str, ok := results[i].(string); ok {
			switch field {
			case "username":
				uc.Username = str
			case "role":
				uc.Role = str
			case "tenant_id":
				uc.TenantId = str
			case "email":
				uc.Email = str
			case "department":
				uc.Department = str
			}
		}
	}

	return uc, nil
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
