# 跨域资源共享（CORS）

`middlewares/cors` 提供跨域资源共享中间件。

## 基本使用

```go
import "github.com/ink-yht-code/gint/middlewares/cors"

r.Use(cors.New(cors.Config{
	AllowOrigins:     []string{"http://localhost:3000"},
	AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
	AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Refresh-Token"},
	ExposeHeaders:    []string{"Authorization", "X-Refresh-Token"},
	AllowCredentials: false,
	MaxAge:           86400,
}))
```

## 配置项

```go
type Config struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}
```

## 说明

- 如果 `AllowCredentials` 为 `true`，不要再使用通配符来源
- 中间件会自动处理 `OPTIONS` 预检请求
- 认证相关 Header 的暴露设置要和前端协议保持一致
