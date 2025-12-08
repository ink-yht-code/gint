# gint

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/ink-yht-code/gint)](https://goreportcard.com/report/github.com/ink-yht-code/gint)

基于 Gin 框架的**轻量级**增强库，专注于核心功能，简单易用。

> 💡 **定位说明**：gint 是 [ginx](https://github.com/ecodeclub/ginx) 的轻量级衍生项目，只保留最常用的核心功能，适合中小型项目和快速开发。如果需要完整的企业级功能，请使用原项目 ginx。

> 📜 **版权声明**：本项目基于 [ginx](https://github.com/ecodeclub/ginx) 开发，遵循 Apache License 2.0。感谢 ginx 原作者及贡献者的优秀工作。详见 [NOTICE](./NOTICE) 文件。

## ✨ 特性

- 🚀 **优雅的 Handler 包装器** - W/B/S/BS 四种包装函数，简化业务逻辑编写
- 🔐 **双 Token 认证** - Access Token + Refresh Token，支持无感刷新
- 🔄 **完善的 Session 管理** - JWT + Redis 混合方案，Token 包含 UserId
- 🛡️ **统一的错误处理** - 内置错误处理机制，代码更简洁
- 📝 **实用的中间件** - 访问日志、限流、活跃连接限制、CORS
- 🔒 **Casbin 权限管理** - 基于 RBAC 的权限管理，接口抽象设计
- 🎯 **类型安全** - 使用泛型提供类型安全的参数绑定
- 💡 **中文注释** - 完整的中文注释，易于理解

## 📦 安装

```bash
go get github.com/ink-yht-code/gint
```

## 🚀 快速开始

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/ink-yht-code/gint"
    "github.com/ink-yht-code/gint/gctx"
)

func main() {
    r := gin.Default()
    
    // 简单接口
    r.GET("/ping", gint.W(func(ctx *gctx.Context) (gint.Result, error) {
        return gint.Result{Code: 0, Msg: "pong"}, nil
    }))
    
    // 带参数绑定
    type LoginReq struct {
        Username string `json:"username" binding:"required"`
        Password string `json:"password" binding:"required"`
    }
    
    r.POST("/login", gint.B(func(ctx *gctx.Context, req LoginReq) (gint.Result, error) {
        // 参数已自动绑定和验证
        return gint.Result{Code: 0, Msg: "登录成功"}, nil
    }))
    
    r.Run(":8080")
}
```

## 📚 文档

- **[项目概览](./docs/项目概览.md)** - 了解 gint 的设计理念和核心特性
- **[响应码规范](./docs/响应码规范.md)** - 统一的响应码定义（0=成功，1=警告，2=错误）
- **[参数校验](./docs/参数校验.md)** - 强大的参数校验功能（策略+建造者+组合模式）
- **[Handler包装器](./docs/Handler包装器.md)** - W/B/S/BS 四种包装器的详细用法
- **[Session管理](./docs/Session管理.md)** - JWT + Redis 混合存储方案
- **[双Token机制](./docs/双Token机制.md)** - Access Token + Refresh Token 详解
- **[Memory-Session](./docs/Memory-Session.md)** - 开发测试用的内存 Session
- **[中间件](./docs/中间件.md)** - 访问日志、限流、CORS 等中间件的使用
- **[活跃连接限制](./docs/活跃连接限制.md)** - 限制同时处理的请求数
- **[Casbin权限管理](./docs/Casbin权限管理.md)** - 基于 RBAC 的权限管理集成
- **[Context增强](./docs/Context增强.md)** - 便捷的参数获取和类型转换

## 💡 核心概念

### Handler 包装器

gint 提供四种包装器，自动处理参数绑定、Session 验证、错误处理等：

- **W** - 基础包装器，只接收 Context
- **B** - 带参数绑定的包装器
- **S** - 带 Session 的包装器
- **BS** - 参数绑定 + Session 的包装器

### Session 管理

JWT + Redis 混合存储方案：
- **JWT** - 存储轻量数据（用户ID、角色等）
- **Redis** - 存储完整会话数据
- **自动续期** - 访问时自动刷新过期时间

### 中间件

- **访问日志** - 记录请求和响应信息
- **限流器** - 支持 IP 限流和用户限流
- **活跃连接限制** - 限制同时处理的请求数
- **CORS** - 处理跨域请求


## 🔄 与 ginx 的对比

| 特性 | ginx | gint |
|------|------|------|
| **定位** | 企业级框架 | 轻量级核心库 |
| **UserID 类型** | `int64` | `string` |
| **注释语言** | 英文 | 中文 |
| **核心功能** | ✅ | ✅ |
| **活跃连接限制** | ✅ | ✅ |
| **Memory Session** | ✅ | ✅ |
| **CORS** | ❌ | ✅ |
| **爬虫检测** | ✅ | ❌ (暂未实现) |
| **依赖** | ekit | 仅 Gin 生态 |

**如何选择？**
- **ginx** - 大型企业项目，需要完整功能
- **gint** - 中小型项目，快速开发，简单易用

详细对比请查看 [项目概览](./docs/项目概览.md#与-ginx-的功能对比)。

## 🙏 致谢

本项目基于 [ginx](https://github.com/ecodeclub/ginx) 开发，感谢原作者 [ecodeclub](https://github.com/ecodeclub) 及所有贡献者的优秀工作。

gint 在 ginx 的基础上进行了以下改进：
- 简化架构，专注核心功能
- 重新设计 Session 管理，实现双 Token 机制
- 采用设计模式重构参数校验系统
- 添加统一的响应码规范
- 完整的中文文档

## 📄 开源协议

本项目采用 [Apache License 2.0](./LICENSE) 开源协议。

### 版权归属

- **gint**: Copyright 2025 Light-ink-yht
- **ginx** (原始项目): Copyright by ecodeclub and contributors

### 合规声明

本项目严格遵守 Apache License 2.0 的所有条款：

✅ 保留了原始项目的版权声明和归属信息  
✅ 在 [NOTICE](./NOTICE) 文件中详细说明了衍生关系  
✅ 标记了所有修改的文件  
✅ 使用相同的 Apache License 2.0 协议  
✅ 提供了完整的修改说明文档

### 第三方依赖

本项目使用的第三方库及其协议：

- [gin-gonic/gin](https://github.com/gin-gonic/gin) - MIT License
- [golang-jwt/jwt](https://github.com/golang-jwt/jwt) - MIT License
- [redis/go-redis](https://github.com/redis/go-redis) - BSD 2-Clause License
- [google/uuid](https://github.com/google/uuid) - BSD 3-Clause License
- [dlclark/regexp2](https://github.com/dlclark/regexp2) - MIT License
- [uber-go/atomic](https://github.com/uber-go/atomic) - MIT License

完整的归属信息请查看 [NOTICE](./NOTICE) 文件。


---

**免责声明**：本项目按"原样"提供，不提供任何明示或暗示的保证。详见 [LICENSE](./LICENSE) 文件。
