# 限流中间件使用指南

## 概述

提供了两种限流算法实现：
- **SimpleLimiter**: 固定窗口算法（性能高，内存占用小）
- **SlidingWindowLimiter**: 滑动窗口算法（更精确，避免临界突刺）

两种实现都是**并发安全**的。

## 使用示例

### 1. 固定窗口限流（推荐用于一般场景）

```go
package main

import (
    "time"
    "github.com/gin-gonic/gin"
    "your-project/middlewares/ratelimit"
)

func main() {
    r := gin.Default()

    // 创建限流器：每分钟最多 100 个请求
    limiter := ratelimit.NewSimpleLimiter(100, time.Minute)

    // 默认使用 IP 作为限流键
    r.Use(ratelimit.NewBuilder(limiter).Build())

    // 或者使用用户 ID 作为限流键
    r.Use(ratelimit.NewBuilder(limiter).WithUserIDKey().Build())

    // 或者使用内置的 IPKeyFunc
    r.Use(ratelimit.NewBuilder(limiter).
        WithKeyFunc(ratelimit.IPKeyFunc).
        Build())

    // 或者使用路径 + IP 作为限流键
    r.Use(ratelimit.NewBuilder(limiter).
        WithKeyFunc(ratelimit.PathKeyFunc).
        Build())

    // 或者自定义限流键
    r.Use(ratelimit.NewBuilder(limiter).WithKeyFunc(func(c *gin.Context) string {
        return "api:" + c.Request.URL.Path + ":user:" + c.GetString("user_id")
    }).Build())

    r.Run(":8080")
}
```

### 2. 滑动窗口限流（推荐用于严格限流场景）

```go
// 创建滑动窗口限流器：每 10 秒最多 50 个请求
limiter := ratelimit.NewSlidingWindowLimiter(50, 10*time.Second)

r.Use(ratelimit.NewBuilder(limiter).Build())
```

### 3. 不同路由使用不同限流策略

```go
r := gin.Default()

// API 接口：每分钟 100 次
apiLimiter := ratelimit.NewSimpleLimiter(100, time.Minute)
apiGroup := r.Group("/api")
apiGroup.Use(ratelimit.NewBuilder(apiLimiter).Build())
{
    apiGroup.GET("/users", getUsers)
}

// 登录接口：每分钟 5 次（防暴力破解）
loginLimiter := ratelimit.NewSlidingWindowLimiter(5, time.Minute)
r.POST("/login", ratelimit.NewBuilder(loginLimiter).Build(), login)

// 敏感操作：每小时 10 次
sensitiveLimit := ratelimit.NewSlidingWindowLimiter(10, time.Hour)
r.POST("/reset-password", 
    ratelimit.NewBuilder(sensitiveLimit).WithUserIDKey().Build(), 
    resetPassword)
```

## 算法对比

| 特性 | SimpleLimiter | SlidingWindowLimiter |
|------|---------------|---------------------|
| 算法 | 固定窗口 | 滑动窗口 |
| 性能 | 高 | 中等 |
| 内存占用 | 低（每个 key 只存储计数） | 较高（存储时间戳列表） |
| 精确度 | 一般（窗口边界可能突刺） | 高（无突刺问题） |
| 适用场景 | 一般限流 | 严格限流、防刷 |

## 并发安全说明

- 使用 `sync.Map` 存储计数器，支持高并发读写
- 每个计数器内部使用 `sync.Mutex` 保护状态
- 自动清理过期数据，防止内存泄漏

## 注意事项

1. **内存占用**: 滑动窗口算法会存储每个请求的时间戳，高流量下内存占用较大
2. **分布式场景**: 当前实现是单机内存限流，分布式场景需要使用 Redis 等
3. **清理策略**: 自动清理过期数据，清理间隔为窗口大小的 2 倍
