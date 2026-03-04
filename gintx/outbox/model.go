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

// Package outbox 提供 Outbox 模式实现
package outbox

import (
	"time"

	"gorm.io/gorm"
)

// Status 状态
type Status string

const (
	StatusPending Status = "pending"
	StatusSent    Status = "sent"
	StatusFailed  Status = "failed"
)

// Outbox Outbox 记录
type Outbox struct {
	ID           string    `gorm:"primaryKey;size:64"`
	Service      string    `gorm:"size:32;not null;index"`
	EventName    string    `gorm:"size:64;not null"`
	Payload      string    `gorm:"type:json;not null"`
	Status       Status    `gorm:"size:16;not null;default:pending;index"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	NextRetryAt  time.Time `gorm:"index"`
	RetryCount   int       `gorm:"default:0"`
	RequestID    string    `gorm:"size:64"`
	ErrorMessage string    `gorm:"size:512"`
}

// TableName 表名
func (Outbox) TableName() string {
	return "outbox"
}

// Repository Outbox 仓库接口
type Repository interface {
	Save(db *gorm.DB, records ...*Outbox) error
	ListPending(db *gorm.DB, limit int) ([]*Outbox, error)
	MarkSent(db *gorm.DB, ids ...string) error
	MarkFailed(db *gorm.DB, id string, errMsg string) error
}

// Repo Outbox 仓库实现
type Repo struct{}

// NewRepo 创建仓库
func NewRepo() *Repo {
	return &Repo{}
}

// Save 保存记录
func (r *Repo) Save(db *gorm.DB, records ...*Outbox) error {
	return db.Create(&records).Error
}

// ListPending 获取待发送记录
func (r *Repo) ListPending(db *gorm.DB, limit int) ([]*Outbox, error) {
	var records []*Outbox
	err := db.Where("status = ? AND next_retry_at <= ?", StatusPending, time.Now()).
		Limit(limit).
		Find(&records).Error
	return records, err
}

// MarkSent 标记为已发送
func (r *Repo) MarkSent(db *gorm.DB, ids ...string) error {
	return db.Model(&Outbox{}).Where("id IN ?", ids).
		Updates(map[string]any{
			"status":    StatusSent,
			"sent_at":  time.Now(),
		}).Error
}

// MarkFailed 标记为失败
func (r *Repo) MarkFailed(db *gorm.DB, id string, errMsg string) error {
	return db.Model(&Outbox{}).Where("id = ?", id).
		Updates(map[string]any{
			"status":        StatusFailed,
			"error_message": errMsg,
			"retry_count":   gorm.Expr("retry_count + 1"),
		}).Error
}
