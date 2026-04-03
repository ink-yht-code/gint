# gint 框架文档目录

> 说明：本文档以当前主分支代码为准。历史版本中的部分示例 API（例如旧版认证配置字段）已不再适用，请优先参考本文档列出的模块说明与代码定义。

- [迁移指南（旧版 -> 当前版本）](./迁移指南.md) - 旧文档路径映射、模块变更与升级建议

## 核心功能

- [Handler包装器](./核心功能/Handler包装器.md) - 简化Handler编写，自动处理参数绑定、校验、响应
- [Context增强](./核心功能/Context增强.md) - 增强的Context，支持UserContext统一用户身份
- [错误处理](./核心功能/错误处理.md) - 统一的错误码和错误处理机制
- [参数校验](./核心功能/参数校验.md) - 基于validator的参数校验，链式API

## 认证授权

- [JWT管理](./认证授权/JWT管理.md) - JWT token生成和验证，支持双Token机制
- [Session管理](./认证授权/Session管理.md) - Memory和Redis存储，支持Cookie和Header传递
- [认证中间件](./认证授权/认证中间件.md) - 统一认证中间件，支持JWT和Session

## 中间件

- [CORS跨域](./中间件/CORS跨域.md) - 跨域资源共享配置
- [CSRF防护](./中间件/CSRF防护.md) - 跨站请求伪造防护
- [限流](./中间件/限流.md) - 基于IP/用户的限流
- [熔断降级](./中间件/熔断降级.md) - 防止服务雪崩
- [链路追踪](./中间件/链路追踪.md) - 自定义链路追踪
- [OpenTelemetry](./中间件/OpenTelemetry.md) - 标准化分布式追踪
- [健康检查](./中间件/健康检查.md) - K8s liveness/readiness探针
- [Prometheus指标](./中间件/Prometheus指标.md) - HTTP请求指标收集
- [访问日志](./中间件/访问日志.md) - 结构化访问日志

## 微服务支持

- [配置管理](./微服务支持/配置管理.md) - 多源配置（文件/YAML/环境变量）
- [优雅关闭](./微服务支持/优雅关闭.md) - 信号处理和资源清理
- [服务间调用](./微服务支持/服务间调用.md) - HTTP客户端，支持重试、超时、追踪传播

## 数据层

- [缓存](./数据层/缓存.md) - 统一缓存接口，支持Memory/Redis
- [数据库](./数据层/数据库.md) - 连接池管理、查询构建器、事务

## 服务治理

- [服务注册发现](./服务治理/服务注册发现.md) - 服务注册、发现、健康检查
- [负载均衡](./服务治理/负载均衡.md) - 轮询、加权轮询、随机、最少连接、一致性哈希

## 开发工具

- [Swagger文档](./开发工具/Swagger文档.md) - OpenAPI文档自动生成
- [日志](./开发工具/日志.md) - zap结构化日志，支持MQ输出

## 快速开始

### 安装

```bash
go get github.com/ink-yht-code/gint
```

### 最小示例

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/ink-yht-code/gint"
    "github.com/ink-yht-code/gint/gctx"
)

func main() {
    r := gin.Default()
    
    // 使用Handler包装器
    r.GET("/hello", gint.W(func(ctx *gctx.Context) (gint.Result, error) {
        return gint.Success("ok", "hello"), nil
    }))
    
    r.Run(":8080")
}
```

### 完整示例

```go
package main

import (
    "context"
    "time"
    
    "github.com/gin-gonic/gin"
    "github.com/ink-yht-code/gint"
    "github.com/ink-yht-code/gint/config"
    "github.com/ink-yht-code/gint/db"
    "github.com/ink-yht-code/gint/cache/memory"
    "github.com/ink-yht-code/gint/middlewares/health"
    "github.com/ink-yht-code/gint/middlewares/metrics"
    "github.com/ink-yht-code/gint/middlewares/circuitbreaker"
    "github.com/ink-yht-code/gint/server"
    "github.com/ink-yht-code/gint/swagger"
)

func main() {
    // 加载配置
    cfg, _ := config.LoadApp()
    
    // 初始化数据库
    database, _ := db.OpenMySQL(cfg.GetString("db.dsn"))
    defer database.Close()
    
    // 初始化缓存
    cache := memory.New()
    defer cache.Close()
    
    // 创建路由
    r := gin.Default()
    
    // 健康检查
    health.Health(r)
    
    // Prometheus指标
    metrics.Setup(r)
    
    // 熔断降级
    r.Use(circuitbreaker.Middleware())
    
    // Swagger文档
    swagger.Setup(r, "My API", "1.0.0")
    
    // 业务路由
    api := r.Group("/api")
    {
        api.GET("/users", gint.S(ListUsers))
        api.POST("/users", gint.BS(CreateUser))
    }
    
    // 优雅关闭
    srv := server.New(":8080", r)
    srv.Run()
}
```
