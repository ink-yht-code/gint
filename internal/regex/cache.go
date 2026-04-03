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

package regex

import (
	"sync"

	"github.com/dlclark/regexp2"
)

// Cache 正则表达式缓存
type Cache struct {
	mu    sync.RWMutex
	cache map[string]*regexp2.Regexp
}

// NewCache 创建新的缓存
func NewCache() *Cache {
	return &Cache{
		cache: make(map[string]*regexp2.Regexp),
	}
}

// Get 获取或编译正则表达式
func (c *Cache) Get(pattern string, options regexp2.RegexOptions) *regexp2.Regexp {
	key := pattern + string(rune(options))

	// 先尝试读锁
	c.mu.RLock()
	if regex, exists := c.cache[key]; exists {
		c.mu.RUnlock()
		return regex
	}
	c.mu.RUnlock()

	// 需要编译，获取写锁
	c.mu.Lock()
	defer c.mu.Unlock()

	// 双重检查，防止并发编译
	if regex, exists := c.cache[key]; exists {
		return regex
	}

	// 编译正则表达式
	regex := regexp2.MustCompile(pattern, options)
	c.cache[key] = regex
	return regex
}

// Clear 清空缓存
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*regexp2.Regexp)
}

// Size 获取缓存大小
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// 全局缓存实例
var globalCache = NewCache()

// MustCompile 编译正则表达式（使用全局缓存）
func MustCompile(pattern string, options regexp2.RegexOptions) *regexp2.Regexp {
	return globalCache.Get(pattern, options)
}

// ClearGlobalCache 清空全局缓存
func ClearGlobalCache() {
	globalCache.Clear()
}

// GlobalCacheSize 获取全局缓存大小
func GlobalCacheSize() int {
	return globalCache.Size()
}
