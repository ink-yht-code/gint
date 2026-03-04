# Context 增强功能详解

## 概述

`gctx.Context` 是对 `gin.Context` 的增强封装，提供了更便捷的参数获取、类型转换和响应方法。它完全兼容 `gin.Context`，可以访问所有原生方法。

## 核心特性

- **链式类型转换** - 通过 `Value` 类型支持链式调用
- **便捷的参数获取** - 统一的 API 获取各种参数
- **用户 ID 管理** - 内置用户 ID 的存取方法
- **SSE 支持** - 简化 Server-Sent Events 的实现
- **便捷响应方法** - 快速返回成功或错误响应

## Value 类型

`Value` 是一个封装了值和错误的结构，支持链式类型转换。

### 结构定义

```go
type Value struct {
    val string
    err error
}
```

### 类型转换方法

#### String

```go
// String 返回字符串值和错误
func (v Value) String() (string, error)

// StringOr 返回字符串值，如果有错误则返回默认值
func (v Value) StringOr(defaultVal string) string
```

使用示例：

```go
// 获取字符串，处理错误
name, err := ctx.Query("name").String()
if err != nil {
    // 处理错误
}

// 获取字符串，使用默认值
name := ctx.Query("name").StringOr("匿名")
```

#### Int

```go
// Int 将值转换为 int
func (v Value) Int() (int, error)

// IntOr 将值转换为 int，如果失败则返回默认值
func (v Value) IntOr(defaultVal int) int
```

使用示例：

```go
// 获取整数，处理错误
page, err := ctx.Query("page").Int()
if err != nil {
    page = 1
}

// 获取整数，使用默认值
page := ctx.Query("page").IntOr(1)
size := ctx.Query("size").IntOr(10)
```

#### Int64

```go
// Int64 将值转换为 int64
func (v Value) Int64() (int64, error)

// Int64Or 将值转换为 int64，如果失败则返回默认值
func (v Value) Int64Or(defaultVal int64) int64
```

使用示例：

```go
// 获取用户 ID
userId := ctx.Param("id").Int64Or(0)

// 获取时间戳
timestamp := ctx.Query("timestamp").Int64Or(time.Now().Unix())
```

#### Bool

```go
// Bool 将值转换为 bool
func (v Value) Bool() (bool, error)

// BoolOr 将值转换为 bool，如果失败则返回默认值
func (v Value) BoolOr(defaultVal bool) bool
```

使用示例：

```go
// 获取布尔值
isPublic := ctx.Query("public").BoolOr(false)
includeDeleted := ctx.Query("include_deleted").BoolOr(false)
```

## 参数获取方法

### Param - 路径参数

从 URL 路径中获取参数。

```go
// 路由定义
r.GET("/users/:id", handler)

// 获取参数
func handler(ctx *gctx.Context) (gint.Result, error) {
    // 获取字符串
    userId := ctx.Param("id").StringOr("")
    
    // 获取整数
    userId := ctx.Param("id").IntOr(0)
    
    return gint.Result{Code: 0, Data: userId}, nil
}
```

### Query - 查询参数

从 URL 查询字符串中获取参数。

```go
// 请求: GET /search?keyword=golang&page=2&size=20

func handler(ctx *gctx.Context) (gint.Result, error) {
    keyword := ctx.Query("keyword").StringOr("")
    page := ctx.Query("page").IntOr(1)
    size := ctx.Query("size").IntOr(10)
    
    results := search(keyword, page, size)
    return gint.Result{Code: 0, Data: results}, nil
}
```

### Cookie - Cookie 值

从请求 Cookie 中获取值。

```go
func handler(ctx *gctx.Context) (gint.Result, error) {
    // 获取 Cookie 值
    token := ctx.Cookie("token").StringOr("")
    
    // 处理错误
    theme, err := ctx.Cookie("theme").String()
    if err != nil {
        theme = "light" // 默认主题
    }
    
    return gint.Result{Code: 0, Data: theme}, nil
}
```

### Header - 请求头

从请求头中获取值。

```go
func handler(ctx *gctx.Context) (gint.Result, error) {
    // 获取 Authorization
    token := ctx.Header("Authorization").StringOr("")
    
    // 获取 User-Agent
    userAgent := ctx.Header("User-Agent").StringOr("")
    
    // 获取自定义头
    apiKey := ctx.Header("X-API-Key").StringOr("")
    
    return gint.Result{Code: 0}, nil
}
```

## 用户 ID 管理

### UserId - 获取用户 ID

从上下文中获取用户 ID，通常由认证中间件设置。

```go
func handler(ctx *gctx.Context) (gint.Result, error) {
    userId := ctx.UserId()
    if userId == "" {
        return gint.Result{Code: 401, Msg: "未登录"}, nil
    }
    
    // 使用 userId 查询用户信息
    user := getUserById(userId)
    return gint.Result{Code: 0, Data: user}, nil
}
```

