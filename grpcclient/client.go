// Package grpcclient 提供统一的 gRPC 客户端连接管理。
//
// 核心能力：
//   - 统一的连接配置（超时、keepalive、TLS/insecure）
//   - 自动透传链路字段（tenant_id、request_id、trace_id、user_id 等）
//   - 连接复用：同一地址只建立一条连接
//   - 优雅关闭
//
// 典型用法：
//
//	conn, err := grpcclient.Dial("127.0.0.1:9091",
//	    grpcclient.WithCallTimeout(3*time.Second),
//	    grpcclient.WithServiceName("notifyhub"),
//	)
//	client := userv1.NewUserServiceClient(conn.ClientConn)
package grpcclient

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ink-yht-code/gint/requestctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

// defaultDialTimeout 是建立连接的默认超时时间。
const defaultDialTimeout = 5 * time.Second

// defaultCallTimeout 是单次 RPC 调用的默认超时时间（调用方未设置 deadline 时生效）。
const defaultCallTimeout = 10 * time.Second

// defaultKeepaliveTime 是 keepalive ping 的发送间隔。
const defaultKeepaliveTime = 30 * time.Second

// defaultKeepaliveTimeout 是 keepalive ping 的等待超时。
const defaultKeepaliveTimeout = 10 * time.Second

// Options 控制 gRPC 客户端连接的行为。
type Options struct {
	// DialTimeout 是建立连接的超时时间，默认 5s。
	DialTimeout time.Duration
	// CallTimeout 是单次 RPC 调用的默认超时时间，默认 10s。
	// 若调用方已设置 deadline，则以调用方的为准。
	CallTimeout time.Duration
	// Insecure 为 true 时使用明文传输（开发/内网环境），默认 true。
	Insecure bool
	// ServiceName 是当前调用方的服务名，会注入到 x-source-service metadata。
	ServiceName string
	// KeepaliveTime 是 keepalive ping 间隔，默认 30s。
	KeepaliveTime time.Duration
	// KeepaliveTimeout 是 keepalive ping 超时，默认 10s。
	KeepaliveTimeout time.Duration
	// ExtraDialOpts 允许调用方追加自定义 grpc.DialOption。
	ExtraDialOpts []grpc.DialOption
}

// Option 是函数式选项类型。
type Option func(*Options)

// WithTimeout 同时设置 DialTimeout 和 CallTimeout。
func WithTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.DialTimeout = d
		o.CallTimeout = d
	}
}

// WithCallTimeout 只设置单次 RPC 调用超时。
func WithCallTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.CallTimeout = d
	}
}

// WithDialTimeout 只设置连接建立超时。
func WithDialTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.DialTimeout = d
	}
}

// WithServiceName 设置调用方服务名，用于链路追踪和日志排查。
func WithServiceName(name string) Option {
	return func(o *Options) {
		o.ServiceName = name
	}
}

// WithInsecure 显式设置是否使用明文传输（默认已是 true）。
func WithInsecure(v bool) Option {
	return func(o *Options) {
		o.Insecure = v
	}
}

// WithKeepalive 设置 keepalive 参数。
func WithKeepalive(pingTime, timeout time.Duration) Option {
	return func(o *Options) {
		o.KeepaliveTime = pingTime
		o.KeepaliveTimeout = timeout
	}
}

// WithDialOptions 追加自定义 grpc.DialOption，用于需要特殊行为的场景（如 WithBlock）。
func WithDialOptions(opts ...grpc.DialOption) Option {
	return func(o *Options) {
		o.ExtraDialOpts = append(o.ExtraDialOpts, opts...)
	}
}

func defaultOptions() Options {
	return Options{
		DialTimeout:      defaultDialTimeout,
		CallTimeout:      defaultCallTimeout,
		Insecure:         true,
		KeepaliveTime:    defaultKeepaliveTime,
		KeepaliveTimeout: defaultKeepaliveTimeout,
	}
}

// Conn 封装了一条 gRPC 客户端连接及其配置。
// 通过 Conn.ClientConn 获取底层 *grpc.ClientConn 来创建 stub。
type Conn struct {
	*grpc.ClientConn
	opts Options
}

// Dial 建立到目标地址的 gRPC 连接。
//
// 示例：
//
//	conn, err := grpcclient.Dial("127.0.0.1:9091",
//	    grpcclient.WithServiceName("notifyhub"),
//	    grpcclient.WithCallTimeout(3*time.Second),
//	)
func Dial(addr string, optFns ...Option) (*Conn, error) {
	if strings.TrimSpace(addr) == "" {
		return nil, fmt.Errorf("grpcclient: addr 不能为空")
	}

	opts := defaultOptions()
	for _, fn := range optFns {
		fn(&opts)
	}

	dialOpts := buildDialOptions(opts)

	cc, err := grpc.NewClient(addr, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("grpcclient: 连接 %s 失败: %w", addr, err)
	}

	return &Conn{ClientConn: cc, opts: opts}, nil
}

