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
// Modifications: Fixed concurrency issues, added sliding window limiter

package ratelimit

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Limiter 限流器接口
type Limiter interface {
	// Allow 检查是否允许请求
	// key: 限流键（如 IP、用户 ID 等）
	// 返回 true 表示允许，false 表示拒绝
	Allow(key string) bool
}

// KeyFunc 生成限流键的函数类型
type KeyFunc func(c *gin.Context) string

// Builder 限流中间件构建器
type Builder struct {
	limiter Limiter // 限流器
	keyFunc KeyFunc // 生成限流键的函数
}

// NewBuilder 创建限流中间件构建器
// 默认使用 IP 作为限流键
func NewBuilder(limiter Limiter) *Builder {
	return &Builder{
		limiter: limiter,
		keyFunc: func(c *gin.Context) string {
			return "ip:" + c.ClientIP()
		},
	}
}

// WithKeyFunc 设置自定义的限流键生成函数
func (b *Builder) WithKeyFunc(keyFunc KeyFunc) *Builder {
	b.keyFunc = keyFunc
	return b
}

// WithUserIDKey 使用用户 ID 作为限流键
func (b *Builder) WithUserIDKey() *Builder {
	b.keyFunc = func(c *gin.Context) string {
		if userId, exists := c.Get("user_id"); exists {
			if uid, ok := userId.(string); ok {
				return "user:" + uid
			}
		}
		// 如果没有用户 ID，使用 IP
		return "ip:" + c.ClientIP()
	}
	return b
}

// Build 构建中间件
func (b *Builder) Build() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := b.keyFunc(c)

		// 检查是否允许请求
		if !b.limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code": 429,
				"msg":  "请求过于频繁，请稍后再试",
			})
			return
		}

		c.Next()
	}
}

// SimpleLimiter 简单的内存限流器（基于固定窗口，并发安全）
type SimpleLimiter struct {
	rate     int           // 每个窗口允许的请求数
	window   time.Duration // 窗口大小
	counters sync.Map      // 并发安全的计数器 map[string]*counter
	cleanup  time.Duration // 清理过期计数器的间隔
}

type counter struct {
	mu          sync.Mutex // 保护 count 和 windowStart
	count       int        // 当前计数
	windowStart time.Time  // 窗口开始时间
}

// NewSimpleLimiter 创建简单限流器
// rate: 每个窗口允许的请求数
// window: 窗口大小
func NewSimpleLimiter(rate int, window time.Duration) *SimpleLimiter {
	limiter := &SimpleLimiter{
		rate:    rate,
		window:  window,
		cleanup: window * 2, // 清理间隔为窗口大小的 2 倍
	}

	// 启动清理协程
	go limiter.cleanupLoop()

	return limiter
}

// Allow 检查是否允许请求（并发安全）
func (l *SimpleLimiter) Allow(key string) bool {
	now := time.Now()

	// 获取或创建计数器
	value, _ := l.counters.LoadOrStore(key, &counter{
		count:       0,
		windowStart: now,
	})
	c := value.(*counter)

	// 加锁保护计数器操作
	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查是否需要重置窗口
	if now.Sub(c.windowStart) >= l.window {
		c.count = 1
		c.windowStart = now
		return true
	}

	// 检查是否超过限制
	if c.count >= l.rate {
		return false
	}

	// 增加计数
	c.count++
	return true
}

// cleanupLoop 清理过期的计数器
func (l *SimpleLimiter) cleanupLoop() {
	ticker := time.NewTicker(l.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		l.counters.Range(func(key, value interface{}) bool {
			c := value.(*counter)
			c.mu.Lock()
			expired := now.Sub(c.windowStart) >= l.window*2
			c.mu.Unlock()

			if expired {
				l.counters.Delete(key)
			}
			return true
		})
	}
}

// IPKeyFunc 使用 IP 作为限流键
func IPKeyFunc(c *gin.Context) string {
	return fmt.Sprintf("ip:%s", c.ClientIP())
}

// UserIDKeyFunc 使用用户 ID 作为限流键
func UserIDKeyFunc(c *gin.Context) string {
	if userId, exists := c.Get("user_id"); exists {
		if uid, ok := userId.(string); ok {
			return fmt.Sprintf("user:%s", uid)
		}
	}
	return IPKeyFunc(c)
}

// PathKeyFunc 使用路径 + IP 作为限流键
func PathKeyFunc(c *gin.Context) string {
	return fmt.Sprintf("path:%s:ip:%s", c.Request.URL.Path, c.ClientIP())
}

// ============ 滑动窗口限流器（更精确，避免临界突刺问题） ============

// SlidingWindowLimiter 滑动窗口限流器
type SlidingWindowLimiter struct {
	rate     int           // 每个窗口允许的请求数
	window   time.Duration // 窗口大小
	counters sync.Map      // map[string]*slidingCounter
	cleanup  time.Duration // 清理间隔
}

type slidingCounter struct {
	mu       sync.Mutex
	requests []time.Time // 请求时间戳列表
}

// NewSlidingWindowLimiter 创建滑动窗口限流器
// rate: 每个窗口允许的请求数
// window: 窗口大小
func NewSlidingWindowLimiter(rate int, window time.Duration) *SlidingWindowLimiter {
	limiter := &SlidingWindowLimiter{
		rate:    rate,
		window:  window,
		cleanup: window * 2,
	}

	go limiter.cleanupLoop()
	return limiter
}

// Allow 检查是否允许请求
func (l *SlidingWindowLimiter) Allow(key string) bool {
	now := time.Now()

	// 获取或创建计数器
	value, _ := l.counters.LoadOrStore(key, &slidingCounter{
		requests: make([]time.Time, 0, l.rate),
	})
	c := value.(*slidingCounter)

	c.mu.Lock()
	defer c.mu.Unlock()

	// 移除过期的请求记录
	cutoff := now.Add(-l.window)
	validRequests := make([]time.Time, 0, len(c.requests))
	for _, t := range c.requests {
		if t.After(cutoff) {
			validRequests = append(validRequests, t)
		}
	}
	c.requests = validRequests

	// 检查是否超过限制
	if len(c.requests) >= l.rate {
		return false
	}

	// 记录本次请求
	c.requests = append(c.requests, now)
	return true
}

// cleanupLoop 清理过期的计数器
func (l *SlidingWindowLimiter) cleanupLoop() {
	ticker := time.NewTicker(l.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		cutoff := now.Add(-l.window * 2)

		l.counters.Range(func(key, value interface{}) bool {
			c := value.(*slidingCounter)
			c.mu.Lock()
			// 如果所有请求都已过期，删除该计数器
			if len(c.requests) == 0 || c.requests[len(c.requests)-1].Before(cutoff) {
				c.mu.Unlock()
				l.counters.Delete(key)
			} else {
				c.mu.Unlock()
			}
			return true
		})
	}
}
