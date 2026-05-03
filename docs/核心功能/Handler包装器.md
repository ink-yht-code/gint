# 处理器包装器

`gint` 提供了一组轻量级包装器，用来减少 Handler 中重复的绑定、Session 获取和响应输出代码。当前主包装器为 `W`、`B`、`S`、`BS`。

## 包装器一览

| 包装器 | 函数签名 | 请求绑定 | Session |
| --- | --- | --- | --- |
| `W` | `func(ctx *gctx.Context) (gint.Result, error)` | 否 | 否 |
| `B` | `func(ctx *gctx.Context, req Req) (gint.Result, error)` | 是 | 否 |
| `S` | `func(ctx *gctx.Context, sess session.Session) (gint.Result, error)` | 否 | 是 |
| `BS` | `func(ctx *gctx.Context, req Req, sess session.Session) (gint.Result, error)` | 是 | 是 |

## W：无请求体

```go
r.GET("/ping", gint.W(func(ctx *gctx.Context) (gint.Result, error) {
	return gint.Success("pong", nil), nil
}))
```

## B：绑定请求体

```go
type CreateUserReq struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

r.POST("/users", gint.B(func(ctx *gctx.Context, req CreateUserReq) (gint.Result, error) {
	return gint.Success("", gin.H{
		"name":  req.Name,
		"email": req.Email,
	}), nil
}))
```

## S：携带会话

```go
r.GET("/profile", gint.S(func(ctx *gctx.Context, sess session.Session) (gint.Result, error) {
	return gint.Success("", gin.H{
		"user_id": sess.Claims().UserId,
	}), nil
}))
```

## BS：绑定请求体并携带会话

```go
type UpdateProfileReq struct {
	Nickname string `json:"nickname" binding:"required"`
}

r.PUT("/profile", gint.BS(func(ctx *gctx.Context, req UpdateProfileReq, sess session.Session) (gint.Result, error) {
	return gint.Success("更新成功", gin.H{
		"user_id":  sess.Claims().UserId,
		"nickname": req.Nickname,
	}), nil
}))
```

## 错误行为

- `error == nil`：正常写出返回的 `gint.Result`
- 普通错误：包装器按框架规则转换为错误响应
- `gint.ErrUnauthorized`：返回 `401`
- `gint.ErrNoResponse`：跳过自动响应写出

## 建议

- 需要绑定请求体时优先使用 `B` 或 `BS`
- 需要登录态时优先使用 `S` 或 `BS`
- 业务层尽量只关注服务调用与结果返回
