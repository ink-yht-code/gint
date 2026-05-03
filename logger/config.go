// 版权所有 2025 ink-yht-code
//
// 专有许可
//
// 重要说明：本软件并非开源软件。
// 未经版权持有人事先书面许可，
// 不得使用、复制、修改、合并、发布、分发、再许可，
// 也不得全部或部分出售本文件的副本。
//
// 本软件按“现状”提供，不附带任何形式的担保。

package logger

// Config 定义日志配置。
type Config struct {
	// Level 日志级别，可选值：debug、info、warn、error。
	Level string `yaml:"level" json:"level"`

	// Format 日志格式，可选值：json、text。
	Format string `yaml:"format" json:"format"`

	// Output 输出位置，可选值：stdout、file、both。
	Output string `yaml:"output" json:"output"`

	// Filename 在输出到文件时指定日志文件路径。
	Filename string `yaml:"filename" json:"filename"`

	// MaxSize 单个日志文件的最大大小，单位为 MB。
	MaxSize int `yaml:"max_size" json:"max_size"`

	// MaxBackups 保留的历史日志文件数量。
	MaxBackups int `yaml:"max_backups" json:"max_backups"`

	// MaxAge 历史日志文件的保留天数。
	MaxAge int `yaml:"max_age" json:"max_age"`

	// Compress 控制轮转后的日志文件是否压缩。
	Compress bool `yaml:"compress" json:"compress"`

	// EnableDB 控制是否同时写入数据库日志。
	EnableDB bool `yaml:"enable_db" json:"enable_db"`

	// DBLevel 指定写入数据库日志的最低级别。
	DBLevel string `yaml:"db_level" json:"db_level"`
}

// DefaultConfig 返回一份安全的默认日志配置。
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
