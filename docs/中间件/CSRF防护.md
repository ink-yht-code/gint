# 跨站请求伪造防护（CSRF）

`middlewares/csrf` 为会修改服务端状态的请求提供 CSRF 防护。

## 基本使用

```go
import "github.com/ink-yht-code/gint/middlewares/csrf"

r.Use(csrf.New(csrf.DefaultConfig()))
```

## 自定义示例

```go
r.Use(csrf.New(csrf.Config{
	Secret:      "your-secret-key",
	TokenLength: 32,
	TokenLookup: "header:X-CSRF-Token",
}))
```

## 令牌查找位置

常见写法：

- `header:X-CSRF-Token`
- `form:_csrf`
- `query:csrf_token`

## 建议

- 主要用于浏览器侧的状态修改接口
- 纯只读 `GET` 接口通常可以跳过
- 生产环境应配合 HTTPS 与安全 Cookie 配置使用
