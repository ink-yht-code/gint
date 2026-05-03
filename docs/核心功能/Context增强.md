# 上下文增强

gint 提供了增强的上下文能力，基于 Gin Context 扩展，支持统一的用户身份信息管理。

## 概述

`gctx.Context` 是对 `gin.Context` 的增强封装，提供：
- 用户上下文
- 链路追踪 ID
- 便捷方法

## 基本使用

### 获取增强上下文

```go
import "github.com/ink-yht-code/gint/gctx"

func Handler(c *gin.Context) {
    // 转换为增强上下文
    ctx := gctx.FromContext(c)
    
    // 或在包装器中直接使用
    // gint.W 会自动注入 *gctx.Context
}
```

### 在处理器包装器中使用

```go
r.GET("/profile", gint.W(func(c *gin.Context, req Req) (gint.Result, error) {
    // c 已经是 *gctx.Context 类型
    ctx := c.(*gctx.Context)
    
    userId := ctx.UserId()
    traceId := ctx.TraceId()
    
    return gint.Result{Data: data}, nil
}))
```

## 用户上下文

### 结构定义

```go
type UserContext struct {
    UserId   string // 用户ID
    Role     string // 角色
    TenantId string // 租户ID
    Username string // 用户名
    Email    string // 邮箱
}
```

### 设置用户上下文

```go
// 手动设置
ctx := gctx.FromContext(c)
ctx.SetUserContext(&gctx.UserContext{
    UserId:   "123",
    Role:     "admin",
    TenantId: "tenant-001",
    Username: "john",
    Email:    "john@example.com",
})
```

### 获取用户上下文

```go
// 获取完整用户上下文
userContext := ctx.UserContext()
if userContext != nil {
    userId := userContext.UserId
    role := userContext.Role
    tenantId := userContext.TenantId
}

// 便捷方法：仅获取用户 ID
userId := ctx.UserId()
```

### 在中间件中注入

配合 trace 中间件自动注入：

```go
import "github.com/ink-yht-code/gint/middlewares/trace"

// trace中间件会自动从请求头提取UserContext
r.Use(trace.Middleware())

r.GET("/api/profile", gint.S(func(c *gin.Context, req Req, session *gint.Session) (gint.Result, error) {
    ctx := c.(*gctx.Context)
    
    // UserContext 已被trace中间件注入
    userContext := ctx.UserContext()
    
    return gint.Result{Data: userContext}, nil
}))
```

## 链路追踪标识

### 设置链路追踪 ID

```go
ctx := gctx.FromContext(c)
ctx.SetTraceId("trace-123")

// 或使用中间件自动生成
r.Use(trace.Middleware())
```

### 获取链路追踪 ID

```go
traceId := ctx.TraceId()
```

### 响应头自动设置

trace 中间件会自动将链路追踪 ID 写入响应头：

```
X-Trace-ID: trace-123
```

## 便捷方法

### 读写键值

```go
// 设置值
ctx.Set("key", "value")

// 获取值
value := ctx.Get("key")

// 获取字符串
value := ctx.GetString("key", "default")

// 获取整数
value := ctx.GetInt("count", 0)
```

### 请求信息

```go
// 获取客户端IP
ip := ctx.ClientIP()

// 获取请求头
auth := ctx.GetHeader("Authorization")

// 获取User-Agent
ua := ctx.GetHeader("User-Agent")
```

## 与 Gin 上下文兼容

`gctx.Context` 完全兼容 `gin.Context`：

```go
type Context struct {
    *gin.Context
}
```

所有 Gin Context 方法都可以直接调用：

```go
ctx.JSON(200, data)
ctx.Query("key")
ctx.Param("id")
ctx.PostForm("field")
ctx.GetRawData()
// ... 所有gin.Context方法
```

## 服务间传递

### 自动传递

trace 中间件 + ghttp 客户端可自动传递用户上下文：

```go
// 服务 A：设置用户上下文
r.Use(trace.Middleware())

r.GET("/api/call", func(c *gin.Context) {
    ctx := c.(*gctx.Context)
    
    // 调用服务 B，用户上下文自动传递
    client := ghttp.NewClient()
    var result Response
    client.Get(ctx, "http://service-b/api/data", &result)
    
    c.JSON(200, result)
})

// 服务 B：接收用户上下文
r.Use(trace.Middleware())

r.GET("/api/data", func(c *gin.Context) {
    ctx := c.(*gctx.Context)
    
    // 用户上下文已自动从请求头提取
    userContext := ctx.UserContext()
    
    c.JSON(200, userContext)
})
```

### 请求头格式

用户上下文通过以下请求头传递：

| 请求头 | 字段 |
|--------|------|
| `X-User-ID` | UserId |
| `X-User-Role` | Role |
| `X-Tenant-ID` | TenantId |
| `X-Username` | Username |
| `X-Email` | Email |
| `X-Trace-ID` | TraceID |

## 完整示例

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/ink-yht-code/gint"
    "github.com/ink-yht-code/gint/gctx"
    "github.com/ink-yht-code/gint/middlewares/trace"
)

func main() {
    r := gin.Default()
    
    // 链路追踪中间件（自动注入UserContext）
    r.Use(trace.Middleware())
    
    // 需要认证的路由
    auth := r.Group("/api")
    auth.Use(AuthMiddleware())
    {
        auth.GET("/profile", gint.W(GetProfile))
        auth.PUT("/profile", gint.BS(UpdateProfile))
    }
    
    r.Run(":8080")
}

// 认证中间件
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        ctx := gctx.FromContext(c)
        
        // 从token解析用户信息
        token := c.GetHeader("Authorization")
        claims, err := ParseToken(token)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"code": 401, "msg": "未授权"})
            return
        }
        
        // 设置UserContext
        ctx.SetUserContext(&gctx.UserContext{
            UserId:   claims.UserId,
            Role:     claims.Role,
            TenantId: claims.TenantId,
            Username: claims.Username,
            Email:    claims.Email,
        })
        
        c.Next()
    }
}

// 获取用户信息
func GetProfile(c *gin.Context, req struct{}) (gint.Result, error) {
    ctx := c.(*gctx.Context)
    
    userContext := ctx.UserContext()
    if userContext == nil {
        return gint.Result{}, gint.NewBizError(401, "未授权")
    }
    
    user, err := userService.GetById(userContext.UserId)
    if err != nil {
        return gint.Result{}, err
    }
    
    return gint.Result{Data: user}, nil
}

// 更新用户信息
type UpdateProfileReq struct {
    Nickname string `json:"nickname" validate:"required"`
    Avatar   string `json:"avatar"`
}

func UpdateProfile(c *gin.Context, req UpdateProfileReq, session *gint.Session) (gint.Result, error) {
    ctx := c.(*gctx.Context)
    
    userId := ctx.UserId()
    
    user, err := userService.GetById(userId)
    if err != nil {
        return gint.Result{}, err
    }
    
    user.Nickname = req.Nickname
    user.Avatar = req.Avatar
    
    if err := userService.Save(user); err != nil {
        return gint.Result{}, err
    }
    
    return gint.Result{
        Msg:  "更新成功",
        Data: user,
    }, nil
}
```
