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

package logger

// Config 日志配置
type Config struct {
	// Level 日志级别: debug, info, warn, error
	Level string `yaml:"level" json:"level"`

	// Format 日志格式: json, text
	Format string `yaml:"format" json:"format"`

	// Output 输出目标: stdout, file, both
	Output string `yaml:"output" json:"output"`

	// Filename 日志文件路径（当 Output 为 file 或 both 时有效）
	Filename string `yaml:"filename" json:"filename"`

	// MaxSize 单个日志文件最大大小（MB）
	MaxSize int `yaml:"max_size" json:"max_size"`

	// MaxBackups 保留的旧日志文件最大数量
	MaxBackups int `yaml:"max_backups" json:"max_backups"`

	// MaxAge 保留旧日志文件的最大天数
	MaxAge int `yaml:"max_age" json:"max_age"`

	// Compress 是否压缩旧日志文件
	Compress bool `yaml:"compress" json:"compress"`

	// EnableDB 是否启用数据库日志
	EnableDB bool `yaml:"enable_db" json:"enable_db"`

	// DBLevel 数据库日志最低级别
	DBLevel string `yaml:"db_level" json:"db_level"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		Level:      "info",
		Format:     "json",
		Output:     "stdout",
		Filename:   "./logs/app.log",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   false,
		EnableDB:   false,
		DBLevel:    "info",
	}
}
