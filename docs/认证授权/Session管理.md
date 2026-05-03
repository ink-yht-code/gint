# 会话管理

当前版本的会话模块以 `session.Provider` 为核心，不再以旧的 manager-first 模式为主。

## 核心接口

### 提供者接口

```go
type Provider interface {
	NewSession(ctx *gctx.Context, userId string, jwtData map[string]string, sessData map[string]any) (Session, error)
	Get(ctx *gctx.Context) (Session, error)
	Destroy(ctx *gctx.Context) error
	RenewToken(ctx *gctx.Context) error
}
```

### 会话接口

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

## 提供者实现方式

- `session/memory`：适合本地开发和测试
- `session/redis`：适合生产环境

## 令牌载体

- `session/cookie.NewCarrier(...)`
- `session/header.NewCarrier()`
- `session/header.NewCarrierWithHeader("X-Token")`

## 设置全局会话提供者

```go
provider := memory.NewProvider(
	"jwt-secret",
	2*time.Hour,
	7*24*time.Hour,
	cookie.NewCarrier("gint_token"),
)
session.SetDefaultProvider(provider)
```

## 创建会话

```go
ctx := &gctx.Context{Context: c}
sess, err := session.NewSession(
	ctx,
	"user-1",
	map[string]string{"role": "admin"},
	map[string]any{"username": "tom"},
)
_ = sess
_ = err
```

## 获取会话

```go
ctx := &gctx.Context{Context: c}
sess, err := session.Get(ctx)
if err != nil {
	c.AbortWithStatus(401)
	return
}
_ = sess
```

## 销毁会话

```go
ctx := &gctx.Context{Context: c}
if err := session.DefaultProvider().Destroy(ctx); err != nil {
	c.AbortWithStatus(401)
	return
}
```

## 建议

- 生产环境优先使用 Redis 会话提供者
- 如果使用 Cookie，注意 `HttpOnly`、`Secure`、`SameSite`
- Token 刷新策略要和前端载体保持一致
