# Session 管理详解

## 概述

gint 提供了完善的 Session 管理方案，采用 JWT + Redis 混合存储设计：

- **JWT** - 存储轻量、不变的数据（如用户ID、角色），编码在 Token 中
- **Redis** - 存储完整、可变的会话数据，支持修改和删除
- **自动续期** - 访问时自动刷新过期时间
- **灵活载体** - 支持 Header 和 Cookie 两种 Token 传输方式

## 核心接口

### Session 接口

```go
type Session interface {
    // Claims 获取 JWT 中的声明数据
    Claims() *jwt.Claims
    
    // Get 从 Session 中获取数据
    Get(ctx context.Context, key string) (any, error)
    
    // Set 设置 Session 数据
    Set(ctx context.Context, key string, val any) error
    
    // Destroy 销毁 Session
    Destroy(ctx context.Context) error
}
```

### Provider 接口

```go
type Provider interface {
    // NewSession 创建新的 Session
    NewSession(ctx context.Context, userId string, jwtData map[string]string, sessData map[string]any) (Session, error)
    
    // Get 获取已存在的 Session
    Get(ctx context.Context) (Session, error)
    
    // UpdateClaims 更新 JWT Claims
    UpdateClaims(ctx context.Context, sess Session) error
}
```

### TokenCarrier 接口

```go
type TokenCarrier interface {
    // Inject 将 Token 注入到响应中
    Inject(ctx *gctx.Context, token string)
    
    // Extract 从请求中提取 Token
    Extract(ctx *gctx.Context) string
    
    // Clear 清除 Token
    Clear(ctx *gctx.Context)
}
```

## 初始化配置

### 基本配置

```go
import (
    "time"
    "github.com/redis/go-redis/v9"
    "github.com/ink-yht-code/gint/session"
    redisSession "github.com/ink-yht-code/gint/session/redis"
    "github.com/ink-yht-code/gint/session/header"
)

func main() {
    // 1. 创建 Redis 客户端
    rdb := redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    })
    
    // 2. 创建 Session Provider
    provider := redisSession.NewProvider(
        rdb,                      // Redis 客户端
        "your-jwt-secret-key",    // JWT 签名密钥（请使用强密钥）
        time.Hour * 24,           // Session 过期时间
        header.NewCarrier(),      // Token 载体
    )
    
    // 3. 设置为默认 Provider
    session.SetDefaultProvider(provider)
}
```

### 环境配置

建议使用环境变量管理配置：

```go
import "os"

func initSession() {
    jwtKey := os.Getenv("JWT_SECRET_KEY")
    if jwtKey == "" {
        jwtKey = "default-dev-key" // 开发环境默认值
    }
    
    redisAddr := os.Getenv("REDIS_ADDR")
    if redisAddr == "" {
        redisAddr = "localhost:6379"
    }
    
    rdb := redis.NewClient(&redis.Options{
        Addr: redisAddr,
    })
    
    provider := redisSession.NewProvider(
        rdb,
        jwtKey,
        time.Hour * 24,
        header.NewCarrier(),
    )
    
    session.SetDefaultProvider(provider)
}
```

## Token 载体

### Header 载体（推荐用于 API）

使用 HTTP Header 传输 Token，适合前后端分离的 API 服务。

#### 默认配置

```go
import "github.com/ink-yht-code/gint/session/header"

// 使用默认的 "Authorization" Header
carrier := header.NewCarrier()
```

客户端请求示例：
```bash
curl http://localhost:8080/profile \
  -H "Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

#### 自定义 Header 名称

```go
// 使用自定义 Header 名称
carrier := header.NewCarrierWithHeader("X-Token")
```

客户端请求示例：
```bash
curl http://localhost:8080/profile \
  -H "X-Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### Cookie 载体（推荐用于 Web）

使用 Cookie 传输 Token，适合传统的 Web 应用。

#### 基本配置

```go
import "github.com/ink-yht-code/gint/session/cookie"

carrier := cookie.NewCarrier("session_token")
```

#### 完整配置

```go
carrier := cookie.NewCarrier("session_token",
    cookie.WithDomain("example.com"),     // Cookie 域名
    cookie.WithPath("/"),                 // Cookie 路径
    cookie.WithMaxAge(86400),             // 最大存活时间（秒）
    cookie.WithSecure(true),              // 仅 HTTPS 传输
    cookie.WithHttpOnly(true),            // 禁止 JavaScript 访问
)
```

配置说明：
- **Domain** - Cookie 的作用域名，留空则为当前域名
- **Path** - Cookie 的作用路径，通常设置为 "/"
- **MaxAge** - Cookie 的最大存活时间（秒），应与 Session 过期时间一致
- **Secure** - 生产环境建议设置为 true，仅在 HTTPS 下传输
- **HttpOnly** - 建议设置为 true，防止 XSS 攻击

