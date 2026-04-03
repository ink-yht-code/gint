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

package discovery

import (
	"context"
	"errors"
	"net"
	"strconv"
	"sync"
	"time"
)

// ServiceInstance 服务实例
type ServiceInstance struct {
	// ID 实例ID
	ID string
	// Name 服务名称
	Name string
	// Address 服务地址
	Address string
	// Port 服务端口
	Port int
	// Metadata 元数据
	Metadata map[string]string
	// Version 服务版本
	Version string
	// Healthy 是否健康
	Healthy bool
	// Weight 权重
	Weight int
	// LastHeartbeat 最后心跳时间
	LastHeartbeat time.Time
}

// Addr 返回完整地址
func (s *ServiceInstance) Addr() string {
	if s.Port > 0 {
		return net.JoinHostPort(s.Address, strconv.Itoa(s.Port))
	}
	return s.Address
}

// FullAddr 返回完整地址（host:port）
func (s *ServiceInstance) FullAddr() string {
	return s.Addr()
}

// WatchHandler 服务变化监听处理器
type WatchHandler interface {
	// OnAdd 实例添加
	OnAdd(instance *ServiceInstance)
	// OnDelete 实例删除
	OnDelete(instance *ServiceInstance)
}

// WatchHandlerFunc 函数类型的 WatchHandler
type WatchHandlerFunc struct {
	AddFunc    func(instance *ServiceInstance)
	DeleteFunc func(instance *ServiceInstance)
}

// OnAdd 实现 WatchHandler 接口
func (h WatchHandlerFunc) OnAdd(instance *ServiceInstance) {
	if h.AddFunc != nil {
		h.AddFunc(instance)
	}
}

// OnDelete 实现 WatchHandler 接口
func (h WatchHandlerFunc) OnDelete(instance *ServiceInstance) {
	if h.DeleteFunc != nil {
		h.DeleteFunc(instance)
	}
}

// Registry 服务注册接口
type Registry interface {
	// Register 注册服务实例
	Register(ctx context.Context, instance *ServiceInstance) error
	// Deregister 注销服务实例
	Deregister(ctx context.Context, instance *ServiceInstance) error
	// Heartbeat 发送心跳
	Heartbeat(ctx context.Context, instance *ServiceInstance) error
}

// Discovery 服务发现接口
type Discovery interface {
	// GetInstance 获取单个实例
	GetInstance(ctx context.Context, serviceName string) (*ServiceInstance, error)
	// GetInstances 获取所有实例
	GetInstances(ctx context.Context, serviceName string) ([]*ServiceInstance, error)
	// Watch 监听服务变化
	Watch(ctx context.Context, serviceName string) (<-chan []*ServiceInstance, error)
	// Close 关闭
	Close() error
}

// RegistryDiscovery 注册与发现组合接口
type RegistryDiscovery interface {
	Registry
	Discovery
}

// Config 配置
type Config struct {
	// HeartbeatInterval 心跳间隔
	HeartbeatInterval time.Duration
	// HeartbeatTimeout 心跳超时
	HeartbeatTimeout time.Duration
	// RefreshInterval 刷新间隔
	RefreshInterval time.Duration
	// TTL 注册过期时间
	TTL time.Duration
}

// DefaultConfig 默认配置
func DefaultConfig() Config {
	return Config{
		HeartbeatInterval: 10 * time.Second,
		HeartbeatTimeout:  30 * time.Second,
		RefreshInterval:   30 * time.Second,
		TTL:               60 * time.Second,
	}
}

// --- 内存实现（用于测试） ---

// MemoryRegistry 内存注册中心
type MemoryRegistry struct {
	instances map[string]map[string]*ServiceInstance
	config    Config
	mu        sync.RWMutex
}

// NewMemoryRegistry 创建内存注册中心
func NewMemoryRegistry(config ...Config) *MemoryRegistry {
	cfg := DefaultConfig()
	if len(config) > 0 {
		cfg = config[0]
	}
	return &MemoryRegistry{
		instances: make(map[string]map[string]*ServiceInstance),
		config:    cfg,
	}
}

// Register 注册服务实例
func (r *MemoryRegistry) Register(ctx context.Context, instance *ServiceInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.instances[instance.Name] == nil {
		r.instances[instance.Name] = make(map[string]*ServiceInstance)
	}
	instance.LastHeartbeat = time.Now()
	instance.Healthy = true
	r.instances[instance.Name][instance.ID] = instance
	return nil
}

