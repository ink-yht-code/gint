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

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ink-yht-code/gint/logger"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Config 配置管理器
type Config struct {
	data    map[string]any
	sources []Source
	mu      sync.RWMutex
}

// Source 配置源接口
type Source interface {
	// Load 加载配置
	Load() (map[string]any, error)
	// Name 配置源名称
	Name() string
}

// Option 配置选项
type Option func(*Config)

// New 创建配置管理器
func New(opts ...Option) *Config {
	c := &Config{
		data:    make(map[string]any),
		sources: make([]Source, 0),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Load 加载所有配置源
func (c *Config) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var loadErrs []error
	for _, source := range c.sources {
		data, err := source.Load()
		if err != nil {
			logger.Warn("配置源加载失败",
				zap.String("source", source.Name()),
				zap.Error(err))
			loadErrs = append(loadErrs, fmt.Errorf("%s: %w", source.Name(), err))
			continue
		}
		c.merge(data)
		logger.Info("配置源加载成功", zap.String("source", source.Name()))
	}

	if len(loadErrs) > 0 {
		return errors.Join(loadErrs...)
	}
	return nil
}

// merge 合并配置（后面的覆盖前面的）
func (c *Config) merge(data map[string]any) {
	for k, v := range data {
		c.data[k] = v
	}
}

// Get 获取配置值
func (c *Config) Get(key string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := strings.Split(key, ".")
	value := any(c.data)

	for _, k := range keys {
		switch v := value.(type) {
		case map[string]any:
			var ok bool
			value, ok = v[k]
			if !ok {
				return nil
			}
		default:
			return nil
		}
	}

	return value
}

// GetString 获取字符串配置
func (c *Config) GetString(key string, defaultValue ...string) string {
	val := c.Get(key)
	if val == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", val)
}

// GetInt 获取整数配置
func (c *Config) GetInt(key string, defaultValue ...int) int {
	val := c.Get(key)
	if val == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

// GetInt64 获取 int64 配置
func (c *Config) GetInt64(key string, defaultValue ...int64) int64 {
	val := c.Get(key)
	if val == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	switch v := val.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}

// GetBool 获取布尔配置
func (c *Config) GetBool(key string, defaultValue ...bool) bool {
	val := c.Get(key)
	if val == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	case string:
		return strings.ToLower(v) == "true" || v == "1"
	default:
		return false
	}
}

// GetFloat64 获取浮点配置
func (c *Config) GetFloat64(key string, defaultValue ...float64) float64 {
	val := c.Get(key)
	if val == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

// GetStringSlice 获取字符串数组配置
func (c *Config) GetStringSlice(key string, defaultValue ...[]string) []string {
	val := c.Get(key)
	if val == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return nil
	}

	switch v := val.(type) {
	case []string:
		return v
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

// GetStringMap 获取字符串映射配置
func (c *Config) GetStringMap(key string, defaultValue ...map[string]string) map[string]string {
	val := c.Get(key)
	if val == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return nil
	}

	if m, ok := val.(map[string]any); ok {
		result := make(map[string]string)
		for k, v := range m {
			if s, ok := v.(string); ok {
				result[k] = s
			}
		}
		return result
	}

	return nil
}

// Unmarshal 将配置解析到结构体
func (c *Config) Unmarshal(key string, target any) error {
	val := c.Get(key)
	if val == nil {
		return fmt.Errorf("配置项 %s 不存在", key)
	}

	// 先转为 JSON 再解析
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, target)
}

// --- 配置源实现 ---

// FileSource 文件配置源
type FileSource struct {
	path string
}

// WithFile 添加文件配置源
func WithFile(path string) Option {
	return func(c *Config) {
		c.sources = append(c.sources, &FileSource{path: path})
	}
}

// Load 加载文件配置
func (s *FileSource) Load() (map[string]any, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}

	result := make(map[string]any)
	ext := strings.ToLower(filepath.Ext(s.path))

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &result); err != nil {
			return nil, err
		}
	case ".json":
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("不支持的配置文件格式: %s", ext)
	}

	return result, nil
}

