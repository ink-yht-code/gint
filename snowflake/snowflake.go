// Copyright 2025 ink-yht-code
//
// Proprietary License

package snowflake

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrInvalidNodeID  = errors.New("invalid node id: must be between 0 and 1023")
	ErrInvalidEpoch   = errors.New("invalid epoch: epoch cannot be in the future")
	ErrClockMovedBack = errors.New("clock moved backwards, refusing to generate id")
	ErrIDOverflow     = errors.New("sequence overflow, wait for next millisecond")
)

const (
	// DefaultEpoch 默认起始时间 (2024-01-01 00:00:00 UTC)
	DefaultEpoch int64 = 1704067200000

	// 位分配
	nodeIDBits   uint64 = 10 // 节点 ID 位数
	sequenceBits uint64 = 12 // 序列号位数

	// 最大值
	maxNodeID   int64 = -1 ^ (-1 << nodeIDBits)   // 1023
	maxSequence int64 = -1 ^ (-1 << sequenceBits) // 4095

	// 位移
	nodeIDShift    uint64 = sequenceBits
	timestampShift uint64 = sequenceBits + nodeIDBits
)

// Generator 雪花算法 ID 生成器
type Generator struct {
	mu sync.Mutex

	epoch     int64 // 起始时间戳（毫秒）
	nodeID    int64 // 节点 ID
	timestamp int64 // 上次生成 ID 的时间戳
	sequence  int64 // 序列号
}

// Option 生成器选项
type Option func(*Generator)

// WithEpoch 设置起始时间
func WithEpoch(epoch int64) Option {
	return func(g *Generator) {
		g.epoch = epoch
	}
}

// New 创建 ID 生成器
func New(nodeID int64, opts ...Option) (*Generator, error) {
	if nodeID < 0 || nodeID > maxNodeID {
		return nil, ErrInvalidNodeID
	}

	g := &Generator{
		epoch:  DefaultEpoch,
		nodeID: nodeID,
	}

	for _, opt := range opts {
		opt(g)
	}

	// 检查 epoch 不能是未来时间
	if g.epoch > time.Now().UnixMilli() {
		return nil, ErrInvalidEpoch
	}

	return g, nil
}

// Generate 生成唯一 ID
func (g *Generator) Generate() (int64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now().UnixMilli() - g.epoch

	// 时钟回拨检测
	if now < g.timestamp {
		return 0, ErrClockMovedBack
	}

	// 同一毫秒内
	if now == g.timestamp {
		g.sequence = (g.sequence + 1) & maxSequence
		// 序列号溢出，等待下一毫秒
		if g.sequence == 0 {
			now = g.waitNextMillis(now)
		}
	} else {
		// 新毫秒，序列号重置
		g.sequence = 0
	}

	g.timestamp = now

	// 组装 ID: 时间戳(41位) | 节点ID(10位) | 序列号(12位)
	id := (now << timestampShift) | (g.nodeID << nodeIDShift) | g.sequence
	return id, nil
}

// GenerateString 生成字符串形式的 ID
func (g *Generator) GenerateString() (string, error) {
	id, err := g.Generate()
	if err != nil {
		return "", err
	}
	return int64ToString(id), nil
}

// Parse 解析 ID
func Parse(id int64) *ID {
	return &ID{
		Timestamp: id >> timestampShift,
		NodeID:    (id >> nodeIDShift) & maxNodeID,
		Sequence:  id & maxSequence,
	}
}

// ParseString 解析字符串 ID
func ParseString(s string) (*ID, error) {
	id, err := stringToInt64(s)
	if err != nil {
		return nil, err
	}
	return Parse(id), nil
}

// ID 解析后的 ID 结构
type ID struct {
	Timestamp int64 // 时间戳部分
	NodeID    int64 // 节点 ID 部分
	Sequence  int64 // 序列号部分
}

// Time 返回 ID 的生成时间
func (id *ID) Time() time.Time {
	// 这里需要外部传入 epoch 才能计算，简化处理
	return time.UnixMilli(id.Timestamp + DefaultEpoch)
}

// waitNextMillis 等待下一毫秒
func (g *Generator) waitNextMillis(lastTimestamp int64) int64 {
	now := time.Now().UnixMilli() - g.epoch
	for now <= lastTimestamp {
		time.Sleep(100 * time.Microsecond)
		now = time.Now().UnixMilli() - g.epoch
	}
	return now
}

// int64ToString 快速 int64 转字符串
func int64ToString(n int64) string {
	if n == 0 {
		return "0"
	}

	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}

	for n > 0 {
		i--
		buf[i] = byte(n%10) + '0'
		n /= 10
	}

	if neg {
		i--
		buf[i] = '-'
	}

	return string(buf[i:])
}

// stringToInt64 字符串转 int64
func stringToInt64(s string) (int64, error) {
	if len(s) == 0 {
		return 0, errors.New("empty string")
	}

	var n int64
	neg := false
	start := 0

	if s[0] == '-' {
		neg = true
		start = 1
	}

	for i := start; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, errors.New("invalid character")
		}
		n = n*10 + int64(s[i]-'0')
	}

	if neg {
		n = -n
	}

	return n, nil
}

// NodeIDFromIP 根据 IP 地址生成节点 ID
// 使用 IP 的后两段来生成，适用于单机多实例场景
func NodeIDFromIP(ip uint32) int64 {
	// 使用后 16 位，取模 1024
	return int64((ip & 0xFFFF) % 1024)
}

// NodeIDFromMac 根据 MAC 地址生成节点 ID
func NodeIDFromMac(mac uint64) int64 {
	// 使用 MAC 地址后 10 位
	return int64(mac & 0x3FF)
}
