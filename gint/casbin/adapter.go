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
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
)

// memoryAdapter 内存适配器实现
// 策略通过代码动态加载，不持久化
type memoryAdapter struct {
	lines []string
}

// NewMemoryAdapter 创建内存适配器
func NewMemoryAdapter() persist.Adapter {
	return &memoryAdapter{
		lines: make([]string, 0),
	}
}

// LoadPolicy 加载策略（从内存中）
func (a *memoryAdapter) LoadPolicy(m model.Model) error {
	// 策略通过代码动态加载，这里不需要实现
	return nil
}

// SavePolicy 保存策略（保存到内存）
func (a *memoryAdapter) SavePolicy(m model.Model) error {
	// 策略保存在内存中，不需要持久化
	return nil
}

// AddPolicy 添加策略
func (a *memoryAdapter) AddPolicy(sec string, ptype string, rule []string) error {
	// 不需要实现，策略通过 Enforcer 管理
	return nil
}

// RemovePolicy 移除策略
func (a *memoryAdapter) RemovePolicy(sec string, ptype string, rule []string) error {
	// 不需要实现，策略通过 Enforcer 管理
	return nil
}

// RemoveFilteredPolicy 移除过滤的策略
func (a *memoryAdapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	// 不需要实现，策略通过 Enforcer 管理
	return nil
}