// Name 返回配置源名称
func (s *FileSource) Name() string {
	return fmt.Sprintf("file:%s", s.path)
}

// EnvSource 环境变量配置源
type EnvSource struct {
	prefix string
}

// WithEnv 添加环境变量配置源
// prefix: 环境变量前缀，如 "APP_" 会读取 APP_DB_HOST 转换为 db.host
func WithEnv(prefix string) Option {
	return func(c *Config) {
		c.sources = append(c.sources, &EnvSource{prefix: prefix})
	}
}

// Load 加载环境变量配置
func (s *EnvSource) Load() (map[string]any, error) {
	result := make(map[string]any)

	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, s.prefix) {
			continue
		}

		// 解析 KEY=VALUE
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimPrefix(parts[0], s.prefix)
		value := parts[1]

		// 转换为小写并替换下划线为点
		// APP_DB_HOST -> db.host
		keys := strings.Split(strings.ToLower(key), "_")
		s.setNested(result, keys, value)
	}

	return result, nil
}

// setNested 设置嵌套值
func (s *EnvSource) setNested(data map[string]any, keys []string, value string) {
	for i, k := range keys {
		if i == len(keys)-1 {
			data[k] = value
			return
		}

		if _, ok := data[k]; !ok {
			data[k] = make(map[string]any)
		}

		if nested, ok := data[k].(map[string]any); ok {
			data = nested
		}
	}
}

// Name 返回配置源名称
func (s *EnvSource) Name() string {
	return fmt.Sprintf("env:%s", s.prefix)
}

// MapSource 内存映射配置源
type MapSource struct {
	name string
	data map[string]any
}

// WithMap 添加内存映射配置源
func WithMap(data map[string]any, name ...string) Option {
	n := "map"
	if len(name) > 0 {
		n = name[0]
	}
	return func(c *Config) {
		c.sources = append(c.sources, &MapSource{name: n, data: data})
	}
}

// Load 加载内存配置
func (s *MapSource) Load() (map[string]any, error) {
	return s.data, nil
}

// Name 返回配置源名称
func (s *MapSource) Name() string {
	return fmt.Sprintf("map:%s", s.name)
}

// --- 全局配置 ---

var defaultConfig *Config
var configOnce sync.Once

// Init 初始化默认配置
func Init(opts ...Option) {
	configOnce.Do(func() {
		defaultConfig = New(opts...)
		defaultConfig.Load()
	})
}

// Get 从默认配置获取值
func Get(key string) any {
	if defaultConfig == nil {
		return nil
	}
	return defaultConfig.Get(key)
}

// GetString 从默认配置获取字符串
func GetString(key string, defaultValue ...string) string {
	if defaultConfig == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return ""
	}
	return defaultConfig.GetString(key, defaultValue...)
}

// GetInt 从默认配置获取整数
func GetInt(key string, defaultValue ...int) int {
	if defaultConfig == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return defaultConfig.GetInt(key, defaultValue...)
}

// GetBool 从默认配置获取布尔值
func GetBool(key string, defaultValue ...bool) bool {
	if defaultConfig == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return false
	}
	return defaultConfig.GetBool(key, defaultValue...)
}

// Default 获取默认配置管理器
func Default() *Config {
	return defaultConfig
}

// --- 快捷加载函数 ---

// LoadFile 快捷加载单个配置文件
func LoadFile(path string) (*Config, error) {
	c := New(WithFile(path))
	if err := c.Load(); err != nil {
		return nil, err
	}
	return c, nil
}

// LoadApp 快捷加载应用配置
// 默认加载 config.yaml 和环境变量（APP_ 前缀）
func LoadApp(configPath ...string) (*Config, error) {
	path := "config.yaml"
	if len(configPath) > 0 {
		path = configPath[0]
	}

	c := New(
		WithFile(path),
		WithEnv("APP_"),
	)

	if err := c.Load(); err != nil {
		return nil, err
	}

	defaultConfig = c
	return c, nil
}
