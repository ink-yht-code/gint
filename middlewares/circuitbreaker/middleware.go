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

package circuitbreaker

import (
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ink-yht-code/gint/logger"
	"go.uber.org/zap"
)

// State 熔断器状态
type State int32

const (
	// StateClosed 关闭状态（正常）
	StateClosed State = iota
	// StateOpen 打开状态（熔断）
	StateOpen
	// StateHalfOpen 半开状态（试探）
	StateHalfOpen
)

// String 返回状态名称
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Config 熔断器配置
type Config struct {
	// FailureThreshold 失败阈值（连续失败次数）
	FailureThreshold int
	// SuccessThreshold 成功阈值（半开状态下连续成功次数）
	SuccessThreshold int
	// Timeout 熔断超时时间（熔断后多久尝试恢复）
	Timeout time.Duration
	// HalfOpenRequests 半开状态下允许的请求数
	HalfOpenRequests int
}

// DefaultConfig 默认配置
func DefaultConfig() Config {
	return Config{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          30 * time.Second,
		HalfOpenRequests: 1,
	}
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	config Config

	state           atomic.Int32 // 当前状态
	failureCount    atomic.Int32 // 连续失败计数
	successCount    atomic.Int32 // 连续成功计数
	lastFailureTime atomic.Int64 // 最后一次失败时间

	requests         atomic.Int64 // 总请求数
	failures         atomic.Int64 // 总失败数
	halfOpenInFlight atomic.Int32 // 半开状态进行中请求数

	mu sync.Mutex
}

// New 创建熔断器
func New(config ...Config) *CircuitBreaker {
	cfg := DefaultConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	cb := &CircuitBreaker{
		config: cfg,
	}
	cb.state.Store(int32(StateClosed))
	return cb
}

// Allow 检查是否允许请求
func (cb *CircuitBreaker) Allow() bool {
	state := State(cb.state.Load())

	switch state {
	case StateClosed:
		return true
	case StateOpen:
		// 检查是否可以尝试恢复
		lastFailure := time.Unix(0, cb.lastFailureTime.Load())
		if time.Since(lastFailure) >= cb.config.Timeout {
			cb.transitionTo(StateHalfOpen)
			return cb.allowHalfOpenRequest()
		}
		return false
	case StateHalfOpen:
		// 半开状态限制请求数
		return cb.allowHalfOpenRequest()
	default:
		return false
	}
}

// RecordSuccess 记录成功
func (cb *CircuitBreaker) RecordSuccess() {
	cb.requests.Add(1)
	state := State(cb.state.Load())

	switch state {
	case StateClosed:
		// 重置失败计数
		cb.failureCount.Store(0)
	case StateHalfOpen:
		cb.releaseHalfOpenRequest()
		count := cb.successCount.Add(1)
		if int(count) >= cb.config.SuccessThreshold {
			cb.transitionTo(StateClosed)
			cb.failureCount.Store(0)
			cb.successCount.Store(0)
		}
	}
}

// RecordFailure 记录失败
func (cb *CircuitBreaker) RecordFailure() {
	cb.requests.Add(1)
	cb.failures.Add(1)
	cb.lastFailureTime.Store(time.Now().UnixNano())

	state := State(cb.state.Load())

	switch state {
	case StateClosed:
		count := cb.failureCount.Add(1)
		if int(count) >= cb.config.FailureThreshold {
			cb.transitionTo(StateOpen)
		}
	case StateHalfOpen:
		cb.releaseHalfOpenRequest()
		// 半开状态下失败，立即熔断
		cb.transitionTo(StateOpen)
		cb.successCount.Store(0)
	}
}

// transitionTo 状态转换
func (cb *CircuitBreaker) transitionTo(state State) {
	oldState := State(cb.state.Swap(int32(state)))
	if oldState != state {
		if state != StateHalfOpen {
			cb.halfOpenInFlight.Store(0)
		}
		logger.Info("熔断器状态变更",
			zap.String("from", oldState.String()),
			zap.String("to", state.String()))
	}
}

func (cb *CircuitBreaker) allowHalfOpenRequest() bool {
	limit := cb.config.HalfOpenRequests
	if limit <= 0 {
		limit = 1
	}
	current := cb.halfOpenInFlight.Add(1)
	if int(current) > limit {
		cb.halfOpenInFlight.Add(-1)
		return false
	}
	return true
}

func (cb *CircuitBreaker) releaseHalfOpenRequest() {
	for {
		current := cb.halfOpenInFlight.Load()
		if current <= 0 {
			return
		}
		if cb.halfOpenInFlight.CompareAndSwap(current, current-1) {
			return
		}
	}
}

// State 获取当前状态
func (cb *CircuitBreaker) State() State {
	return State(cb.state.Load())
}

