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

import "errors"

var (
	// ErrNoResponse 表示不需要返回响应
	// 当你已经手动处理了响应时，可以返回这个错误
	ErrNoResponse = errors.New("不需要返回响应")

	// ErrUnauthorized 表示未授权
	// 返回这个错误会自动返回 401 状态码
	ErrUnauthorized = errors.New("未授权")

	// ErrSessionNotFound 表示 Session 不存在
	ErrSessionNotFound = errors.New("会话不存在")

	// ErrSessionExpired 表示 Session 已过期
	ErrSessionExpired = errors.New("会话已过期")

	// ErrInvalidToken 表示无效的 Token
	ErrInvalidToken = errors.New("无效的令牌")
)
