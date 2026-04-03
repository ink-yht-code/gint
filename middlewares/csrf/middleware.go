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

package csrf

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Config CSRF 中间件配置
type Config struct {
	// Secret 用于生成 CSRF token 的密钥
	Secret string

	// TokenLength CSRF token 长度（字节数）
	TokenLength int

	// TokenLookup 从请求中查找 token 的方式
	// 支持格式：
	// - "header:<name>" 从 header 中获取
	// - "form:<name>" 从表单中获取
	// - "query:<name>" 从查询参数中获取
	TokenLookup string

	// CookieName 存储 CSRF token 的 cookie 名称
	CookieName string

	// CookieDomain cookie 域名
	CookieDomain string

	// CookiePath cookie 路径
	CookiePath string

	// CookieHTTPOnly 是否设置 HttpOnly
	CookieHTTPOnly bool

	// CookieSecure 是否设置 Secure
	CookieSecure bool

	// CookieSameSite cookie SameSite 属性
	CookieSameSite http.SameSite

	// CookieMaxAge cookie 过期时间（秒）
	CookieMaxAge int

	// SkipFunc 跳过 CSRF 检查的函数
	SkipFunc func(c *gin.Context) bool

	// ErrorHandler 错误处理函数
	ErrorHandler func(c *gin.Context, err error)
}

// DefaultConfig 默认配置
func DefaultConfig() Config {
	return Config{
		TokenLength:    32,
		TokenLookup:    "header:X-CSRF-Token",
		CookieName:     "_csrf",
		CookiePath:     "/",
		CookieHTTPOnly: true,
		CookieSecure:   false,
		CookieSameSite: http.SameSiteLaxMode,
		CookieMaxAge:   86400, // 24 hours
		SkipFunc: func(c *gin.Context) bool {
			// 默认跳过 GET, HEAD, OPTIONS 请求
			method := c.Request.Method
			return method == "GET" || method == "HEAD" || method == "OPTIONS"
		},
		ErrorHandler: func(c *gin.Context, err error) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code": 403,
				"msg":  "CSRF token 验证失败",
			})
		},
	}
}

// New 创建 CSRF 中间件
func New(config Config) gin.HandlerFunc {
	// 设置默认值
	if config.Secret == "" {
		panic("CSRF secret cannot be empty")
	}
	if config.TokenLength <= 0 {
		config.TokenLength = 32
	}
	if config.TokenLookup == "" {
		config.TokenLookup = "header:X-CSRF-Token"
	}
	if config.CookieName == "" {
		config.CookieName = "_csrf"
	}
	if config.CookiePath == "" {
		config.CookiePath = "/"
	}
	if config.CookieMaxAge <= 0 {
		config.CookieMaxAge = 86400
	}
	if config.SkipFunc == nil {
		config.SkipFunc = func(c *gin.Context) bool {
			method := c.Request.Method
			return method == "GET" || method == "HEAD" || method == "OPTIONS"
		}
	}
	if config.ErrorHandler == nil {
		config.ErrorHandler = func(c *gin.Context, err error) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code": 403,
				"msg":  "CSRF token 验证失败",
			})
		}
	}

	// 解析 TokenLookup
	parts := strings.Split(config.TokenLookup, ":")
	if len(parts) != 2 {
		panic("Invalid TokenLookup format")
	}
	lookupMethod := parts[0]
	lookupKey := parts[1]

	return func(c *gin.Context) {
		// 检查是否跳过
		if config.SkipFunc(c) {
			// 即使跳过验证，也要设置 token 到 cookie 中
			token := generateToken(config.TokenLength, config.Secret)
			setTokenCookie(c, config, token)
			c.Set("csrf_token", token)
			c.Next()
			return
		}

		// 从 cookie 中获取存储的 token
		storedToken, err := c.Cookie(config.CookieName)
		if err != nil {
			// 没有 cookie，生成新的 token
			token := generateToken(config.TokenLength, config.Secret)
			setTokenCookie(c, config, token)
			config.ErrorHandler(c, fmt.Errorf("CSRF token not found in cookie"))
			return
		}

		// 从请求中获取提交的 token
		var submittedToken string
		switch lookupMethod {
		case "header":
			submittedToken = c.GetHeader(lookupKey)
		case "form":
			submittedToken = c.PostForm(lookupKey)
		case "query":
			submittedToken = c.Query(lookupKey)
		default:
			config.ErrorHandler(c, fmt.Errorf("unsupported token lookup method: %s", lookupMethod))
			return
		}

		// 验证 token
		if !validateToken(storedToken, submittedToken, config.Secret) {
			config.ErrorHandler(c, fmt.Errorf("CSRF token validation failed"))
			return
		}

		// 验证通过，继续处理请求
		c.Set("csrf_token", storedToken)
		c.Next()
	}
}

