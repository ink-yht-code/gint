# Memory Session 使用指南

## 概述

Memory Session 是基于内存的 Session 实现，适合**开发测试环境**使用。数据存储在服务器内存中，重启后会丢失。

> ⚠️ **重要提示**：Memory Session 仅用于开发测试，生产环境请使用 [Redis Session](./session.md)。

## 特点

### 优点

- ✅ **零依赖** - 不需要 Redis 等外部服务
- ✅ **快速启动** - 适合快速开发和测试
- ✅ **简单直接** - 无需配置外部服务

### 缺点

- ❌ **数据易失** - 服务重启后数据丢失
- ❌ **不支持分布式** - 多实例部署时 Session 不共享
- ❌ **内存占用** - 大量 Session 会占用较多内存
- ❌ **不适合生产** - 无法保证数据持久性

## 基本用法

### 初始化

```go
import (
    "time"
    "github.com/ink-yht-code/gint/session"
    "github.com/ink-yht-code/gint/session/memory"
    "github.com/ink-yht-code/gint/session/header"
)

func main() {
    r := gin.Default()
    
    // 创建 Memory Session Provider
    provider := memory.NewProvider(
        "your-jwt-secret-key",  // JWT 签名密钥
        time.Hour * 24,         // Session 过期时间
        header.NewCarrier(),    // Token 载体
    )
    
    // 设置为默认 Provider
    session.SetDefaultProvider(provider)
    
    // 注册路由...
    r.Run(":8080")
}
```

### 创建 Session

```go
r.POST("/login", gint.B(func(ctx *gctx.Context, req LoginReq) (gint.Result, error) {
    // 验证用户名密码...
    
    // 创建 Session
    sess, err := session.NewSession(ctx, "user_123",
        map[string]string{"role": "admin"},
        map[string]any{
            "login_time": time.Now().Unix(),
            "login_ip":   ctx.ClientIP(),
        },
    )
    if err != nil {
        return gint.Result{Code: 500, Msg: "创建会话失败"}, err
    }
    
    return gint.Result{Code: 0, Msg: "登录成功"}, nil
}))
```

### 使用 Session

```go
r.GET("/profile", gint.S(func(ctx *gctx.Context, sess session.Session) (gint.Result, error) {
    userId := sess.Claims().UserId
    role := sess.Claims().Data["role"]
    
    loginTime, _ := sess.Get(ctx, "login_time")
    
    return gint.Result{
        Code: 0,
        Data: map[string]any{
            "user_id":    userId,
            "role":       role,
            "login_time": loginTime,
        },
    }, nil
}))
```

## 完整示例

```go
package main

import (
    "time"
    
    "github.com/gin-gonic/gin"
    "github.com/ink-yht-code/gint"
    "github.com/ink-yht-code/gint/gctx"
    "github.com/ink-yht-code/gint/session"
    "github.com/ink-yht-code/gint/session/memory"
    "github.com/ink-yht-code/gint/session/header"
)

type LoginReq struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}

func main() {
    r := gin.Default()
    
    // 初始化 Memory Session
    provider := memory.NewProvider(
        "dev-secret-key",
        time.Hour * 2,
        header.NewCarrier(),
    )
    session.SetDefaultProvider(provider)
    
    // 登录
    r.POST("/login", gint.B(func(ctx *gctx.Context, req LoginReq) (gint.Result, error) {
        if req.Username != "admin" || req.Password != "123456" {
            return gint.Result{Code: 401, Msg: "用户名或密码错误"}, nil
        }
        
        sess, err := session.NewSession(ctx, "user_123",
            map[string]string{"role": "admin"},
            map[string]any{"login_time": time.Now().Unix()},
        )
        if err != nil {
            return gint.Result{Code: 500}, err
        }
        
        return gint.Result{Code: 0, Msg: "登录成功"}, nil
    }))
    
    // 获取用户信息
    r.GET("/profile", gint.S(func(ctx *gctx.Context, sess session.Session) (gint.Result, error) {
        return gint.Result{
            Code: 0,
            Data: map[string]any{
                "user_id": sess.Claims().UserId,
                "role":    sess.Claims().Data["role"],
            },
        }, nil
    }))
    
    // 退出登录
    r.POST("/logout", gint.S(func(ctx *gctx.Context, sess session.Session) (gint.Result, error) {
        if err := sess.Destroy(ctx); err != nil {
            return gint.Result{Code: 500}, err
        }
        return gint.Result{Code: 0, Msg: "退出成功"}, nil
    }))
    
    r.Run(":8080")
}
```

