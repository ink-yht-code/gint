// Copyright 2025 ink-yht-code
//
// Proprietary License

package etcd

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/ink-yht-code/gint/discovery"
)

var (
	ErrNotRegistered = errors.New("service not registered")
	ErrKeyNotFound   = errors.New("key not found")
)

// Registry etcd 服务注册中心
type Registry struct {
	client   *clientv3.Client
	leaseID  clientv3.LeaseID
	kv       clientv3.KV
	lease    clientv3.Lease
	mu       sync.RWMutex
	services map[string]*discovery.ServiceInstance
	prefix   string
	ttl      int64
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// Option 注册中心选项
type Option func(*Registry)

// WithPrefix 设置 key 前缀
func WithPrefix(prefix string) Option {
	return func(r *Registry) {
		r.prefix = prefix
	}
}

// WithTTL 设置租约 TTL（秒）
func WithTTL(ttl int64) Option {
	return func(r *Registry) {
		r.ttl = ttl
	}
}

// New 创建 etcd 服务注册中心
func New(endpoints []string, opts ...Option) (*Registry, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:            endpoints,
		DialTimeout:          5 * time.Second,
		DialKeepAliveTime:    10 * time.Second,
		DialKeepAliveTimeout: 3 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	r := &Registry{
		client:   client,
		kv:       clientv3.NewKV(client),
		lease:    clientv3.NewLease(client),
		services: make(map[string]*discovery.ServiceInstance),
		prefix:   "/services/",
		ttl:      30,
	}

	for _, opt := range opts {
		opt(r)
	}

	r.ctx, r.cancel = context.WithCancel(context.Background())

	return r, nil
}

// Register 注册服务
func (r *Registry) Register(ctx context.Context, service *discovery.ServiceInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := r.serviceKey(service.Name, service.ID)
	value, err := json.Marshal(service)
	if err != nil {
		return err
	}

	// 创建租约
	if r.leaseID == 0 {
		resp, err := r.lease.Grant(ctx, r.ttl)
		if err != nil {
			return err
		}
		r.leaseID = resp.ID

		// 启动保活
		r.wg.Add(1)
		go r.keepAlive()
	}

	// 注册服务
	_, err = r.kv.Put(ctx, key, string(value), clientv3.WithLease(r.leaseID))
	if err != nil {
		return err
	}

	r.services[key] = service
	return nil
}

// Deregister 注销服务
func (r *Registry) Deregister(ctx context.Context, service *discovery.ServiceInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := r.serviceKey(service.Name, service.ID)

	_, err := r.kv.Delete(ctx, key)
	if err != nil {
		return err
	}

	delete(r.services, key)
	return nil
}

// GetInstance 获取单个服务实例
func (r *Registry) GetInstance(ctx context.Context, serviceName string) (*discovery.ServiceInstance, error) {
	instances, err := r.GetInstances(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	if len(instances) == 0 {
		return nil, ErrKeyNotFound
	}

	// 简单轮询，实际应使用负载均衡
	return instances[0], nil
}

// GetInstances 获取所有服务实例
func (r *Registry) GetInstances(ctx context.Context, serviceName string) ([]*discovery.ServiceInstance, error) {
	prefix := r.prefix + serviceName + "/"

	resp, err := r.kv.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	instances := make([]*discovery.ServiceInstance, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var instance discovery.ServiceInstance
		if err := json.Unmarshal(kv.Value, &instance); err != nil {
			continue
		}
		instances = append(instances, &instance)
	}

	return instances, nil
}

// Watch 监听服务变化
func (r *Registry) Watch(ctx context.Context, serviceName string, handler discovery.WatchHandler) error {
	prefix := r.prefix + serviceName + "/"

	watcher := clientv3.NewWatcher(r.client)
	defer watcher.Close()

	watchChan := watcher.Watch(ctx, prefix, clientv3.WithPrefix())

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case resp, ok := <-watchChan:
			if !ok {
				return errors.New("watch channel closed")
			}

			for _, event := range resp.Events {
				var instance discovery.ServiceInstance
				if err := json.Unmarshal(event.Kv.Value, &instance); err != nil {
					continue
				}

				switch event.Type {
				case clientv3.EventTypePut:
					handler.OnAdd(&instance)
				case clientv3.EventTypeDelete:
					handler.OnDelete(&instance)
				}
			}
		}
	}
}

