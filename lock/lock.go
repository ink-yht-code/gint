// Copyright 2025 ink-yht-code
//
// Proprietary License

package lock

import (
	"context"
	"time"
)

// Locker 分布式锁接口
type Locker interface {
	// TryLock 尝试获取锁，不阻塞
	// 返回 true 表示获取成功
	TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error)

	// Lock 获取锁，阻塞直到成功或超时
	Lock(ctx context.Context, key string, ttl time.Duration, opts ...Option) error

	// Unlock 释放锁
	Unlock(ctx context.Context, key string) error

	// TryLockWithRetry 尝试获取锁，带重试
	TryLockWithRetry(ctx context.Context, key string, ttl time.Duration, retryInterval time.Duration, maxRetries int) (bool, error)

	// IsLocked 检查锁是否被持有
	IsLocked(ctx context.Context, key string) (bool, error)

	// Extend 延长锁的 TTL
	Extend(ctx context.Context, key string, ttl time.Duration) error
}

// Option 锁选项
type Option func(*Options)

// Options 锁配置选项
type Options struct {
	// RetryInterval 重试间隔
	RetryInterval time.Duration
	// Timeout 总超时时间
	Timeout time.Duration
	// Value 锁的值（用于可重入锁）
	Value string
	// Watchdog 是否启用看门狗自动续期
	Watchdog bool
	// WatchdogInterval 看门狗续期间隔
	WatchdogInterval time.Duration
}

// WithRetryInterval 设置重试间隔
func WithRetryInterval(d time.Duration) Option {
	return func(o *Options) {
		o.RetryInterval = d
	}
}

// WithTimeout 设置总超时时间
func WithTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.Timeout = d
	}
}

// WithValue 设置锁的值
func WithValue(value string) Option {
	return func(o *Options) {
		o.Value = value
	}
}

// WithWatchdog 启用看门狗自动续期
func WithWatchdog(enable bool, interval time.Duration) Option {
	return func(o *Options) {
		o.Watchdog = enable
		o.WatchdogInterval = interval
	}
}

// DefaultOptions 默认配置
func DefaultOptions() *Options {
	return &Options{
		RetryInterval:    100 * time.Millisecond,
		Timeout:          10 * time.Second,
		WatchdogInterval: 5 * time.Second,
	}
}

// ApplyOptions 应用选项
func ApplyOptions(opts ...Option) *Options {
	o := DefaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	return o
}
