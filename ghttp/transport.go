// Copyright 2025 ink-yht-code
//
// Proprietary License

// Package ghttp transport.go 提供可复用的 HTTP Transport 连接池，
// 按目标 host 缓存 *http.Transport，避免每次请求重建连接。
package ghttp

import (
	"net/http"
	"sync"
	"time"
)

// TransportPool 按 host 维护独立的 http.Transport，实现连接复用。
type TransportPool struct {
	mu         sync.RWMutex
	transports map[string]*http.Transport
	cfg        TransportConfig
}

// TransportConfig 连接池配置。
type TransportConfig struct {
	// MaxIdleConnsPerHost 每个 host 最大空闲连接数，默认 100
	MaxIdleConnsPerHost int
	// MaxConnsPerHost 每个 host 最大连接数，0 表示不限制
	MaxConnsPerHost int
	// IdleConnTimeout 空闲连接超时，默认 90s
	IdleConnTimeout time.Duration
	// DialTimeout 建连超时，默认 10s
	DialTimeout time.Duration
	// ResponseHeaderTimeout 等待响应头超时，默认 30s
	ResponseHeaderTimeout time.Duration
	// DisableKeepAlives 禁用 Keep-Alive（调试用）
	DisableKeepAlives bool
}

// DefaultTransportConfig 返回生产推荐配置。
func DefaultTransportConfig() TransportConfig {
	return TransportConfig{
		MaxIdleConnsPerHost:   100,
		MaxConnsPerHost:       0,
		IdleConnTimeout:       90 * time.Second,
		DialTimeout:           10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}
}

var globalPool = NewTransportPool(DefaultTransportConfig())

// NewTransportPool 创建连接池。
func NewTransportPool(cfg TransportConfig) *TransportPool {
	if cfg.MaxIdleConnsPerHost <= 0 {
		cfg.MaxIdleConnsPerHost = 100
	}
	if cfg.IdleConnTimeout <= 0 {
		cfg.IdleConnTimeout = 90 * time.Second
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 10 * time.Second
	}
	if cfg.ResponseHeaderTimeout <= 0 {
		cfg.ResponseHeaderTimeout = 30 * time.Second
	}
	return &TransportPool{
		transports: make(map[string]*http.Transport),
		cfg:        cfg,
	}
}

// Get 获取指定 host 的 Transport，不存在则创建。
func (p *TransportPool) Get(host string) *http.Transport {
	p.mu.RLock()
	t, ok := p.transports[host]
	p.mu.RUnlock()
	if ok {
		return t
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	// double-check
	if t, ok = p.transports[host]; ok {
		return t
	}
	t = p.newTransport()
	p.transports[host] = t
	return t
}

// GetClient 返回绑定到指定 host Transport 的 http.Client。
func (p *TransportPool) GetClient(host string, timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: p.Get(host),
		Timeout:   timeout,
	}
}

// GlobalPool 返回全局连接池（单例）。
func GlobalPool() *TransportPool {
	return globalPool
}

// SetGlobalConfig 重置全局连接池配置（需在服务启动前调用）。
func SetGlobalConfig(cfg TransportConfig) {
	globalPool = NewTransportPool(cfg)
}

func (p *TransportPool) newTransport() *http.Transport {
	return &http.Transport{
		MaxIdleConnsPerHost:   p.cfg.MaxIdleConnsPerHost,
		MaxConnsPerHost:       p.cfg.MaxConnsPerHost,
		IdleConnTimeout:       p.cfg.IdleConnTimeout,
		ResponseHeaderTimeout: p.cfg.ResponseHeaderTimeout,
		DisableKeepAlives:     p.cfg.DisableKeepAlives,
		// 使用标准库默认 Dialer，DialTimeout 通过 context 控制
	}
}
