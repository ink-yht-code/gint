# gint - HTTP框架库

[![License](https://img.shields.io/badge/License-Proprietary-red.svg)](#license)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://golang.org)
[![Version](https://img.shields.io/badge/version-v0.1.0-blue.svg)](https://github.com/ink-yht-code/gint/releases)

gint 是一个基于 Gin 的企业级 HTTP 框架库，提供了丰富的中间件、Session管理、JWT认证、权限控制等功能，帮助开发者快速构建高性能的Web应用和微服务。

## 目录

- [特性](#特性)
- [快速开始](#快速开始)
- [核心功能](#核心功能)
  - [Handler包装器](#handler包装器)
  - [中间件](#中间件)
  - [Session管理](#session管理)
  - [JWT管理](#jwt管理)
  - [参数校验](#参数校验)
  - [错误处理](#错误处理)
- [分布式组件](#分布式组件)
  - [分布式锁](#分布式锁)
  - [分布式ID生成](#分布式id生成)
  - [服务注册发现](#服务注册发现)
  - [负载均衡](#负载均衡)
  - [消息队列](#消息队列)
  - [延迟队列](#延迟队列)
  - [定时任务](#定时任务)
  - [MapReduce并发](#mapreduce并发)
- [文档](#文档)
- [示例](#示例)
- [贡献](#贡献)
- [许可证](#许可证)

## 特性

- **Handler包装器** - 简化Handler编写，自动处理参数绑定、校验、响应
- **丰富的中间件** - CORS、CSRF、限流、熔断降级、访问日志、认证等
- **Session管理** - Memory和Redis存储，支持Cookie和Header传递
- **JWT管理** - JWT token生成和验证，支持双Token机制
- **参数校验** - 基于validator的参数校验，链式API
- **错误处理** - 统一的错误码和错误处理
- **Context增强** - 增强的Context，支持UserContext统一用户身份
- **微服务支持** - 健康检查、优雅关闭、Prometheus指标、服务间调用
- **配置管理** - 多源配置（文件/YAML/环境变量）
- **熔断降级** - 防止服务雪崩，支持自定义降级
- **分布式追踪** - OpenTelemetry 集成，支持 Jaeger/Zipkin
- **API文档** - Swagger/OpenAPI 自动生成
- **缓存抽象** - 统一接口，支持 Memory/Redis
- **数据库工具** - 连接池管理、查询构建器、事务
- **服务注册发现** - Memory/etcd/Nacos 服务注册、发现、健康检查
- **负载均衡** - 轮询、加权轮询、随机、最少连接、一致性哈希
- **分布式锁** - Redis 分布式锁，支持看门狗自动续期
- **分布式ID生成** - 雪花算法，支持 1024 节点，每毫秒 4096 ID
- **消息队列** - 统一抽象层，支持 Kafka/RabbitMQ/Memory
- **延迟队列** - Redis ZSET 实现，支持精确延迟
- **定时任务** - Cron 表达式、固定间隔、每日调度
- **MapReduce并发** - Map/Filter/ForEach/FlatMap/Partition 并发处理

## 最近更新（2026-04）

### 新增功能
- **分布式锁** - Redis 实现，支持 `TryLock`、`Lock`、`Unlock`、`Extend`，看门狗自动续期
- **分布式ID生成** - 雪花算法实现，支持自定义 epoch，提供 `NodeIDFromIP`/`NodeIDFromMac`
- **定时任务** - `Scheduler` 调度器，支持 Cron 表达式、固定间隔、每日执行
- **延迟队列** - Redis ZSET 实现 + Memory 实现，支持精确延迟消息
- **消息队列抽象层** - 统一 `Producer`/`Consumer` 接口，支持 Kafka/RabbitMQ/Memory
- **MapReduce工具** - `MapReduce`、`Map`、`Filter`、`ForEach`、`FlatMap`、`Partition` 并发处理
- **etcd服务发现** - 租约保活、Watch 监听、本地缓存
- **Nacos服务发现** - 服务注册/发现、订阅变更

### Bug修复
- 修复 `cache` 默认实例未初始化时的 panic，现返回 `ErrCacheNotInitialized`
- 修复 `auth` 中间件配置缺失时的空指针风险（缺失 `JWTManager` 返回 500）
- 修复网关认证透传行为，用户信息会写入请求头供下游转发链路使用
- 修复熔断器半开状态限流逻辑，`HalfOpenRequests` 现已生效
- 修复配置加载错误被吞的问题，`config.Load()` 现在会返回聚合错误
- 修复指标中间件重复初始化导致重复注册 panic 的问题（`Init` 幂等）
- 修复随机负载均衡并发访问风险（`Random.Select` 已并发安全）
- 移除 `lumberjack` 的 `+incompatible` 依赖，改用自实现日志轮转器
- 新增多组单元/集成测试，覆盖上述核心行为

## 快速开始

### 安装

```bash
go get github.com/ink-yht-code/gint
```

### 基本使用

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/ink-yht-code/gint"
    "github.com/ink-yht-code/gint/gctx"
)

func main() {
    r := gin.Default()
    
    // 使用包装器
    r.GET("/ping", gint.W(func(ctx *gctx.Context) (gint.Result, error) {
        return gint.Result{Code: 0, Msg: "pong"}, nil
    }))
    
    r.Run(":8080")
}
```

## 核心功能

### Handler包装器

gint提供了多种包装器，简化Handler编写：

#### W - 无参数包装器

```go
r.GET("/ping", gint.W(func(ctx *gctx.Context) (gint.Result, error) {
    return gint.Result{Code: 0, Msg: "pong"}, nil
}))
```

#### B - 带参数绑定包装器

```go
type HelloReq struct {
    Name string `json:"name" validate:"required"`
}

r.POST("/hello", gint.B(func(ctx *gctx.Context, req *HelloReq) (gint.Result, error) {
    return gint.Result{Code: 0, Data: "Hello, " + req.Name}, nil
}))
```

#### S - 带Session包装器

```go
r.GET("/profile", gint.S(func(ctx *gctx.Context) (gint.Result, error) {
    session := ctx.Session()
    userID := session.Get("user_id")
    return gint.Result{Code: 0, Data: userID}, nil
}))
```

### 中间件

#### CORS跨域

```go
import "github.com/ink-yht-code/gint/middlewares/cors"

// 使用默认配置
r.Use(cors.Default())

// 或自定义配置
r.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"http://localhost:3000"},
    AllowMethods:     []string{"GET", "POST"},
    AllowCredentials: true,
}))
```

#### 限流

```go
import (
    "time"
    "github.com/ink-yht-code/gint/middlewares/ratelimit"
)

limiter := ratelimit.NewSimpleLimiter(100, time.Second)
r.Use(ratelimit.NewBuilder(limiter).Build())
```

#### CSRF防护

```go
import "github.com/ink-yht-code/gint/middlewares/csrf"

r.Use(csrf.New(csrf.DefaultConfig()))
```

#### 活跃连接限制

```go
import "github.com/ink-yht-code/gint/middlewares/activelimit"

r.Use(activelimit.NewBuilder(1000).Build()) // 最多1000个活跃连接
```

#### 访问日志

```go
import (
    "github.com/ink-yht-code/gint/middlewares/accesslog"
    "github.com/ink-yht-code/gint/logger"
)

r.Use(accesslog.NewBuilder(accesslog.ZapLogFunc()).Build())
```

### Session管理

#### Memory存储

```go
import (
    "github.com/ink-yht-code/gint/session"
    "github.com/ink-yht-code/gint/session/memory"
)

provider := memory.NewProvider()
sessionManager := session.NewManager(provider)

// 设置Session
sessionManager.Set(ctx, "user_id", "12345", time.Hour*24)

// 获取Session
userID := sessionManager.Get(ctx, "user_id")

// 删除Session
sessionManager.Delete(ctx, "user_id")
```

#### Redis存储

```go
import (
    "github.com/ink-yht-code/gint/session"
    "github.com/ink-yht-code/gint/session/redis"
    "github.com/redis/go-redis/v9"
)

redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

provider := redis.NewProvider(redisClient)
sessionManager := session.NewManager(provider)
```

#### Session传递方式

```go
// Cookie传递（默认）
carrier := cookie.NewCarrier("session_id")

// Header传递
carrier := header.NewCarrier("X-Session-Id")

sessionManager := session.NewManager(provider, session.WithCarrier(carrier))
```

### JWT管理

#### 创建JWT Manager

```go
import "github.com/ink-yht-code/gint/jwt"

opts := jwt.NewOptions("your-secret-key", time.Hour*2, time.Hour*24*7)
manager := jwt.NewManager(opts)
```

#### 生成Token

```go
// 生成单个 Token
claims := jwt.Claims{UserId: "12345"}
token, err := manager.GenerateToken(claims)

// 生成 Token 对（Access + Refresh）
tokenPair, err := manager.GenerateTokenPair(claims)
// tokenPair.AccessToken, tokenPair.RefreshToken
```

#### 验证Token

```go
// 验证 Token
claims, err := manager.VerifyToken(token)

// 验证 Refresh Token
claims, err := manager.VerifyRefreshToken(refreshToken)
```

#### 刷新Token

```go
// 验证 Refresh Token 后重新生成
claims, err := manager.VerifyRefreshToken(refreshToken)
if err == nil {
    newTokenPair, err := manager.GenerateTokenPair(*claims)
}
```

### 参数校验

#### 链式API

```go
import "github.com/ink-yht-code/gint"

validator := gint.NewValidatorBuilder().
    Field("name", req.Name).
    AddRule(gint.Required()).
    AddRule(gint.MinLength(2)).
    AddRule(gint.MaxLength(50)).
    Validate()

if !validator.IsValid() {
    return gint.InvalidParam(validator.GetFirstError()), nil
}
```

#### 内置校验规则

```go
gint.Required()           // 必填
gint.Min(0)             // 最小值
gint.Max(100)           // 最大值
gint.MinLength(6)        // 最小长度
gint.MaxLength(50)        // 最大长度
gint.Email()             // 邮箱格式
gint.Phone()             // 手机号格式
gint.URL()              // URL格式
gint.Regexp(`^\d+$`)   // 正则表达式
```

### 错误处理

#### 标准错误码

```go
gint.Success()                    // 成功 (0)
gint.InvalidParam("参数错误")      // 参数错误 (1)
gint.Unauthorized("未授权")       // 未授权 (2)
gint.Forbidden("无权限")          // 无权限 (3)
gint.NotFound("资源不存在")        // 未找到 (4)
gint.Conflict("资源冲突")        // 冲突 (5)
gint.InternalError("内部错误")    // 内部错误 (9999)
```

#### 自定义错误

```go
// 创建业务错误
err := &gint.BizError{
    Code:  1001,
    Msg:   "用户不存在",
    Cause: nil,
}

// 使用自定义错误
return gint.Result{}, err
```

## 文档

- [文档总览](docs/README.md)
- [迁移指南（旧版 -> 当前版本）](docs/迁移指南.md)
- [核心功能/Handler包装器](docs/核心功能/Handler包装器.md)
- [核心功能/参数校验](docs/核心功能/参数校验.md)
- [核心功能/错误处理](docs/核心功能/错误处理.md)
- [认证授权/JWT管理](docs/认证授权/JWT管理.md)
- [认证授权/Session管理](docs/认证授权/Session管理.md)
- [认证授权/认证中间件](docs/认证授权/认证中间件.md)

## 微服务支持

### 健康检查

```go
import "github.com/ink-yht-code/gint/middlewares/health"

// K8s 存活检查（仅检查进程存活）
r.GET("/live", health.Liveness())

// K8s 就绪检查（检查依赖服务）
r.GET("/ready", health.Readiness(
    health.NewChecker("db", health.CheckDB(db.Ping), 5*time.Second),
    health.NewChecker("redis", health.CheckRedis(rdb.Ping), 3*time.Second),
))

// 或一键配置
health.Health(r, checkers...)
```

### 优雅关闭

```go
import "github.com/ink-yht-code/gint/server"

func main() {
    r := gin.Default()
    
    // 创建服务器
    srv := server.New(":8080", r, server.WithShutdownTimeout(30*time.Second))
    
    // 添加关闭钩子
    shutdown := server.NewGracefulShutdown(30*time.Second).
        AddHook(server.HookCloseDB(db.Close)).
        AddHook(server.HookCloseRedis(rdb.Close))
    
    // 启动并等待信号
    go srv.Start()
    shutdown.Wait()
}
```

### Prometheus 指标

```go
import "github.com/ink-yht-code/gint/middlewares/metrics"

// 自动注册 /metrics 端点和收集中间件
metrics.Setup(r)

// 自定义指标
counter := metrics.NewCounter("business_ops_total", "业务操作计数", []string{"type"})
counter.Inc("login")
```

### 服务间调用

```go
import "github.com/ink-yht-code/gint/ghttp"

// 创建客户端（自动传递链路追踪和用户身份）
client := ghttp.NewClient(
    ghttp.WithTimeout(5*time.Second),
    ghttp.WithRetry(3, 100*time.Millisecond),
)

// 调用下游服务
var result UserResponse
resp, err := client.Get(ctx, "http://user-service/api/users/123", &result)

// 或使用便捷方法
ghttp.Post(ctx, "http://order-service/api/orders", orderReq, &result)
```

### 配置管理

```go
import "github.com/ink-yht-code/gint/config"

// 快捷加载（config.yaml + 环境变量 APP_）
cfg, err := config.LoadApp()

// 或自定义配置源
cfg := config.New(
    config.WithFile("config.yaml"),
    config.WithFile("config.local.yaml"),
    config.WithEnv("APP_"),
)

// 获取配置
dbHost := cfg.GetString("db.host", "localhost")
dbPort := cfg.GetInt("db.port", 3306)
debug := cfg.GetBool("debug", false)

// 解析到结构体
var dbConfig DatabaseConfig
cfg.Unmarshal("db", &dbConfig)
```

### 熔断降级

```go
import "github.com/ink-yht-code/gint/middlewares/circuitbreaker"

// 基本熔断（连续5次失败后熔断30秒）
r.Use(circuitbreaker.Middleware())

// 带降级处理
r.Use(circuitbreaker.WithFallback(func(c *gin.Context) {
    c.JSON(200, gin.H{"code": 0, "msg": "服务降级", "data": nil})
}))

// 自定义配置
r.Use(circuitbreaker.NewBuilder(circuitbreaker.Config{
    FailureThreshold: 10,    // 连续失败10次触发
    SuccessThreshold: 5,     // 半开状态连续成功5次恢复
    Timeout:          60*time.Second,
}).Build())

// 按路径分组熔断
r.Use(circuitbreaker.NewBuilder().WithKeyFunc(func(c *gin.Context) string {
    return c.FullPath() // 每个路径独立熔断
}).Build())
```

### OpenTelemetry 分布式追踪

```go
import "github.com/ink-yht-code/gint/middlewares/otel"

// 初始化（连接 Jaeger/Zipkin 等）
tracer, err := otel.Init(otel.Config{
    ServiceName:    "user-service",
    Environment:    "production",
    ExporterType:   "grpc",  // 或 "http"
    Endpoint:       "localhost:4317",
    SampleRate:     1.0,
})
defer tracer.Shutdown(ctx)

// 使用中间件
r.Use(otel.Middleware())

// 服务间调用自动传播追踪上下文
client := ghttp.NewClient() // 自动注入 trace
```

### Swagger API 文档

```go
import "github.com/ink-yht-code/gint/swagger"

// 快速设置（自动注册所有路由）
swagger.Setup(r, "My API", "1.0.0")

// 访问: http://localhost:8080/swagger.json
// UI: http://localhost:8080/swagger/ui

// 或手动构建
b := swagger.NewBuilder().
    Title("My API").
    Version("1.0.0").
    BearerAuth("bearer").
    GET("/users", swagger.PathItem{
        Summary: "获取用户列表",
        Tags:    []string{"用户"},
        Responses: map[string]swagger.Response{
            "200": swagger.OKResponse("成功", nil),
        },
    })
r.GET("/swagger.json", b.Handler())
```

### 缓存

```go
import "github.com/ink-yht-code/gint/cache"
import "github.com/ink-yht-code/gint/cache/memory"
import "github.com/ink-yht-code/gint/cache/redis"

// 内存缓存
memCache := memory.New()

// Redis 缓存
redisCache := redis.New(redisClient, "app")

// 使用缓存
cache.Set(ctx, "user:123", []byte("data"), 10*time.Minute)
data, err := cache.Get(ctx, "user:123")

// 类型安全缓存
userCache := cache.NewTypedCache[User](redisCache, "user")
userCache.Set(ctx, "123", user, 10*time.Minute)
```

### 数据库

```go
import "github.com/ink-yht-code/gint/db"

// 打开连接
database, err := db.OpenMySQL("user:pass@tcp(localhost:3306)/db")

// 查询构建器
users, _ := db.Scan[User](database.Table("users").
    Where("status = ?", 1).
    OrderBy("created_at DESC").
    Limit(10).
    Get(ctx, database))

// 插入
database.Insert("users").
    Fields("name", "email").
    Values("John", "john@example.com").
    Exec(ctx, database)

// 事务
database.Transaction(ctx, func(tx *db.Tx) error {
    tx.Table("users").Update().Set("balance", 100).Exec(ctx, tx)
    return nil
})
```

### 服务注册发现

```go
import "github.com/ink-yht-code/gint/discovery"

// 创建注册中心
registry := discovery.NewMemoryRegistry()

// 注册服务
registry.Register(ctx, &discovery.ServiceInstance{
    ID:      "user-1",
    Name:    "user-service",
    Address: "192.168.1.10:8080",
})

// 发现服务
instances, _ := registry.GetInstances(ctx, "user-service")
```

### 负载均衡

```go
import "github.com/ink-yht-code/gint/loadbalance"

// 轮询
lb := loadbalance.NewRoundRobin()
instance, _ := lb.Select(instances)

// 加权轮询
wlb := loadbalance.NewWeightedRoundRobin()

// 最少连接
lclb := loadbalance.NewLeastConnections()
defer lclb.Release(instance)
```

## 分布式组件

### 分布式锁

基于 Redis 的分布式锁实现，支持看门狗自动续期：

```go
import (
    "context"
    "time"
    "github.com/redis/go-redis/v9"
    "github.com/ink-yht-code/gint/lock"
    "github.com/ink-yht-code/gint/lock/redis"
)

// 创建 Redis 客户端
rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

// 创建分布式锁
locker := redislock.New(rdb, "myapp")

// 尝试获取锁（非阻塞）
ctx := context.Background()
acquired, err := locker.TryLock(ctx, "resource:123", 30*time.Second)
if acquired {
    defer locker.Unlock(ctx, "resource:123")
    // 执行业务逻辑
}

// 阻塞等待获取锁
err = locker.Lock(ctx, "resource:123", 30*time.Second, 
    lock.WithWaitInterval(100*time.Millisecond),
    lock.WithMaxWait(10*time.Second),
)

// 看门狗自动续期（防止业务执行时间超过锁过期时间）
locker.Lock(ctx, "resource:123", 30*time.Second, lock.WithWatchdog())

// 手动延长锁时间
locker.Extend(ctx, "resource:123", 30*time.Second)
```

### 分布式ID生成

雪花算法实现，适用于分布式系统唯一 ID 生成：

```go
import "github.com/ink-yht-code/gint/snowflake"

// 创建节点（nodeID: 0-1023）
node, _ := snowflake.NewNode(1)

// 生成 ID
id := node.Generate()
fmt.Println(id.Int64())  // 1234567890123456789
fmt.Println(id.String()) // "1234567890123456789"
fmt.Println(id.Base62()) // "1A2B3C4D5E"

// 解析 ID 信息
fmt.Println(id.Timestamp()) // 生成时间
fmt.Println(id.NodeID())    // 节点 ID
fmt.Println(id.Sequence())  // 序列号

// 从 IP 或 MAC 地址生成 NodeID
nodeID := snowflake.NodeIDFromIP("192.168.1.100")
nodeID := snowflake.NodeIDFromMAC("00:1A:2B:3C:4D:5E")

// 自定义 epoch（开始时间）
node, _ := snowflake.NewNode(1, snowflake.WithEpoch(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)))
```

### 服务注册发现

支持 Memory、etcd、Nacos 三种注册中心：

```go
import "github.com/ink-yht-code/gint/discovery"
import "github.com/ink-yht-code/gint/discovery/etcd"
import "github.com/ink-yht-code/gint/discovery/nacos"

// ========== etcd 注册中心 ==========
etcdRegistry, _ := etcddiscovery.New(etcddiscovery.Config{
    Endpoints:   []string{"localhost:2379"},
    DialTimeout: 5 * time.Second,
    TTL:         10,
})

// 注册服务
etcdRegistry.Register(ctx, &discovery.ServiceInstance{
    ID:       "user-service-1",
    Name:     "user-service",
    Address:  "192.168.1.10:8080",
    Weight:   100,
    Metadata: map[string]string{"version": "v1.0.0"},
})

// 发现服务
instances, _ := etcdRegistry.GetInstances(ctx, "user-service")

// 监听服务变化
etcdRegistry.Watch(ctx, "user-service", discovery.WatchHandlerFunc{
    AddFunc:    func(inst *discovery.ServiceInstance) { fmt.Println("新增:", inst.Address) },
    DeleteFunc: func(inst *discovery.ServiceInstance) { fmt.Println("删除:", inst.Address) },
})

// ========== Nacos 注册中心 ==========
nacosRegistry, _ := nacosdiscovery.New(nacosdiscovery.Config{
    ServerAddr: "localhost:8848",
    Namespace:  "dev",
    GroupName: "DEFAULT_GROUP",
})

// 注册、发现、监听同上
```

### 消息队列

统一的消息队列抽象层，支持 Kafka、RabbitMQ、Memory：

```go
import "github.com/ink-yht-code/gint/mq"
import "github.com/ink-yht-code/gint/mq/kafka"
import "github.com/ink-yht-code/gint/mq/rabbitmq"

// ========== Kafka ==========
kafkaProducer, _ := kafkamq.NewProducer(kafkamq.ProducerConfig{
    Brokers: []string{"localhost:9092"},
    Async:   true, // 异步发送
})

// 发送消息
kafkaProducer.Send(ctx, mq.Message{
    Topic: "orders",
    Key:   []byte("order-123"),
    Value: []byte(`{"id": 123, "status": "created"}`),
})

// 消费消息
kafkaConsumer, _ := kafkamq.NewConsumer(kafkamq.ConsumerConfig{
    Brokers: []string{"localhost:9092"},
    GroupID: "order-processor",
    Topics:  []string{"orders"},
})
kafkaConsumer.Subscribe(ctx, func(msg mq.Message) error {
    fmt.Printf("收到消息: %s\n", string(msg.Value))
    return nil
})

// ========== RabbitMQ ==========
rabbitProducer, _ := rabbitmq.NewProducer(rabbitmq.ProducerConfig{
    URL:      "amqp://guest:guest@localhost:5672/",
    Exchange: "orders",
})

rabbitConsumer, _ := rabbitmq.NewConsumer(rabbitmq.ConsumerConfig{
    URL:      "amqp://guest:guest@localhost:5672/",
    Queue:    "order-queue",
})
```

### 延迟队列

基于 Redis ZSET 实现的延迟队列：

```go
import "github.com/ink-yht-code/gint/delayqueue"
import "github.com/redis/go-redis/v9"

rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

// 创建延迟队列
dq := delayqueue.NewRedis(rdb, delayqueue.Config{
    QueueKey:    "delay-queue:orders",
    BufferSize:  100,
    PollInterval: 100 * time.Millisecond,
})

// 投递延迟消息（5分钟后执行）
dq.Enqueue(ctx, delayqueue.Message{
    ID:        "order-cancel-123",
    Payload:   []byte(`{"orderId": 123}`),
    DelayTime: time.Now().Add(5 * time.Minute),
})

// 消费延迟消息
dq.Consume(ctx, func(msg delayqueue.Message) error {
    // 处理超时未支付的订单
    return processOrderCancel(msg.Payload)
})
```

### 定时任务

支持 Cron 表达式、固定间隔、每日调度：

```go
import "github.com/ink-yht-code/gint/cron"

scheduler := cron.NewScheduler()

// 固定间隔执行（每 5 分钟）
scheduler.AddJob("cleanup", cron.NewIntervalSchedule(5*time.Minute), func(ctx context.Context) error {
    return cleanupExpiredSessions()
})

// 每日固定时间执行（每天凌晨 2 点）
scheduler.AddJob("daily-report", cron.NewDailySchedule(2, 0, 0), func(ctx context.Context) error {
    return generateDailyReport()
})

// Cron 表达式（每分钟执行）
scheduler.AddJob("health-check", cron.NewCronSchedule("* * * * *"), func(ctx context.Context) error {
    return checkServiceHealth()
})

// 启动调度器
scheduler.Start()
defer scheduler.Stop()

// 手动触发
scheduler.Trigger("cleanup")

// 暂停/恢复
scheduler.Pause("daily-report")
scheduler.Resume("daily-report")
```

### MapReduce并发

并发数据处理工具，简化并发编程：

```go
import "github.com/ink-yht-code/gint/mr"

// MapReduce: Map -> Reduce
result, _ := mr.MapReduce(
    context.Background(),
    []int{1, 2, 3, 4, 5},
    // Map: 每个元素平方
    func(ctx context.Context, v int) (int, error) {
        return v * v, nil
    },
    // Reduce: 求和
    func(ctx context.Context, results []int) (int, error) {
        sum := 0
        for _, r := range results {
            sum += r
        }
        return sum, nil
    },
    mr.WithWorkers(4), // 4 个并发 worker
)
// result = 55 (1+4+9+16+25)

// Map: 并发映射
doubled, _ := mr.Map(context.Background(), []int{1, 2, 3}, 
    func(ctx context.Context, v int) (int, error) {
        return v * 2, nil
    },
)
// doubled = [2, 4, 6]

// Filter: 并发过滤
evens, _ := mr.Filter(context.Background(), []int{1, 2, 3, 4, 5, 6},
    func(ctx context.Context, v int) (bool, error) {
        return v%2 == 0, nil
    },
)
// evens = [2, 4, 6]

// ForEach: 并发遍历（无返回值）
mr.ForEach(context.Background(), []string{"a", "b", "c"},
    func(ctx context.Context, v string) error {
        fmt.Println(v)
        return nil
    },
)

// FlatMap: 映射后展平
flat, _ := mr.FlatMap(context.Background(), []string{"hello", "world"},
    func(ctx context.Context, v string) ([]string, error) {
        return strings.Split(v, ""), nil
    },
)
// flat = ["h", "e", "l", "l", "o", "w", "o", "r", "l", "d"]

// Partition: 分区处理
mr.Partition(context.Background(), bigSlice, 100,
    func(ctx context.Context, chunk []int) error {
        return processBatch(chunk)
    },
)
```

## 示例

### 完整的RESTful API

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/ink-yht-code/gint"
    "github.com/ink-yht-code/gint/gctx"
    "github.com/ink-yht-code/gint/middlewares/cors"
    "github.com/ink-yht-code/gint/middlewares/ratelimit"
)

type CreateUserReq struct {
    Name  string `json:"name" validate:"required"`
    Email string `json:"email" validate:"required,email"`
}

type User struct {
    ID    int64  `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    r := gin.Default()
    
    // 使用中间件
    r.Use(cors.Middleware())
    r.Use(ratelimit.Middleware(100))
    
    // 路由
    r.POST("/users", gint.B(createUser))
    r.GET("/users/:id", gint.W(getUser))
    
    r.Run(":8080")
}

func createUser(ctx *gctx.Context, req *CreateUserReq) (gint.Result, error) {
    // 参数校验
    validator := gint.NewValidatorBuilder().
        Field("name", req.Name).AddRule(gint.Required()).
        Field("email", req.Email).AddRule(gint.Required()).AddRule(gint.Email()).
        Validate()
    
    if !validator.IsValid() {
        return gint.InvalidParam(validator.GetFirstError()), nil
    }
    
    // 业务逻辑
    user := &User{
        ID:    1,
        Name:  req.Name,
        Email: req.Email,
    }
    
    return gint.Result{Code: 0, Data: user}, nil
}

func getUser(ctx *gctx.Context) (gint.Result, error) {
    userID := ctx.Param("id")
    
    // 业务逻辑
    user := &User{
        ID:    1,
        Name:  "John Doe",
        Email: "john@example.com",
    }
    
    return gint.Result{Code: 0, Data: user}, nil
}
```

## 贡献

欢迎贡献代码！请遵循以下步骤：

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 许可证

Proprietary License

未经版权所有者书面授权，不得使用、复制、修改或分发本项目的任何部分。

## 联系方式

- 项目主页: [https://github.com/ink-yht-code/gint](https://github.com/ink-yht-code/gint)
- 问题反馈: [https://github.com/ink-yht-code/gint/issues](https://github.com/ink-yht-code/gint/issues)

---

Made with ❤️ by [ink-yht-code](https://github.com/ink-yht-code)
