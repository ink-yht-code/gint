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

import (
	"testing"
)

func TestCodeConstants(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected int
	}{
		{"CodeSuccess", CodeSuccess, 0},
		{"CodeWarning", CodeWarning, 1},
		{"CodeError", CodeError, 2},
		{"CodeInvalidParam", CodeInvalidParam, 10000},
		{"CodeInternalError", CodeInternalError, 20000},
		{"CodeUnauthorized", CodeUnauthorized, 20001},
		{"CodeForbidden", CodeForbidden, 20003},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.expected {
				t.Errorf("%s = %d, want %d", tt.name, tt.code, tt.expected)
			}
		})
	}
}

func TestGetCodeMessage(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected string
	}{
		{"CodeSuccess", CodeSuccess, "成功"},
		{"CodeWarning", CodeWarning, "警告"},
		{"CodeError", CodeError, "错误"},
		{"CodeInvalidParam", CodeInvalidParam, "参数错误"},
		{"CodeInternalError", CodeInternalError, "系统繁忙"},
		{"CodeUnauthorized", CodeUnauthorized, "未授权"},
		{"CodeForbidden", CodeForbidden, "没有权限"},
		{"Unknown", 99999, "未知错误"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCodeMessage(tt.code); got != tt.expected {
				t.Errorf("GetCodeMessage(%d) = %s, want %s", tt.code, got, tt.expected)
			}
		})
	}
}

func TestSuccess(t *testing.T) {
	tests := []struct {
		name     string
		msg      string
		data     any
		expected Result
	}{
		{
			name: "with message and data",
			msg:  "操作成功",
			data: "test_data",
			expected: Result{
				Code: CodeSuccess,
				Msg:  "操作成功",
				Data: "test_data",
			},
		},
		{
			name: "empty message uses default",
			msg:  "",
			data: nil,
			expected: Result{
				Code: CodeSuccess,
				Msg:  "成功",
				Data: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Success(tt.msg, tt.data)
			if got.Code != tt.expected.Code || got.Msg != tt.expected.Msg || got.Data != tt.expected.Data {
				t.Errorf("Success() = %+v, want %+v", got, tt.expected)
			}
		})
	}
}

func TestWarning(t *testing.T) {
	tests := []struct {
		name     string
		msg      string
		data     any
		expected Result
	}{
		{
			name: "with message and data",
			msg:  "部分数据未更新",
			data: map[string]int{"count": 5},
			expected: Result{
				Code: CodeWarning,
				Msg:  "部分数据未更新",
				Data: map[string]int{"count": 5},
			},
		},
		{
			name: "empty message uses default",
			msg:  "",
			data: nil,
			expected: Result{
				Code: CodeWarning,
				Msg:  "警告",
				Data: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Warning(tt.msg, tt.data)
			if got.Code != tt.expected.Code || got.Msg != tt.expected.Msg {
				t.Errorf("Warning() = %+v, want %+v", got, tt.expected)
			}
		})
	}
}

func TestError(t *testing.T) {
	tests := []struct {
		name     string
		msg      string
		expected Result
	}{
		{
			name: "with message",
			msg:  "操作失败",
			expected: Result{
				Code: CodeError,
				Msg:  "操作失败",
				Data: nil,
			},
		},
		{
			name: "empty message uses default",
			msg:  "",
			expected: Result{
				Code: CodeError,
				Msg:  "错误",
				Data: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Error(tt.msg)
			if got.Code != tt.expected.Code || got.Msg != tt.expected.Msg || got.Data != nil {
				t.Errorf("Error() = %+v, want %+v", got, tt.expected)
			}
		})
	}
}

func TestErrorWithCode(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		msg      string
		expected Result
	}{
		{
			name: "with code and message",
			code: 10001,
			msg:  "用户名或密码错误",
			expected: Result{
				Code: 10001,
				Msg:  "用户名或密码错误",
				Data: nil,
			},
		},
		{
			name: "empty message uses default",
			code: CodeInvalidParam,
			msg:  "",
			expected: Result{
				Code: CodeInvalidParam,
				Msg:  "参数错误",
				Data: nil,
			},
		},
		{
			name: "unknown code uses default message",
			code: 99999,
			msg:  "",
			expected: Result{
				Code: 99999,
				Msg:  "未知错误",
				Data: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ErrorWithCode(tt.code, tt.msg)
			if got.Code != tt.expected.Code || got.Msg != tt.expected.Msg {
				t.Errorf("ErrorWithCode() = %+v, want %+v", got, tt.expected)
			}
		})
	}
}

func TestInvalidParam(t *testing.T) {
	tests := []struct {
		name     string
		msg      string
		expected Result
	}{
		{
			name: "with message",
			msg:  "用户名不能为空",
			expected: Result{
				Code: CodeInvalidParam,
				Msg:  "用户名不能为空",
				Data: nil,
			},
		},
		{
			name: "empty message uses default",
			msg:  "",
			expected: Result{
				Code: CodeInvalidParam,
				Msg:  "参数错误",
				Data: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InvalidParam(tt.msg)
			if got.Code != tt.expected.Code || got.Msg != tt.expected.Msg {
				t.Errorf("InvalidParam() = %+v, want %+v", got, tt.expected)
			}
		})
	}
}

func TestInternalError(t *testing.T) {
	got := InternalError()
	expected := Result{
		Code: CodeInternalError,
		Msg:  "系统繁忙",
		Data: nil,
	}

	if got.Code != expected.Code || got.Msg != expected.Msg || got.Data != nil {
		t.Errorf("InternalError() = %+v, want %+v", got, expected)
	}
}

func TestUnauthorized(t *testing.T) {
	got := Unauthorized()
	expected := Result{
		Code: CodeUnauthorized,
		Msg:  "未授权",
		Data: nil,
	}

	if got.Code != expected.Code || got.Msg != expected.Msg {
		t.Errorf("Unauthorized() = %+v, want %+v", got, expected)
	}
}

func TestForbidden(t *testing.T) {
	got := Forbidden()
	expected := Result{
		Code: CodeForbidden,
		Msg:  "没有权限",
		Data: nil,
	}

	if got.Code != expected.Code || got.Msg != expected.Msg {
		t.Errorf("Forbidden() = %+v, want %+v", got, expected)
	}
}
