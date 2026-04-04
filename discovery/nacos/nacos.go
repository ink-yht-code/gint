// Copyright 2025 ink-yht-code
//
// Proprietary License

// Package nacos 提供 Nacos 服务注册与发现功能
// 使用 HTTP API 实现，无第三方 SDK 依赖问题
package nacos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ink-yht-code/gint/discovery"
)

var (
	ErrServiceNotFound = errors.New("service not found")
	ErrInvalidConfig   = errors.New("invalid config")
	ErrRequestFailed   = errors.New("request failed")
)

// Config Nacos 配置
type Config struct {
	// 服务端地址 (如: http://localhost:8848)
	ServerAddr string
	// 命名空间
	Namespace string
	// 分组
	Group string
	// 用户名
	Username string
	// 密码
	Password string
	// HTTP 超时
	Timeout time.Duration
}

// Registry Nacos 服务注册中心
type Registry struct {
	config      *Config
	httpClient  *http.Client
	accessToken string
	tokenMu     sync.RWMutex
	mu          sync.RWMutex
	services    map[string]*discovery.ServiceInstance
	groupName   string
	stopCh      chan struct{}
	wg          sync.WaitGroup
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
	if config.ServerAddr == "" {
		return nil, ErrInvalidConfig
	}

	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}

	r := &Registry{
		config:   config,
		services: make(map[string]*discovery.ServiceInstance),
		stopCh:   make(chan struct{}),
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		groupName: config.Group,
	}

	if r.groupName == "" {
		r.groupName = "DEFAULT_GROUP"
	}

	for _, opt := range opts {
		opt(r)
	}

	// 登录获取 token
	if config.Username != "" && config.Password != "" {
		if err := r.login(); err != nil {
			return nil, fmt.Errorf("login failed: %w", err)
		}
		// 启动 token 续期
		r.wg.Add(1)
		go r.refreshToken()
	}

	return r, nil
}

// login 登录获取 accessToken
func (r *Registry) login() error {
	values := url.Values{}
	values.Set("username", r.config.Username)
	values.Set("password", r.config.Password)

	resp, err := r.httpClient.PostForm(
		r.config.ServerAddr+"/nacos/v1/auth/login",
		values,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"accessToken"`
		TokenTtl    int64  `json:"tokenTtl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	r.tokenMu.Lock()
	r.accessToken = result.AccessToken
	r.tokenMu.Unlock()

	return nil
}

// refreshToken 定期刷新 token
func (r *Registry) refreshToken() {
	defer r.wg.Done()

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = r.login()
		case <-r.stopCh:
			return
		}
	}
}

// getAccessToken 获取当前 token
func (r *Registry) getAccessToken() string {
	r.tokenMu.RLock()
	defer r.tokenMu.RUnlock()
	return r.accessToken
}

