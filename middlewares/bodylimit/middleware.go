// Copyright 2025 ink-yht-code
//
// Proprietary License

// Package bodylimit 提供请求体大小限制中间件，防止超大 body 打爆内存。
package bodylimit

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Middleware 限制请求体最大字节数，超出返回 413。
// maxBytes 单位为字节，例如 4<<20 表示 4MB。
func Middleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"code":  413,
				"error": "request body too large",
			})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