## 创建 Session

### 登录接口

```go
type LoginReq struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}

r.POST("/login", gint.B(func(ctx *gctx.Context, req LoginReq) (gint.Result, error) {
    // 1. 验证用户名密码
    user, err := validateUser(req.Username, req.Password)
    if err != nil {
        return gint.Result{Code: 401, Msg: "用户名或密码错误"}, nil
    }
    
    // 2. 创建 Session
    sess, err := session.NewSession(ctx, user.ID,
        // JWT 数据（轻量、不变）
        map[string]string{
            "role": user.Role,
            "dept": user.Department,
        },
        // Session 数据（完整、可变）
        map[string]any{
            "login_time": time.Now().Unix(),
            "login_ip":   ctx.ClientIP(),
            "device":     ctx.GetHeader("User-Agent"),
        },
    )
    if err != nil {
        return gint.Result{Code: 500, Msg: "创建会话失败"}, err
    }
    
    return gint.Result{
        Code: 0,
        Msg:  "登录成功",
        Data: map[string]any{
            "user_id": sess.Claims().UserId,
            "role":    sess.Claims().Data["role"],
        },
    }, nil
}))
```

### JWT 数据 vs Session 数据

**JWT 数据**（存储在 Token 中）：
- ✅ 轻量级数据（用户ID、角色、部门等）
- ✅ 不会改变的数据
- ✅ 可以在客户端解析的数据
- ❌ 不要存储敏感信息（会被 Base64 解码）

**Session 数据**（存储在 Redis 中）：
- ✅ 完整的会话数据
- ✅ 可能会改变的数据
- ✅ 敏感信息（权限列表、个人信息等）
- ✅ 大量数据

## 使用 Session

### 获取 Session 数据

```go
r.GET("/profile", gint.S(func(ctx *gctx.Context, sess session.Session) (gint.Result, error) {
    // 1. 从 JWT Claims 获取数据
    userId := sess.Claims().UserId
    role := sess.Claims().Data["role"]
    dept := sess.Claims().Data["dept"]
    
    // 2. 从 Session 获取数据
    loginTime, _ := sess.Get(ctx, "login_time")
    loginIP, _ := sess.Get(ctx, "login_ip")
    
    return gint.Result{
        Code: 0,
        Data: map[string]any{
            "user_id":    userId,
            "role":       role,
            "dept":       dept,
            "login_time": loginTime,
            "login_ip":   loginIP,
        },
    }, nil
}))
```

### 修改 Session 数据

```go
r.POST("/settings", gint.BS(func(ctx *gctx.Context, req UpdateSettingsReq, sess session.Session) (gint.Result, error) {
    // 更新 Session 数据
    if err := sess.Set(ctx, "theme", req.Theme); err != nil {
        return gint.Result{Code: 500, Msg: "更新失败"}, err
    }
    
    if err := sess.Set(ctx, "language", req.Language); err != nil {
        return gint.Result{Code: 500, Msg: "更新失败"}, err
    }
    
    return gint.Result{Code: 0, Msg: "更新成功"}, nil
}))
```

### 权限验证

```go
r.DELETE("/users/:id", gint.S(func(ctx *gctx.Context, sess session.Session) (gint.Result, error) {
    // 检查角色权限
    role := sess.Claims().Data["role"]
    if role != "admin" {
        return gint.Result{Code: 403, Msg: "无权限"}, nil
    }
    
    // 执行删除操作
    userId := ctx.Param("id").String()
    if err := deleteUser(userId); err != nil {
        return gint.Result{Code: 500, Msg: "删除失败"}, err
    }
    
    return gint.Result{Code: 0, Msg: "删除成功"}, nil
}))
```

## 销毁 Session

### 退出登录

```go
r.POST("/logout", gint.S(func(ctx *gctx.Context, sess session.Session) (gint.Result, error) {
    // 销毁 Session（清除 Redis 数据和 Token）
    if err := sess.Destroy(ctx); err != nil {
        return gint.Result{Code: 500, Msg: "退出失败"}, err
    }
    
    return gint.Result{Code: 0, Msg: "退出成功"}, nil
}))
```

### 强制下线

```go
// 管理员强制用户下线
r.POST("/admin/kick/:userId", gint.S(func(ctx *gctx.Context, sess session.Session) (gint.Result, error) {
    // 检查管理员权限
    if sess.Claims().Data["role"] != "admin" {
        return gint.Result{Code: 403, Msg: "无权限"}, nil
    }
    
    targetUserId := ctx.Param("userId").String()
    
    // 删除目标用户的所有 Session
    // 注意：这需要自己实现，可以通过 Redis 的 key 模式匹配
    if err := forceLogout(targetUserId); err != nil {
        return gint.Result{Code: 500, Msg: "操作失败"}, err
    }
    
    return gint.Result{Code: 0, Msg: "已强制下线"}, nil
}))
```

