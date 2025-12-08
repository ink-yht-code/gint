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
//
// This file is derived from ginx (https://github.com/ecodeclub/ginx)
// Original Copyright by ecodeclub and contributors
// Modifications: Fixed MaxAge type conversion

package cors

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Config CORS 配置
type Config struct {
	// AllowOrigins 允许的源列表，如 ["http://localhost:3000", "https://example.com"]
	// 使用 "*" 表示允许所有源（不推荐用于生产环境）
	AllowOrigins []string

	// AllowMethods 允许的 HTTP 方法
	AllowMethods []string

	// AllowHeaders 允许的请求头
	AllowHeaders []string

	// ExposeHeaders 暴露给客户端的响应头
	ExposeHeaders []string

	// AllowCredentials 是否允许携带凭证（Cookie、HTTP 认证等）
	AllowCredentials bool

	// MaxAge 预检请求的缓存时间（秒）
	MaxAge int
}

// DefaultConfig 返回默认的 CORS 配置
func DefaultConfig() Config {
	return Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{},
		AllowCredentials: false,
		MaxAge:           86400, // 24 小时
	}
}

// New 创建 CORS 中间件
func New(config Config) gin.HandlerFunc {
	// 如果没有配置，使用默认配置
	if len(config.AllowOrigins) == 0 {
		config = DefaultConfig()
	}

	// 预处理配置
	allowAllOrigins := false
	for _, origin := range config.AllowOrigins {
		if origin == "*" {
			allowAllOrigins = true
			break
		}
	}

	allowMethodsStr := strings.Join(config.AllowMethods, ", ")
	allowHeadersStr := strings.Join(config.AllowHeaders, ", ")
	exposeHeadersStr := strings.Join(config.ExposeHeaders, ", ")

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 设置 Access-Control-Allow-Origin
		if allowAllOrigins {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin != "" {
			// 检查 origin 是否在允许列表中
			for _, allowedOrigin := range config.AllowOrigins {
				if origin == allowedOrigin {
					c.Header("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		// 设置其他 CORS 头
		if len(config.AllowMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", allowMethodsStr)
		}

		if len(config.AllowHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", allowHeadersStr)
		}

		if len(config.ExposeHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", exposeHeadersStr)
		}

		if config.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			if config.MaxAge > 0 {
				c.Header("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Default 返回使用默认配置的 CORS 中间件
func Default() gin.HandlerFunc {
	return New(DefaultConfig())
}
