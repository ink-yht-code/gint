# CORS跨域

gint 提供跨域资源共享（CORS）中间件，用于处理跨域请求。

## 概述

CORS中间件特性：
- 支持自定义允许的域名
- 支持自定义允许的方法
- 支持自定义允许的请求头
- 支持预检请求缓存

## 基本使用

### 默认配置

```go
import "github.com/ink-yht-code/gint/middlewares/cors"

// 使用默认配置（允许所有域名）
r.Use(cors.Middleware())
```

### 自定义配置

```go
r.Use(cors.Middleware(cors.Config{
    AllowOrigins:     []string{"https://example.com", "https://app.example.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
    ExposeHeaders:    []string{"Content-Length", "X-Request-Id"},
    AllowCredentials: true,
    MaxAge:           12 * time.Hour,
}))
```

## 配置选项

```go
type Config struct {
    // AllowOrigins 允许的域名
    // ["*"] 表示允许所有域名
    // ["https://example.com"] 表示只允许指定域名
    AllowOrigins []string
    
    // AllowMethods 允许的HTTP方法
    AllowMethods []string
    
    // AllowHeaders 允许的请求头
    AllowHeaders []string
    
    // ExposeHeaders 暴露给客户端的响应头
    ExposeHeaders []string
    
    // AllowCredentials 是否允许携带凭证（Cookie）
    AllowCredentials bool
    
    // MaxAge 预检请求缓存时间
    MaxAge time.Duration
}
```

### 默认值

```go
AllowOrigins:     []string{"*"}
AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"}
ExposeHeaders:    []string{}
AllowCredentials: false
MaxAge:           12 * time.Hour
```

## 动态域名

### 基于配置文件

```go
// 从配置加载允许的域名
allowOrigins := cfg.GetStringSlice("cors.allow_origins")

r.Use(cors.Middleware(cors.Config{
    AllowOrigins: allowOrigins,
}))
```

### 基于函数判断

```go
r.Use(cors.Middleware(cors.Config{
    AllowOriginFunc: func(origin string) bool {
        // 允许所有 example.com 子域名
        if strings.HasSuffix(origin, ".example.com") {
            return true
        }
        return false
    },
}))
```

## 完整示例

```go
package main

import (
    "time"
    
    "github.com/gin-gonic/gin"
    "github.com/ink-yht-code/gint/middlewares/cors"
)

func main() {
    r := gin.Default()
    
    // CORS中间件
    r.Use(cors.Middleware(cors.Config{
        AllowOrigins: []string{
            "https://example.com",
            "https://app.example.com",
            "http://localhost:3000", // 开发环境
        },
        AllowMethods: []string{
            "GET",
            "POST",
            "PUT",
            "DELETE",
            "OPTIONS",
        },
        AllowHeaders: []string{
            "Origin",
            "Content-Type",
            "Authorization",
            "X-Request-Id",
        },
        ExposeHeaders: []string{
            "Content-Length",
            "X-Request-Id",
        },
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    }))
    
    // 业务路由
    r.GET("/api/data", func(c *gin.Context) {
        c.JSON(200, gin.H{"data": "ok"})
    })
    
    r.Run(":8080")
}
```

## 常见问题

### 1. AllowCredentials 与 AllowOrigins "*" 冲突

当 `AllowCredentials: true` 时，`AllowOrigins` 不能为 `["*"]`：

```go
// ❌ 错误配置
r.Use(cors.Middleware(cors.Config{
    AllowOrigins:     []string{"*"},
    AllowCredentials: true, // 冲突！
}))

// ✅ 正确配置
r.Use(cors.Middleware(cors.Config{
    AllowOrigins:     []string{"https://example.com"},
    AllowCredentials: true,
}))
```

### 2. 预检请求处理

CORS中间件会自动处理 OPTIONS 预检请求，无需额外处理：

```
OPTIONS /api/data HTTP/1.1
Origin: https://example.com
Access-Control-Request-Method: POST
Access-Control-Request-Headers: Content-Type, Authorization

HTTP/1.1 204 No Content
Access-Control-Allow-Origin: https://example.com
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Origin, Content-Type, Authorization
Access-Control-Max-Age: 43200
```
