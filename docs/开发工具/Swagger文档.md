# Swagger 接口文档

`gint/swagger` 提供 OpenAPI 构建器和轻量级 Swagger 界面支持。

## 常见用法

```go
b := swagger.NewBuilder().
	Title("用户服务 API").
	Version("v0.1.0").
	Server("http://127.0.0.1:8080", "本地环境")

r.GET("/swagger.json", b.Handler())
r.GET("/swagger/ui", b.UI("/swagger.json"))
```

## 构建器示例

```go
b := swagger.NewBuilder().
	Title("我的 API").
	Version("1.0.0").
	BearerAuth("bearerAuth").
	GET("/users/{id}", swagger.PathItem{
		Summary: "获取用户详情",
		Tags:    []string{"user"},
		Parameters: []swagger.Parameter{
			swagger.PathParam("id", "用户 ID", &swagger.Schema{Type: "string"}),
		},
		Responses: map[string]swagger.Response{
			"200": swagger.OKResponse("成功", nil),
		},
	})
```

## 常见路径

- `/swagger.json`
- `/swagger/ui`
- `/swagger/index.html`

## 常用辅助方法

- `swagger.OKResponse`
- `swagger.ErrorResponse`
- `swagger.JSONBody`
- `swagger.PathParam`
- `swagger.HeaderParam`
- `swagger.QueryParam`

## 建议

- 文档最好显式注册，便于稳定维护
- 重复响应结构建议抽成可复用的模式定义
- 认证请求头和标签命名在各模块中保持一致
