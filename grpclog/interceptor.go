package grpclog

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	gintlogger "github.com/ink-yht-code/gint/logger"
	"github.com/ink-yht-code/gint/requestctx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	// DefaultMaxPayloadBytes 是默认的请求/响应日志截断长度。
	DefaultMaxPayloadBytes = 4096
)

// Options 描述 gRPC 访问日志拦截器的可配置项。
type Options struct {
	// ServiceName 是当前服务名，会写入访问日志。
	ServiceName string
	// MaxPayloadBytes 控制请求体和响应体日志的最大长度。
	MaxPayloadBytes int
	// DefaultTenantID 用于单租户服务或上游未透传 x-tenant-id 时的回填值。
	DefaultTenantID string
	// WriteResponseHeaders 指定是否把 request_id、trace_id、tenant_id 回写到 gRPC 响应头。
	WriteResponseHeaders bool
}

// UnaryServerInterceptor 创建一个 gRPC 服务端访问日志拦截器。
// 它会提取链路字段、记录请求和响应载荷，并把元信息写回 context 供业务层复用。
func UnaryServerInterceptor(opts Options) grpc.UnaryServerInterceptor {
	maxPayloadBytes := opts.MaxPayloadBytes
	if maxPayloadBytes <= 0 {
		maxPayloadBytes = DefaultMaxPayloadBytes
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		startAt := time.Now()
		meta := extractRequestMeta(ctx)

		if meta.RequestID == "" {
			meta.RequestID = uuid.NewString()
		}
		if meta.TraceID == "" {
			meta.TraceID = meta.RequestID
		}
		if strings.TrimSpace(meta.TenantID) == "" {
			meta.TenantID = strings.TrimSpace(opts.DefaultTenantID)
		}

		ctx = requestctx.WithMeta(ctx, meta)
		if opts.WriteResponseHeaders {
			_ = grpc.SetHeader(ctx, metadata.Pairs(
				"x-request-id", meta.RequestID,
				"x-trace-id", meta.TraceID,
				"x-tenant-id", meta.TenantID,
			))
		}

		resp, err := handler(ctx, req)

		cost := time.Since(startAt)
		st, ok := status.FromError(err)
		if !ok {
			st = status.New(codes.OK, "")
		}

		fields := []zap.Field{
			zap.String("service", strings.TrimSpace(opts.ServiceName)),
			zap.String("protocol", "grpc"),
			zap.String("method", info.FullMethod),
			zap.String("code", st.Code().String()),
			zap.Int64("duration_ms", cost.Milliseconds()),
			zap.String("tenant_id", meta.TenantID),
			zap.String("trace_id", meta.TraceID),
			zap.String("request_id", meta.RequestID),
			zap.String("user_id", meta.UserID),
			zap.String("user_name", meta.UserName),
			zap.String("user_type", meta.UserType),
			zap.String("source_service", meta.SourceService),
			zap.String("client_ip", meta.ClientIP),
			zap.String("peer_addr", meta.PeerAddr),
			zap.String("user_agent", meta.UserAgent),
			zap.Bool("has_authorization", meta.HasAuthorization),
			zap.Int64("deadline_unix_ms", meta.DeadlineUnixMS),
			zap.Bool("deadline_set", meta.DeadlineUnixMS > 0),
			zap.Strings("metadata_keys", meta.MetadataKeys),
			zap.Any("metadata", meta.Metadata),
			zap.String("request_payload", marshalPayload(req, maxPayloadBytes)),
			zap.String("response_payload", marshalPayload(resp, maxPayloadBytes)),
		}
		if err != nil {
			fields = append(fields, zap.Error(err), zap.String("error_message", st.Message()))
		}

		logByCode(st.Code(), fields...)
		return resp, err
	}
}