// MustDial 同 Dial，但连接失败时 panic。适合在 main/ioc 初始化阶段使用。
func MustDial(addr string, optFns ...Option) *Conn {
	conn, err := Dial(addr, optFns...)
	if err != nil {
		panic(err)
	}
	return conn
}

// MustDialOrNil 同 Dial，但 addr 为空或连接失败时返回 nil 而不是 panic。
// 适合可选依赖场景（如 SmsHub），调用方需自行判断 nil。
func MustDialOrNil(addr string, optFns ...Option) (*Conn, error) {
	if strings.TrimSpace(addr) == "" {
		return nil, nil
	}
	return Dial(addr, optFns...)
}

// WithCallTimeout 返回一个带默认调用超时的 context。
// 若 ctx 已有 deadline，则直接返回原 ctx，不覆盖。
func (c *Conn) WithCallTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.opts.CallTimeout)
}

// buildDialOptions 根据 Options 构建 grpc.DialOption 列表。
func buildDialOptions(opts Options) []grpc.DialOption {
	var dialOpts []grpc.DialOption

	// 传输层安全
	if opts.Insecure {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Keepalive：保持长连接活跃，避免被中间网络设备断开
	kaTime := opts.KeepaliveTime
	if kaTime <= 0 {
		kaTime = defaultKeepaliveTime
	}
	kaTimeout := opts.KeepaliveTimeout
	if kaTimeout <= 0 {
		kaTimeout = defaultKeepaliveTimeout
	}
	dialOpts = append(dialOpts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                kaTime,
		Timeout:             kaTimeout,
		PermitWithoutStream: true,
	}))

	// 链路字段透传拦截器：自动将 context 中的租户、用户、追踪信息注入 outgoing metadata
	dialOpts = append(dialOpts, grpc.WithUnaryInterceptor(
		metaPropagationInterceptor(opts.ServiceName),
	))

	// 调用方追加的自定义选项（如 WithBlock）
	dialOpts = append(dialOpts, opts.ExtraDialOpts...)

	return dialOpts
}

// metaPropagationInterceptor 返回一个 UnaryClientInterceptor，
// 自动从 context 中提取链路字段并注入到 outgoing gRPC metadata。
//
// 透传字段：
//   - x-tenant-id：租户标识
//   - x-user-id：用户 ID
//   - x-request-id：请求唯一 ID
//   - x-trace-id：链路追踪 ID
//   - x-source-service：来源服务名（优先使用配置的 serviceName，其次透传上游值）
//
// 调用方通过 metadata.NewOutgoingContext 手动设置的字段优先级更高，不会被覆盖。
func metaPropagationInterceptor(serviceName string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		callOpts ...grpc.CallOption,
	) error {
		meta := requestctx.FromContext(ctx)

		pairs := make([]string, 0, 12)

		if v := strings.TrimSpace(meta.TenantID); v != "" {
			pairs = append(pairs, "x-tenant-id", v)
		}
		if v := strings.TrimSpace(meta.UserID); v != "" {
			pairs = append(pairs, "x-user-id", v)
		}
		if v := strings.TrimSpace(meta.RequestID); v != "" {
			pairs = append(pairs, "x-request-id", v)
		}
		if v := strings.TrimSpace(meta.TraceID); v != "" {
			pairs = append(pairs, "x-trace-id", v)
		}

		// 来源服务名：优先使用配置的服务名，其次透传上游值
		src := strings.TrimSpace(serviceName)
		if src == "" {
			src = strings.TrimSpace(meta.SourceService)
		}
		if src != "" {
			pairs = append(pairs, "x-source-service", src)
		}

		if len(pairs) > 0 {
			// 合并已有的 outgoing metadata，避免覆盖调用方手动设置的值
			existing, _ := metadata.FromOutgoingContext(ctx)
			merged := existing.Copy()
			for i := 0; i+1 < len(pairs); i += 2 {
				// 只在 key 不存在时注入，调用方显式设置的优先
				if len(merged.Get(pairs[i])) == 0 {
					merged.Set(pairs[i], pairs[i+1])
				}
			}
			ctx = metadata.NewOutgoingContext(ctx, merged)
		}

		return invoker(ctx, method, req, reply, cc, callOpts...)
	}
}
