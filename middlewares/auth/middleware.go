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

package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ink-yht-code/gint/gctx"
	"github.com/ink-yht-code/gint/jwt"
	"github.com/ink-yht-code/gint/session"
)

// TokenExtractor 从请求中提取 Token 的函数
type TokenExtractor func(c *gin.Context) string

// HeaderExtractor 从 Header 提取 Token
func HeaderExtractor(headerName string) TokenExtractor {
	return func(c *gin.Context) string {
		return c.GetHeader(headerName)
	}
}

// BearerExtractor 从 Authorization: Bearer <token> 提取
func BearerExtractor() TokenExtractor {
	return func(c *gin.Context) string {
		auth := c.GetHeader("Authorization")
		if len(auth) > 7 && auth[:7] == "Bearer " {
			return auth[7:]
		}
		return ""
	}
}

// CookieExtractor 从 Cookie 提取
func CookieExtractor(cookieName string) TokenExtractor {
	return func(c *gin.Context) string {
		token, _ := c.Cookie(cookieName)
		return token
	}
}

// Config 认证中间件配置
type Config struct {
	// JWTManager JWT 管理器
	JWTManager jwt.Manager
	// TokenExtractor Token 提取器，默认从 Authorization: Bearer 提取
	TokenExtractor TokenExtractor
	// SessionProvider Session 提供者（可选，不设置则使用全局默认）
	SessionProvider session.Provider
}

func respondMisconfigured(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
		"code": 50001,
		"msg":  "认证中间件配置错误",
	})
}

// Middleware 创建认证中间件
// 流程：验证 JWT → 获取 Session → 构建 UserContext → 注入到 ctx
func Middleware(cfg Config) gin.HandlerFunc {
	extractor := cfg.TokenExtractor
	if extractor == nil {
		extractor = BearerExtractor()
	}

	return func(c *gin.Context) {
		if cfg.JWTManager == nil {
			respondMisconfigured(c)
			return
		}
		ctx := &gctx.Context{Context: c}

		// 1. 提取 Token
		token := extractor(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 20001,
				"msg":  "未提供认证令牌",
			})
			return
		}

		// 2. 验证 JWT
		claims, err := cfg.JWTManager.VerifyToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 20001,
				"msg":  "令牌无效或已过期",
			})
			return
		}

		// 3. 设置 Session Provider（如果配置了）
		if cfg.SessionProvider != nil {
			session.SetProvider(ctx, cfg.SessionProvider)
		}

		// 4. 获取 Session（从 JWT claims 中的 SSID）
		sess, err := session.Get(ctx)
		if err != nil {
			// Session 不存在，但 JWT 有效，创建基本 UserContext
			uc := &gctx.UserContext{
				UserId: claims.UserId,
			}
			ctx.SetUserContext(uc)
			ctx.SetUserId(claims.UserId)
			c.Next()
			return
		}

		// 5. 从 Session 构建 UserContext
		uc, ucErr := sess.UserContext(c.Request.Context())
		if ucErr == nil && uc != nil {
			ctx.SetUserContext(uc)
		} else {
			// 即使获取详情失败，也设置基本的 userId
			ctx.SetUserId(claims.UserId)
		}

		c.Next()
	}
}

// GatewayMiddleware 网关认证中间件
// 验证 JWT 后，将 UserContext 信息注入到请求头，传递给下游服务
func GatewayMiddleware(cfg Config) gin.HandlerFunc {
	extractor := cfg.TokenExtractor
	if extractor == nil {
		extractor = BearerExtractor()
	}

	return func(c *gin.Context) {
		if cfg.JWTManager == nil {
			respondMisconfigured(c)
			return
		}
		ctx := &gctx.Context{Context: c}

		// 1. 提取 Token
		token := extractor(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 20001,
				"msg":  "未提供认证令牌",
			})
			return
		}

		// 2. 验证 JWT
		claims, err := cfg.JWTManager.VerifyToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 20001,
				"msg":  "令牌无效或已过期",
			})
			return
		}

		// 3. 设置 Session Provider（如果配置了）
		if cfg.SessionProvider != nil {
			session.SetProvider(ctx, cfg.SessionProvider)
		}

		// 4. 获取 Session 并构建 UserContext
		var uc *gctx.UserContext
		sess, err := session.Get(ctx)
		if err == nil {
			uc, _ = sess.UserContext(c.Request.Context())
		}

		if uc == nil {
			uc = &gctx.UserContext{UserId: claims.UserId}
		}

		// 5. 注入到 ctx
		ctx.SetUserContext(uc)

		// 6. 注入 Header，供网关后续转发链路使用
		c.Request.Header.Set("X-User-ID", uc.UserId)
		c.Writer.Header().Set("X-User-ID", uc.UserId)
		if uc.Role != "" {
			c.Request.Header.Set("X-User-Role", uc.Role)
			c.Writer.Header().Set("X-User-Role", uc.Role)
		}
		if uc.TenantId != "" {
			c.Request.Header.Set("X-Tenant-ID", uc.TenantId)
			c.Writer.Header().Set("X-Tenant-ID", uc.TenantId)
		}
		if uc.Username != "" {
			c.Request.Header.Set("X-Username", uc.Username)
			c.Writer.Header().Set("X-Username", uc.Username)
		}

		c.Next()
	}
}

// OptionalMiddleware 可选认证中间件
// 不强制要求认证，有 Token 则验证，无 Token 则跳过
func OptionalMiddleware(cfg Config) gin.HandlerFunc {
	extractor := cfg.TokenExtractor
	if extractor == nil {
		extractor = BearerExtractor()
	}

	return func(c *gin.Context) {
		if cfg.JWTManager == nil {
			c.Next()
			return
		}
		ctx := &gctx.Context{Context: c}

		token := extractor(c)
		if token == "" {
			c.Next()
			return
		}

		claims, err := cfg.JWTManager.VerifyToken(token)
		if err != nil {
			c.Next()
			return
		}

		if cfg.SessionProvider != nil {
			session.SetProvider(ctx, cfg.SessionProvider)
		}

		sess, err := session.Get(ctx)
		if err == nil {
			uc, ucErr := sess.UserContext(c.Request.Context())
			if ucErr == nil && uc != nil {
				ctx.SetUserContext(uc)
			}
		} else {
			ctx.SetUserId(claims.UserId)
		}

		c.Next()
	}
}
