// Copyright 2025 ink-yht-code
//
// Proprietary License

// Package etcd 提供基于 etcd 的服务注册与发现功能。
// Registry 同时实现了 discovery.RegistryDiscovery 接口，
// 可直接通过 discovery.SetDefault(r) 设为全局注册中心。
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
	ErrNotRegistered = errors.New("etcd: service not registered")
	ErrKeyNotFound   = errors.New("etcd: key not found")
)

// Registry 基于 etcd 的服务注册与发现，实现 discovery.RegistryDiscovery 接口。
type Registry struct {
	client  *clientv3.Client
	kv      clientv3.KV
	lease   clientv3.Lease
	prefix  string
	ttl     int64
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.RWMutex
	// leases 记录每个实例对应的租约，key 为 serviceKey
	leases  map[string]clientv3.LeaseID
	// services 记录已注册的实例，key 为 serviceKey
	services map[string]*discovery.ServiceInstance
}

// Option 注册中心选项
type Option func(*Registry)

// WithPrefix 设置 key 前缀，默认 "/services/"
func WithPrefix(prefix string) Option {
	return func(r *Registry) {
		r.prefix = prefix
	}
}

// WithTTL 设置租约 TTL（秒），默认 30
func WithTTL(ttl int64) Option {
	return func(r *Registry) {
		r.ttl = ttl
	}
}

// New 创建 etcd 注册中心。
//
// 用法：
//
//	r, err := etcd.New([]string{"127.0.0.1:2379"})
//	discovery.SetDefault(r)
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
		prefix:   "/services/",
		ttl:      30,
		leases:   make(map[string]clientv3.LeaseID),
		services: make(map[string]*discovery.ServiceInstance),
	}
	for _, opt := range opts {
		opt(r)
	}
	r.ctx, r.cancel = context.WithCancel(context.Background())
	return r, nil
}

// ---- discovery.Registry 接口 ----

// Register 注册服务实例，使用独立租约并自动保活。
func (r *Registry) Register(ctx context.Context, instance *discovery.ServiceInstance) error {
	key := r.serviceKey(instance.Name, instance.ID)

	value, err := json.Marshal(instance)
	if err != nil {
		return err
	}

	// 为每个实例创建独立租约
	resp, err := r.lease.Grant(ctx, r.ttl)
	if err != nil {
		return err
	}
	leaseID := resp.ID

	if _, err = r.kv.Put(ctx, key, string(value), clientv3.WithLease(leaseID)); err != nil {
		_, _ = r.lease.Revoke(context.Background(), leaseID)
		return err
	}

	r.mu.Lock()
	r.leases[key] = leaseID
	r.services[key] = instance
	r.mu.Unlock()

	// 启动保活协程
	r.wg.Add(1)
	go r.keepAlive(key, leaseID)

	return nil
}

// Deregister 注销服务实例，撤销租约并删除 key。
func (r *Registry) Deregister(ctx context.Context, instance *discovery.ServiceInstance) error {
	key := r.serviceKey(instance.Name, instance.ID)

	r.mu.Lock()
	leaseID, ok := r.leases[key]
	if ok {
		delete(r.leases, key)
		delete(r.services, key)
	}
	r.mu.Unlock()

	if ok && leaseID != 0 {
		_, _ = r.lease.Revoke(ctx, leaseID)
	}
	_, err := r.kv.Delete(ctx, key)
	return err
}

// Heartbeat 手动续约（通常不需要调用，Register 已自动保活）。
func (r *Registry) Heartbeat(ctx context.Context, instance *discovery.ServiceInstance) error {
	key := r.serviceKey(instance.Name, instance.ID)

	r.mu.RLock()
	leaseID, ok := r.leases[key]
	r.mu.RUnlock()

	if !ok {
		return ErrNotRegistered
	}
	_, err := r.lease.KeepAliveOnce(ctx, leaseID)
	return err
}

// ---- discovery.Discovery 接口 ----

