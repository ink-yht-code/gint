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
	"context"
)

// Permission 权限定义
type Permission struct {
	Resource string // 资源路径，如 /api/users
	Action   string // 操作，如 GET, POST, DELETE, PUT
}

// PolicyProvider 策略提供者接口
// 引用项目需要实现此接口，提供策略数据
type PolicyProvider interface {
	// LoadPolicies 加载所有策略规则
	// 返回策略规则列表，格式：["p, admin, /api/users, GET", "p, user, /api/profile, GET"]
	// 或者返回空切片，表示从其他方式加载策略
	LoadPolicies(ctx context.Context) ([]string, error)

	// LoadRolePolicies 加载角色继承关系
	// 返回角色继承规则列表，格式：["g, alice, admin", "g, bob, user"]
	LoadRolePolicies(ctx context.Context) ([]string, error)
}

// UserRoleProvider 用户角色提供者接口
// 用于获取用户的角色信息
type UserRoleProvider interface {
	// GetUserRoles 获取用户的角色列表
	// userId: 用户 ID
	// 返回角色列表，如 ["admin", "user"]
	GetUserRoles(ctx context.Context, userId string) ([]string, error)
}

// ResourceMatcher 资源匹配器接口
// 用于将请求路径和方法转换为 Casbin 资源格式
type ResourceMatcher interface {
	// Match 将请求路径和方法转换为资源标识
	// path: 请求路径，如 /api/users/123
	// method: HTTP 方法，如 GET, POST
	// 返回资源标识，如 /api/users
	Match(path string, method string) string
}

// DefaultResourceMatcher 默认资源匹配器
// 直接使用路径和方法作为资源标识
type DefaultResourceMatcher struct{}

// Match 实现 ResourceMatcher 接口
func (m *DefaultResourceMatcher) Match(path string, method string) string {
	return path + ":" + method
}

// PathResourceMatcher 路径资源匹配器
// 只使用路径作为资源标识，忽略方法
type PathResourceMatcher struct{}

// Match 实现 ResourceMatcher 接口
func (m *PathResourceMatcher) Match(path string, method string) string {
	return path
}

// Options Casbin 配置选项
type Options struct {
	// ModelPath Casbin 模型文件路径
	// 如果为空，使用默认的 RBAC 模型
	ModelPath string

	// PolicyProvider 策略提供者
	// 必须实现，用于加载策略规则
	PolicyProvider PolicyProvider

	// UserRoleProvider 用户角色提供者
	// 可选，如果不提供，将从 Session 的 Claims 中获取角色
	UserRoleProvider UserRoleProvider

	// ResourceMatcher 资源匹配器
	// 可选，默认使用 DefaultResourceMatcher
	ResourceMatcher ResourceMatcher

	// AutoLoad 是否自动加载策略
	// 如果为 true，每次请求都会重新加载策略（不推荐，性能差）
	// 如果为 false，需要手动调用 RefreshPolicies() 刷新策略
	AutoLoad bool

	// CacheEnabled 是否启用策略缓存
	// 如果为 true，策略会被缓存，需要手动刷新
	CacheEnabled bool
}