// Close 关闭注册中心
func (r *Registry) Close() error {
	r.cancel()
	r.wg.Wait()

	// 注销所有服务
	r.mu.RLock()
	for _, service := range r.services {
		key := r.serviceKey(service.Name, service.ID)
		_, _ = r.kv.Delete(context.Background(), key)
	}
	r.mu.RUnlock()

	// 撤销租约
	if r.leaseID != 0 {
		_, _ = r.lease.Revoke(context.Background(), r.leaseID)
	}

	return r.client.Close()
}

// keepAlive 保持租约
func (r *Registry) keepAlive() {
	defer r.wg.Done()

	ticker := time.NewTicker(time.Duration(r.ttl/3) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			_, err := r.lease.KeepAliveOnce(r.ctx, r.leaseID)
			if err != nil {
				// 租约过期，尝试重新创建
				resp, err := r.lease.Grant(r.ctx, r.ttl)
				if err != nil {
					continue
				}
				r.leaseID = resp.ID

				// 重新注册所有服务
				r.mu.RLock()
				for key, service := range r.services {
					value, _ := json.Marshal(service)
					_, _ = r.kv.Put(r.ctx, key, string(value), clientv3.WithLease(r.leaseID))
				}
				r.mu.RUnlock()
			}
		}
	}
}

// serviceKey 生成服务 key
func (r *Registry) serviceKey(name, id string) string {
	return r.prefix + name + "/" + id
}

// Discovery etcd 服务发现客户端
type Discovery struct {
	registry *Registry
	cache    map[string][]*discovery.ServiceInstance
	mu       sync.RWMutex
}

// NewDiscovery 创建服务发现客户端
func NewDiscovery(registry *Registry) *Discovery {
	return &Discovery{
		registry: registry,
		cache:    make(map[string][]*discovery.ServiceInstance),
	}
}

// GetInstance 获取单个实例
func (d *Discovery) GetInstance(ctx context.Context, serviceName string) (*discovery.ServiceInstance, error) {
	return d.registry.GetInstance(ctx, serviceName)
}

// GetInstances 获取所有实例
func (d *Discovery) GetInstances(ctx context.Context, serviceName string) ([]*discovery.ServiceInstance, error) {
	// 先查缓存
	d.mu.RLock()
	if instances, ok := d.cache[serviceName]; ok && len(instances) > 0 {
		d.mu.RUnlock()
		return instances, nil
	}
	d.mu.RUnlock()

	// 从 etcd 获取
	instances, err := d.registry.GetInstances(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	d.mu.Lock()
	d.cache[serviceName] = instances
	d.mu.Unlock()

	return instances, nil
}

// Watch 监听服务变化
func (d *Discovery) Watch(ctx context.Context, serviceName string, handler discovery.WatchHandler) error {
	return d.registry.Watch(ctx, serviceName, &cacheHandler{
		discovery: d,
		handler:   handler,
		service:   serviceName,
	})
}

// cacheHandler 带缓存更新的处理器
type cacheHandler struct {
	discovery *Discovery
	handler   discovery.WatchHandler
	service   string
}

func (h *cacheHandler) OnAdd(instance *discovery.ServiceInstance) {
	h.discovery.mu.Lock()
	instances := h.discovery.cache[h.service]
	instances = append(instances, instance)
	h.discovery.cache[h.service] = instances
	h.discovery.mu.Unlock()

	h.handler.OnAdd(instance)
}

func (h *cacheHandler) OnDelete(instance *discovery.ServiceInstance) {
	h.discovery.mu.Lock()
	instances := h.discovery.cache[h.service]
	for i, ins := range instances {
		if ins.ID == instance.ID {
			instances = append(instances[:i], instances[i+1:]...)
			break
		}
	}
	h.discovery.cache[h.service] = instances
	h.discovery.mu.Unlock()

	h.handler.OnDelete(instance)
}
