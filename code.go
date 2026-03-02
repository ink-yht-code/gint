// Copyright 2025 ink-yht-code
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
)

// CodeMessage 响应码对应的默认消息
// 注意：此 map 为只读，不要在运行时修改
var CodeMessage = map[int]string{
	CodeSuccess:       "成功",
	CodeWarning:       "警告",
	CodeError:         "错误",
	CodeInvalidParam:  "参数错误",
	CodeInternalError: "系统繁忙",
	CodeUnauthorized:  "未授权",
	CodeForbidden:     "没有权限",
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
