# Prometheus指标

gint 提供 Prometheus 指标中间件，自动收集 HTTP 请求指标。

## 兼容性说明

- 当前版本不提供 `metrics.Config` 形式的中间件参数
- 请使用 `metrics.Setup`、`metrics.Middleware` 与自定义指标构造函数

## 概述

Prometheus指标特性：
- 自动收集请求指标
- 内置常用指标
- 支持 `/metrics` 端点
- 支持自定义指标

## 基本使用

### 快速启用

```go
import "github.com/ink-yht-code/gint/middlewares/metrics"

// 自动注册 /metrics 端点
metrics.Setup(r)
```

### 手动挂载

```go
r.GET("/metrics", metrics.Handler())
r.Use(metrics.Middleware())
```

## 内置指标

| 指标名称 | 类型 | 说明 |
|---------|------|------|
| `gint_http_requests_total` | Counter | 请求总数 |
| `gint_http_request_duration_seconds` | Histogram | 请求延迟分布 |
| `gint_http_inflight_requests` | Gauge | 正在处理的请求数 |
| `gint_http_response_size_bytes` | Histogram | 响应大小分布 |

## 指标示例

```
# HELP gint_http_requests_total Total number of HTTP requests
# TYPE gint_http_requests_total counter
gint_http_requests_total{method="GET",path="/api/users",status="200"} 1234

# HELP gint_http_request_duration_seconds HTTP request latency
# TYPE gint_http_request_duration_seconds histogram
gint_http_request_duration_seconds_bucket{method="GET",path="/api/users",le="0.005"} 100
gint_http_request_duration_seconds_bucket{method="GET",path="/api/users",le="0.01"} 200
gint_http_request_duration_seconds_bucket{method="GET",path="/api/users",le="0.025"} 500
gint_http_request_duration_seconds_sum{method="GET",path="/api/users"} 12.5
gint_http_request_duration_seconds_count{method="GET",path="/api/users"} 1234
```

## 自定义指标

```go
counter := metrics.NewCounter("business_ops_total", "业务操作计数", []string{"type"})
counter.Inc("login")

gauge := metrics.NewGauge("queue_depth", "队列深度", []string{"queue"})
gauge.Set(42, "email")

hist := metrics.NewHistogram("job_latency_seconds", "任务耗时", []string{"job"})
hist.Observe(0.12, "sync_user")
```

## Prometheus配置

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'myapp'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: /metrics
```

## Grafana看板

推荐 Grafana 看板指标：

```promql
# 请求速率
rate(http_requests_total[5m])

# P99延迟
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))

# 错误率
rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])

# 正在处理的请求
http_requests_in_flight
```

## 完整示例

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/ink-yht-code/gint/middlewares/metrics"
    "github.com/prometheus/client_golang/prometheus"
)

var (
    activeUsers = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "active_users",
        Help: "Number of active users",
    })
)

func init() {
    prometheus.MustRegister(activeUsers)
}

func main() {
    r := gin.Default()
    
    // Prometheus指标
    metrics.Setup(r)
    
    // 业务路由
    r.GET("/api/users", GetUsers)
    
    r.Run(":8080")
}

func GetUsers(c *gin.Context) {
    // 更新自定义指标
    activeUsers.Set(float64(userService.GetActiveCount()))
    
    // 业务逻辑...
    c.JSON(200, users)
}
```