// Deregister 注销服务实例
func (r *MemoryRegistry) Deregister(ctx context.Context, instance *ServiceInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.instances[instance.Name] != nil {
		delete(r.instances[instance.Name], instance.ID)
	}
	return nil
}

// Heartbeat 发送心跳
func (r *MemoryRegistry) Heartbeat(ctx context.Context, instance *ServiceInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.instances[instance.Name] == nil || r.instances[instance.Name][instance.ID] == nil {
		return ErrInstanceNotFound
	}
	inst := r.instances[instance.Name][instance.ID]
	inst.LastHeartbeat = time.Now()
	inst.Healthy = true
	return nil
}

// GetInstance 获取单个实例
func (r *MemoryRegistry) GetInstance(ctx context.Context, serviceName string) (*ServiceInstance, error) {
	instances := r.getHealthyInstances(serviceName)
	if len(instances) == 0 {
		return nil, ErrNoInstance
	}
	// 简单轮询
	return instances[0], nil
}

// GetInstances 获取所有实例
func (r *MemoryRegistry) GetInstances(ctx context.Context, serviceName string) ([]*ServiceInstance, error) {
	instances := r.getHealthyInstances(serviceName)
	if len(instances) == 0 {
		return nil, ErrNoInstance
	}
	return instances, nil
}

// Watch 监听服务变化
func (r *MemoryRegistry) Watch(ctx context.Context, serviceName string) (<-chan []*ServiceInstance, error) {
	ch := make(chan []*ServiceInstance, 10)
	go func() {
		ticker := time.NewTicker(r.config.RefreshInterval)
		defer ticker.Stop()
		defer close(ch)

		for {
			select {
			case <-ticker.C:
				instances := r.getHealthyInstances(serviceName)
				select {
				case ch <- instances:
				default:
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

// getHealthyInstances 获取健康实例
func (r *MemoryRegistry) getHealthyInstances(serviceName string) []*ServiceInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*ServiceInstance

	if r.instances[serviceName] == nil {
		return result
	}

	now := time.Now()
	for _, inst := range r.instances[serviceName] {
		// 检查是否过期（在返回副本上标记健康状态，避免在读锁下写共享对象）
		healthy := inst.Healthy && now.Sub(inst.LastHeartbeat) <= r.config.HeartbeatTimeout
		if !healthy {
			continue
		}
		copyInst := *inst
		copyInst.Healthy = true
		result = append(result, &copyInst)
	}

	return result
}

// Close 关闭
func (r *MemoryRegistry) Close() error {
	return nil
}

// --- 错误定义 ---

var (
	// ErrInstanceNotFound 实例未找到
	ErrInstanceNotFound = errors.New("discovery: instance not found")
	// ErrNoInstance 没有可用实例
	ErrNoInstance = errors.New("discovery: no available instance")
	// ErrRegisterFailed 注册失败
	ErrRegisterFailed = errors.New("discovery: register failed")
	// ErrDeregisterFailed 注销失败
	ErrDeregisterFailed = errors.New("discovery: deregister failed")
)

// --- 全局注册中心 ---

var defaultRegistry RegistryDiscovery

// SetDefault 设置默认注册中心
func SetDefault(r RegistryDiscovery) {
	defaultRegistry = r
}

// Default 获取默认注册中心
func Default() RegistryDiscovery {
	return defaultRegistry
}

// Register 注册到默认注册中心
func Register(ctx context.Context, instance *ServiceInstance) error {
	if defaultRegistry == nil {
		return ErrNoInstance
	}
	return defaultRegistry.Register(ctx, instance)
}

// GetInstance 从默认注册中心获取实例
func GetInstance(ctx context.Context, serviceName string) (*ServiceInstance, error) {
	if defaultRegistry == nil {
		return nil, ErrNoInstance
	}
	return defaultRegistry.GetInstance(ctx, serviceName)
}

// GetInstances 从默认注册中心获取所有实例
func GetInstances(ctx context.Context, serviceName string) ([]*ServiceInstance, error) {
	if defaultRegistry == nil {
		return nil, ErrNoInstance
	}
	return defaultRegistry.GetInstances(ctx, serviceName)
}
