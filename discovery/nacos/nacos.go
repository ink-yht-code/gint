// Copyright 2025 ink-yht-code
//
// Proprietary License

package nacos

import (
	"context"
	"errors"
	"sync"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	"github.com/ink-yht-code/gint/discovery"
)

var (
	ErrServiceNotFound = errors.New("service not found")
	ErrInvalidConfig   = errors.New("invalid config")
)

// Config Nacos 配置
type Config struct {
	// 服务端地址
	ServerAddrs []string
	// 命名空间
	Namespace string
	// 分组
	Group string
	// 用户名
	Username string
	// 密码
	Password string
	// 缓存目录
	CacheDir string
	// 日志目录
	LogDir string
	// 是否使用缓存
	UpdateCacheWhenEmpty bool
}

// Registry Nacos 服务注册中心
type Registry struct {
	client    naming_client.INamingClient
	config    *Config
	mu        sync.RWMutex
	services  map[string]*discovery.ServiceInstance
	groupName string
}

// Option 注册中心选项
type Option func(*Registry)

// WithGroup 设置分组名
func WithGroup(group string) Option {
	return func(r *Registry) {
		r.groupName = group
	}
}

// New 创建 Nacos 服务注册中心
func New(config *Config, opts ...Option) (*Registry, error) {
	if len(config.ServerAddrs) == 0 {
		return nil, ErrInvalidConfig
	}

	// 构建服务端配置
	serverConfigs := make([]constant.ServerConfig, 0, len(config.ServerAddrs))
	for _, addr := range config.ServerAddrs {
		serverConfigs = append(serverConfigs, constant.ServerConfig{
			IpAddr: addr,
			Port:   8848, // 默认端口
		})
	}

	// 构建客户端配置
	clientConfig := constant.ClientConfig{
		NamespaceId:          config.Namespace,
		Username:             config.Username,
		Password:             config.Password,
		CacheDir:             config.CacheDir,
		LogDir:               config.LogDir,
		UpdateCacheWhenEmpty: config.UpdateCacheWhenEmpty,
	}

	// 创建命名客户端
	client, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return nil, err
	}

	r := &Registry{
		client:    client,
		config:    config,
		services:  make(map[string]*discovery.ServiceInstance),
		groupName: config.Group,
	}

	if r.groupName == "" {
		r.groupName = "DEFAULT_GROUP"
	}

	for _, opt := range opts {
		opt(r)
	}

	return r, nil
}

// Register 注册服务
func (r *Registry) Register(ctx context.Context, service *discovery.ServiceInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.client.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          getIPFromAddress(service.Address),
		Port:        getPortFromAddress(service.Address),
		ServiceName: service.Name,
		Weight:      float64(service.Weight),
		Enable:      true,
		Healthy:     true,
		Metadata:    service.Metadata,
		GroupName:   r.groupName,
		Ephemeral:   true,
	})

	if err != nil {
		return err
	}

	r.services[service.ID] = service
	return nil
}

// Deregister 注销服务
func (r *Registry) Deregister(ctx context.Context, service *discovery.ServiceInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.client.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          getIPFromAddress(service.Address),
		Port:        getPortFromAddress(service.Address),
		ServiceName: service.Name,
		GroupName:   r.groupName,
		Ephemeral:   true,
	})

	if err != nil {
		return err
	}

	delete(r.services, service.ID)
	return nil
}

// GetInstance 获取单个服务实例
func (r *Registry) GetInstance(ctx context.Context, serviceName string) (*discovery.ServiceInstance, error) {
	instances, err := r.GetInstances(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	if len(instances) == 0 {
		return nil, ErrServiceNotFound
	}

	return instances[0], nil
}

// GetInstances 获取所有服务实例
func (r *Registry) GetInstances(ctx context.Context, serviceName string) ([]*discovery.ServiceInstance, error) {
	resp, err := r.client.SelectInstances(vo.SelectInstancesParam{
		ServiceName: serviceName,
		GroupName:   r.groupName,
		HealthyOnly: true,
	})

	if err != nil {
		return nil, err
	}

	instances := make([]*discovery.ServiceInstance, 0, len(resp))
	for _, instance := range resp {
		instances = append(instances, &discovery.ServiceInstance{
			ID:       instance.InstanceId,
			Name:     instance.ServiceName,
			Address:  formatAddress(instance.Ip, instance.Port),
			Weight:   int(instance.Weight),
			Metadata: instance.Metadata,
		})
	}

	return instances, nil
}

// Watch 监听服务变化
func (r *Registry) Watch(ctx context.Context, serviceName string, handler discovery.WatchHandler) error {
	return r.client.Subscribe(&vo.SubscribeParam{
		ServiceName: serviceName,
		GroupName:   r.groupName,
		SubscribeCallback: func(services []model.Instance, err error) {
			if err != nil {
				return
			}

			for _, instance := range services {
				serviceInstance := &discovery.ServiceInstance{
					ID:       instance.InstanceId,
					Name:     serviceName,
					Address:  formatAddress(instance.Ip, instance.Port),
					Weight:   int(instance.Weight),
					Metadata: instance.Metadata,
				}

				if instance.Enable {
					handler.OnAdd(serviceInstance)
				} else {
					handler.OnDelete(serviceInstance)
				}
			}
		},
	})
}

// Close 关闭注册中心
func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 注销所有服务
	for _, service := range r.services {
		_, _ = r.client.DeregisterInstance(vo.DeregisterInstanceParam{
			Ip:          getIPFromAddress(service.Address),
			Port:        getPortFromAddress(service.Address),
			ServiceName: service.Name,
			GroupName:   r.groupName,
			Ephemeral:   true,
		})
	}

	return nil
}

// Discovery Nacos 服务发现客户端
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

	// 从 Nacos 获取
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
	return d.registry.Watch(ctx, serviceName, &discoveryHandler{
		discovery: d,
		handler:   handler,
		service:   serviceName,
	})
}

// discoveryHandler 带缓存更新的处理器
type discoveryHandler struct {
	discovery *Discovery
	handler   discovery.WatchHandler
	service   string
}

func (h *discoveryHandler) OnAdd(instance *discovery.ServiceInstance) {
	h.discovery.mu.Lock()
	instances := h.discovery.cache[h.service]
	instances = append(instances, instance)
	h.discovery.cache[h.service] = instances
	h.discovery.mu.Unlock()

	h.handler.OnAdd(instance)
}

func (h *discoveryHandler) OnDelete(instance *discovery.ServiceInstance) {
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

// 辅助函数

// getIPFromAddress 从地址中提取 IP
func getIPFromAddress(address string) string {
	for i := 0; i < len(address); i++ {
		if address[i] == ':' {
			return address[:i]
		}
	}
	return address
}

// getPortFromAddress 从地址中提取端口
func getPortFromAddress(address string) uint64 {
	for i := 0; i < len(address); i++ {
		if address[i] == ':' {
			var port uint64
			for j := i + 1; j < len(address); j++ {
				port = port*10 + uint64(address[j]-'0')
			}
			return port
		}
	}
	return 8080
}

// formatAddress 格式化地址
func formatAddress(ip string, port uint64) string {
	return ip + ":" + formatUint(port)
}

// formatUint 快速格式化 uint64
func formatUint(n uint64) string {
	if n == 0 {
		return "0"
	}

	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte(n%10) + '0'
		n /= 10
	}

	return string(buf[i:])
}
