# Prometheus 指标采集

`middlewares/metrics` 提供 Prometheus 风格的 HTTP 指标采集能力。

## 基本使用

```go
import "github.com/ink-yht-code/gint/middlewares/metrics"

metrics.Setup(r)
```

通常会挂载：

- `/metrics`

并采集常见 HTTP 指标，例如：

- 请求总数
- 请求耗时
- 正在处理的请求数

## 手动挂载

```go
r.GET("/metrics", metrics.Handler())
r.Use(metrics.Middleware())
```

## 建议

- 生产环境要合理保护 `/metrics`
- 标签维度不要过高
- 路径统计优先使用路由模板而不是原始 URL
