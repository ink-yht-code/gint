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
//
// This file is derived from ginx (https://github.com/ecodeclub/ginx)
// Original Copyright by ecodeclub and contributors
// Modifications: Support for dual-token injection

package header

import (
	"github.com/ink-yht-code/gint/gint/gctx"
	"github.com/ink-yht-code/gint/gint/session"
	"strings"
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
