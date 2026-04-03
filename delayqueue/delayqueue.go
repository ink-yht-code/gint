// Copyright 2025 ink-yht-code
//
// Proprietary License

package delayqueue

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrQueueClosed   = errors.New("queue is closed")
	ErrMessageExists = errors.New("message already exists")
)

// Message 延迟消息
type Message struct {
	ID       string        // 消息 ID
	Topic    string        // 主题
	Body     string        // 消息体
	Delay    time.Duration // 延迟时间
	ExecTime int64         // 执行时间戳（毫秒）
}

// Handler 消息处理器
type Handler func(ctx context.Context, msg *Message) error

// DelayQueue 延迟队列接口
type DelayQueue interface {
	// Push 添加延迟消息
	Push(ctx context.Context, msg *Message) error
	// Start 启动消费者
	Start(ctx context.Context) error
	// Stop 停止消费者
	Stop() error
	// RegisterHandler 注册消息处理器
	RegisterHandler(topic string, handler Handler)
}

// RedisDelayQueue Redis 实现的延迟队列
type RedisDelayQueue struct {
	client   redis.Cmdable
	key      string
	handlers map[string]Handler
	mu       sync.RWMutex
	running  bool
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	pollSize int
}

// Option 延迟队列选项
type Option func(*RedisDelayQueue)

// WithPollSize 设置轮询大小
func WithPollSize(size int) Option {
	return func(q *RedisDelayQueue) {
		q.pollSize = size
	}
}

// NewRedisDelayQueue 创建 Redis 延迟队列
func NewRedisDelayQueue(client redis.Cmdable, key string, opts ...Option) *RedisDelayQueue {
	q := &RedisDelayQueue{
		client:   client,
		key:      key,
		handlers: make(map[string]Handler),
		pollSize: 100,
	}
	for _, opt := range opts {
		opt(q)
	}
	return q
}

// Push 添加延迟消息
func (q *RedisDelayQueue) Push(ctx context.Context, msg *Message) error {
	if msg.Delay > 0 {
		msg.ExecTime = time.Now().Add(msg.Delay).UnixMilli()
	}

	// 使用 ZADD 添加到有序集合
	member := encodeMessage(msg)
	return q.client.ZAdd(ctx, q.key, redis.Z{
		Score:  float64(msg.ExecTime),
		Member: member,
	}).Err()
}

// Start 启动消费者
func (q *RedisDelayQueue) Start(ctx context.Context) error {
	q.mu.Lock()
	if q.running {
		q.mu.Unlock()
		return nil
	}
	q.running = true
	q.mu.Unlock()

	ctx, q.cancel = context.WithCancel(ctx)

	q.wg.Add(1)
	go q.consume(ctx)

	return nil
}

// Stop 停止消费者
func (q *RedisDelayQueue) Stop() error {
	q.mu.Lock()
	if !q.running {
		q.mu.Unlock()
		return nil
	}
	q.running = false
	q.cancel()
	q.mu.Unlock()

	q.wg.Wait()
	return nil
}

// RegisterHandler 注册消息处理器
func (q *RedisDelayQueue) RegisterHandler(topic string, handler Handler) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[topic] = handler
}

// consume 消费循环
func (q *RedisDelayQueue) consume(ctx context.Context) {
	defer q.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			q.poll(ctx)
		}
	}
}

// poll 轮询到期消息
func (q *RedisDelayQueue) poll(ctx context.Context) {
	now := time.Now().UnixMilli()

	// 获取到期的消息
	results, err := q.client.ZRangeByScore(ctx, q.key, &redis.ZRangeBy{
		Min:   "0",
		Max:   formatInt64(now),
		Count: int64(q.pollSize),
	}).Result()

	if err != nil {
		return
	}

	for range results {
		// 尝试获取消息所有权（使用 Lua 脚本保证原子性）
		// 先检查消息是否仍在队列中，然后移除
		script := `
			local item = redis.call('ZRANGE', KEYS[1], ARGV[1], ARGV[1], 'BYSCORE')
			if #item > 0 then
				redis.call('ZREM', KEYS[1], item[1])
				return item[1]
			end
			return nil
		`

		msgStr, err := q.client.Eval(ctx, script, []string{q.key}, now).Result()
		if err != nil || msgStr == nil {
			continue
		}

		msg, err := decodeMessage(msgStr.(string))
		if err != nil {
			continue
		}

		// 处理消息
		q.handleMessage(ctx, msg)
	}
}