## 自动清理机制

Memory Session 会自动清理过期的 Session：

- **清理频率**：每 5 分钟
- **清理对象**：已过期的 Session
- **后台运行**：在独立的 goroutine 中运行

这个机制可以防止内存泄漏，但也意味着：
- 过期的 Session 不会立即被删除
- 最多可能有 5 分钟的延迟

## 使用 Cookie 载体

```go
import "github.com/ink-yht-code/gint/session/cookie"

provider := memory.NewProvider(
    "dev-secret-key",
    time.Hour * 2,
    cookie.NewCarrier("session_token",
        cookie.WithHttpOnly(true),
        cookie.WithPath("/"),
    ),
)
```

## 迁移到 Redis

当你准备部署到生产环境时，只需要替换 Provider：

```go
// 开发环境
import "github.com/ink-yht-code/gint/session/memory"
provider := memory.NewProvider(jwtKey, expiration, carrier)

// 生产环境
import (
    "github.com/redis/go-redis/v9"
    redisSession "github.com/ink-yht-code/gint/session/redis"
)

rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
provider := redisSession.NewProvider(rdb, jwtKey, expiration, carrier)
```

其他代码无需修改！

## 注意事项

### 1. 仅用于开发测试

```go
// ✅ 开发环境
if os.Getenv("ENV") == "development" {
    provider := memory.NewProvider(jwtKey, expiration, carrier)
}

// ✅ 生产环境
if os.Getenv("ENV") == "production" {
    rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
    provider := redisSession.NewProvider(rdb, jwtKey, expiration, carrier)
}
```

### 2. 内存占用

大量 Session 会占用较多内存：

```go
// 假设每个 Session 占用 1KB
// 10000 个 Session = 10MB
// 100000 个 Session = 100MB

// 建议：开发环境限制并发用户数
```

### 3. 不支持分布式

如果你有多个服务实例，Session 不会共享：

```
实例 A：用户登录，Session 存储在实例 A 的内存
实例 B：用户请求被路由到实例 B，找不到 Session
```

解决方案：使用 Redis Session。

### 4. 服务重启数据丢失

```go
// 服务重启前
用户 A 已登录，Session 在内存中

// 服务重启后
所有 Session 丢失，用户需要重新登录
```

## 最佳实践

### 1. 环境区分

```go
func initSession() session.Provider {
    jwtKey := os.Getenv("JWT_KEY")
    expiration := time.Hour * 24
    carrier := header.NewCarrier()
    
    if os.Getenv("ENV") == "production" {
        // 生产环境使用 Redis
        rdb := redis.NewClient(&redis.Options{
            Addr: os.Getenv("REDIS_ADDR"),
        })
        return redisSession.NewProvider(rdb, jwtKey, expiration, carrier)
    }
    
    // 开发环境使用 Memory
    return memory.NewProvider(jwtKey, expiration, carrier)
}
```

### 2. 设置合理的过期时间

```go
// 开发环境可以设置较短的过期时间
provider := memory.NewProvider(
    jwtKey,
    time.Minute * 30,  // 30 分钟
    carrier,
)
```

### 3. 监控内存使用

```go
import "runtime"

func printMemStats() {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    fmt.Printf("Alloc = %v MB", m.Alloc / 1024 / 1024)
}
```

## 常见问题

### Q: Memory Session 适合哪些场景？

A: 
- ✅ 本地开发
- ✅ 单元测试
- ✅ 快速原型验证
- ❌ 生产环境
- ❌ 分布式部署

### Q: 如何查看当前有多少 Session？

A: Memory Session 没有提供查询接口，这是设计上的简化。如果需要监控，建议使用 Redis Session。

### Q: 可以手动清理 Session 吗？

A: Session 会自动清理，无需手动操作。如果需要立即删除某个 Session，调用 `sess.Destroy(ctx)` 即可。

### Q: 性能如何？

A: Memory Session 性能很好，因为数据在内存中。但要注意：
- 大量 Session 会占用内存
- 没有持久化保证
- 不适合生产环境

## 总结

Memory Session 是一个轻量级的 Session 实现，非常适合开发测试环境。它的优点是简单、快速、零依赖，缺点是数据易失、不支持分布式。

**记住**：Memory Session 仅用于开发测试，生产环境请使用 Redis Session！
