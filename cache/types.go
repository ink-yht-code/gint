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

package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// Cache 缓存接口
type Cache interface {
	// Get 获取缓存
	Get(ctx context.Context, key string) ([]byte, error)
	// Set 设置缓存
	Set(ctx context.Context, key string, value []byte, expiration time.Duration) error
	// SetNX 仅当 key 不存在时设置
	SetNX(ctx context.Context, key string, value []byte, expiration time.Duration) (bool, error)
	// GetOrSet 获取缓存，不存在则调用 fn 设置
	GetOrSet(ctx context.Context, key string, fn func() ([]byte, error), expiration time.Duration) ([]byte, error)
	// Delete 删除缓存
	Delete(ctx context.Context, key string) error
	// Exists 检查 key 是否存在
	Exists(ctx context.Context, key string) (bool, error)
	// Expire 设置过期时间
	Expire(ctx context.Context, key string, expiration time.Duration) error
	// TTL 获取剩余过期时间
	TTL(ctx context.Context, key string) (time.Duration, error)
	// Clear 清空缓存
	Clear(ctx context.Context) error
}

// --- 类型安全的封装 ---

// TypedCache 类型安全的缓存
type TypedCache[T any] struct {
	cache  Cache
	prefix string
}

// NewTypedCache 创建类型安全的缓存
func NewTypedCache[T any](cache Cache, prefix string) *TypedCache[T] {
	return &TypedCache[T]{cache: cache, prefix: prefix}
}

// Get 获取缓存
func (c *TypedCache[T]) Get(ctx context.Context, key string) (*T, error) {
	data, err := c.cache.Get(ctx, c.prefix+key)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Set 设置缓存
func (c *TypedCache[T]) Set(ctx context.Context, key string, value T, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.cache.Set(ctx, c.prefix+key, data, expiration)
}

// GetOrSet 获取或设置缓存
func (c *TypedCache[T]) GetOrSet(ctx context.Context, key string, fn func() (*T, error), expiration time.Duration) (*T, error) {
	data, err := c.cache.GetOrSet(ctx, c.prefix+key, func() ([]byte, error) {
		v, err := fn()
		if err != nil {
			return nil, err
		}
		return json.Marshal(v)
	}, expiration)
	if err != nil {
		return nil, err
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete 删除缓存
func (c *TypedCache[T]) Delete(ctx context.Context, key string) error {
	return c.cache.Delete(ctx, c.prefix+key)
}

// --- 批量操作接口 ---

// BatchCache 批量操作接口
type BatchCache interface {
	Cache
	// MGet 批量获取
	MGet(ctx context.Context, keys []string) (map[string][]byte, error)
	// MSet 批量设置
	MSet(ctx context.Context, items map[string][]byte, expiration time.Duration) error
	// MDelete 批量删除
	MDelete(ctx context.Context, keys []string) error
}

// --- 计数器接口 ---

// Counter 计数器接口
type Counter interface {
	// Incr 自增
	Incr(ctx context.Context, key string) (int64, error)
	// IncrBy 自增指定值
	IncrBy(ctx context.Context, key string, value int64) (int64, error)
	// Decr 自减
	Decr(ctx context.Context, key string) (int64, error)
	// DecrBy 自减指定值
	DecrBy(ctx context.Context, key string, value int64) (int64, error)
	// Get 获取值
	Get(ctx context.Context, key string) (int64, error)
	// Set 设置值
	Set(ctx context.Context, key string, value int64) error
}

// --- 集合接口 ---

// SetCache 集合接口
type SetCache interface {
	// SAdd 添加成员
	SAdd(ctx context.Context, key string, members ...string) error
	// SRem 移除成员
	SRem(ctx context.Context, key string, members ...string) error
	// SMembers 获取所有成员
	SMembers(ctx context.Context, key string) ([]string, error)
	// SIsMember 检查是否是成员
	SIsMember(ctx context.Context, key, member string) (bool, error)
	// SCard 获取成员数量
	SCard(ctx context.Context, key string) (int64, error)
}

// --- 哈希接口 ---

// HashCache 哈希接口
type HashCache interface {
	// HSet 设置字段
	HSet(ctx context.Context, key string, field string, value []byte) error
	// HGet 获取字段
	HGet(ctx context.Context, key, field string) ([]byte, error)
	// HGetAll 获取所有字段
	HGetAll(ctx context.Context, key string) (map[string][]byte, error)
	// HDel 删除字段
	HDel(ctx context.Context, key string, fields ...string) error
	// HExists 检查字段是否存在
	HExists(ctx context.Context, key, field string) (bool, error)
	// HLen 获取字段数量
	HLen(ctx context.Context, key string) (int64, error)
}

// --- 过期策略 ---

// ExpireStrategy 过期策略
type ExpireStrategy int

const (
	// ExpireNever 永不过期
	ExpireNever ExpireStrategy = iota
	// ExpireAfterWrite 写入后过期
	ExpireAfterWrite
	// ExpireAfterAccess 访问后过期
	ExpireAfterAccess
	// ExpireAfterWriteOrAccess 写入或访问后过期
	ExpireAfterWriteOrAccess
)

// Config 缓存配置
type Config struct {
	// DefaultExpiration 默认过期时间
	DefaultExpiration time.Duration
	// CleanupInterval 清理间隔（内存缓存）
	CleanupInterval time.Duration
	// MaxSize 最大容量（内存缓存）
	MaxSize int
	// ExpireStrategy 过期策略
	ExpireStrategy ExpireStrategy
}

// DefaultConfig 默认配置
func DefaultConfig() Config {
	return Config{
		DefaultExpiration: 30 * time.Minute,
		CleanupInterval:   1 * time.Minute,
		MaxSize:           10000,
		ExpireStrategy:    ExpireAfterWrite,
	}
}

// --- 错误定义 ---

var (
	// ErrCacheNotFound 缓存不存在
	ErrCacheNotFound = errors.New("cache: key not found")
	// ErrCacheExpired 缓存已过期
	ErrCacheExpired = errors.New("cache: key expired")
	// ErrCacheFull 缓存已满
	ErrCacheFull = errors.New("cache: full")
	// ErrNilValue 空值
	ErrNilValue = errors.New("cache: nil value")
	// ErrCacheNotInitialized 默认缓存未初始化
	ErrCacheNotInitialized = errors.New("cache: default cache not initialized")
)

// --- 全局缓存 ---

var defaultCache Cache

// SetDefault 设置默认缓存
func SetDefault(cache Cache) {
	defaultCache = cache
}

// Default 获取默认缓存
func Default() Cache {
	return defaultCache
}

// Get 从默认缓存获取
func Get(ctx context.Context, key string) ([]byte, error) {
	if defaultCache == nil {
		return nil, ErrCacheNotInitialized
	}
	return defaultCache.Get(ctx, key)
}

// Set 设置默认缓存
func Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	if defaultCache == nil {
		return ErrCacheNotInitialized
	}
	return defaultCache.Set(ctx, key, value, expiration)
}

// Delete 删除默认缓存
func Delete(ctx context.Context, key string) error {
	if defaultCache == nil {
		return ErrCacheNotInitialized
	}
	return defaultCache.Delete(ctx, key)
}
