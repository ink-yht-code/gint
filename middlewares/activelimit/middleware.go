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
