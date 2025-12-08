# Handler 包装器详解

## 概述

gint 提供了四种 Handler 包装器（W、B、S、BS），用于简化业务逻辑的编写。这些包装器自动处理参数绑定、Session 验证、错误处理和响应格式化等重复工作。

## 核心概念

### 统一的响应结构

所有包装器都返回统一的 `Result` 结构：

```go
type Result struct {
    Code int    `json:"code"` // 业务状态码，0 表示成功
    Msg  string `json:"msg"`  // 响应消息
    Data any    `json:"data"` // 响应数据
}
```

### 错误处理机制

包装器会自动处理不同类型的错误：

- **特殊错误** - 如 `ErrNoResponse`、`ErrUnauthorized`，有特殊处理逻辑
- **普通错误** - 自动记录日志并返回 500
- **业务错误** - 通过 `Result.Code` 返回给客户端

## W - 基础包装器

### 函数签名

```go
func W(fn func(ctx *gctx.Context) (Result, error)) gin.HandlerFunc
```

### 适用场景

- 不需要参数绑定
- 不需要 Session 验证
- 简单的查询接口

### 使用示例

```go
// 健康检查接口
r.GET("/ping", gint.W(func(ctx *gctx.Context) (gint.Result, error) {
    return gint.Result{
        Code: 0,
        Msg:  "pong",
        Data: time.Now(),
    }, nil
}))

// 获取服务器时间
r.GET("/time", gint.W(func(ctx *gctx.Context) (gint.Result, error) {
    return gint.Result{
        Code: 0,
        Data: map[string]any{
            "timestamp": time.Now().Unix(),
            "timezone":  "UTC+8",
        },
    }, nil
}))

// 手动处理响应
r.GET("/download", gint.W(func(ctx *gctx.Context) (gint.Result, error) {
    ctx.File("./file.pdf")
    return gint.Result{}, gint.ErrNoResponse // 不需要返回 JSON
}))
```

## B - 带参数绑定的包装器

### 函数签名

```go
func B[Req any](fn func(ctx *gctx.Context, req Req) (Result, error)) gin.HandlerFunc
```

### 适用场景

- 需要绑定请求参数（JSON、Form、Query 等）
- 需要参数验证
- 不需要 Session 验证的业务接口

### 使用示例

#### 基本用法

```go
type LoginReq struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}

r.POST("/login", gint.B(func(ctx *gctx.Context, req LoginReq) (gint.Result, error) {
    // req 已经自动绑定和验证
    if req.Username != "admin" || req.Password != "123456" {
        return gint.Result{Code: 401, Msg: "用户名或密码错误"}, nil
    }
    
    return gint.Result{Code: 0, Msg: "登录成功"}, nil
}))
```

#### 参数验证

使用 Gin 的 binding 标签进行参数验证：

```go
type CreateUserReq struct {
    Username string `json:"username" binding:"required,min=3,max=20"`
    Email    string `json:"email" binding:"required,email"`
    Age      int    `json:"age" binding:"required,gte=18,lte=100"`
    Phone    string `json:"phone" binding:"required,len=11"`
}

r.POST("/users", gint.B(func(ctx *gctx.Context, req CreateUserReq) (gint.Result, error) {
    // 参数已经通过验证
    // 创建用户...
    return gint.Result{Code: 0, Msg: "创建成功"}, nil
}))
```

#### 分页查询

```go
r.GET("/users", gint.B(func(ctx *gctx.Context, req gint.PageRequest) (gint.Result, error) {
    req.Validate() // 验证并设置默认值（page=1, size=10）
    
    // 从数据库查询
    users := getUsersFromDB(req.Offset(), req.Size)
    total := getTotalCount()
    
    return gint.Result{
        Code: 0,
        Data: gint.PageData[User]{
            List:  users,
            Total: total,
            Page:  req.Page,
            Size:  req.Size,
        },
    }, nil
}))
```

#### Query 参数绑定

