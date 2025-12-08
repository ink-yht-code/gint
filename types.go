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
	"github.com/gin-gonic/gin"
	"github.com/ink-yht-code/gint/gctx"
)

// Handler 定义了路由处理器接口
// 用于组织和注册路由
type Handler interface {
	// PrivateRoutes 注册需要认证的路由
	PrivateRoutes(server *gin.Engine)
	// PublicRoutes 注册公开的路由
	PublicRoutes(server *gin.Engine)
}

// Result 统一的响应结构
type Result struct {
	Code int    `json:"code"` // 业务状态码，0 表示成功
	Msg  string `json:"msg"`  // 响应消息
	Data any    `json:"data"` // 响应数据
}

// PageData 用于返回分页查询的数据
type PageData[T any] struct {
	List  []T   `json:"list"`  // 数据列表
	Total int64 `json:"total"` // 总数
	Page  int   `json:"page"`  // 当前页码
	Size  int   `json:"size"`  // 每页大小
}

// PageRequest 通用的分页请求参数
type PageRequest struct {
	Page int `json:"page" form:"page"` // 页码，从 1 开始
	Size int `json:"size" form:"size"` // 每页大小
}

// Validate 验证分页参数
func (p *PageRequest) Validate() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Size < 1 {
		p.Size = 10
	}
	if p.Size > 100 {
		p.Size = 100
	}
}

// Offset 计算偏移量
func (p *PageRequest) Offset() int {
	return (p.Page - 1) * p.Size
}

type Context = gctx.Context
