# OpenTelemetry分布式追踪

gint 提供 OpenTelemetry 集成，支持 Jaeger、Zipkin 等分布式追踪系统。

## 概述

OpenTelemetry特性：
- 标准化分布式追踪
- 支持 Jaeger/Zipkin/OTLP
- 自动传播追踪上下文
- 支持服务间调用追踪

## 基本使用

### 初始化

```go
import "github.com/ink-yht-code/gint/middlewares/otel"

func main() {
    // 初始化OpenTelemetry
    tracer, err := otel.Init(otel.Config{
        ServiceName:    "user-service",
        ServiceVersion: "1.0.0",
        Environment:    "production",
        ExporterType:   "grpc",  // 或 "http"
        Endpoint:       "localhost:4317",
        SampleRate:     1.0,
    })
    if err != nil {
        panic(err)
    }
    defer tracer.Shutdown(context.Background())
    
    r := gin.Default()
    
    // 使用中间件
    r.Use(otel.Middleware())
    
    r.Run(":8080")
}
```

### 配置选项

```go
type Config struct {
    ServiceName    string  // 服务名称
    ServiceVersion string  // 服务版本
    Environment    string  // 环境
    ExporterType   string  // 导出器类型：grpc/http
    Endpoint       string  // 导出器地址
    Insecure       bool    // 是否不使用TLS
    SampleRate     float64 // 采样率（0-1）
}
```

## Jaeger集成

### Docker启动Jaeger

```bash
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 4317:4317 \
  -p 4318:4318 \
  jaegertracing/all-in-one:latest
```

### 配置连接Jaeger

```go
tracer, err := otel.Init(otel.Config{
    ServiceName:  "my-service",
    ExporterType: "grpc",
    Endpoint:     "localhost:4317",
    Insecure:     true,
})
```

### 访问Jaeger UI

```
http://localhost:16686
```

## 自定义Span

```go
func GetUser(c *gin.Context) {
    ctx := c.Request.Context()
    
    // 创建自定义Span
    ctx, span := otel.StartSpan(ctx, "GetUser")
    defer span.End()
    
    // 添加属性
    span.SetAttributes(
        attribute.String("user.id", "123"),
        attribute.String("user.name", "john"),
    )
    
    // 数据库查询
    ctx, dbSpan := otel.StartSpan(ctx, "db.query")
    user, err := db.GetUser(ctx, userId)
    dbSpan.End()
    
    if err != nil {
        span.RecordError(err)
    }
    
    c.JSON(200, user)
}
```

## 服务间调用追踪

### 自动传播追踪上下文

```go
import "github.com/ink-yht-code/gint/ghttp"

func CallOrderService(ctx context.Context) (*Order, error) {
    // ghttp客户端自动传播追踪上下文
    client := ghttp.NewClient()
    
    var order Order
    _, err := client.Get(ctx, "http://order-service/api/orders/123", &order)
    
    return &order, err
}
```

### 手动注入追踪上下文

```go
import (
    "net/http"
    "go.opentelemetry.io/otel"
)

func CallExternalAPI(ctx context.Context) {
    req, _ := http.NewRequest("GET", "http://external-api/data", nil)
    
    // 注入追踪上下文到请求头
    otel.GetTextMapPropagator().Inject(ctx, otel.NewHTTPHeaderCarrier(req.Header))
    
    resp, err := http.DefaultClient.Do(req)
    // ...
}
```

## 追踪信息获取

```go
func Handler(c *gin.Context) {
    // 获取TraceID
    traceID := otel.TraceIDFromContext(c.Request.Context())
    
    // 获取当前Span
    span := otel.SpanFromContext(c.Request.Context())
    
    // 添加事件
    span.AddEvent("processing", trace.WithAttributes(
        attribute.String("step", "validation"),
    ))
    
    // ...
}
```

## 完整示例

```go
package main

import (
    "context"
    
    "github.com/gin-gonic/gin"
    "github.com/ink-yht-code/gint"
    "github.com/ink-yht-code/gint/ghttp"
    "github.com/ink-yht-code/gint/middlewares/otel"
    "go.opentelemetry.io/otel/attribute"
)

func main() {
    // 初始化OpenTelemetry
    tracer, err := otel.Init(otel.Config{
        ServiceName:    "user-service",
        ServiceVersion: "1.0.0",
        Environment:    "production",
        ExporterType:   "grpc",
        Endpoint:       "localhost:4317",
        Insecure:       true,
        SampleRate:     1.0,
    })
    if err != nil {
        panic(err)
    }
    defer tracer.Shutdown(context.Background())
    
    r := gin.Default()
    
    // 追踪中间件
    r.Use(otel.Middleware())
    
    // 业务路由
    r.GET("/api/users/:id", gint.U(GetUser))
    
    r.Run(":8080")
}

type GetUserReq struct {
    ID int64 `uri:"id" validate:"required"`
}

func GetUser(c *gin.Context, req GetUserReq) (gint.Result, error) {
    ctx := c.Request.Context()
    
    // 创建子Span
    ctx, span := otel.StartSpan(ctx, "GetUser")
    defer span.End()
    
    // 查询用户
    user, err := userService.GetById(ctx, req.ID)
    if err != nil {
        span.RecordError(err)
        return gint.Result{}, err
    }
    
    // 调用订单服务
    ctx, orderSpan := otel.StartSpan(ctx, "CallOrderService")
    orders, _ := ghttp.Get(ctx, "http://order-service/api/orders?user_id="+string(req.ID), nil)
    orderSpan.End()
    
    span.SetAttributes(
        attribute.Int64("user.id", req.ID),
        attribute.String("user.name", user.Name),
    )
    
    return gint.Result{Data: gin.H{
        "user":   user,
        "orders": orders,
    }}, nil
}
```
