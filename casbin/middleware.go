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

package casbin

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ink-yht-code/gint"
	"github.com/ink-yht-code/gint/gctx"
	"github.com/ink-yht-code/gint/session"
)

// Builder Casbin 中间件构建器
type Builder struct {
	manager *Manager
	opts    Options
}

// NewBuilder 创建 Casbin 中间件构建器
func NewBuilder(manager *Manager) *Builder {
	return &Builder{
		manager: manager,
		opts:    manager.opts,
	}
}

// Build 构建中间件
func (b *Builder) Build() gin.HandlerFunc {
	// 获取资源匹配器
	matcher := b.opts.ResourceMatcher
	if matcher == nil {
		matcher = &DefaultResourceMatcher{}
	}

	return func(c *gin.Context) {
		ctx := &gctx.Context{Context: c}

		// 如果启用了自动加载，每次请求都重新加载策略（不推荐）
		if b.opts.AutoLoad {
			if err := b.manager.LoadPolicies(ctx); err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gint.InternalError())
				return
			}
		}

		// 获取 Session（必须已通过 gint.S() 或 gint.BS() 验证）
		sess, err := session.Get(ctx)
		if err != nil {
			// 如果没有 Session，说明未登录，返回 401
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 获取用户 ID
		userId := sess.Claims().UserId
		if userId == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 获取用户角色
		var roles []string
		if b.opts.UserRoleProvider != nil {
			// 从 UserRoleProvider 获取角色
			roles, err = b.opts.UserRoleProvider.GetUserRoles(ctx, userId)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gint.InternalError())
				return
			}
		} else {
			// 从 Session Claims 中获取角色
			if roleStr, ok := sess.Claims().Data["roles"]; ok {
				roles = parseRoles(roleStr)
			}
		}

		// 如果没有角色，尝试直接使用用户 ID
		if len(roles) == 0 {
			roles = []string{userId}
		}

		// 获取资源标识
		resource := matcher.Match(c.Request.URL.Path, c.Request.Method)
		action := c.Request.Method

		// 检查权限（检查用户是否有权限访问该资源）
		allowed := false
		for _, role := range roles {
			ok, err := b.manager.Enforce(role, resource, action)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gint.InternalError())
				return
			}
			if ok {
				allowed = true
				break
			}
		}

		// 如果所有角色都没有权限，也尝试直接用用户 ID 检查
		if !allowed {
			ok, err := b.manager.Enforce(userId, resource, action)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gint.InternalError())
				return
			}
			allowed = ok
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gint.Forbidden())
			return
		}

		// 权限检查通过，继续处理请求
		c.Next()
	}
}

// parseRoles 解析角色字符串
// 支持逗号分隔的字符串，如 "admin,user" 或 "admin, user"
func parseRoles(roleStr interface{}) []string {
	var roles []string

	switch v := roleStr.(type) {
	case string:
		if v == "" {
			return roles
		}
		parts := strings.Split(v, ",")
		for _, part := range parts {
			role := strings.TrimSpace(part)
			if role != "" {
				roles = append(roles, role)
			}
		}
	case []string:
		roles = v
	}

	return roles
}

// RequirePermission 创建需要特定权限的中间件
// 用于在路由中直接指定需要的权限
func RequirePermission(manager *Manager, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := &gctx.Context{Context: c}

		// 获取 Session
		sess, err := session.Get(ctx)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 获取用户 ID
		userId := sess.Claims().UserId
		if userId == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 检查权限
		allowed, err := manager.Enforce(userId, resource, action)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gint.InternalError())
			return
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gint.Forbidden())
			return
		}

		c.Next()
	}
}
