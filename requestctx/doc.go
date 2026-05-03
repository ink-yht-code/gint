// Package requestctx 提供请求级上下文工具。
//
// 这个包专门用于在 context.Context 中传递链路字段，
// 例如 tenant_id、trace_id、request_id、user_id 等。
// 这样 service、repository、integration 等各层在记录业务日志时，
// 都可以复用同一套上下文字段，而不需要重复解析 metadata。
package requestctx
