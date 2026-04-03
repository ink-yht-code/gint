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

package activelimit

import (
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

// Builder 活跃连接限制中间件构建器
type Builder struct {
	maxActive int64 // 最大活跃连接数
}

// NewBuilder 创建活跃连接限制中间件构建器
// maxActive: 最大允许的同时活跃连接数
func NewBuilder(maxActive int64) *Builder {
	return &Builder{
		maxActive: maxActive,
	}
}

// Build 构建中间件
func (b *Builder) Build() gin.HandlerFunc {
	var currentActive int64

	return func(c *gin.Context) {
		// 增加活跃连接计数
		current := atomic.AddInt64(&currentActive, 1)

		// 请求结束后减少计数
		defer func() {
			atomic.AddInt64(&currentActive, -1)
		}()

		// 检查是否超过限制
		if current > b.maxActive {
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}

		c.Next()
	}
}
