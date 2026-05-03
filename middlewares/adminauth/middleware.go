// Copyright 2025 ink-yht-code
//
// Proprietary License

// Package adminauth 提供管理接口鉴权中间件，支持静态 Token 和 IP 白名单两种方式。
package adminauth

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TokenMiddleware 静态 Bearer Token 鉴权。
// 请求头需携带 Authorization: Bearer <token>。
func TokenMiddleware(token string) gin.HandlerFunc {
	expected := "Bearer " + token
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") != expected {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":  401,
				"error": "unauthorized",
			})
			return
		}
		c.Next()
	}
}

// IPWhitelistMiddleware IP 白名单鉴权，支持 CIDR 和精确 IP。
// 例如：[]string{"127.0.0.1", "10.0.0.0/8"}
func IPWhitelistMiddleware(allowedCIDRs []string) gin.HandlerFunc {
	nets := parseCIDRs(allowedCIDRs)
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if !isAllowed(clientIP, nets) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":  403,
				"error": "forbidden",
			})
			return
		}
		c.Next()
	}
}

// CombinedMiddleware Token 或 IP 白名单，满足其一即可通过。
func CombinedMiddleware(token string, allowedCIDRs []string) gin.HandlerFunc {
	expected := "Bearer " + token
	nets := parseCIDRs(allowedCIDRs)
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") == expected {
			c.Next()
			return
		}
		if isAllowed(c.ClientIP(), nets) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"code":  401,
			"error": "unauthorized",
		})
	}
}

func parseCIDRs(cidrs []string) []*net.IPNet {
	var nets []*net.IPNet
	for _, cidr := range cidrs {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		// 纯 IP 补全为 /32 或 /128
		if !strings.Contains(cidr, "/") {
			if strings.Contains(cidr, ":") {
				cidr += "/128"
			} else {
				cidr += "/32"
			}
		}
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			nets = append(nets, ipNet)
		}
	}
	return nets
}

func isAllowed(clientIP string, nets []*net.IPNet) bool {
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
