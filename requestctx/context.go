package requestctx

import (
	"context"
	"strings"

	"go.uber.org/zap"
)

type contextKey string

const metaContextKey contextKey = "request_meta"

// Meta 表示一次请求在服务内部流转时需要复用的链路字段。
type Meta struct {
	// TenantID 是当前请求所属租户标识。
	TenantID string
	// UserID 是当前登录用户或调用主体 ID。
	UserID string
	// UserName 是当前用户展示名，便于日志排查时直接识别。
	UserName string
	// UserType 表示用户类型或调用方类型，例如 admin、member、system。
	UserType string
	// RequestID 是单次请求 ID，通常用于一次入口请求的唯一标识。
	RequestID string
	// TraceID 是链路追踪 ID，用于串联跨服务调用日志。
	TraceID string
	// SourceService 是上游来源服务名，用于判断请求从哪个系统进入。
	SourceService string
	// ClientIP 是客户端来源 IP，通常取自转发头或真实客户端地址。
	ClientIP string
	// PeerAddr 是当前 gRPC 连接对端地址。
	PeerAddr string
	// UserAgent 是调用方上报的 user-agent 信息。
	UserAgent string
	// HasAuthorization 表示当前请求是否携带鉴权头。
	HasAuthorization bool
	// DeadlineUnixMS 是请求 deadline 的毫秒时间戳，未设置时为 0。
	DeadlineUnixMS int64
	// MetadataKeys 记录收到的 metadata key 列表，便于调试入口透传情况。
	MetadataKeys []string
	// Metadata 是请求 metadata 的快照，用于访问日志或问题排查。
	Metadata map[string]string
}

// WithMeta 把请求元信息写入 context，供后续各层复用。
func WithMeta(ctx context.Context, meta Meta) context.Context {
	return context.WithValue(ctx, metaContextKey, meta)
}

// FromContext 从 context 中提取请求元信息。
// 如果上游未写入，则返回零值结构，调用方不需要额外判空。
func FromContext(ctx context.Context) Meta {
	if ctx == nil {
		return Meta{}
	}

	meta, ok := ctx.Value(metaContextKey).(Meta)
	if !ok {
		return Meta{}
	}
	return meta
}

// TenantID 返回上下文中的 tenant_id。
func TenantID(ctx context.Context) string {
	return strings.TrimSpace(FromContext(ctx).TenantID)
}

// UserID 返回上下文中的 user_id。
func UserID(ctx context.Context) string {
	return strings.TrimSpace(FromContext(ctx).UserID)
}

// UserName 返回上下文中的 user_name。
func UserName(ctx context.Context) string {
	return strings.TrimSpace(FromContext(ctx).UserName)
}

// UserType 返回上下文中的 user_type。
func UserType(ctx context.Context) string {
	return strings.TrimSpace(FromContext(ctx).UserType)
}

// RequestID 返回上下文中的 request_id。
func RequestID(ctx context.Context) string {
	return strings.TrimSpace(FromContext(ctx).RequestID)
}

// TraceID 返回上下文中的 trace_id。
func TraceID(ctx context.Context) string {
	return strings.TrimSpace(FromContext(ctx).TraceID)
}

// SourceService 返回上下文中的 source_service。
func SourceService(ctx context.Context) string {
	return strings.TrimSpace(FromContext(ctx).SourceService)
}

// UserIDOr 返回上下文中的 user_id；若为空则回退到 fallback。
func UserIDOr(ctx context.Context, fallback string) string {
	if value := UserID(ctx); value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}

// TenantIDOr 返回上下文中的 tenant_id；若为空则回退到 fallback。
func TenantIDOr(ctx context.Context, fallback string) string {
	if value := TenantID(ctx); value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}

// RequestIDOr 返回上下文中的 request_id；若为空则回退到 fallback。
func RequestIDOr(ctx context.Context, fallback string) string {
	if value := RequestID(ctx); value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}

// TraceIDOr 返回上下文中的 trace_id；若为空则回退到 fallback。
func TraceIDOr(ctx context.Context, fallback string) string {
	if value := TraceID(ctx); value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}

// ZapFields 把 context 中的链路字段转换成统一的 zap 字段。
func ZapFields(ctx context.Context) []zap.Field {
	meta := FromContext(ctx)
	return Fields(meta)
}

// Fields 把元信息转换成适合日志打点的结构化字段。
func Fields(meta Meta) []zap.Field {
	return []zap.Field{
		zap.String("tenant_id", strings.TrimSpace(meta.TenantID)),
		zap.String("trace_id", strings.TrimSpace(meta.TraceID)),
		zap.String("request_id", strings.TrimSpace(meta.RequestID)),
		zap.String("user_id", strings.TrimSpace(meta.UserID)),
		zap.String("user_name", strings.TrimSpace(meta.UserName)),
		zap.String("user_type", strings.TrimSpace(meta.UserType)),
		zap.String("source_service", strings.TrimSpace(meta.SourceService)),
		zap.String("client_ip", strings.TrimSpace(meta.ClientIP)),
		zap.String("peer_addr", strings.TrimSpace(meta.PeerAddr)),
		zap.String("user_agent", strings.TrimSpace(meta.UserAgent)),
		zap.Bool("has_authorization", meta.HasAuthorization),
		zap.Int64("deadline_unix_ms", meta.DeadlineUnixMS),
		zap.Bool("deadline_set", meta.DeadlineUnixMS > 0),
		zap.Strings("metadata_keys", meta.MetadataKeys),
		zap.Any("metadata", meta.Metadata),
	}
}