### SetUserId - 设置用户 ID

在认证中间件中设置用户 ID。

```go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        ctx := &gctx.Context{Context: c}
        
        // 验证 Token
        token := ctx.Header("Authorization").StringOr("")
        userId, err := validateToken(token)
        if err != nil {
            c.JSON(401, gin.H{"code": 401, "msg": "未授权"})
            c.Abort()
            return
        }
        
        // 设置用户 ID
        ctx.SetUserId(userId)
        
        c.Next()
    }
}
```

## SSE（Server-Sent Events）

### EventStream - 创建事件流

`EventStream` 方法返回一个通道，用于发送 SSE 事件。

```go
func handler(ctx *gctx.Context) (gint.Result, error) {
    eventCh := ctx.EventStream()
    
    // 在协程中发送事件
    go func() {
        for i := 0; i < 10; i++ {
            // 发送事件
            eventCh <- []byte(fmt.Sprintf("data: message %d\n\n", i))
            time.Sleep(time.Second)
        }
        // 关闭通道表示结束
        close(eventCh)
    }()
    
    // 返回 ErrNoResponse 表示不需要返回 JSON
    return gint.Result{}, gint.ErrNoResponse
}
```

### SSE 事件格式

SSE 事件必须遵循以下格式：

```
data: 消息内容\n\n
```

可以包含多行数据：

```
data: 第一行\n
data: 第二行\n\n
```

可以指定事件类型：

```
event: custom-event\n
data: 消息内容\n\n
```

### 完整示例

```go
// 实时日志推送
r.GET("/logs", gint.W(func(ctx *gctx.Context) (gint.Result, error) {
    eventCh := ctx.EventStream()
    
    go func() {
        defer close(eventCh)
        
        // 模拟日志流
        logs := []string{
            "Starting application...",
            "Loading configuration...",
            "Connecting to database...",
            "Server started on :8080",
        }
        
        for _, log := range logs {
            select {
            case eventCh <- []byte(fmt.Sprintf("data: %s\n\n", log)):
                time.Sleep(500 * time.Millisecond)
            case <-ctx.Request.Context().Done():
                // 客户端断开连接
                return
            }
        }
    }()
    
    return gint.Result{}, gint.ErrNoResponse
}))

// 客户端代码（JavaScript）
const eventSource = new EventSource('/logs');
eventSource.onmessage = (event) => {
    console.log('Log:', event.data);
};
```

### 进度推送示例

```go
// 文件处理进度
r.POST("/process", gint.W(func(ctx *gctx.Context) (gint.Result, error) {
    eventCh := ctx.EventStream()
    
    go func() {
        defer close(eventCh)
        
        for progress := 0; progress <= 100; progress += 10 {
            data := map[string]any{
                "progress": progress,
                "message":  fmt.Sprintf("Processing... %d%%", progress),
            }
            
            jsonData, _ := json.Marshal(data)
            eventCh <- []byte(fmt.Sprintf("data: %s\n\n", jsonData))
            
            time.Sleep(time.Second)
        }
    }()
    
    return gint.Result{}, gint.ErrNoResponse
}))
```

## 便捷响应方法

### Success - 成功响应

快速返回成功响应（状态码 200，code 0）。

```go
func handler(ctx *gctx.Context) (gint.Result, error) {
    data := map[string]any{
        "name": "张三",
        "age":  25,
    }
    
    // 等同于 ctx.JSON(200, gin.H{"code": 0, "msg": "success", "data": data})
    ctx.Success(data)
    
    return gint.Result{}, gint.ErrNoResponse
}
```

### Error - 错误响应

快速返回错误响应（状态码 200，自定义 code）。

```go
func handler(ctx *gctx.Context) (gint.Result, error) {
    // 等同于 ctx.JSON(200, gin.H{"code": 404, "msg": "用户不存在", "data": nil})
    ctx.Error(404, "用户不存在")
    
    return gint.Result{}, gint.ErrNoResponse
}
```

### 使用场景

这两个方法适合在需要提前返回响应的场景：

```go
func handler(ctx *gctx.Context) (gint.Result, error) {
    userId := ctx.Param("id").StringOr("")
    if userId == "" {
        ctx.Error(400, "缺少用户 ID")
        return gint.Result{}, gint.ErrNoResponse
    }
    
    user, err := getUserById(userId)
    if err != nil {
        ctx.Error(404, "用户不存在")
        return gint.Result{}, gint.ErrNoResponse
    }
    
    ctx.Success(user)
    return gint.Result{}, gint.ErrNoResponse
}
```

**注意**：使用这些方法后，必须返回 `gint.ErrNoResponse`，否则会重复返回响应。

