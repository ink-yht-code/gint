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

// TokenExtractor 从请求中提取 Token。
type TokenExtractor func(c *gin.Context) string

// HeaderExtractor 从 Header 提取 Token。
func HeaderExtractor(headerName string) TokenExtractor {
	return func(c *gin.Context) string {
		return c.GetHeader(headerName)
	}
}

// BearerExtractor 从 Authorization: Bearer <token> 提取 Token。
func BearerExtractor() TokenExtractor {
	return func(c *gin.Context) string {
		auth := c.GetHeader("Authorization")
		if len(auth) > 7 && auth[:7] == "Bearer " {
			return auth[7:]
		}
		return ""
	}
}

// CookieExtractor 从 Cookie 提取 Token。
func CookieExtractor(cookieName string) TokenExtractor {
	return func(c *gin.Context) string {
		token, _ := c.Cookie(cookieName)
		return token
	}
}

// Config 是认证中间件配置。
type Config struct {
	JWTManager      jwt.Manager
	TokenExtractor  TokenExtractor
	SessionProvider session.Provider
}

func respondMisconfigured(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
		"code": 50001,
		"msg":  "认证中间件配置错误",
	})
}

func unauthorized(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"code": 20001,
		"msg":  msg,
	})
}

func buildUserContextFromClaims(claims *jwt.Claims) *gctx.UserContext {
	if claims == nil {
		return &gctx.UserContext{}
	}

	uc := &gctx.UserContext{
		UserId:   claims.UserId,
		Username: claims.Username,
		Role:     claims.Role,
		TenantId: claims.TenantId,
		Email:    claims.Email,
		Extra:    map[string]string{},
	}

	for key, value := range claims.Data {
		switch key {
		case "tenant_id":
			if uc.TenantId == "" {
				uc.TenantId = value
			}
		case "role":
			if uc.Role == "" {
				uc.Role = value
			}
		case "username":
			if uc.Username == "" {
				uc.Username = value
			}
		case "email":
			if uc.Email == "" {
				uc.Email = value
			}
		default:
			uc.Extra[key] = value
		}
	}

	return uc
}

func ensureUserContext(ctx *gctx.Context, uc *gctx.UserContext, claims *jwt.Claims) *gctx.UserContext {
	claimsUC := buildUserContextFromClaims(claims)
	if uc == nil {
		ctx.SetUserContext(claimsUC)
		ctx.SetUserId(claimsUC.UserId)
		return claimsUC
	}

	if uc.UserId == "" {
		uc.UserId = claimsUC.UserId
	}
	if uc.Username == "" {
		uc.Username = claimsUC.Username
	}
	if uc.Role == "" {
		uc.Role = claimsUC.Role
	}
	if uc.TenantId == "" {
		uc.TenantId = claimsUC.TenantId
	}
	if uc.Email == "" {
		uc.Email = claimsUC.Email
	}
	if uc.Extra == nil {
		uc.Extra = map[string]string{}
	}
	for key, value := range claimsUC.Extra {
		if _, ok := uc.Extra[key]; !ok {
			uc.Extra[key] = value
		}
	}

	ctx.SetUserContext(uc)
	ctx.SetUserId(uc.UserId)
	return uc
}

// Middleware 创建认证中间件。
// 流程：验证 JWT -> 获取 Session -> 构建 UserContext -> 注入到 ctx。
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

		token := extractor(c)
		if token == "" {
			unauthorized(c, "未提供认证令牌")
			return
		}

		claims, err := cfg.JWTManager.VerifyToken(token)
		if err != nil {
			unauthorized(c, "令牌无效或已过期")
			return
		}

		if cfg.SessionProvider != nil {
			session.SetProvider(ctx, cfg.SessionProvider)
		}

		sess, err := session.Get(ctx)
		if err != nil {
			ensureUserContext(ctx, nil, claims)
			c.Next()
			return
		}

		uc, ucErr := sess.UserContext(c.Request.Context())
		if ucErr != nil {
			ensureUserContext(ctx, nil, claims)
			c.Next()
			return
		}

		ensureUserContext(ctx, uc, claims)
		c.Next()
	}
}

// GatewayMiddleware 创建网关认证中间件。
// 验证 JWT 后，将 UserContext 信息注入到请求头，传递给下游服务。
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

		token := extractor(c)
		if token == "" {
			unauthorized(c, "未提供认证令牌")
			return
		}

		claims, err := cfg.JWTManager.VerifyToken(token)
		if err != nil {
			unauthorized(c, "令牌无效或已过期")
			return
		}

		if cfg.SessionProvider != nil {
			session.SetProvider(ctx, cfg.SessionProvider)
		}

		var uc *gctx.UserContext
		sess, err := session.Get(ctx)
		if err == nil {
			uc, _ = sess.UserContext(c.Request.Context())
		}

		uc = ensureUserContext(ctx, uc, claims)

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
		if uc.Email != "" {
			c.Request.Header.Set("X-User-Email", uc.Email)
			c.Writer.Header().Set("X-User-Email", uc.Email)
		}

		c.Next()
	}
}

// OptionalMiddleware 创建可选认证中间件。
// 不强制要求认证，有 Token 则验证，无 Token 则跳过。
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
			if ucErr == nil {
				ensureUserContext(ctx, uc, claims)
				c.Next()
				return
			}
		}

		ensureUserContext(ctx, nil, claims)
		c.Next()
	}
}
