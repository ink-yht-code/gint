# Swagger文档

Swagger文档模块用于自动生成OpenAPI文档。

## 功能特性

- 自动生成OpenAPI文档
- 支持Swagger UI
- 与Handler包装器集成
- 支持自定义文档

## 使用示例

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/ink-yht-code/gint"
    "github.com/ink-yht-code/gint/swagger"
)

func main() {
    r := gin.Default()
    
    // 快速初始化Swagger（推荐）
    // 自动注册 /swagger.json 和 /swagger/ui
    swagger.Setup(r, "User Service API", "v0.1.0")
    
    // 使用Handler包装器
    r.GET("/ping", gint.W(func(ctx *gctx.Context) (gint.Result, error) {
        return gint.Result{Code: 0, Msg: "pong"}, nil
    }))
    
    r.Run(":8080")
}
```

## 访问Swagger UI

启动服务后，可以通过以下地址访问Swagger UI：

```
http://localhost:8080/swagger/ui
http://localhost:8080/swagger/index.html
```

## 自定义文档

```go
// 自定义接口文档（手工构建 OpenAPI）
b := swagger.NewBuilder().
    Title("My API").
    Version("1.0.0").
    BearerAuth("bearer").
    GET("/users/:id", swagger.PathItem{
        Summary: "获取用户信息",
        Tags:    []string{"用户"},
        Parameters: []swagger.Parameter{
            swagger.PathParam("id", "用户ID", &swagger.Schema{Type: "string"}),
        },
        Responses: map[string]swagger.Response{
            "200": swagger.OKResponse("成功", nil),
        },
    })

r.GET("/swagger.json", b.Handler())
r.GET("/swagger/ui", b.UI("/swagger.json"))
```

## 配置选项

可以通过配置结构体自定义完整的文档元信息：

```go
type Config struct {
    // Title 文档标题
    Title string
    
    // Description 文档详细描述
    Description string
    
    // Version 文档版本
    Version string
    
    // TermsOfService 服务条款链接
    TermsOfService string
    
    // Contact 维护者联系信息
    Contact struct {
        Name  string // 维护者名称
        Email string // 联系邮箱
        URL   string // 个人主页
    }
    
    // License 许可证信息
    License struct {
        Name string // 许可证名称，如 MIT
        URL  string // 许可证官方链接
    }
}
```

### 使用配置初始化Swagger

```go
// 使用完整配置初始化Swagger服务
    cfg := swagger.Config{
        Title:       "User Service API",
        Version:     "v0.1.0",
        Description: "这是一个用户管理服务的接口文档",
        Contact: struct {
            Name  string
            Email string
            URL   string
        }{
            Name:  "API开发团队",
            Email: "api-support@example.com",
        },
        License: struct {
            Name string
            URL  string
        }{
            Name: "MIT License",
            URL:  "https://opensource.org/licenses/MIT",
        }
    }
    
    // 使用配置初始化
    swagger.InitWithConfig(cfg)
    // 注册路由
    swagger.Register(r)
```
```