## 访问原生 gin.Context

`gctx.Context` 嵌入了 `gin.Context`，可以直接访问所有原生方法：

```go
func handler(ctx *gctx.Context) (gint.Result, error) {
    // 访问原生方法
    ctx.JSON(200, gin.H{"message": "hello"})
    ctx.File("./file.pdf")
    ctx.Redirect(302, "/login")
    ctx.SetCookie("name", "value", 3600, "/", "", false, true)
    
    // 获取请求信息
    method := ctx.Request.Method
    path := ctx.Request.URL.Path
    ip := ctx.ClientIP()
    
    return gint.Result{}, gint.ErrNoResponse
}
```

## 实用示例

### 分页查询

```go
func listUsers(ctx *gctx.Context) (gint.Result, error) {
    page := ctx.Query("page").IntOr(1)
    size := ctx.Query("size").IntOr(10)
    
    // 验证参数
    if page < 1 {
        page = 1
    }
    if size < 1 || size > 100 {
        size = 10
    }
    
    offset := (page - 1) * size
    users, total := getUsersFromDB(offset, size)
    
    return gint.Result{
        Code: 0,
        Data: gint.PageData[User]{
            List:  users,
            Total: total,
            Page:  page,
            Size:  size,
        },
    }, nil
}
```

### 文件上传

```go
func uploadFile(ctx *gctx.Context) (gint.Result, error) {
    file, err := ctx.FormFile("file")
    if err != nil {
        return gint.Result{Code: 400, Msg: "文件上传失败"}, err
    }
    
    // 验证文件大小
    if file.Size > 10*1024*1024 { // 10MB
        return gint.Result{Code: 400, Msg: "文件过大"}, nil
    }
    
    // 保存文件
    dst := "./uploads/" + file.Filename
    if err := ctx.SaveUploadedFile(file, dst); err != nil {
        return gint.Result{Code: 500, Msg: "保存文件失败"}, err
    }
    
    return gint.Result{
        Code: 0,
        Data: map[string]any{
            "filename": file.Filename,
            "size":     file.Size,
        },
    }, nil
}
```

### 条件查询

```go
func searchArticles(ctx *gctx.Context) (gint.Result, error) {
    // 获取查询条件
    keyword := ctx.Query("keyword").StringOr("")
    category := ctx.Query("category").StringOr("")
    status := ctx.Query("status").IntOr(0)
    startDate := ctx.Query("start_date").StringOr("")
    endDate := ctx.Query("end_date").StringOr("")
    
    // 构建查询
    query := buildQuery(keyword, category, status, startDate, endDate)
    articles := executeQuery(query)
    
    return gint.Result{Code: 0, Data: articles}, nil
}
```

### IP 限制

```go
func adminOnly(ctx *gctx.Context) (gint.Result, error) {
    // 获取客户端 IP
    ip := ctx.ClientIP()
    
    // 检查 IP 白名单
    allowedIPs := []string{"127.0.0.1", "192.168.1.100"}
    if !contains(allowedIPs, ip) {
        return gint.Result{Code: 403, Msg: "禁止访问"}, nil
    }
    
    // 执行管理操作
    return gint.Result{Code: 0, Msg: "操作成功"}, nil
}
```

## 最佳实践

### 1. 使用链式调用

```go
// ✅ 推荐：使用 Or 方法提供默认值
page := ctx.Query("page").IntOr(1)
size := ctx.Query("size").IntOr(10)

// ❌ 不推荐：手动处理错误
page, err := ctx.Query("page").Int()
if err != nil {
    page = 1
}
```

### 2. 参数验证

```go
// ✅ 推荐：验证参数范围
page := ctx.Query("page").IntOr(1)
if page < 1 {
    page = 1
}

size := ctx.Query("size").IntOr(10)
if size < 1 || size > 100 {
    size = 10
}
```

### 3. 错误处理

```go
// ✅ 推荐：区分业务错误和系统错误
user, err := getUserById(userId)
if err != nil {
    if errors.Is(err, ErrUserNotFound) {
        return gint.Result{Code: 404, Msg: "用户不存在"}, nil
    }
    return gint.Result{Code: 500}, err
}
```

### 4. SSE 资源清理

```go
// ✅ 推荐：监听客户端断开
go func() {
    defer close(eventCh)
    
    for {
        select {
        case eventCh <- data:
            // 发送数据
        case <-ctx.Request.Context().Done():
            // 客户端断开，清理资源
            return
        }
    }
}()
```

## 总结

`gctx.Context` 通过提供便捷的方法和链式 API，大大简化了参数获取和类型转换的代码。合理使用这些增强功能，可以让你的代码更加简洁和易读。同时，它完全兼容 `gin.Context`，可以无缝使用 Gin 的所有原生功能。
