# gintx - 运行时框架

gintx 提供微服务运行时基础设施组件。

## 安装

```bash
go get github.com/ink-yht-code/gintx
```

## 模块

### log - 日志

基于 zap 的结构化日志，支持 ctx 注入 request_id。

```go
import "github.com/ink-yht-code/gintx/log"

// 初始化
log.Init(log.Config{
    Level:    "info",
    Encoding: "json",
    Output:   "stdout",
})

// 使用
log.Info("message", zap.String("key", "value"))
log.InfoCtx(ctx, "message with request_id")
```

### tx - 事务管理

事务管理器，支持 ctx 注入事务 DB。

```go
import "github.com/ink-yht-code/gintx/tx"

// 创建事务管理器
txMgr := tx.NewManager(db)

// 在事务中执行
err := txMgr.Do(ctx, func(ctx context.Context) error {
    // 获取事务 DB
    txDB := tx.FromContext(ctx, db)
    return txDB.Create(&model).Error
})

// 在 DAO 层获取 DB
func (d *DAO) Create(ctx context.Context, m *Model) error {
    db := tx.GetDB(ctx, d.db) // 自动使用事务 DB
    return db.Create(m).Error
}
```

### db - 数据库

GORM 数据库初始化。

```go
import "github.com/ink-yht-code/gintx/db"

db, err := db.New(db.Config{
    DSN:      "user:pass@tcp(127.0.0.1:3306)/db",
    MaxOpen:  100,
    MaxIdle:  10,
    LogLevel: "info",
})
```

### redis - Redis

Redis 客户端初始化。

```go
import "github.com/ink-yht-code/gintx/redis"

client := redis.New(redis.Config{
    Addr:     "127.0.0.1:6379",
    Password: "",
    DB:       0,
})
```

### httpx - HTTP 服务器

Gin HTTP 服务器，内置中间件。

```go
import "github.com/ink-yht-code/gintx/httpx"

server := httpx.NewServer(httpx.Config{
    Enabled: true,
    Addr:    ":8080",
})

// 注册路由
server.Engine.GET("/api/v1/hello", handler)

// 启动
server.Run()

// 关闭
server.Shutdown(ctx)
```

内置中间件：
- `RequestID()` - 请求 ID
- `Logger()` - 访问日志
- `Recovery()` - Panic 恢复

### rpc - gRPC 服务器

gRPC 服务器，内置拦截器。

```go
import "github.com/ink-yht-code/gintx/rpc"

server := rpc.NewServer(rpc.Config{
    Enabled: true,
    Addr:    ":9090",
})

// 注册服务
pb.RegisterUserServiceServer(server.Server, userService)

// 启动
server.Run()

// 关闭
server.Shutdown(ctx)
```

### health - 健康检查

HTTP 和 gRPC 健康检查。

```go
import "github.com/ink-yht-code/gintx/health"

// HTTP 健康检查
server.Engine.GET("/health", health.HTTPHandler())

// HTTP 就绪检查
server.Engine.GET("/ready", health.ReadyHandler(
    checkDB,
    checkRedis,
))

// gRPC 健康检查
healthServer := health.NewHealthServer()
healthServer.Register("user-service", &DBChecker{db})
grpc_health_v1.RegisterHealthServer(server.Server, healthServer)
```

### error - 错误映射

业务错误到 HTTP/gRPC 响应映射。

```go
import "github.com/ink-yht-code/gintx/error"

// 实现 BizError 接口
type BizError struct {
    code int
    msg  string
}

func (e *BizError) BizCode() int { return e.code }
func (e *BizError) BizMsg() string { return e.msg }
func (e *BizError) Error() string { return e.msg }

// 映射到 HTTP
error.MapToHTTP(c, err)
```

### outbox - Outbox 模式

事件发布 Outbox 模式（TODO）。

### app - 应用启动器

应用生命周期管理。

```go
import "github.com/ink-yht-code/gintx/app"

app, err := app.New(&app.Config{
    Service: app.ServiceConfig{ID: 101, Name: "user"},
    HTTP:    httpx.Config{Enabled: true, Addr: ":8080"},
    Log:     log.Config{Level: "info", Encoding: "json"},
    DB:      db.Config{DSN: "..."},
    Redis:   redis.Config{Addr: "..."},
})

app.Run()
app.Shutdown(ctx)
```

## 错误码规范

业务码 = ServiceID * 10000 + BizCode

| BizCode | 含义 |
|---------|------|
| 0 | 成功 |
| 1 | 参数错误 |
| 2 | 未授权 |
| 3 | 无权限 |
| 4 | 未找到 |
| 5 | 冲突 |
| 9999 | 内部错误 |

## License

Apache License 2.0