// GetInstance 获取单个健康实例（返回第一个）。
func (r *Registry) GetInstance(ctx context.Context, serviceName string) (*discovery.ServiceInstance, error) {
	instances, err := r.GetInstances(ctx, serviceName)
	if err != nil {
		return nil, err
	}
	if len(instances) == 0 {
		return nil, discovery.ErrNoInstance
	}
	return instances[0], nil
}

// GetInstances 获取指定服务的所有实例。
func (r *Registry) GetInstances(ctx context.Context, serviceName string) ([]*discovery.ServiceInstance, error) {
	prefix := r.prefix + serviceName + "/"
	resp, err := r.kv.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	instances := make([]*discovery.ServiceInstance, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var inst discovery.ServiceInstance
		if err := json.Unmarshal(kv.Value, &inst); err != nil {
			continue
		}
		instances = append(instances, &inst)
	}
	return instances, nil
}

// Watch 监听服务实例变化，返回一个 channel，每次变化推送最新实例列表。
// 调用方应在不再需要时 cancel 传入的 ctx 以释放资源。
func (r *Registry) Watch(ctx context.Context, serviceName string) (<-chan []*discovery.ServiceInstance, error) {
	prefix := r.prefix + serviceName + "/"
	ch := make(chan []*discovery.ServiceInstance, 8)

	// 先推送一次当前快照
	instances, err := r.GetInstances(ctx, serviceName)
	if err != nil {
		return nil, err
	}
	ch <- instances

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer close(ch)

		watcher := clientv3.NewWatcher(r.client)
		defer watcher.Close()

		watchCh := watcher.Watch(ctx, prefix, clientv3.WithPrefix())
		for {
			select {
			case <-ctx.Done():
				return
			case <-r.ctx.Done():
				return
			case resp, ok := <-watchCh:
				if !ok {
					return
				}
				if resp.Err() != nil {
					continue
				}
				// 有变化时重新拉取完整列表
				latest, err := r.GetInstances(ctx, serviceName)
				if err != nil {
					continue
				}
				select {
				case ch <- latest:
				default:
					// 消费方来不及消费时丢弃旧快照，保留最新
					select {
					case <-ch:
					default:
					}
					ch <- latest
				}
			}
		}
	}()

	return ch, nil
}

// Close 关闭注册中心，撤销所有租约并等待后台协程退出。
func (r *Registry) Close() error {
	r.cancel()

	// 撤销所有租约（会触发 etcd 自动删除对应 key）
	r.mu.RLock()
	leases := make(map[string]clientv3.LeaseID, len(r.leases))
	for k, v := range r.leases {
		leases[k] = v
	}
	r.mu.RUnlock()

	for _, leaseID := range leases {
		_, _ = r.lease.Revoke(context.Background(), leaseID)
	}

	r.wg.Wait()
	return r.client.Close()
}

// ---- 内部方法 ----

// keepAlive 为单个实例的租约持续保活，租约过期时自动重新注册。
func (r *Registry) keepAlive(key string, leaseID clientv3.LeaseID) {
	defer r.wg.Done()

	interval := time.Duration(r.ttl/3) * time.Second
	if interval < time.Second {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			_, err := r.lease.KeepAliveOnce(r.ctx, leaseID)
			if err == nil {
				continue
			}
			// 租约过期，重新申请并写回
			r.mu.RLock()
			instance, ok := r.services[key]
			r.mu.RUnlock()
			if !ok {
				return // 已被 Deregister，退出
			}

			newResp, err := r.lease.Grant(r.ctx, r.ttl)
			if err != nil {
				continue
			}
			newLeaseID := newResp.ID

			value, _ := json.Marshal(instance)
			if _, err = r.kv.Put(r.ctx, key, string(value), clientv3.WithLease(newLeaseID)); err != nil {
				_, _ = r.lease.Revoke(context.Background(), newLeaseID)
				continue
			}

			r.mu.Lock()
			r.leases[key] = newLeaseID
			r.mu.Unlock()
			leaseID = newLeaseID
		}
	}
}

func (r *Registry) serviceKey(name, id string) string {
	return r.prefix + name + "/" + id
}
