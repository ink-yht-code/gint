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

package otel

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ink-yht-code/gint/gctx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// Config OpenTelemetry 配置
type Config struct {
	// ServiceName 服务名称
	ServiceName string
	// ServiceVersion 服务版本
	ServiceVersion string
	// Environment 环境（dev/staging/prod）
	Environment string
	// ExporterType 导出器类型：grpc/http
	ExporterType string
	// Endpoint 导出器地址
	Endpoint string
	// Insecure 是否不使用 TLS
	Insecure bool
	// SampleRate 采样率（0-1）
	SampleRate float64
}

// DefaultConfig 默认配置
func DefaultConfig() Config {
	return Config{
		ServiceName:    "gint-service",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		ExporterType:   "grpc",
		Endpoint:       "localhost:4317",
		Insecure:       true,
		SampleRate:     1.0,
	}
}

// Tracer OpenTelemetry 追踪器
type Tracer struct {
	config   Config
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
	shutdown func(context.Context) error
}

var globalTracer *Tracer

// Init 初始化 OpenTelemetry
func Init(cfg Config) (*Tracer, error) {
	// 创建资源
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			attribute.String("environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, err
	}

	// 创建导出器
	var exporter sdktrace.SpanExporter
	switch cfg.ExporterType {
	case "http":
		opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(cfg.Endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		exporter, err = otlptrace.New(context.Background(), otlptracehttp.NewClient(opts...))
	default: // grpc
		opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(cfg.Endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		exporter, err = otlptracegrpc.New(context.Background(), opts...)
	}
	if err != nil {
		return nil, err
	}

	// 创建采样器
	sampler := sdktrace.ParentBased(
		sdktrace.TraceIDRatioBased(cfg.SampleRate),
	)

	// 创建 TracerProvider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// 设置全局 TracerProvider
	otel.SetTracerProvider(provider)

	// 设置全局 Propagator（用于跨服务传播）
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	t := &Tracer{
		config:   cfg,
		provider: provider,
		tracer:   provider.Tracer(cfg.ServiceName),
		shutdown: provider.Shutdown,
	}

	globalTracer = t
	return t, nil
}

// Shutdown 关闭追踪器
func (t *Tracer) Shutdown(ctx context.Context) error {
	return t.shutdown(ctx)
}

// Tracer 返回 trace.Tracer
func (t *Tracer) Tracer() trace.Tracer {
	return t.tracer
}

// --- 中间件 ---

// Middleware 创建 OpenTelemetry 中间件
func Middleware(serviceName ...string) gin.HandlerFunc {
	_ = serviceName // 可选的服务名参数，当前使用全局配置

	return func(c *gin.Context) {
		if globalTracer == nil {
			c.Next()
			return
		}

		// 从请求头提取追踪上下文
		ctx := otel.GetTextMapPropagator().Extract(
			c.Request.Context(),
			&headerCarrier{c.Request.Header},
		)

		// 开始 Span
		spanName := c.Request.Method + " " + c.FullPath()
		if spanName == "" {
			spanName = c.Request.Method + " " + c.Request.URL.Path
		}

		ctx, span := globalTracer.tracer.Start(ctx, spanName,
			trace.WithAttributes(
				semconv.HTTPMethod(c.Request.Method),
				semconv.HTTPURL(c.Request.URL.String()),
				semconv.HTTPRoute(c.FullPath()),
				attribute.String("http.scheme", c.Request.URL.Scheme),
				attribute.String("http.host", c.Request.Host),
				attribute.String("http.user_agent", c.Request.UserAgent()),
				attribute.Int64("http.request_content_length", c.Request.ContentLength),
			),
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		// 设置上下文
		c.Request = c.Request.WithContext(ctx)

		// 注入 traceID 到 gctx
		spanCtx := span.SpanContext()
		if spanCtx.IsValid() {
			ginCtx := &gctx.Context{Context: c}
			ginCtx.SetTraceId(spanCtx.TraceID().String())
		}

		// 执行请求
		c.Next()

		// 记录状态码
		status := c.Writer.Status()
		span.SetAttributes(semconv.HTTPStatusCode(status))

		if status >= 400 {
			span.SetStatus(codes.Error, http.StatusText(status))
			if status >= 500 {
				span.RecordError(c.Errors.Last())
			}
		} else {
			span.SetStatus(codes.Ok, "OK")
		}
	}
}

// headerCarrier 实现 propagation.TextMapCarrier
type headerCarrier struct {
	http.Header
}

func (h *headerCarrier) Keys() []string {
	keys := make([]string, 0, len(h.Header))
	for k := range h.Header {
		keys = append(keys, k)
	}
	return keys
}

// --- 辅助函数 ---

// StartSpan 开始一个新的 Span
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if globalTracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return globalTracer.tracer.Start(ctx, name, opts...)
}

// SpanFromContext 从上下文获取当前 Span
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// TraceIDFromContext 从上下文获取 TraceID
func TraceIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// InjectHTTP 将追踪上下文注入到 HTTP 请求
func InjectHTTP(ctx context.Context, req *http.Request) {
	otel.GetTextMapPropagator().Inject(ctx, &headerCarrier{req.Header})
}

// ExtractHTTP 从 HTTP 请求提取追踪上下文
func ExtractHTTP(req *http.Request) context.Context {
	return otel.GetTextMapPropagator().Extract(
		req.Context(),
		&headerCarrier{req.Header},
	)
}

// --- 用于 ghttp Client 的追踪 ---

// TraceHTTPClient 包装 HTTP 客户端，自动注入追踪
func TraceHTTPClient(client *http.Client) *http.Client {
	if client == nil {
		client = http.DefaultClient
	}

	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	client.Transport = &tracingTransport{
		Transport: transport,
		tracer:    globalTracer.tracer,
	}

	return client
}

type tracingTransport struct {
	Transport http.RoundTripper
	tracer    trace.Tracer
}

func (t *tracingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.tracer == nil {
		return t.Transport.RoundTrip(req)
	}

	ctx := req.Context()
	ctx, span := t.tracer.Start(ctx, req.Method+" "+req.URL.Host,
		trace.WithAttributes(
			semconv.HTTPMethod(req.Method),
			semconv.HTTPURL(req.URL.String()),
			attribute.String("http.host", req.Host),
		),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	// 注入追踪上下文
	otel.GetTextMapPropagator().Inject(ctx, &headerCarrier{req.Header})

	req = req.WithContext(ctx)
	resp, err := t.Transport.RoundTrip(req)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return resp, err
	}

	span.SetAttributes(semconv.HTTPStatusCode(resp.StatusCode))
	if resp.StatusCode >= 400 {
		span.SetStatus(codes.Error, http.StatusText(resp.StatusCode))
	} else {
		span.SetStatus(codes.Ok, "OK")
	}

	return resp, nil
}

// --- 全局函数 ---

// GetTracer 获取全局追踪器
func GetTracer() *Tracer {
	return globalTracer
}

// Shutdown 关闭全局追踪器
func Shutdown(ctx context.Context) error {
	if globalTracer != nil {
		return globalTracer.Shutdown(ctx)
	}
	return nil
}
