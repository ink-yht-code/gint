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

package redis

import (
	"context"
	"time"

	"github.com/ink-yht-code/gint/cache"
	"github.com/redis/go-redis/v9"
)

// Redis Redis 缓存
type Redis struct {
	client *redis.Client
	prefix string
}

// New 创建 Redis 缓存
func New(client *redis.Client, prefix ...string) *Redis {
	p := ""
	if len(prefix) > 0 {
		p = prefix[0]
	}
	return &Redis{
		client: client,
		prefix: p,
	}
}

// key 添加前缀
func (r *Redis) key(key string) string {
	if r.prefix == "" {
		return key
	}
	return r.prefix + ":" + key
}

// Get 获取缓存
func (r *Redis) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, r.key(key)).Bytes()
	if err == redis.Nil {
		return nil, cache.ErrCacheNotFound
	}
	if err != nil {
		return nil, err
	}
	return val, nil
}

// Set 设置缓存
func (r *Redis) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	return r.client.Set(ctx, r.key(key), value, expiration).Err()
}

// SetNX 仅当 key 不存在时设置
func (r *Redis) SetNX(ctx context.Context, key string, value []byte, expiration time.Duration) (bool, error) {
	ok, err := r.client.SetNX(ctx, r.key(key), value, expiration).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

// GetOrSet 获取或设置缓存
func (r *Redis) GetOrSet(ctx context.Context, key string, fn func() ([]byte, error), expiration time.Duration) ([]byte, error) {
	// 先尝试获取
	if val, err := r.Get(ctx, key); err == nil {
		return val, nil
	}

	// 调用 fn 获取值
	val, err := fn()
	if err != nil {
		return nil, err
	}

	// 设置缓存
	if err := r.Set(ctx, key, val, expiration); err != nil {
		return nil, err
	}

	return val, nil
}

// Delete 删除缓存
func (r *Redis) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, r.key(key)).Err()
}

// Exists 检查 key 是否存在
func (r *Redis) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, r.key(key)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Expire 设置过期时间
func (r *Redis) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, r.key(key), expiration).Err()
}

// TTL 获取剩余过期时间
func (r *Redis) TTL(ctx context.Context, key string) (time.Duration, error) {
	d, err := r.client.TTL(ctx, r.key(key)).Result()
	if err != nil {
		return 0, err
	}
	if d < 0 {
		return -1, nil // 永不过期或不存在
	}
	return d, nil
}

// Clear 清空缓存（慎用）
func (r *Redis) Clear(ctx context.Context) error {
	if r.prefix == "" {
		return r.client.FlushDB(ctx).Err()
	}
	// 删除匹配前缀的所有 key
	iter := r.client.Scan(ctx, 0, r.prefix+":*", 0).Iterator()
	for iter.Next(ctx) {
		if err := r.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

// --- 实现 BatchCache ---

// MGet 批量获取
func (r *Redis) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	fullKeys := make([]string, len(keys))
	for i, k := range keys {
		fullKeys[i] = r.key(k)
	}

	vals, err := r.client.MGet(ctx, fullKeys...).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string][]byte)
	for i, val := range vals {
		if val != nil {
			if s, ok := val.(string); ok {
				result[keys[i]] = []byte(s)
			}
		}
	}
	return result, nil
}

// MSet 批量设置
func (r *Redis) MSet(ctx context.Context, items map[string][]byte, expiration time.Duration) error {
	pipe := r.client.Pipeline()
	for k, v := range items {
		pipe.Set(ctx, r.key(k), v, expiration)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// MDelete 批量删除
func (r *Redis) MDelete(ctx context.Context, keys []string) error {
	fullKeys := make([]string, len(keys))
	for i, k := range keys {
		fullKeys[i] = r.key(k)
	}
	return r.client.Del(ctx, fullKeys...).Err()
}

// --- 实现 Counter ---

// Incr 自增
func (r *Redis) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, r.key(key)).Result()
}

// IncrBy 自增指定值
func (r *Redis) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return r.client.IncrBy(ctx, r.key(key), value).Result()
}

// Decr 自减
func (r *Redis) Decr(ctx context.Context, key string) (int64, error) {
	return r.client.Decr(ctx, r.key(key)).Result()
}

// DecrBy 自减指定值
func (r *Redis) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	return r.client.DecrBy(ctx, r.key(key), value).Result()
}

// GetCounter 获取计数器值
func (r *Redis) GetCounter(ctx context.Context, key string) (int64, error) {
	return r.client.Get(ctx, r.key(key)).Int64()
}

// SetCounter 设置计数器值
func (r *Redis) SetCounter(ctx context.Context, key string, value int64) error {
	return r.client.Set(ctx, r.key(key), value, 0).Err()
}

// --- 实现 SetCache ---

// SAdd 添加集合成员
func (r *Redis) SAdd(ctx context.Context, key string, members ...string) error {
	vals := make([]interface{}, len(members))
	for i, m := range members {
		vals[i] = m
	}
	return r.client.SAdd(ctx, r.key(key), vals...).Err()
}

// SRem 移除集合成员
func (r *Redis) SRem(ctx context.Context, key string, members ...string) error {
	vals := make([]interface{}, len(members))
	for i, m := range members {
		vals[i] = m
	}
	return r.client.SRem(ctx, r.key(key), vals...).Err()
}

// SMembers 获取集合所有成员
func (r *Redis) SMembers(ctx context.Context, key string) ([]string, error) {
	return r.client.SMembers(ctx, r.key(key)).Result()
}

// SIsMember 检查是否是集合成员
func (r *Redis) SIsMember(ctx context.Context, key, member string) (bool, error) {
	return r.client.SIsMember(ctx, r.key(key), member).Result()
}

// SCard 获取集合成员数量
func (r *Redis) SCard(ctx context.Context, key string) (int64, error) {
	return r.client.SCard(ctx, r.key(key)).Result()
}

// --- 实现 HashCache ---

// HSet 设置哈希字段
func (r *Redis) HSet(ctx context.Context, key string, field string, value []byte) error {
	return r.client.HSet(ctx, r.key(key), field, value).Err()
}

// HGet 获取哈希字段
func (r *Redis) HGet(ctx context.Context, key, field string) ([]byte, error) {
	val, err := r.client.HGet(ctx, r.key(key), field).Bytes()
	if err == redis.Nil {
		return nil, cache.ErrCacheNotFound
	}
	return val, err
}

// HGetAll 获取哈希所有字段
func (r *Redis) HGetAll(ctx context.Context, key string) (map[string][]byte, error) {
	vals, err := r.client.HGetAll(ctx, r.key(key)).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string][]byte)
	for k, v := range vals {
		result[k] = []byte(v)
	}
	return result, nil
}

// HDel 删除哈希字段
func (r *Redis) HDel(ctx context.Context, key string, fields ...string) error {
	return r.client.HDel(ctx, r.key(key), fields...).Err()
}

// HExists 检查哈希字段是否存在
func (r *Redis) HExists(ctx context.Context, key, field string) (bool, error) {
	return r.client.HExists(ctx, r.key(key), field).Result()
}

// HLen 获取哈希字段数量
func (r *Redis) HLen(ctx context.Context, key string) (int64, error) {
	return r.client.HLen(ctx, r.key(key)).Result()
}