// Default 使用默认配置创建 CSRF 中间件
func Default(secret string) gin.HandlerFunc {
	config := DefaultConfig()
	config.Secret = secret
	return New(config)
}

// generateToken 生成随机 token
func generateToken(length int, secret string) string {
	nonce := make([]byte, length)
	if _, err := rand.Read(nonce); err != nil {
		panic(fmt.Sprintf("Failed to generate CSRF token: %v", err))
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(nonce)
	sig := mac.Sum(nil)

	return base64.URLEncoding.EncodeToString(nonce) + "." + base64.URLEncoding.EncodeToString(sig)
}

// setTokenCookie 设置 token 到 cookie
func setTokenCookie(c *gin.Context, config Config, token string) {
	c.SetCookie(
		config.CookieName,
		token,
		config.CookieMaxAge,
		config.CookiePath,
		config.CookieDomain,
		config.CookieSecure,
		config.CookieHTTPOnly,
	)

	// 设置 SameSite 属性
	if config.CookieSameSite != 0 {
		cookie := &http.Cookie{
			Name:     config.CookieName,
			Value:    token,
			MaxAge:   config.CookieMaxAge,
			Path:     config.CookiePath,
			Domain:   config.CookieDomain,
			Secure:   config.CookieSecure,
			HttpOnly: config.CookieHTTPOnly,
			SameSite: config.CookieSameSite,
		}
		http.SetCookie(c.Writer, cookie)
	}
}

// validateToken 验证 token
func validateToken(storedToken, submittedToken, secret string) bool {
	if storedToken == "" || submittedToken == "" {
		return false
	}

	// 先要求提交 token 与 cookie 中 token 一致（Double Submit Cookie 模式）
	// 使用恒定时间比较防止时序攻击
	if subtle.ConstantTimeCompare([]byte(storedToken), []byte(submittedToken)) != 1 {
		return false
	}

	// 再校验 token 签名，确保 token 确实由 secret 派生（避免 secret 形同虚设）
	parts := strings.SplitN(storedToken, ".", 2)
	if len(parts) != 2 {
		return false
	}
	nonce, err1 := base64.URLEncoding.DecodeString(parts[0])
	sig, err2 := base64.URLEncoding.DecodeString(parts[1])
	if err1 != nil || err2 != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(nonce)
	expected := mac.Sum(nil)
	return hmac.Equal(sig, expected)
}

// GetToken 从上下文中获取 CSRF token
func GetToken(c *gin.Context) string {
	if token, exists := c.Get("csrf_token"); exists {
		if tokenStr, ok := token.(string); ok {
			return tokenStr
		}
	}
	return ""
}

// Builder CSRF 中间件构建器
type Builder struct {
	config Config
}

// NewBuilder 创建构建器
func NewBuilder(secret string) *Builder {
	config := DefaultConfig()
	config.Secret = secret
	return &Builder{config: config}
}

// WithTokenLength 设置 token 长度
func (b *Builder) WithTokenLength(length int) *Builder {
	b.config.TokenLength = length
	return b
}

// WithTokenLookup 设置 token 查找方式
func (b *Builder) WithTokenLookup(lookup string) *Builder {
	b.config.TokenLookup = lookup
	return b
}

// WithCookieName 设置 cookie 名称
func (b *Builder) WithCookieName(name string) *Builder {
	b.config.CookieName = name
	return b
}

// WithCookieConfig 设置 cookie 配置
func (b *Builder) WithCookieConfig(domain, path string, httpOnly, secure bool, sameSite http.SameSite, maxAge int) *Builder {
	b.config.CookieDomain = domain
	b.config.CookiePath = path
	b.config.CookieHTTPOnly = httpOnly
	b.config.CookieSecure = secure
	b.config.CookieSameSite = sameSite
	b.config.CookieMaxAge = maxAge
	return b
}

// WithSkipFunc 设置跳过函数
func (b *Builder) WithSkipFunc(skipFunc func(c *gin.Context) bool) *Builder {
	b.config.SkipFunc = skipFunc
	return b
}

// WithErrorHandler 设置错误处理函数
func (b *Builder) WithErrorHandler(handler func(c *gin.Context, err error)) *Builder {
	b.config.ErrorHandler = handler
	return b
}

// Build 构建中间件
func (b *Builder) Build() gin.HandlerFunc {
	return New(b.config)
}