```go
type SearchReq struct {
    Keyword string `form:"keyword" binding:"required"`
    Page    int    `form:"page" binding:"omitempty,gte=1"`
    Size    int    `form:"size" binding:"omitempty,gte=1,lte=100"`
}

r.GET("/search", gint.B(func(ctx *gctx.Context, req SearchReq) (gint.Result, error) {
    // 从 Query 参数绑定
    results := search(req.Keyword, req.Page, req.Size)
    return gint.Result{Code: 0, Data: results}, nil
}))
```

## S - 带 Session 的包装器

### 函数签名

```go
func S(fn func(ctx *gctx.Context, sess session.Session) (Result, error)) gin.HandlerFunc
```

### 适用场景

- 需要验证用户登录态
- 不需要参数绑定
- 需要访问 Session 数据

### 使用示例

#### 获取用户信息

```go
r.GET("/profile", gint.S(func(ctx *gctx.Context, sess session.Session) (gint.Result, error) {
    // Session 已自动验证
    userId := sess.Claims().UserId
    role := sess.Claims().Data["role"]
    
    // 从 Session 中获取数据
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

#### 退出登录

```go
r.POST("/logout", gint.S(func(ctx *gctx.Context, sess session.Session) (gint.Result, error) {
    if err := sess.Destroy(ctx); err != nil {
        return gint.Result{Code: 500, Msg: "退出失败"}, err
    }
    
    return gint.Result{Code: 0, Msg: "退出成功"}, nil
}))
```

#### 刷新 Session

```go
r.POST("/refresh", gint.S(func(ctx *gctx.Context, sess session.Session) (gint.Result, error) {
    // Session 会自动续期
    return gint.Result{Code: 0, Msg: "刷新成功"}, nil
}))
```

## BS - 参数绑定 + Session 的包装器

### 函数签名

```go
func BS[Req any](fn func(ctx *gctx.Context, req Req, sess session.Session) (Result, error)) gin.HandlerFunc
```

### 适用场景

- 需要验证用户登录态
- 需要参数绑定
- 最常用的业务接口场景

### 使用示例

#### 更新用户资料

```go
type UpdateProfileReq struct {
    Nickname string `json:"nickname" binding:"required,min=2,max=20"`
    Avatar   string `json:"avatar" binding:"required,url"`
    Bio      string `json:"bio" binding:"omitempty,max=200"`
}

r.POST("/profile", gint.BS(func(ctx *gctx.Context, req UpdateProfileReq, sess session.Session) (gint.Result, error) {
    userId := sess.Claims().UserId
    
    // 更新用户信息
    if err := updateUserProfile(userId, req); err != nil {
        return gint.Result{Code: 500, Msg: "更新失败"}, err
    }
    
    return gint.Result{Code: 0, Msg: "更新成功"}, nil
}))
```

#### 创建文章

```go
type CreateArticleReq struct {
    Title   string `json:"title" binding:"required,min=5,max=100"`
    Content string `json:"content" binding:"required,min=10"`
    Tags    []string `json:"tags" binding:"omitempty,max=5"`
}

r.POST("/articles", gint.BS(func(ctx *gctx.Context, req CreateArticleReq, sess session.Session) (gint.Result, error) {
    userId := sess.Claims().UserId
    
    // 创建文章
    articleId, err := createArticle(userId, req)
    if err != nil {
        return gint.Result{Code: 500, Msg: "创建失败"}, err
    }
    
    return gint.Result{
        Code: 0,
        Msg:  "创建成功",
        Data: map[string]any{"article_id": articleId},
    }, nil
}))
```

#### 删除资源

```go
type DeleteReq struct {
    ID string `json:"id" binding:"required"`
}

