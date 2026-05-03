// Copyright 2025 ink-yht-code
//
// Proprietary License

// Package timeout 提供请求超时控制中间件。
// 超时后自动返回 504 Gateway Timeout，并取消下游 context。
package timeout

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Middleware 创建超时中间件，超时后返回 504。
// d 为超时时长，建议网关场景设置 10-30s。
func Middleware(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		// 用 channel 等待 handler 完成或超时
		done := make(chan struct{}, 1)
		panicCh := make(chan any, 1)

		go func() {
			defer func() {
				if p := recover(); p != nil {
					panicCh <- p
				}
			}()
			c.Next()
			done <- struct{}{}
		}()

		select {
		case <-done:
			// 正常完成
		case p := <-panicCh:
			panic(p)
		case <-ctx.Done():
			c.Abort()
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"code":  504,
				"error": "gateway timeout",
			})
		}
	}
}

// PerRoute 为不同路由设置不同超时，通过 keyFunc 返回超时时长。
// 若 keyFunc 返回 0，则不限制超时。
func PerRoute(keyFunc func(c *gin.Context) time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		d := keyFunc(c)
		if d <= 0 {
			c.Next()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{}, 1)
		panicCh := make(chan any, 1)

		go func() {
			defer func() {
				if p := recover(); p != nil {
					panicCh <- p
				}
			}()
			c.Next()
			done <- struct{}{}
		}()

		select {
		case <-done:
		case p := <-panicCh:
			panic(p)
		case <-ctx.Done():
			c.Abort()
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"code":  504,
				"error": "gateway timeout",
			})
		}
	}
}