// doRequest 执行 HTTP 请求
func (r *Registry) doRequest(method, path string, values url.Values) ([]byte, error) {
	var req *http.Request
	var err error

	fullURL := r.config.ServerAddr + path

	// 添加 token
	if token := r.getAccessToken(); token != "" {
		values.Set("accessToken", token)
	}

	if method == http.MethodGet {
		fullURL += "?" + values.Encode()
		req, err = http.NewRequest(method, fullURL, nil)
	} else {
		req, err = http.NewRequest(method, fullURL, strings.NewReader(values.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	if err != nil {
		return nil, err
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: status=%d body=%s", ErrRequestFailed, resp.StatusCode, string(body))
	}

	return body, nil
}

// Register 注册服务
func (r *Registry) Register(ctx context.Context, service *discovery.ServiceInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ip, port := parseAddress(service.Address)

	values := url.Values{}
	values.Set("ip", ip)
	values.Set("port", strconv.FormatUint(port, 10))
	values.Set("serviceName", service.Name)
	values.Set("weight", strconv.Itoa(service.Weight))
	values.Set("enable", "true")
	values.Set("healthy", "true")
	values.Set("ephemeral", "true")
	values.Set("groupName", r.groupName)
	values.Set("namespaceId", r.config.Namespace)

	if len(service.Metadata) > 0 {
		metadataJSON, _ := json.Marshal(service.Metadata)
		values.Set("metadata", string(metadataJSON))
	}

	_, err := r.doRequest(http.MethodPost, "/nacos/v1/ns/instance", values)
	if err != nil {
		return err
	}

	r.services[service.ID] = service

	// 启动心跳
	r.wg.Add(1)
	go r.heartbeat(service)

	return nil
}

// heartbeat 发送心跳
func (r *Registry) heartbeat(service *discovery.ServiceInstance) {
	defer r.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	ip, port := parseAddress(service.Address)

	for {
		select {
		case <-ticker.C:
			values := url.Values{}
			values.Set("ip", ip)
			values.Set("port", strconv.FormatUint(port, 10))
			values.Set("serviceName", service.Name)
			values.Set("groupName", r.groupName)
			values.Set("namespaceId", r.config.Namespace)
			values.Set("beat", fmt.Sprintf(`{"ip":"%s","port":%d,"serviceName":"%s","weight":%d}`,
				ip, port, service.Name, service.Weight))

			_, _ = r.doRequest(http.MethodPut, "/nacos/v1/ns/instance/beat", values)

		case <-r.stopCh:
			return
		}
	}
}

// Deregister 注销服务
func (r *Registry) Deregister(ctx context.Context, service *discovery.ServiceInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ip, port := parseAddress(service.Address)

	values := url.Values{}
	values.Set("ip", ip)
	values.Set("port", strconv.FormatUint(port, 10))
	values.Set("serviceName", service.Name)
	values.Set("groupName", r.groupName)
	values.Set("namespaceId", r.config.Namespace)
	values.Set("ephemeral", "true")

	_, err := r.doRequest(http.MethodDelete, "/nacos/v1/ns/instance", values)
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
	values := url.Values{}
	values.Set("serviceName", serviceName)
	values.Set("groupName", r.groupName)
	values.Set("namespaceId", r.config.Namespace)
	values.Set("healthyOnly", "true")

	body, err := r.doRequest(http.MethodGet, "/nacos/v1/ns/instance/list", values)
	if err != nil {
		return nil, err
	}

	var result struct {
		Hosts []nacosInstance `json:"hosts"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	instances := make([]*discovery.ServiceInstance, 0, len(result.Hosts))
	for _, inst := range result.Hosts {
		instances = append(instances, &discovery.ServiceInstance{
			ID:       inst.InstanceID,
			Name:     serviceName,
			Address:  fmt.Sprintf("%s:%d", inst.IP, inst.Port),
			Weight:   int(inst.Weight),
			Metadata: inst.Metadata,
		})
	}

	return instances, nil
}

// nacosInstance Nacos 实例响应结构
type nacosInstance struct {
	InstanceID string            `json:"instanceId"`
	IP         string            `json:"ip"`
	Port       int               `json:"port"`
	Weight     float64           `json:"weight"`
	Healthy    bool              `json:"healthy"`
	Enabled    bool              `json:"enabled"`
	Metadata   map[string]string `json:"metadata"`
}

// Watch 监听服务变化
func (r *Registry) Watch(ctx context.Context, serviceName string, handler discovery.WatchHandler) error {
	r.wg.Add(1)
	go r.watch(ctx, serviceName, handler)
	return nil
}

// watch 长轮询监听服务变化
func (r *Registry) watch(ctx context.Context, serviceName string, handler discovery.WatchHandler) {
	defer r.wg.Done()

	// 先获取初始列表
	instances, _ := r.GetInstances(ctx, serviceName)
	for _, inst := range instances {
		handler.OnAdd(inst)
	}

	// 长轮询
	for {
		select {
		case <-r.stopCh:
			return
		case <-ctx.Done():
			return
		default:
		}

		values := url.Values{}
		values.Set("serviceName", serviceName)
		values.Set("groupName", r.groupName)
		values.Set("namespaceId", r.config.Namespace)
		values.Set("healthyOnly", "true")

		// 长轮询 30 秒
		values.Set("listeningConfig", "30")

		_, err := r.doRequest(http.MethodGet, "/nacos/v1/ns/instance/list", values)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		// 获取最新实例列表
		newInstances, err := r.GetInstances(ctx, serviceName)
		if err != nil {
			continue
		}

		// 简化处理：重新通知所有实例
		for _, inst := range newInstances {
			handler.OnAdd(inst)
		}
	}
}

// Close 关闭注册中心
func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	close(r.stopCh)

	// 注销所有服务
	for _, service := range r.services {
		_ = r.Deregister(context.Background(), service)
	}

	r.wg.Wait()
	r.httpClient.CloseIdleConnections()

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

// parseAddress 解析地址
func parseAddress(address string) (string, uint64) {
	for i := 0; i < len(address); i++ {
		if address[i] == ':' {
			port, _ := strconv.ParseUint(address[i+1:], 10, 64)
			return address[:i], port
		}
	}
	return address, 8080
}
