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

// Package app 提供应用启动器
package app

import (
	"context"
	"fmt"

	"github.com/ink-yht-code/gintx/db"
	"github.com/ink-yht-code/gintx/httpx"
	"github.com/ink-yht-code/gintx/log"
	"github.com/ink-yht-code/gintx/redis"
	"github.com/ink-yht-code/gintx/tx"
	redislib "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config 应用配置
type Config struct {
	Service ServiceConfig
	HTTP    httpx.Config
	Log     log.Config
	DB      db.Config
	Redis   redis.Config
}

type ServiceConfig struct {
	ID   int
	Name string
}

// App 应用
type App struct {
	Config    *Config
	DB        *gorm.DB
	Redis     *redislib.Client
	TxManager *tx.Manager
	HTTP      *httpx.Server
}

// New 创建应用
func New(cfg *Config) (*App, error) {
	// 初始化日志
	if err := log.Init(cfg.Log); err != nil {
		return nil, fmt.Errorf("init log: %w", err)
	}

	app := &App{Config: cfg}

	// 初始化数据库
	if cfg.DB.DSN != "" {
		database, err := db.New(cfg.DB)
		if err != nil {
			return nil, fmt.Errorf("init db: %w", err)
		}
		app.DB = database
		app.TxManager = tx.NewManager(database)
	}

	// 初始化 Redis
	if cfg.Redis.Addr != "" {
		app.Redis = redislib.NewClient(&redislib.Options{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
	}

	// 初始化 HTTP
	if cfg.HTTP.Enabled {
		app.HTTP = httpx.NewServer(cfg.HTTP)
	}

	return app, nil
}

// Run 启动应用
func (a *App) Run() error {
	if a.HTTP != nil {
		log.Info("HTTP server starting", zap.String("addr", a.Config.HTTP.Addr))
		if err := a.HTTP.Run(); err != nil {
			return fmt.Errorf("http server: %w", err)
		}
	}
	return nil
}

// Shutdown 关闭应用
func (a *App) Shutdown(ctx context.Context) error {
	if a.HTTP != nil {
		if err := a.HTTP.Shutdown(ctx); err != nil {
			return err
		}
	}

	if a.DB != nil {
		sqlDB, _ := a.DB.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	if a.Redis != nil {
		a.Redis.Close()
	}

	return log.Sync()
}
