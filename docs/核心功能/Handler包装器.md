# Handler包装器

`wrapper` 是 gint 的核心能力，用于统一参数绑定、Session 获取和响应输出。

## 兼容性说明

- 当前版本主要包装器为：`W`、`B`、`S`、`BS`
- 文档中不再使用旧示例里的 `Q/QS/U/US` 与 `Extra` 字段写法
- 处理函数统一使用 `*gctx.Context`

## 包装器一览

| 包装器 | 处理函数签名 | 参数绑定 | Session |
|---|---|---|---|
| `W` | `func(ctx *gctx.Context) (Result, error)` | 否 | 否 |
| `B` | `func(ctx *gctx.Context, req Req) (Result, error)` | 是 | 否 |
| `S` | `func(ctx *gctx.Context, sess session.Session) (Result, error)` | 否 | 是 |
| `BS` | `func(ctx *gctx.Context, req Req, sess session.Session) (Result, error)` | 是 | 是 |

## 基本示例

### W：无参数

```go
r.GET("/ping", gint.W(func(ctx *gctx.Context) (gint.Result, error) {
    return gint.Success("pong", nil), nil
}))
```

### B：自动绑定请求参数

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

### S：自动获取 Session

```go
r.GET("/profile", gint.S(func(ctx *gctx.Context, sess session.Session) (gint.Result, error) {
    claims := sess.Claims()
    return gint.Success("", gin.H{
        "user_id": claims.UserId,
    }), nil
}))
```

### BS：绑定 + Session

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

## 错误处理约定

包装器会统一处理错误响应：

- 返回 `nil` 错误：按 `Result` 正常返回
- 返回普通错误：按框架错误码规范返回
- 返回 `gint.ErrUnauthorized`：HTTP `401`
- 返回 `gint.ErrNoResponse`：包装器不再写响应（适合你已手动写响应的场景）

## 最佳实践

- 使用 `binding` tag 做参数校验，避免手写重复校验代码
- 需要登录态的接口优先用 `S/BS`
- 业务里读取用户身份时优先用 `sess.Claims().UserId` 或 `ctx.UserId()`
- 保持 `Result` 结构简洁：`code/msg/data`
