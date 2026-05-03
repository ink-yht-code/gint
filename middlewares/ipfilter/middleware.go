// Copyright 2025 ink-yht-code
//
// Proprietary License

// Package ipfilter 提供基于 IP 的黑白名单中间件，支持精确 IP 和 CIDR。
package ipfilter

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Mode 过滤模式
type Mode int

const (
	// Whitelist 白名单模式：只允许列表中的 IP
	Whitelist Mode = iota
	// Blacklist 黑名单模式：拒绝列表中的 IP
	Blacklist
)

// Config IP 过滤配置
type Config struct {
	Mode Mode
	// IPs 精确 IP 或 CIDR 列表，例如 ["192.168.1.1", "10.0.0.0/8"]
	IPs []string
	// TrustProxy 是否信任 X-Forwarded-For / X-Real-IP 头
	TrustProxy bool
}

// Builder IP 过滤中间件构建器
type Builder struct {
	cfg  Config
	nets []*net.IPNet
	ips  map[string]struct{}
}

// NewBuilder 创建 IP 过滤构建器
func NewBuilder(cfg Config) *Builder {
	b := &Builder{
		cfg: cfg,
		ips: make(map[string]struct{}),
	}
	for _, raw := range cfg.IPs {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if strings.Contains(raw, "/") {
			_, ipNet, err := net.ParseCIDR(raw)
			if err == nil {
				b.nets = append(b.nets, ipNet)
			}
		} else {
			b.ips[raw] = struct{}{}
		}
	}
	return b
}

// Build 构建中间件
func (b *Builder) Build() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := b.getClientIP(c)
		matched := b.isMatch(clientIP)

		switch b.cfg.Mode {
		case Whitelist:
			if !matched {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"code":  403,
					"error": "ip not allowed",
				})
				return
			}
		case Blacklist:
			if matched {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"code":  403,
					"error": "ip blocked",
				})
				return
			}
		}
		c.Next()
	}
}

func (b *Builder) getClientIP(c *gin.Context) string {
	if b.cfg.TrustProxy {
		if ip := c.GetHeader("X-Real-IP"); ip != "" {
			return strings.TrimSpace(ip)
		}
		if forwarded := c.GetHeader("X-Forwarded-For"); forwarded != "" {
			parts := strings.Split(forwarded, ",")
			return strings.TrimSpace(parts[0])
		}
	}
	return c.ClientIP()
}

func (b *Builder) isMatch(clientIP string) bool {
	if _, ok := b.ips[clientIP]; ok {
		return true
	}
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}
	for _, n := range b.nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// NewWhitelist 快速创建白名单中间件
func NewWhitelist(ips []string, trustProxy ...bool) gin.HandlerFunc {
	trust := len(trustProxy) > 0 && trustProxy[0]
	return NewBuilder(Config{Mode: Whitelist, IPs: ips, TrustProxy: trust}).Build()
}

// NewBlacklist 快速创建黑名单中间件
func NewBlacklist(ips []string, trustProxy ...bool) gin.HandlerFunc {
	trust := len(trustProxy) > 0 && trustProxy[0]
	return NewBuilder(Config{Mode: Blacklist, IPs: ips, TrustProxy: trust}).Build()
}