r.DELETE("/articles", gint.BS(func(ctx *gctx.Context, req DeleteReq, sess session.Session) (gint.Result, error) {
    userId := sess.Claims().UserId
    
    // 验证权限
    if !canDelete(userId, req.ID) {
        return gint.Result{Code: 403, Msg: "无权限删除"}, nil
    }
    
    // 删除文章
    if err := deleteArticle(req.ID); err != nil {
        return gint.Result{Code: 500, Msg: "删除失败"}, err
    }
    
    return gint.Result{Code: 0, Msg: "删除成功"}, nil
}))
```

## 错误处理

### 特殊错误

#### ErrNoResponse

当你已经手动处理了响应（如文件下载、SSE 等），不需要返回 JSON 时使用：

```go
r.GET("/download", gint.W(func(ctx *gctx.Context) (gint.Result, error) {
    ctx.File("./file.pdf")
    return gint.Result{}, gint.ErrNoResponse
}))
```

#### ErrUnauthorized

返回 401 状态码：

```go
r.GET("/admin", gint.S(func(ctx *gctx.Context, sess session.Session) (gint.Result, error) {
    if sess.Claims().Data["role"] != "admin" {
        return gint.Result{}, gint.ErrUnauthorized
    }
    // 管理员操作...
    return gint.Result{Code: 0}, nil
}))
```

### 业务错误

通过 `Result.Code` 返回业务错误码：

```go
const (
    CodeSuccess      = 0
    CodeParamError   = 400
    CodeUnauthorized = 401
    CodeNotFound     = 404
    CodeServerError  = 500
)

r.GET("/user/:id", gint.W(func(ctx *gctx.Context) (gint.Result, error) {
    id := ctx.Param("id").String()
    user, err := getUserById(id)
    if err != nil {
        if errors.Is(err, ErrUserNotFound) {
            return gint.Result{Code: CodeNotFound, Msg: "用户不存在"}, nil
        }
        return gint.Result{Code: CodeServerError}, err
    }
    
    return gint.Result{Code: CodeSuccess, Data: user}, nil
}))
```

### 系统错误

返回非 nil 的 error，会自动记录日志并返回 500：

```go
r.POST("/users", gint.B(func(ctx *gctx.Context, req CreateUserReq) (gint.Result, error) {
    if err := db.Create(&user).Error; err != nil {
        // 自动记录日志并返回 500
        return gint.Result{Code: 500, Msg: "创建失败"}, err
    }
    
    return gint.Result{Code: 0, Msg: "创建成功"}, nil
}))
```

## 最佳实践

### 1. 选择合适的包装器

- 简单查询 → 使用 **W**
- 需要参数 → 使用 **B**
- 需要登录 → 使用 **S**
- 需要参数 + 登录 → 使用 **BS**

### 2. 错误处理原则

- **业务错误**（用户不存在、权限不足等）→ 返回 `Result.Code` + `nil error`
- **系统错误**（数据库错误、网络错误等）→ 返回 `error`

### 3. 参数验证

充分利用 Gin 的 binding 标签进行参数验证：

```go
type Req struct {
    Field string `json:"field" binding:"required,min=3,max=20,alphanum"`
}
```

常用验证标签：
- `required` - 必填
- `omitempty` - 可选
- `min=N` / `max=N` - 长度限制
- `gte=N` / `lte=N` - 数值范围
- `email` - 邮箱格式
- `url` - URL 格式
- `alphanum` - 字母数字

### 4. 响应格式统一

始终使用 `Result` 结构返回响应，保持 API 格式统一：

```go
// ✅ 推荐
return gint.Result{Code: 0, Msg: "成功", Data: data}, nil

// ❌ 不推荐
ctx.JSON(200, map[string]any{"status": "ok"})
return gint.Result{}, gint.ErrNoResponse
```

### 5. 分离业务逻辑

Handler 应该保持简洁，复杂的业务逻辑应该放在 Service 层：

```go
// ✅ 推荐
r.POST("/users", gint.B(func(ctx *gctx.Context, req CreateUserReq) (gint.Result, error) {
    user, err := userService.Create(req)
    if err != nil {
        return gint.Result{Code: 500, Msg: "创建失败"}, err
    }
    return gint.Result{Code: 0, Data: user}, nil
}))

// ❌ 不推荐：在 Handler 中写大量业务逻辑
r.POST("/users", gint.B(func(ctx *gctx.Context, req CreateUserReq) (gint.Result, error) {
    // 100 行业务逻辑...
}))
```

## 总结

gint 的 Handler 包装器通过简洁的 API 设计，让你专注于业务逻辑，而不是重复的样板代码。选择合适的包装器，遵循最佳实践，可以让你的代码更加清晰和易于维护。
