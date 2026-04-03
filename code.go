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

package gint

// 统一的响应码定义
const (
	// CodeSuccess 成功
	CodeSuccess = 0

	// CodeWarning 警告
	// 请求处理成功，但有需要注意的信息
	CodeWarning = 1

	// CodeError 错误
	// 请求处理失败
	CodeError = 2

	// CodeInvalidParam 参数错误
	// 用于请求参数不符合接口契约（通常对应 HTTP 400）
	CodeInvalidParam = 10000

	// CodeInternalError 系统错误
	// 用于系统异常（通常对应 HTTP 500），不应将内部错误信息直接暴露给客户端
	CodeInternalError = 20000

	// CodeUnauthorized 未授权
	// 用于未登录或 Token 无效（通常对应 HTTP 401）
	CodeUnauthorized = 20001

	// CodeForbidden 禁止访问
	// 用于已登录但无权限（通常对应 HTTP 403）
	CodeForbidden = 20003

	// CodeNotFound 资源不存在
	// 用于请求的资源不存在（通常对应 HTTP 404）
	CodeNotFound = 20004

	// CodeConflict 资源冲突
	// 用于资源已存在或状态冲突（通常对应 HTTP 409）
	CodeConflict = 20005

	// CodeTooManyRequests 请求过于频繁
	// 用于触发限流（通常对应 HTTP 429）
	CodeTooManyRequests = 20006

	// CodeServiceUnavailable 服务不可用
	// 用于服务暂时不可用（通常对应 HTTP 503）
	CodeServiceUnavailable = 20007
)

// CodeMessage 响应码对应的默认消息
// 注意：此 map 为只读，不要在运行时修改
var CodeMessage = map[int]string{
	CodeSuccess:            "成功",
	CodeWarning:            "警告",
	CodeError:              "错误",
	CodeInvalidParam:       "参数错误",
	CodeInternalError:      "系统繁忙",
	CodeUnauthorized:       "未授权",
	CodeForbidden:          "没有权限",
	CodeNotFound:           "资源不存在",
	CodeConflict:           "资源冲突",
	CodeTooManyRequests:    "请求过于频繁",
	CodeServiceUnavailable: "服务不可用",
}

// GetCodeMessage 获取响应码对应的默认消息
func GetCodeMessage(code int) string {
	if msg, ok := CodeMessage[code]; ok {
		return msg
	}

	// 根据范围返回默认消息
	switch {
	case code == 0:
		return "成功"
	case code == 1:
		return "警告"
	case code == 2:
		return "错误"
	default:
		return "未知错误"
	}
}

// Success 创建成功响应
func Success(msg string, data any) Result {
	if msg == "" {
		msg = "成功"
	}
	return Result{
		Code: CodeSuccess,
		Msg:  msg,
		Data: data,
	}
}

// Warning 创建警告响应
func Warning(msg string, data any) Result {
	if msg == "" {
		msg = "警告"
	}
	return Result{
		Code: CodeWarning,
		Msg:  msg,
		Data: data,
	}
}

// Error 创建错误响应
func Error(msg string) Result {
	if msg == "" {
		msg = "错误"
	}
	return Result{
		Code: CodeError,
		Msg:  msg,
		Data: nil,
	}
}

// ErrorWithCode 创建带自定义错误码的响应
func ErrorWithCode(code int, msg string) Result {
	if msg == "" {
		msg = GetCodeMessage(code)
	}
	return Result{
		Code: code,
		Msg:  msg,
		Data: nil,
	}
}

// InvalidParam 创建参数错误响应
func InvalidParam(msg string) Result {
	if msg == "" {
		msg = GetCodeMessage(CodeInvalidParam)
	}
	return Result{Code: CodeInvalidParam, Msg: msg, Data: nil}
}

// InternalError 创建系统错误响应
// 注意：该响应用于对外返回统一文案，内部错误细节应记录在日志中
func InternalError() Result {
	return Result{Code: CodeInternalError, Msg: GetCodeMessage(CodeInternalError), Data: nil}
}

// Unauthorized 创建未授权响应
func Unauthorized() Result {
	return Result{Code: CodeUnauthorized, Msg: GetCodeMessage(CodeUnauthorized), Data: nil}
}

// Forbidden 创建禁止访问响应
func Forbidden() Result {
	return Result{Code: CodeForbidden, Msg: GetCodeMessage(CodeForbidden), Data: nil}
}

// NotFound 创建资源不存在响应
func NotFound(msg string) Result {
	if msg == "" {
		msg = GetCodeMessage(CodeNotFound)
	}
	return Result{Code: CodeNotFound, Msg: msg, Data: nil}
}

// Conflict 创建资源冲突响应
func Conflict(msg string) Result {
	if msg == "" {
		msg = GetCodeMessage(CodeConflict)
	}
	return Result{Code: CodeConflict, Msg: msg, Data: nil}
}

// TooManyRequests 创建请求过于频繁响应
func TooManyRequests() Result {
	return Result{Code: CodeTooManyRequests, Msg: GetCodeMessage(CodeTooManyRequests), Data: nil}
}

// ServiceUnavailable 创建服务不可用响应
func ServiceUnavailable() Result {
	return Result{Code: CodeServiceUnavailable, Msg: GetCodeMessage(CodeServiceUnavailable), Data: nil}
}
