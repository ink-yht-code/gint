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

package store

import (
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Service 服务注册记录
type Service struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `gorm:"uniqueIndex;size:64;not null"`
	ServiceID int       `gorm:"uniqueIndex;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// TableName 表名
func (Service) TableName() string {
	return "services"
}

// Store 存储接口
type Store interface {
	// Allocate 分配 ServiceID（幂等）
	Allocate(name string) (*Service, error)
	// Get 获取服务
	Get(name string) (*Service, error)
	// List 列出所有服务
	List() ([]Service, error)
}

// SQLiteStore SQLite 存储
type SQLiteStore struct {
	db *gorm.DB
}

// NewSQLiteStore 创建 SQLite 存储
func NewSQLiteStore(dsn string) (*SQLiteStore, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 自动迁移
	if err := db.AutoMigrate(&Service{}); err != nil {
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

// Allocate 分配 ServiceID
func (s *SQLiteStore) Allocate(name string) (*Service, error) {
	// 先检查是否已存在
	var svc Service
	if err := s.db.Where("name = ?", name).First(&svc).Error; err == nil {
		return &svc, nil
	}

	// 获取下一个 ID
	var maxID int
	s.db.Model(&Service{}).Select("COALESCE(MAX(service_id), 100)").Scan(&maxID)
	nextID := maxID + 1
	if nextID < 101 {
		nextID = 101
	}

	// 创建新记录
	svc = Service{
		Name:      name,
		ServiceID: nextID,
	}
	if err := s.db.Create(&svc).Error; err != nil {
		return nil, err
	}

	return &svc, nil
}

// Get 获取服务
func (s *SQLiteStore) Get(name string) (*Service, error) {
	var svc Service
	if err := s.db.Where("name = ?", name).First(&svc).Error; err != nil {
		return nil, err
	}
	return &svc, nil
}

// List 列出所有服务
func (s *SQLiteStore) List() ([]Service, error) {
	var services []Service
	if err := s.db.Find(&services).Error; err != nil {
		return nil, err
	}
	return services, nil
}
