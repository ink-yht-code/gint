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
	"sync"
	"time"

	"github.com/ink-yht-code/gint/cache"
)

// item 缓存项
type item struct {
	value      []byte
	expiration int64
}

// expired 检查是否过期
func (i *item) expired() bool {
	if i.expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > i.expiration
}

// Memory 内存缓存
type Memory struct {
	items    map[string]*item
	mu       sync.RWMutex
	config   cache.Config
	stopChan chan struct{}
}

// New 创建内存缓存
func New(config ...cache.Config) *Memory {
	cfg := cache.DefaultConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	m := &Memory{
		items:    make(map[string]*item),
		config:   cfg,
		stopChan: make(chan struct{}),
	}

	// 启动清理协程
	if cfg.CleanupInterval > 0 {
		go m.cleanup(cfg.CleanupInterval)
	}

	return m
}

// cleanup 定期清理过期项
func (m *Memory) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.DeleteExpired()
		case <-m.stopChan:
			return
		}
	}
}

// DeleteExpired 删除所有过期项
func (m *Memory) DeleteExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, item := range m.items {
		if item.expired() {
			delete(m.items, key)
		}
	}
}

// Close 关闭缓存
func (m *Memory) Close() {
	close(m.stopChan)
}

// Get 获取缓存
func (m *Memory) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, ok := m.items[key]
	if !ok {
		return nil, cache.ErrCacheNotFound
	}

	if item.expired() {
		return nil, cache.ErrCacheExpired
	}

	return item.value, nil
}

// Set 设置缓存
func (m *Memory) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查容量
	if m.config.MaxSize > 0 && len(m.items) >= m.config.MaxSize {
		// 删除最早过期的项
		m.deleteOldest()
	}

	var exp int64
	if expiration > 0 {
		exp = time.Now().Add(expiration).UnixNano()
	} else if m.config.DefaultExpiration > 0 {
		exp = time.Now().Add(m.config.DefaultExpiration).UnixNano()
	}

	m.items[key] = &item{
		value:      value,
		expiration: exp,
	}

	return nil
}

// deleteOldest 删除最早过期的项
func (m *Memory) deleteOldest() {
	var oldestKey string
	var oldestExp int64 = -1

	for key, item := range m.items {
		if oldestExp == -1 || item.expiration < oldestExp {
			oldestKey = key
			oldestExp = item.expiration
		}
	}

	if oldestKey != "" {
		delete(m.items, oldestKey)
	}
}

// SetNX 仅当 key 不存在时设置
func (m *Memory) SetNX(ctx context.Context, key string, value []byte, expiration time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if item, ok := m.items[key]; ok && !item.expired() {
		return false, nil
	}

	var exp int64
	if expiration > 0 {
		exp = time.Now().Add(expiration).UnixNano()
	}

	m.items[key] = &item{
		value:      value,
		expiration: exp,
	}

	return true, nil
}

// GetOrSet 获取或设置缓存
func (m *Memory) GetOrSet(ctx context.Context, key string, fn func() ([]byte, error), expiration time.Duration) ([]byte, error) {
	// 先尝试获取
	if val, err := m.Get(ctx, key); err == nil {
		return val, nil
	}

	// 调用 fn 获取值
	val, err := fn()
	if err != nil {
		return nil, err
	}

	// 设置缓存
	if err := m.Set(ctx, key, val, expiration); err != nil {
		return nil, err
	}

	return val, nil
}

// Delete 删除缓存
func (m *Memory) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.items, key)
	return nil
}

// Exists 检查 key 是否存在
func (m *Memory) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, ok := m.items[key]
	if !ok {
		return false, nil
	}

	return !item.expired(), nil
}

// Expire 设置过期时间
func (m *Memory) Expire(ctx context.Context, key string, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	item, ok := m.items[key]
	if !ok {
		return cache.ErrCacheNotFound
	}

	item.expiration = time.Now().Add(expiration).UnixNano()
	return nil
}

// TTL 获取剩余过期时间
func (m *Memory) TTL(ctx context.Context, key string) (time.Duration, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, ok := m.items[key]
	if !ok {
		return 0, cache.ErrCacheNotFound
	}

	if item.expiration == 0 {
		return -1, nil // 永不过期
	}

	remaining := time.Until(time.Unix(0, item.expiration))
	if remaining < 0 {
		return 0, cache.ErrCacheExpired
	}

	return remaining, nil
}

// Clear 清空缓存
func (m *Memory) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.items = make(map[string]*item)
	return nil
}

// Count 获取缓存项数量
func (m *Memory) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, item := range m.items {
		if !item.expired() {
			count++
		}
	}
	return count
}

// Keys 获取所有 key
func (m *Memory) Keys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.items))
	for key, item := range m.items {
		if !item.expired() {
			keys = append(keys, key)
		}
	}
	return keys
}
