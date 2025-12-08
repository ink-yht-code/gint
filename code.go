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
)

// CodeMessage 响应码对应的默认消息
// 注意：此 map 为只读，不要在运行时修改
var CodeMessage = map[int]string{
	CodeSuccess: "成功",
	CodeWarning: "警告",
	CodeError:   "错误",
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
