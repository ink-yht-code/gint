# Session管理

当前版本 Session 模块基于 `session.Provider` 工作，不再使用旧版 Manager API。

## 核心接口

### Provider

```go
type Provider interface {
    NewSession(ctx *gctx.Context, userId string, jwtData map[string]string, sessData map[string]any) (Session, error)
    Get(ctx *gctx.Context) (Session, error)
    Destroy(ctx *gctx.Context) error
    RenewToken(ctx *gctx.Context) error
}
```

### Session

```go
type Session interface {
    Set(ctx context.Context, key string, val any) error
    Get(ctx context.Context, key string) (any, error)
    Del(ctx context.Context, key string) error
    Destroy(ctx context.Context) error
    Claims() *jwt.Claims
    UserContext(ctx context.Context) (*gctx.UserContext, error)
    Refresh(ctx context.Context) error
}
```

## 提供者实现

- `session/memory`：内存实现，适合开发与测试
- `session/redis`：Redis 实现，适合生产环境

两者都使用双 token 机制（access + refresh），并通过 `session.TokenCarrier` 进行 token 注入与提取。

## TokenCarrier

- `session/cookie.NewCarrier(...)`
- `session/header.NewCarrier()` 或 `session/header.NewCarrierWithHeader("X-Token")`

## 初始化方式

### 全局 provider

```go
import (
    "time"

    "github.com/ink-yht-code/gint/session"
    "github.com/ink-yht-code/gint/session/cookie"
    "github.com/ink-yht-code/gint/session/memory"
)

provider := memory.NewProvider(
    "jwt-secret",
    2*time.Hour,        // access token 过期
    7*24*time.Hour,     // refresh token / session 过期
    cookie.NewCarrier("gint_token"),
)
session.SetDefaultProvider(provider)
```

### 按请求覆盖 provider

```go
func middleware(provider session.Provider) gin.HandlerFunc {
    return func(c *gin.Context) {
        ctx := &gctx.Context{Context: c}
        session.SetProvider(ctx, provider)
        c.Next()
    }
}
```

## 常见操作

### 创建会话（登录）

```go
ctx := &gctx.Context{Context: c}
sess, err := session.NewSession(
    ctx,
    "user-1",
    map[string]string{"role": "admin"},        // jwtData
    map[string]any{"username": "tom"},         // sessData
)
_ = sess
_ = err
```

### 获取会话

```go
ctx := &gctx.Context{Context: c}
sess, err := session.Get(ctx)
if err != nil {
    c.AbortWithStatus(401)
    return
}
uc, _ := sess.UserContext(c.Request.Context())
_ = uc
```

### 销毁会话（登出）

```go
ctx := &gctx.Context{Context: c}
sess, err := session.Get(ctx)
if err == nil {
    _ = sess.Destroy(c.Request.Context())
}
```

> 若需要同时清理载体中的 token，推荐直接调用当前 provider 的 `Destroy(ctx)`（例如在统一登出逻辑中）。

## 使用建议

- 生产环境优先用 Redis Provider
- 若使用 Cookie 载体，务必评估 `HttpOnly/Secure/SameSite`
- 刷新 token 依赖 `X-Refresh-Token` 头，网关层需允许透传