// Stats 获取统计信息
func (cb *CircuitBreaker) Stats() Stats {
	return Stats{
		State:           cb.State().String(),
		TotalRequests:   cb.requests.Load(),
		TotalFailures:   cb.failures.Load(),
		FailureCount:    cb.failureCount.Load(),
		SuccessCount:    cb.successCount.Load(),
		LastFailureTime: time.Unix(0, cb.lastFailureTime.Load()),
	}
}

// Stats 统计信息
type Stats struct {
	State           string    `json:"state"`
	TotalRequests   int64     `json:"total_requests"`
	TotalFailures   int64     `json:"total_failures"`
	FailureCount    int32     `json:"failure_count"`
	SuccessCount    int32     `json:"success_count"`
	LastFailureTime time.Time `json:"last_failure_time"`
}

// --- 中间件 ---

// Builder 熔断中间件构建器
type Builder struct {
	breaker    *CircuitBreaker
	fallback   gin.HandlerFunc
	judgeFunc  func(c *gin.Context, err error) bool
	keyFunc    func(c *gin.Context) string
	breakers   sync.Map // 按 key 存储多个熔断器
	singleMode bool     // 单熔断器模式
}

// NewBuilder 创建熔断中间件构建器
func NewBuilder(config ...Config) *Builder {
	return &Builder{
		breaker:    New(config...),
		singleMode: true,
	}
}

// WithFallback 设置降级处理函数
func (b *Builder) WithFallback(fn gin.HandlerFunc) *Builder {
	b.fallback = fn
	return b
}

// WithJudge 设置失败判断函数
// 默认：状态码 >= 500 视为失败
func (b *Builder) WithJudge(fn func(c *gin.Context, err error) bool) *Builder {
	b.judgeFunc = fn
	return b
}

// WithKeyFunc 设置按 key 分组熔断
// 例如：按路径、按服务名等
func (b *Builder) WithKeyFunc(fn func(c *gin.Context) string) *Builder {
	b.keyFunc = fn
	b.singleMode = false
	return b
}

// Build 构建中间件
func (b *Builder) Build() gin.HandlerFunc {
	return func(c *gin.Context) {
		breaker := b.getBreaker(c)

		// 检查是否允许请求
		if !breaker.Allow() {
			logger.Warn("熔断器打开，拒绝请求",
				zap.String("path", c.Request.URL.Path),
				zap.String("state", breaker.State().String()))

			if b.fallback != nil {
				b.fallback(c)
				return
			}
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"code":  503,
				"msg":   "服务暂时不可用，请稍后重试",
				"state": breaker.State().String(),
			})
			return
		}

		// 执行请求
		c.Next()

		// 判断是否失败
		isFailure := false
		if b.judgeFunc != nil {
			isFailure = b.judgeFunc(c, nil)
		} else {
			// 默认：状态码 >= 500 视为失败
			isFailure = c.Writer.Status() >= 500
		}

		if isFailure {
			breaker.RecordFailure()
		} else {
			breaker.RecordSuccess()
		}
	}
}

// getBreaker 获取熔断器
func (b *Builder) getBreaker(c *gin.Context) *CircuitBreaker {
	if b.singleMode {
		return b.breaker
	}

	key := b.keyFunc(c)
	if key == "" {
		return b.breaker
	}

	val, ok := b.breakers.Load(key)
	if ok {
		return val.(*CircuitBreaker)
	}

	breaker := New(b.breaker.config)
	actual, _ := b.breakers.LoadOrStore(key, breaker)
	return actual.(*CircuitBreaker)
}

// --- 便捷函数 ---

// Middleware 创建默认熔断中间件
func Middleware(config ...Config) gin.HandlerFunc {
	return NewBuilder(config...).Build()
}

// WithFallback 创建带降级的熔断中间件
func WithFallback(fallback gin.HandlerFunc, config ...Config) gin.HandlerFunc {
	return NewBuilder(config...).WithFallback(fallback).Build()
}

// --- 错误定义 ---

var (
	// ErrCircuitBreakerOpen 熔断器打开
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
)

// --- 用于 HTTP 客户端的熔断器 ---

// HTTPBreaker HTTP 客户端熔断器
type HTTPBreaker struct {
	breaker *CircuitBreaker
}

// NewHTTPBreaker 创建 HTTP 熔断器
func NewHTTPBreaker(config ...Config) *HTTPBreaker {
	return &HTTPBreaker{
		breaker: New(config...),
	}
}

// Do 执行 HTTP 请求（带熔断保护）
func (h *HTTPBreaker) Do(fn func() error) error {
	if !h.breaker.Allow() {
		return ErrCircuitBreakerOpen
	}

	err := fn()
	if err != nil {
		h.breaker.RecordFailure()
		return err
	}

	h.breaker.RecordSuccess()
	return nil
}

// State 获取状态
func (h *HTTPBreaker) State() State {
	return h.breaker.State()
}

// Stats 获取统计
func (h *HTTPBreaker) Stats() Stats {
	return h.breaker.Stats()
}
