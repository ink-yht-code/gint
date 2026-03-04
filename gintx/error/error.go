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

// Package error 提供错误映射功能
package error

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// BizError 业务错误接口
type BizError interface {
	BizCode() int
	BizMsg() string
	Error() string
}

// MapToHTTP 将错误映射到 HTTP 响应
func MapToHTTP(c *gin.Context, err error) {
	if err == nil {
		return
	}

	var biz BizError
	if errors.As(err, &biz) {
		status, resp := mapBizError(biz)
		c.JSON(status, resp)
		return
	}

	// 非业务错误
	c.JSON(http.StatusInternalServerError, gin.H{
		"code":    0,
		"message": "internal error",
	})
}

// mapBizError 映射业务错误
func mapBizError(biz BizError) (int, gin.H) {
	bizCode := biz.BizCode()
	suffix := bizCode % 10000

	var status int
	switch suffix {
	case 1: // InvalidParam
		status = http.StatusBadRequest
	case 2: // Unauthorized
		status = http.StatusUnauthorized
	case 3: // Forbidden
		status = http.StatusForbidden
	case 4: // NotFound
		status = http.StatusNotFound
	case 5: // Conflict
		status = http.StatusConflict
	default:
		status = http.StatusInternalServerError
	}

	return status, gin.H{
		"code":    bizCode,
		"message": biz.BizMsg(),
	}
}

// Handler 错误处理中间件
func Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 检查是否有错误
		if len(c.Errors) > 0 {
			err := c.Errors[0].Err
			MapToHTTP(c, err)
		}
	}
}
