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
	"fmt"
	"strings"
	"sync"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
)

// Manager Casbin 管理器
type Manager struct {
	enforcer     *casbin.Enforcer
	opts         Options
	adapter      persist.Adapter
	mu           sync.RWMutex
	cacheEnabled bool
}

// NewManager 创建 Casbin 管理器
func NewManager(opts Options) (*Manager, error) {
	if opts.PolicyProvider == nil {
		return nil, fmt.Errorf("PolicyProvider 不能为空")
	}

	// 加载模型
	var m model.Model
	var err error
	if opts.ModelPath != "" {
		m, err = model.NewModelFromFile(opts.ModelPath)
		if err != nil {
			return nil, fmt.Errorf("加载模型文件失败: %w", err)
		}
	} else {
		// 使用默认的 RBAC 模型
		m, err = getDefaultModel()
		if err != nil {
			return nil, fmt.Errorf("创建默认模型失败: %w", err)
		}
	}

	// 创建内存适配器
	adapter := NewMemoryAdapter()

	// 创建 Enforcer
	enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("创建 Enforcer 失败: %w", err)
	}

	manager := &Manager{
		enforcer:     enforcer,
		opts:         opts,
		adapter:      adapter,
		cacheEnabled: opts.CacheEnabled,
	}

	// 加载策略
	if err := manager.LoadPolicies(context.Background()); err != nil {
		return nil, fmt.Errorf("加载策略失败: %w", err)
	}

	return manager, nil
}

// LoadPolicies 加载策略
func (m *Manager) LoadPolicies(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 加载策略规则
	policies, err := m.opts.PolicyProvider.LoadPolicies(ctx)
	if err != nil {
		return fmt.Errorf("加载策略规则失败: %w", err)
	}

	// 加载角色继承关系
	rolePolicies, err := m.opts.PolicyProvider.LoadRolePolicies(ctx)
	if err != nil {
		return fmt.Errorf("加载角色策略失败: %w", err)
	}

	// 清空现有策略
	m.enforcer.ClearPolicy()

	// 添加策略规则
	for _, policy := range policies {
		if policy != "" {
			// 解析策略字符串，格式：p, sub, obj, act
			parts := parsePolicyString(policy)
			if len(parts) >= 4 && parts[0] == "p" {
				sub, obj, act := parts[1], parts[2], parts[3]
				_, err := m.enforcer.AddPolicy(sub, obj, act)
				if err != nil {
					return fmt.Errorf("添加策略失败: %w", err)
				}
			}
		}
	}

	// 添加角色继承关系
	for _, rolePolicy := range rolePolicies {
		if rolePolicy != "" {
			// 解析角色策略字符串，格式：g, user, role
			parts := parsePolicyString(rolePolicy)
			if len(parts) >= 3 && parts[0] == "g" {
				user, role := parts[1], parts[2]
				_, err := m.enforcer.AddGroupingPolicy(user, role)
				if err != nil {
					return fmt.Errorf("添加角色策略失败: %w", err)
				}
			}
		}
	}

	// 保存策略
	if err := m.enforcer.SavePolicy(); err != nil {
		return fmt.Errorf("保存策略失败: %w", err)
	}

	return nil
}

// RefreshPolicies 刷新策略
func (m *Manager) RefreshPolicies(ctx context.Context) error {
	return m.LoadPolicies(ctx)
}

// Enforce 执行权限检查
// subject: 主体（通常是用户 ID 或角色）
// object: 对象（资源）
// action: 操作
func (m *Manager) Enforce(subject, object, action string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	allowed, err := m.enforcer.Enforce(subject, object, action)
	if err != nil {
		return false, fmt.Errorf("权限检查失败: %w", err)
	}

	return allowed, nil
}

// GetEnforcer 获取 Casbin Enforcer（用于高级用法）
func (m *Manager) GetEnforcer() *casbin.Enforcer {
	return m.enforcer
}

// getDefaultModel 获取默认的 RBAC 模型
func getDefaultModel() (model.Model, error) {
	// 默认的 RBAC 模型配置
	modelText := `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
`

	m, err := model.NewModelFromString(modelText)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// parsePolicyString 解析策略字符串
// 格式：p, sub, obj, act 或 g, user, role
func parsePolicyString(policy string) []string {
	parts := strings.Split(policy, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
