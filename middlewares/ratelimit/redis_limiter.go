// Copyright 2025 ink-yht-code
//
// Proprietary License

package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisLimiter 基于 Redis Lua 脚本的分布式滑动窗口限流器。
// 多实例部署时共享同一个计数，适合集群级限流。
type RedisLimiter struct {
	client *redis.Client
	rate   int
	window time.Duration
	prefix string
	script *redis.Script
}

// slidingWindowLua 滑动窗口 Lua 脚本：
// KEYS[1] = 限流 key
// ARGV[1] = 当前时间戳（纳秒）
// ARGV[2] = 窗口大小（纳秒）
// ARGV[3] = 最大请求数
// ARGV[4] = key 过期时间（秒）
var slidingWindowLua = redis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local expire = tonumber(ARGV[4])

-- 移除窗口外的旧记录
redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)

-- 统计当前窗口内的请求数
local count = redis.call('ZCARD', key)

if count >= limit then
    return 0
end

-- 记录本次请求（score=时间戳，member=时间戳+随机数避免重复）
redis.call('ZADD', key, now, now .. '-' .. math.random(1, 1000000))
redis.call('EXPIRE', key, expire)
return 1
`)

// NewRedisLimiter 创建 Redis 分布式滑动窗口限流器。
// rate: 窗口内最大请求数
// window: 窗口大小
// prefix: key 前缀，用于隔离不同业务
func NewRedisLimiter(client *redis.Client, rate int, window time.Duration, prefix ...string) *RedisLimiter {
	p := "gw:rl"
	if len(prefix) > 0 && prefix[0] != "" {
		p = prefix[0]
	}
	return &RedisLimiter{
		client: client,
		rate:   rate,
		window: window,
		prefix: p,
		script: slidingWindowLua,
	}
}

// Allow 检查是否允许请求，返回 true 表示允许。
func (l *RedisLimiter) Allow(key string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	fullKey := fmt.Sprintf("%s:%s", l.prefix, key)
	now := time.Now().UnixNano()
	windowNs := l.window.Nanoseconds()
	expireSec := int(l.window.Seconds()) + 1

	result, err := l.script.Run(ctx, l.client,
		[]string{fullKey},
		now, windowNs, l.rate, expireSec,
	).Int()
	if err != nil {
		// Redis 不可用时降级为允许，避免限流器故障影响业务
		return true
	}
	return result == 1
}
