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

// Package tx 提供事务管理，支持 ctx 注入事务 DB
package tx

import (
	"context"

	"gorm.io/gorm"
)

type ctxKey struct{}

// Manager 事务管理器
type Manager struct {
	db *gorm.DB
}

// NewManager 创建事务管理器
func NewManager(db *gorm.DB) *Manager {
	return &Manager{db: db}
}

// Do 在事务中执行函数
func (m *Manager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 将事务 DB 注入 ctx
		ctx = context.WithValue(ctx, ctxKey{}, tx)
		return fn(ctx)
	})
}

// FromContext 从 ctx 获取 DB（优先返回事务 DB）
func FromContext(ctx context.Context, defaultDB *gorm.DB) *gorm.DB {
	if ctx == nil {
		return defaultDB
	}
	if tx, ok := ctx.Value(ctxKey{}).(*gorm.DB); ok {
		return tx
	}
	return defaultDB
}

// GetDB 获取 DB（用于 dao 层）
func GetDB(ctx context.Context, defaultDB *gorm.DB) *gorm.DB {
	return FromContext(ctx, defaultDB)
}
