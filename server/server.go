// 版权所有 2025 ink-yht-code
//
// 专有许可
//
// 重要说明：本软件并非开源软件。
// 未经版权持有人事先书面许可，
// 不得使用、复制、修改、合并、发布、分发、再许可，
// 也不得全部或部分出售本文件的副本。
//
// 本软件按“现状”提供，不附带任何形式的担保。

package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ink-yht-code/gint/logger"
	"go.uber.org/zap"
)

// Server 封装了带优雅关闭能力的 HTTP 服务。
type Server struct {
	*http.Server
	shutdownTS time.Duration
}

// Option 用于定制服务实例。
type Option func(*Server)

// WithShutdownTimeout 设置优雅关闭超时时间。
func WithShutdownTimeout(d time.Duration) Option {
	return func(s *Server) {
		s.shutdownTS = d
	}
}

// New 创建一个新的 HTTP 服务包装器。
func New(addr string, engine *gin.Engine, opts ...Option) *Server {
	s := &Server{
		Server: &http.Server{
			Addr:    addr,
			Handler: engine,
		},
		shutdownTS: 30 * time.Second,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Start 以异步方式启动 HTTP 服务。
func (s *Server) Start() error {
	go func() {
		logger.Info("HTTP 服务已启动", zap.String("addr", s.Addr))
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP 服务启动失败", zap.Error(err))
		}
	}()
	return nil
}

// Run 启动服务并阻塞，直到收到退出信号。
func (s *Server) Run() error {
	go func() {
		logger.Info("HTTP 服务已启动", zap.String("addr", s.Addr))
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP 服务启动失败", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭 HTTP 服务")

	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTS)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		logger.Error("HTTP 服务关闭失败", zap.Error(err))
		return fmt.Errorf("HTTP 服务关闭失败: %w", err)
	}

	logger.Info("HTTP 服务已停止")
	return nil
}

// Shutdown 优雅关闭底层 HTTP 服务。
func (s *Server) Shutdown(ctx context.Context) error {
	return s.Server.Shutdown(ctx)
}

// Run 使用默认参数启动服务。
func Run(addr string, engine *gin.Engine) error {
	return New(addr, engine).Run()
}

// RunWithTimeout 使用自定义关闭超时启动服务。
func RunWithTimeout(addr string, engine *gin.Engine, timeout time.Duration) error {
	return New(addr, engine, WithShutdownTimeout(timeout)).Run()
}

// ShutdownHook 表示关闭阶段执行的钩子函数。
type ShutdownHook func(ctx context.Context) error

// GracefulShutdown 管理优雅关闭钩子。
type GracefulShutdown struct {
	hooks    []ShutdownHook
	timeout  time.Duration
	onSignal func()
}

// NewGracefulShutdown 创建优雅关闭管理器。
func NewGracefulShutdown(timeout time.Duration) *GracefulShutdown {
	return &GracefulShutdown{
		hooks:   make([]ShutdownHook, 0),
		timeout: timeout,
	}
}

// AddHook 注册关闭钩子。
func (g *GracefulShutdown) AddHook(hook ShutdownHook) *GracefulShutdown {
	g.hooks = append(g.hooks, hook)
	return g
}

// OnSignal 注册收到退出信号后的回调函数。
func (g *GracefulShutdown) OnSignal(fn func()) *GracefulShutdown {
	g.onSignal = fn
	return g
}

// Wait 等待退出信号并依次执行关闭钩子。
func (g *GracefulShutdown) Wait() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	if g.onSignal != nil {
		g.onSignal()
	}

	logger.Info("开始执行关闭钩子")

	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()

	for i, hook := range g.hooks {
		logger.Info("执行关闭钩子", zap.Int("index", i+1), zap.Int("total", len(g.hooks)))
		if err := hook(ctx); err != nil {
			logger.Error("关闭钩子执行失败", zap.Int("index", i+1), zap.Error(err))
		}
	}

	logger.Info("所有关闭钩子执行完成")
	return nil
}

// HookCloseDB 返回一个用于关闭数据库连接的钩子。
func HookCloseDB(closeFunc func() error) ShutdownHook {
	return func(ctx context.Context) error {
		logger.Info("关闭数据库连接")
		return closeFunc()
	}
}

// HookCloseRedis 返回一个用于关闭 Redis 连接的钩子。
func HookCloseRedis(closeFunc func() error) ShutdownHook {
	return func(ctx context.Context) error {
		logger.Info("关闭 Redis 连接")
		return closeFunc()
	}
}

// HookCloseMQ 返回一个用于关闭消息队列连接的钩子。
func HookCloseMQ(closeFunc func() error) ShutdownHook {
	return func(ctx context.Context) error {
		logger.Info("关闭消息队列连接")
		return closeFunc()
	}
}
