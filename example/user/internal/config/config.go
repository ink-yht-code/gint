package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Service ServiceConfig `yaml:"service"`
	HTTP    HTTPConfig    `yaml:"http"`
	GRPC    GRPCConfig    `yaml:"grpc"`
	DB      DBConfig      `yaml:"db"`
	Redis   RedisConfig   `yaml:"redis"`
	Log     LogConfig     `yaml:"log"`
	Outbox  OutboxConfig  `yaml:"outbox"`
}

type ServiceConfig struct {
	ID   int    `yaml:"id"`
	Name string `yaml:"name"`
}

type HTTPConfig struct {
	Enabled bool   `yaml:"enabled"`
	Addr    string `yaml:"addr"`
}

type GRPCConfig struct {
	Enabled bool   `yaml:"enabled"`
	Addr    string `yaml:"addr"`
}

type DBConfig struct {
	DSN      string `yaml:"dsn"`
	MaxOpen  int    `yaml:"max_open"`
	MaxIdle  int    `yaml:"max_idle"`
	LogLevel string `yaml:"log_level"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type LogConfig struct {
	Level    string `yaml:"level"`
	Encoding string `yaml:"encoding"`
	Output   string `yaml:"output"`
}

type OutboxConfig struct {
	Enabled      bool          `yaml:"enabled"`
	BatchSize    int           `yaml:"batch_size"`
	PollInterval time.Duration `yaml:"poll_interval"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
