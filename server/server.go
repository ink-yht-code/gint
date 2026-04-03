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

// Server HTTP 服务器封装
type Server struct {
	*http.Server
	shutdownTS time.Duration // 关闭超时时间
}

// Option 服务器配置选项
type Option func(*Server)

// WithShutdownTimeout 设置关闭超时时间
func WithShutdownTimeout(d time.Duration) Option {
	return func(s *Server) {
		s.shutdownTS = d
	}
}

// New 创建 HTTP 服务器
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

// Start 启动服务器（非阻塞）
func (s *Server) Start() error {
	go func() {
		logger.Info("HTTP 服务器启动", zap.String("addr", s.Addr))
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP 服务器错误", zap.Error(err))
		}
	}()
	return nil
}

// Run 启动服务器并阻塞，支持优雅关闭
// 监听 SIGINT (Ctrl+C) 和 SIGTERM (K8s 停止信号)
func (s *Server) Run() error {
	// 启动服务器
	go func() {
		logger.Info("HTTP 服务器启动", zap.String("addr", s.Addr))
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP 服务器错误", zap.Error(err))
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务器...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTS)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		logger.Error("服务器关闭错误", zap.Error(err))
		return fmt.Errorf("服务器关闭错误: %w", err)
	}

	logger.Info("服务器已关闭")
	return nil
}

// Shutdown 优雅关闭
// 1. 停止接受新连接
// 2. 等待现有请求处理完成
// 3. 超时后强制关闭
func (s *Server) Shutdown(ctx context.Context) error {
	return s.Server.Shutdown(ctx)
}

// --- 简便函数 ---

// Run 快速启动服务器
func Run(addr string, engine *gin.Engine) error {
	return New(addr, engine).Run()
}

// RunWithTimeout 快速启动服务器（自定义超时）
func RunWithTimeout(addr string, engine *gin.Engine, timeout time.Duration) error {
	return New(addr, engine, WithShutdownTimeout(timeout)).Run()
}

// --- 关闭钩子 ---

// ShutdownHook 关闭钩子函数
type ShutdownHook func(ctx context.Context) error

// GracefulShutdown 优雅关闭管理器
type GracefulShutdown struct {
	hooks    []ShutdownHook
	timeout  time.Duration
	onSignal func()
}

// NewGracefulShutdown 创建优雅关闭管理器
func NewGracefulShutdown(timeout time.Duration) *GracefulShutdown {
	return &GracefulShutdown{
		hooks:   make([]ShutdownHook, 0),
		timeout: timeout,
	}
}

// AddHook 添加关闭钩子
// 钩子按添加顺序执行
func (g *GracefulShutdown) AddHook(hook ShutdownHook) *GracefulShutdown {
	g.hooks = append(g.hooks, hook)
	return g
}

// OnSignal 设置信号处理回调
func (g *GracefulShutdown) OnSignal(fn func()) *GracefulShutdown {
	g.onSignal = fn
	return g
}

// Wait 等待信号并执行关闭
func (g *GracefulShutdown) Wait() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	if g.onSignal != nil {
		g.onSignal()
	}

	logger.Info("开始执行关闭钩子...")

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

// --- 预置钩子 ---

// HookCloseDB 关闭数据库连接
func HookCloseDB(closeFunc func() error) ShutdownHook {
	return func(ctx context.Context) error {
		logger.Info("关闭数据库连接")
		return closeFunc()
	}
}

// HookCloseRedis 关闭 Redis 连接
func HookCloseRedis(closeFunc func() error) ShutdownHook {
	return func(ctx context.Context) error {
		logger.Info("关闭 Redis 连接")
		return closeFunc()
	}
}

// HookCloseMQ 关闭消息队列连接
func HookCloseMQ(closeFunc func() error) ShutdownHook {
	return func(ctx context.Context) error {
		logger.Info("关闭消息队列连接")
		return closeFunc()
	}
}