## 自动续期

Session 在每次访问时会自动刷新过期时间，无需手动处理。

```go
r.GET("/api/data", gint.S(func(ctx *gctx.Context, sess session.Session) (gint.Result, error) {
    // Session 会自动续期
    // 无需手动调用任何方法
    
    return gint.Result{Code: 0, Data: "data"}, nil
}))
```

## 错误处理

### Session 相关错误

```go
var (
    ErrSessionNotFound = errors.New("session not found")
    ErrSessionExpired  = errors.New("session expired")
    ErrInvalidToken    = errors.New("invalid token")
)
```

### 自定义错误处理

如果需要自定义 Session 错误的处理逻辑，可以在中间件中拦截：

```go
func CustomSessionMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        ctx := &gctx.Context{Context: c}
        
        // 尝试获取 Session
        sess, err := session.Get(ctx)
        if err != nil {
            if errors.Is(err, gint.ErrSessionExpired) {
                c.JSON(401, gin.H{"code": 401, "msg": "会话已过期，请重新登录"})
                c.Abort()
                return
            }
            if errors.Is(err, gint.ErrInvalidToken) {
                c.JSON(401, gin.H{"code": 401, "msg": "无效的令牌"})
                c.Abort()
                return
            }
        }
        
        c.Next()
    }
}
```

## 高级用法

### 多设备登录管理

通过在 Session 中存储设备信息，实现多设备登录管理：

```go
// 登录时记录设备信息
sess, err := session.NewSession(ctx, userId,
    map[string]string{"role": "user"},
    map[string]any{
        "device_id":   generateDeviceID(ctx),
        "device_name": ctx.GetHeader("User-Agent"),
        "login_time":  time.Now().Unix(),
    },
)

// 查询用户的所有在线设备
r.GET("/devices", gint.S(func(ctx *gctx.Context, sess session.Session) (gint.Result, error) {
    userId := sess.Claims().UserId
    devices := getAllDevices(userId) // 需要自己实现
    
    return gint.Result{Code: 0, Data: devices}, nil
}))
```

### Session 数据缓存

对于频繁访问的数据，可以缓存在 JWT 中：

```go
// 用户角色变更时，更新 JWT
r.POST("/admin/change-role", gint.BS(func(ctx *gctx.Context, req ChangeRoleReq, sess session.Session) (gint.Result, error) {
    // 更新数据库
    if err := updateUserRole(req.UserId, req.NewRole); err != nil {
        return gint.Result{Code: 500}, err
    }
    
    // 更新 JWT Claims（如果是当前用户）
    if req.UserId == sess.Claims().UserId {
        sess.Claims().Data["role"] = req.NewRole
        if err := session.UpdateClaims(ctx, sess); err != nil {
            return gint.Result{Code: 500}, err
        }
    }
    
    return gint.Result{Code: 0, Msg: "更新成功"}, nil
}))
```

## 安全建议

### 1. JWT 密钥管理

- ✅ 使用强随机密钥（至少 32 字节）
- ✅ 通过环境变量管理，不要硬编码
- ✅ 定期更换密钥
- ❌ 不要将密钥提交到代码仓库

### 2. Token 传输

- ✅ 生产环境使用 HTTPS
- ✅ Cookie 设置 `HttpOnly` 和 `Secure`
- ✅ 设置合理的过期时间
- ❌ 不要在 URL 中传输 Token

### 3. Session 数据

- ✅ 敏感数据存储在 Redis，不要放在 JWT
- ✅ 定期清理过期的 Session
- ✅ 限制 Session 数据大小
- ❌ 不要在 JWT 中存储密码等敏感信息

### 4. 权限控制

- ✅ 每次请求都验证权限
- ✅ 使用中间件统一处理权限
- ✅ 权限变更时及时更新 Session
- ❌ 不要仅依赖客户端的权限判断

## 最佳实践

1. **选择合适的载体** - API 服务用 Header，Web 应用用 Cookie
2. **合理分配数据** - 轻量数据放 JWT，完整数据放 Redis
3. **设置合理的过期时间** - 根据业务需求平衡安全性和用户体验
4. **实现权限中间件** - 统一处理权限验证逻辑
5. **监控 Session 状态** - 记录登录日志，监控异常登录

## 总结

gint 的 Session 管理方案结合了 JWT 的无状态特性和 Redis 的灵活存储，既保证了性能，又提供了完整的会话管理功能。通过合理使用，可以轻松实现用户认证、权限控制、多设备管理等功能。