// handleMessage 处理消息
func (q *RedisDelayQueue) handleMessage(ctx context.Context, msg *Message) {
	q.mu.RLock()
	handler, ok := q.handlers[msg.Topic]
	q.mu.RUnlock()

	if !ok {
		return
	}

	// 执行处理器
	_ = handler(ctx, msg)
}

// MemoryDelayQueue 内存实现的延迟队列（用于测试）
type MemoryDelayQueue struct {
	messages []*Message
	handlers map[string]Handler
	mu       sync.RWMutex
	running  bool
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewMemoryDelayQueue 创建内存延迟队列
func NewMemoryDelayQueue() *MemoryDelayQueue {
	return &MemoryDelayQueue{
		messages: make([]*Message, 0),
		handlers: make(map[string]Handler),
	}
}

// Push 添加延迟消息
func (q *MemoryDelayQueue) Push(ctx context.Context, msg *Message) error {
	if msg.Delay > 0 {
		msg.ExecTime = time.Now().Add(msg.Delay).UnixMilli()
	} else if msg.ExecTime == 0 {
		msg.ExecTime = time.Now().UnixMilli()
	}

	q.mu.Lock()
	q.messages = append(q.messages, msg)
	q.mu.Unlock()
	return nil
}

// Start 启动消费者
func (q *MemoryDelayQueue) Start(ctx context.Context) error {
	q.mu.Lock()
	if q.running {
		q.mu.Unlock()
		return nil
	}
	q.running = true
	q.mu.Unlock()

	ctx, q.cancel = context.WithCancel(ctx)
	q.wg.Add(1)
	go q.consume(ctx)

	return nil
}

// Stop 停止消费者
func (q *MemoryDelayQueue) Stop() error {
	q.mu.Lock()
	if !q.running {
		q.mu.Unlock()
		return nil
	}
	q.running = false
	q.cancel()
	q.mu.Unlock()

	q.wg.Wait()
	return nil
}

// RegisterHandler 注册消息处理器
func (q *MemoryDelayQueue) RegisterHandler(topic string, handler Handler) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[topic] = handler
}

// consume 消费循环
func (q *MemoryDelayQueue) consume(ctx context.Context) {
	defer q.wg.Done()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			q.poll(ctx)
		}
	}
}

// poll 轮询到期消息
func (q *MemoryDelayQueue) poll(ctx context.Context) {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now().UnixMilli()
	var remaining []*Message

	for _, msg := range q.messages {
		if msg.ExecTime <= now {
			// 处理消息
			if handler, ok := q.handlers[msg.Topic]; ok {
				go handler(ctx, msg)
			}
		} else {
			remaining = append(remaining, msg)
		}
	}

	q.messages = remaining
}

// encodeMessage 编码消息
func encodeMessage(msg *Message) string {
	return msg.ID + ":" + msg.Topic + ":" + msg.Body + ":" + formatInt64(msg.ExecTime)
}

// decodeMessage 解码消息
func decodeMessage(s string) (*Message, error) {
	parts := splitMessage(s)
	if len(parts) < 4 {
		return nil, errors.New("invalid message format")
	}

	execTime := parseInt64(parts[3])
	return &Message{
		ID:       parts[0],
		Topic:    parts[1],
		Body:     parts[2],
		ExecTime: execTime,
	}, nil
}

// 辅助函数
func formatInt64(n int64) string {
	return time.UnixMilli(n).Format("20060102150405")
}

func parseInt64(s string) int64 {
	t, _ := time.Parse("20060102150405", s)
	return t.UnixMilli()
}

func splitMessage(s string) []string {
	result := make([]string, 0, 4)
	start := 0
	for i := 0; i < len(s) && len(result) < 3; i++ {
		if s[i] == ':' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}
