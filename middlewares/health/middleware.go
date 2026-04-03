// Copyright 2025 ink-yht-code
//
// Proprietary License
//
// IMPORTANT: This software is NOT open source.
// You may NOT use, copy, modify, merge, publish, distribute, sublicense,
// or sell copies of this file, in whole or in part, without prior written
// permission from the copyright holder.
//
// This software is provided "AS IS", without warranty of any kind.

package health

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthStatus 健康状态
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusDegraded  HealthStatus = "degraded"
)

// CheckFunc 健康检查函数
type CheckFunc func(ctx context.Context) error

// CheckResult 检查结果
type CheckResult struct {
	Status  HealthStatus   `json:"status"`
	Details map[string]any `json:"details,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// Checker 健康检查器
type Checker struct {
	name    string
	check   CheckFunc
	timeout time.Duration
}

// NewChecker 创建健康检查器
func NewChecker(name string, check CheckFunc, timeout time.Duration) *Checker {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Checker{
		name:    name,
		check:   check,
		timeout: timeout,
	}
}

// Check 执行健康检查
func (c *Checker) Check(ctx context.Context) CheckResult {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	err := c.check(ctx)
	if err != nil {
		return CheckResult{
			Status: StatusUnhealthy,
			Error:  err.Error(),
		}
	}
	return CheckResult{
		Status: StatusHealthy,
	}
}

// Builder 健康检查中间件构建器
type Builder struct {
	checkers []*Checker
	liveOnly bool // 仅存活检查（不检查依赖）
	mu       sync.RWMutex
}

// NewBuilder 创建健康检查构建器
func NewBuilder() *Builder {
	return &Builder{
		checkers: make([]*Checker, 0),
	}
}

// AddChecker 添加健康检查器
func (b *Builder) AddChecker(name string, check CheckFunc, timeout ...time.Duration) *Builder {
	t := 5 * time.Second
	if len(timeout) > 0 {
		t = timeout[0]
	}
	b.checkers = append(b.checkers, NewChecker(name, check, t))
	return b
}

// LiveOnly 设置仅存活检查（K8s liveness probe）
// 不检查依赖服务，仅检查进程存活
func (b *Builder) LiveOnly() *Builder {
	b.liveOnly = true
	return b
}

// Build 构建健康检查中间件
func (b *Builder) Build() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// 存活检查：仅检查进程存活
		if b.liveOnly {
			c.JSON(http.StatusOK, gin.H{
				"status": string(StatusHealthy),
			})
			return
		}

		// 就绪检查：检查所有依赖
		b.mu.RLock()
		checkers := b.checkers
		b.mu.RUnlock()

		results := make(map[string]CheckResult)
		overallStatus := StatusHealthy

		for _, checker := range checkers {
			result := checker.Check(ctx)
			results[checker.name] = result

			// 更新整体状态
			if result.Status == StatusUnhealthy {
				overallStatus = StatusUnhealthy
			} else if result.Status == StatusDegraded && overallStatus != StatusUnhealthy {
				overallStatus = StatusDegraded
			}
		}

		statusCode := http.StatusOK
		if overallStatus == StatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, gin.H{
			"status": string(overallStatus),
			"checks": results,
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// --- 预置检查器 ---

// CheckDB 数据库检查
func CheckDB(pingFunc func(ctx context.Context) error) CheckFunc {
	return func(ctx context.Context) error {
		return pingFunc(ctx)
	}
}

// CheckRedis Redis 检查
func CheckRedis(pingFunc func(ctx context.Context) error) CheckFunc {
	return func(ctx context.Context) error {
		return pingFunc(ctx)
	}
}

// CheckHTTP HTTP 服务检查
func CheckHTTP(url string, timeout time.Duration) CheckFunc {
	return func(ctx context.Context) error {
		client := &http.Client{Timeout: timeout}
		resp, err := client.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 500 {
			return ErrServiceUnavailable
		}
		return nil
	}
}

// --- 简便函数 ---

// Liveness 存活检查中间件（K8s liveness probe）
func Liveness() gin.HandlerFunc {
	return NewBuilder().LiveOnly().Build()
}

// Readiness 就绪检查中间件（K8s readiness probe）
func Readiness(checkers ...*Checker) gin.HandlerFunc {
	builder := NewBuilder()
	for _, c := range checkers {
		builder.checkers = append(builder.checkers, c)
	}
	return builder.Build()
}

// Health 综合健康检查（同时提供 /live 和 /ready）
func Health(r *gin.Engine, checkers ...*Checker) {
	r.GET("/live", Liveness())
	r.GET("/ready", Readiness(checkers...))
	r.GET("/health", NewBuilder().AddChecker("self", func(ctx context.Context) error {
		return nil // 进程自身总是健康
	}).Build())
}

// --- 错误定义 ---

var (
	// ErrServiceUnavailable 服务不可用
	ErrServiceUnavailable = errors.New("service unavailable")
	// ErrTimeout 检查超时
	ErrTimeout = errors.New("health check timeout")
)
