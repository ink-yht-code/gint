// Copyright 2025 ink-yht-code
//
// Proprietary License

package redis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ink-yht-code/gint/lock"
)

var (
	ErrLockNotHeld    = errors.New("lock not held by current holder")
	ErrLockConflict   = errors.New("lock conflict")
	ErrInvalidTTL     = errors.New("invalid ttl")
	ErrWatchdogFailed = errors.New("watchdog extend failed")
)

// RedisLocker Redis 分布式锁实现
type RedisLocker struct {
	client    redis.Cmdable
	keyPrefix string

	// 看门狗管理
	watchdogMu  sync.RWMutex
	watchdogMap map[string]context.CancelFunc
}

// Option Redis 锁选项
type Option func(*RedisLocker)

// WithKeyPrefix 设置 key 前缀
func WithKeyPrefix(prefix string) Option {
	return func(l *RedisLocker) {
		l.keyPrefix = prefix
	}
}

// New 创建 Redis 分布式锁
func New(client redis.Cmdable, opts ...Option) lock.Locker {
	l := &RedisLocker{
		client:      client,
		keyPrefix:   "lock:",
		watchdogMap: make(map[string]context.CancelFunc),
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// generateValue 生成唯一锁值
func (l *RedisLocker) generateValue() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// fullKey 获取完整 key
func (l *RedisLocker) fullKey(key string) string {
	return l.keyPrefix + key
}

// TryLock 尝试获取锁
func (l *RedisLocker) TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		return false, ErrInvalidTTL
	}

	fullKey := l.fullKey(key)
	value := l.generateValue()

	// SET NX EX 原子操作
	ok, err := l.client.SetNX(ctx, fullKey, value, ttl).Result()
	if err != nil {
		return false, err
	}

	if ok {
		// 存储锁值用于后续解锁
		l.storeLockValue(key, value)
	}
	return ok, nil
}

// Lock 获取锁，阻塞直到成功或超时
func (l *RedisLocker) Lock(ctx context.Context, key string, ttl time.Duration, opts ...lock.Option) error {
	options := lock.ApplyOptions(opts...)

	// 如果设置了总超时，创建带超时的 context
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	// 使用传入的 value 或生成新值
	value := options.Value
	if value == "" {
		value = l.generateValue()
	}

	fullKey := l.fullKey(key)
	retryInterval := options.RetryInterval
	if retryInterval <= 0 {
		retryInterval = 100 * time.Millisecond
	}

	ticker := time.NewTicker(retryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			ok, err := l.client.SetNX(ctx, fullKey, value, ttl).Result()
			if err != nil {
				return err
			}
			if ok {
				l.storeLockValue(key, value)

				// 启动看门狗
				if options.Watchdog {
					l.startWatchdog(ctx, key, value, ttl, options.WatchdogInterval)
				}
				return nil
			}
		}
	}
}

// Unlock 释放锁
func (l *RedisLocker) Unlock(ctx context.Context, key string) error {
	// 停止看门狗
	l.stopWatchdog(key)

	fullKey := l.fullKey(key)
	value := l.getLockValue(key)

	if value == "" {
		return ErrLockNotHeld
	}

	// Lua 脚本保证原子性：只有持有锁的客户端才能解锁
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`

	result, err := l.client.Eval(ctx, script, []string{fullKey}, value).Int64()
	if err != nil {
		return err
	}

	// 清理本地存储的锁值
	l.deleteLockValue(key)

	if result == 0 {
		return ErrLockNotHeld
	}
	return nil
}

// TryLockWithRetry 带重试的尝试获取锁
func (l *RedisLocker) TryLockWithRetry(ctx context.Context, key string, ttl time.Duration, retryInterval time.Duration, maxRetries int) (bool, error) {
	for i := 0; i < maxRetries; i++ {
		ok, err := l.TryLock(ctx, key, ttl)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}

		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(retryInterval):
		}
	}
	return false, nil
}

// IsLocked 检查锁是否被持有
func (l *RedisLocker) IsLocked(ctx context.Context, key string) (bool, error) {
	fullKey := l.fullKey(key)
	n, err := l.client.Exists(ctx, fullKey).Result()
	return n > 0, err
}

// Extend 延长锁的 TTL
func (l *RedisLocker) Extend(ctx context.Context, key string, ttl time.Duration) error {
	fullKey := l.fullKey(key)
	value := l.getLockValue(key)

	if value == "" {
		return ErrLockNotHeld
	}

	// Lua 脚本：只有持有锁的客户端才能续期
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("PEXPIRE", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	result, err := l.client.Eval(ctx, script, []string{fullKey}, value, ttl.Milliseconds()).Int64()
	if err != nil {
		return err
	}

	if result == 0 {
		return ErrLockNotHeld
	}
	return nil
}

// startWatchdog 启动看门狗
func (l *RedisLocker) startWatchdog(ctx context.Context, key, value string, ttl, interval time.Duration) {
	l.watchdogMu.Lock()
	defer l.watchdogMu.Unlock()

	// 如果已经存在，先停止
	if cancel, exists := l.watchdogMap[key]; exists {
		cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	l.watchdogMap[key] = cancel

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := l.Extend(context.Background(), key, ttl); err != nil {
					// 续期失败，停止看门狗
					l.stopWatchdog(key)
					return
				}
			}
		}
	}()
}

// stopWatchdog 停止看门狗
func (l *RedisLocker) stopWatchdog(key string) {
	l.watchdogMu.Lock()
	defer l.watchdogMu.Unlock()

	if cancel, exists := l.watchdogMap[key]; exists {
		cancel()
		delete(l.watchdogMap, key)
	}
}

// 本地锁值存储（简化实现，生产环境应使用更可靠的方式）
var (
	lockValueMu sync.RWMutex
	lockValues  = make(map[string]string)
)

func (l *RedisLocker) storeLockValue(key, value string) {
	lockValueMu.Lock()
	lockValues[l.fullKey(key)] = value
	lockValueMu.Unlock()
}

func (l *RedisLocker) getLockValue(key string) string {
	lockValueMu.RLock()
	v := lockValues[l.fullKey(key)]
	lockValueMu.RUnlock()
	return v
}

func (l *RedisLocker) deleteLockValue(key string) {
	lockValueMu.Lock()
	delete(lockValues, l.fullKey(key))
	lockValueMu.Unlock()
}
