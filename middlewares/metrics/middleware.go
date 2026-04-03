// Copyright 2025 ink-yht-code
//
// Proprietary License
//
// IMPORTANT: This software is NOT open source.
// You may NOT use, copy, modify, merge, publish, distribute, sublicense,
// or sell copies of this file, in whole or in part, without prior written
// permission from the copyright holder.
//
// This software is provided "AS IS", without warranty of any kind.

package metrics

import (
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	namespace = "gint"
	subsystem = "http"
)

// Metrics 指标收集器
type Metrics struct {
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestSize      *prometheus.HistogramVec
	responseSize     *prometheus.HistogramVec
	inflightRequests *prometheus.GaugeVec
}

// Option 指标配置选项
type Option func(*Metrics)

var defaultMetrics *Metrics
var initOnce sync.Once

// Init 初始化默认指标收集器
func Init(opts ...Option) {
	initOnce.Do(func() {
		defaultMetrics = NewMetrics(opts...)
		prometheus.MustRegister(
			defaultMetrics.requestsTotal,
			defaultMetrics.requestDuration,
			defaultMetrics.requestSize,
			defaultMetrics.responseSize,
			defaultMetrics.inflightRequests,
		)
	})
}

// NewMetrics 创建指标收集器
func NewMetrics(opts ...Option) *Metrics {
	m := &Metrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		requestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "request_size_bytes",
				Help:      "HTTP request size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 7),
			},
			[]string{"method", "path"},
		),
		responseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "response_size_bytes",
				Help:      "HTTP response size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 7),
			},
			[]string{"method", "path"},
		),
		inflightRequests: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "inflight_requests",
				Help:      "Current number of inflight HTTP requests",
			},
			[]string{"method"},
		),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Middleware 创建指标收集中间件
func Middleware() gin.HandlerFunc {
	if defaultMetrics == nil {
		Init()
	}

	return func(c *gin.Context) {
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		// 记录请求大小
		reqSize := float64(c.Request.ContentLength)
		if reqSize < 0 {
			reqSize = 0
		}

		// 增加进行中的请求计数
		defaultMetrics.inflightRequests.WithLabelValues(c.Request.Method).Inc()
		defer defaultMetrics.inflightRequests.WithLabelValues(c.Request.Method).Dec()

		// 处理请求
		c.Next()

		// 记录响应大小
		respSize := float64(c.Writer.Size())

		// 记录请求耗时
		duration := time.Since(start).Seconds()

		// 记录状态码
		status := strconv.Itoa(c.Writer.Status())

		// 更新指标
		defaultMetrics.requestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		defaultMetrics.requestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
		defaultMetrics.requestSize.WithLabelValues(c.Request.Method, path).Observe(reqSize)
		defaultMetrics.responseSize.WithLabelValues(c.Request.Method, path).Observe(respSize)
	}
}

// Handler 返回 Prometheus 指标暴露端点
func Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	}
}

// Setup 快速设置指标收集
// 自动注册 /metrics 端点
func Setup(r *gin.Engine) {
	r.GET("/metrics", Handler())
	r.Use(Middleware())
}

// --- 自定义指标 ---

// Counter 自定义计数器
type Counter struct {
	vec *prometheus.CounterVec
}

// NewCounter 创建自定义计数器
func NewCounter(name, help string, labels []string) *Counter {
	c := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      name,
			Help:      help,
		},
		labels,
	)
	prometheus.MustRegister(c)
	return &Counter{vec: c}
}

// Inc 增加计数
func (c *Counter) Inc(labels ...string) {
	c.vec.WithLabelValues(labels...).Inc()
}

// Add 增加指定值
func (c *Counter) Add(value float64, labels ...string) {
	c.vec.WithLabelValues(labels...).Add(value)
}

// Gauge 自定义仪表
type Gauge struct {
	vec *prometheus.GaugeVec
}

// NewGauge 创建自定义仪表
func NewGauge(name, help string, labels []string) *Gauge {
	g := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      name,
			Help:      help,
		},
		labels,
	)
	prometheus.MustRegister(g)
	return &Gauge{vec: g}
}

// Set 设置值
func (g *Gauge) Set(value float64, labels ...string) {
	g.vec.WithLabelValues(labels...).Set(value)
}

// Inc 增加值
func (g *Gauge) Inc(labels ...string) {
	g.vec.WithLabelValues(labels...).Inc()
}

// Dec 减少值
func (g *Gauge) Dec(labels ...string) {
	g.vec.WithLabelValues(labels...).Dec()
}

// Histogram 自定义直方图
type Histogram struct {
	vec *prometheus.HistogramVec
}

// NewHistogram 创建自定义直方图
func NewHistogram(name, help string, labels []string, buckets ...float64) *Histogram {
	b := prometheus.DefBuckets
	if len(buckets) > 0 {
		b = buckets
	}
	h := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      name,
			Help:      help,
			Buckets:   b,
		},
		labels,
	)
	prometheus.MustRegister(h)
	return &Histogram{vec: h}
}

// Observe 记录观测值
func (h *Histogram) Observe(value float64, labels ...string) {
	h.vec.WithLabelValues(labels...).Observe(value)
}
