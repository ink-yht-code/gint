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
//
// This file is derived from ginx (https://github.com/ecodeclub/ginx)
// Original Copyright by ecodeclub and contributors
// Modifications: Support for dual-token injection

package header

import (
	"strings"

	"github.com/ink-yht-code/gint/gctx"
	"github.com/ink-yht-code/gint/session"
)

var _ session.TokenCarrier = (*Carrier)(nil)

// Carrier 基于 HTTP Header 的 Token 载体
type Carrier struct {
	headerName string // Header 名称
}

// NewCarrier 创建 Header Token 载体
// 默认使用 "Authorization" 作为 Header 名称
func NewCarrier() *Carrier {
	return &Carrier{
		headerName: "Authorization",
	}
}

// NewCarrierWithHeader 创建自定义 Header 名称的 Token 载体
func NewCarrierWithHeader(headerName string) *Carrier {
	return &Carrier{
		headerName: headerName,
	}
}

// Inject 将 Token 注入到响应 Header 中
func (c *Carrier) Inject(ctx *gctx.Context, token string) {
	ctx.Context.Header(c.headerName, token)
}

// Extract 从请求 Header 中提取 Token
func (c *Carrier) Extract(ctx *gctx.Context) string {
	val := strings.TrimSpace(ctx.GetHeader(c.headerName))
	if val == "" {
		return ""
	}

	// 支持标准格式: Authorization: Bearer <token>
	// 同时兼容旧格式：直接传 token
	if len(val) >= 7 && strings.EqualFold(val[:6], "bearer") {
		rest := strings.TrimSpace(val[6:])
		if rest != "" {
			return rest
		}
	}

	return val
}

// Clear 清除 Token（设置为空）
func (c *Carrier) Clear(ctx *gctx.Context) {
	ctx.Context.Header(c.headerName, "")
}