// extractRequestMeta 从 gRPC metadata、peer 和 deadline 中提取统一链路字段。
func extractRequestMeta(ctx context.Context) requestctx.Meta {
	meta := requestctx.Meta{}
	if deadline, ok := ctx.Deadline(); ok {
		meta.DeadlineUnixMS = deadline.UnixMilli()
	}

	if p, ok := peer.FromContext(ctx); ok && p != nil && p.Addr != nil {
		meta.PeerAddr = p.Addr.String()
	}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		meta.TenantID = firstNonEmpty(md, "x-tenant-id")
		meta.UserID = firstNonEmpty(md, "x-user-id", "user-id")
		meta.UserName = firstNonEmpty(md, "x-user-name", "user-name")
		meta.UserType = firstNonEmpty(md, "x-user-type", "user-type", "x-user-role", "user-role")
		meta.RequestID = firstNonEmpty(md, "x-request-id", "request-id")
		meta.TraceID = firstNonEmpty(md, "x-trace-id", "trace-id", "x-b3-traceid")
		if meta.TraceID == "" {
			meta.TraceID = parseTraceIDFromTraceParent(firstNonEmpty(md, "traceparent"))
		}
		meta.SourceService = firstNonEmpty(md, "x-source-service", "x-service-name", "source-service", "x-app-id")
		meta.UserAgent = firstNonEmpty(md, "user-agent")
		meta.ClientIP = firstNonEmpty(md, "x-forwarded-for", "x-real-ip")
		meta.HasAuthorization = hasAnyValue(md, "authorization", "x-authorization")
		meta.MetadataKeys = metadataKeys(md)
		meta.Metadata = snapshotMetadata(md)
	}

	if meta.ClientIP == "" {
		meta.ClientIP = meta.PeerAddr
	}
	return meta
}

// logByCode 根据 gRPC 状态码选择访问日志级别。
func logByCode(code codes.Code, fields ...zap.Field) {
	switch code {
	case codes.OK:
		gintlogger.Info("grpc access", fields...)
	case codes.InvalidArgument, codes.NotFound, codes.AlreadyExists, codes.PermissionDenied,
		codes.Unauthenticated, codes.ResourceExhausted, codes.FailedPrecondition,
		codes.Aborted, codes.OutOfRange, codes.Canceled, codes.DeadlineExceeded:
		gintlogger.Warn("grpc access", fields...)
	default:
		gintlogger.Error("grpc access", fields...)
	}
}

// marshalPayload 序列化请求或响应载荷，并按长度上限截断。
func marshalPayload(v any, maxBytes int) string {
	if v == nil {
		return ""
	}

	var raw []byte
	var err error

	if msg, ok := v.(proto.Message); ok {
		raw, err = protojson.MarshalOptions{
			UseProtoNames:   true,
			EmitUnpopulated: false,
		}.Marshal(msg)
	} else {
		raw, err = json.Marshal(v)
	}
	if err != nil {
		return truncateString(`{"marshal_error":"`+safeString(err.Error())+`"}`, maxBytes)
	}

	return truncateString(string(raw), maxBytes)
}

// snapshotMetadata 复制 metadata 到普通 map，便于统一打日志。
func snapshotMetadata(md metadata.MD) map[string]string {
	if len(md) == 0 {
		return nil
	}

	out := make(map[string]string, len(md))
	for key, vals := range md {
		lowerKey := strings.ToLower(strings.TrimSpace(key))
		if lowerKey == "" {
			continue
		}
		out[lowerKey] = truncateString(strings.Join(vals, ","), 512)
	}
	return out
}

// metadataKeys 返回 metadata 中存在的 key 列表。
func metadataKeys(md metadata.MD) []string {
	if len(md) == 0 {
		return nil
	}

	keys := make([]string, 0, len(md))
	for key := range md {
		key = strings.ToLower(strings.TrimSpace(key))
		if key != "" {
			keys = append(keys, key)
		}
	}
	return keys
}

// firstNonEmpty 按顺序返回第一个非空 metadata 值。
func firstNonEmpty(md metadata.MD, keys ...string) string {
	for _, key := range keys {
		values := md.Get(key)
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value != "" {
				return value
			}
		}
	}
	return ""
}

// hasAnyValue 判断给定 metadata key 中是否至少有一个非空值。
func hasAnyValue(md metadata.MD, keys ...string) bool {
	return firstNonEmpty(md, keys...) != ""
}

// truncateString 按最大长度截断字符串，避免日志载荷过大。
func truncateString(value string, maxBytes int) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	return value[:maxBytes] + "...(truncated)"
}

// safeString 对错误字符串做最小转义，避免拼接 JSON 时破坏格式。
func safeString(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return replacer.Replace(strings.TrimSpace(value))
}

// parseTraceIDFromTraceParent 从 W3C traceparent 格式中提取 trace_id。
func parseTraceIDFromTraceParent(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	parts := strings.Split(value, "-")
	if len(parts) < 4 {
		return ""
	}

	traceID := strings.TrimSpace(parts[1])
	if len(traceID) != 32 {
		return ""
	}

	for _, ch := range traceID {
		if !strings.ContainsRune("0123456789abcdefABCDEF", ch) {
			return ""
		}
	}

	return strings.ToLower(traceID)
}
