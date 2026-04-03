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
