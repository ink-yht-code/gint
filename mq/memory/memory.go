// Copyright 2025 ink-yht-code
//
// Proprietary License

// Package memory 提供基于内存的 MQ 实现，适合单元测试和本地开发，无需外部依赖。
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/ink-yht-code/gint/mq"
)

const (
	defaultPartitions = 3
	defaultChanSize   = 1000
	pollInterval      = 10 * time.Millisecond
)

// MemoryMQ 内存 MQ，实现 mq.MQ 接口
type MemoryMQ struct {
	mu        sync.RWMutex
	closed    bool
	topics    map[string]*memTopic
	producers []*MemoryProducer
	consumers []*MemoryConsumer
}

// NewMQ 创建内存 MQ
//
//	m := memory.NewMQ()
//	_ = m.CreateTopic(ctx, "orders", 3)
//	p, _ := m.Producer("orders")
//	c, _ := m.Consumer("orders", "group-a")
func NewMQ() mq.MQ {
	return &MemoryMQ{
		topics: make(map[string]*memTopic),
	}
}

func (m *MemoryMQ) CreateTopic(ctx context.Context, name string, partitions int) error {
	if name == "" {
		return mq.ErrInvalidTopic
	}
	if partitions <= 0 {
		return mq.ErrInvalidPartition
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return mq.ErrMQClosed
	}
	if _, ok := m.topics[name]; !ok {
		m.topics[name] = newMemTopic(name, partitions)
	}
	return nil
}

func (m *MemoryMQ) DeleteTopics(ctx context.Context, topics ...string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return mq.ErrMQClosed
	}
	for _, name := range topics {
		delete(m.topics, name)
	}
	return nil
}

func (m *MemoryMQ) Producer(topicName string) (mq.Producer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, mq.ErrMQClosed
	}
	t := m.getOrCreateTopic(topicName)
	p := &MemoryProducer{t: t}
	m.producers = append(m.producers, p)
	return p, nil
}

func (m *MemoryMQ) Consumer(topicName string, groupID string) (mq.Consumer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, mq.ErrMQClosed
	}
	t := m.getOrCreateTopic(topicName)
	c := newMemoryConsumer(t, groupID)
	m.consumers = append(m.consumers, c)
	return c, nil
}

func (m *MemoryMQ) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	for _, p := range m.producers {
		_ = p.Close()
	}
	for _, c := range m.consumers {
		_ = c.Close()
	}
	return nil
}

func (m *MemoryMQ) getOrCreateTopic(name string) *memTopic {
	t, ok := m.topics[name]
	if !ok {
		t = newMemTopic(name, defaultPartitions)
		m.topics[name] = t
	}
	return t
}

// --- memTopic ---

type memTopic struct {
	name       string
	partitions []*memPartition
}

func newMemTopic(name string, partitions int) *memTopic {
	pts := make([]*memPartition, partitions)
	for i := range pts {
		pts[i] = &memPartition{
			msgs:     make([]*mq.Message, 0, 64),
			notifyCh: make(chan struct{}, 1),
		}
	}
	return &memTopic{name: name, partitions: pts}
}

func (t *memTopic) selectPartition(key []byte) int32 {
	if len(key) == 0 {
		return 0
	}
	h := 0
	for _, b := range key {
		h = h*31 + int(b)
	}
	if h < 0 {
		h = -h
	}
	return int32(h % len(t.partitions))
}

func (t *memTopic) append(msg *mq.Message, partition int32) {
	if int(partition) >= len(t.partitions) || partition < 0 {
		partition = 0
	}
	p := t.partitions[partition]
	p.mu.Lock()
	msg.Partition = partition
	msg.Offset = int64(len(p.msgs))
	p.msgs = append(p.msgs, msg)
	p.mu.Unlock()
	select {
	case p.notifyCh <- struct{}{}:
	default:
	}
}

// --- memPartition ---

type memPartition struct {
	mu       sync.RWMutex
	msgs     []*mq.Message
	notifyCh chan struct{}
}

func (p *memPartition) get(offset int64) []*mq.Message {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if offset >= int64(len(p.msgs)) {
		return nil
	}
	result := make([]*mq.Message, len(p.msgs)-int(offset))
	copy(result, p.msgs[offset:])
	return result
}

// --- MemoryProducer ---

// MemoryProducer 内存生产者
type MemoryProducer struct {
	mu        sync.RWMutex
	t         *memTopic
	closed    bool
	closeOnce sync.Once
}

func (p *MemoryProducer) Send(ctx context.Context, msg *mq.Message) (*mq.ProducerResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		return nil, mq.ErrProducerClosed
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	partition := p.t.selectPartition(msg.Key)
	p.t.append(msg, partition)
	return &mq.ProducerResult{Partition: msg.Partition, Offset: msg.Offset}, nil
}

func (p *MemoryProducer) SendWithPartition(ctx context.Context, msg *mq.Message, partition int32) (*mq.ProducerResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		return nil, mq.ErrProducerClosed
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	p.t.append(msg, partition)
	return &mq.ProducerResult{Partition: msg.Partition, Offset: msg.Offset}, nil
}

func (p *MemoryProducer) Close() error {
	p.closeOnce.Do(func() {
		p.mu.Lock()
		p.closed = true
		p.mu.Unlock()
	})
	return nil
}

// --- MemoryConsumer ---

// MemoryConsumer 内存消费者，维护每个分区的独立消费进度
type MemoryConsumer struct {
	t         *memTopic
	groupID   string
	offsets   []int64
	msgCh     chan *mq.Message
	closeCtx  context.Context
	cancelFn  context.CancelFunc
	closeOnce sync.Once
	closeErr  error
}

func newMemoryConsumer(t *memTopic, groupID string) *MemoryConsumer {
	ctx, cancel := context.WithCancel(context.Background())
	c := &MemoryConsumer{
		t:        t,
		groupID:  groupID,
		offsets:  make([]int64, len(t.partitions)),
		msgCh:    make(chan *mq.Message, defaultChanSize),
		closeCtx: ctx,
		cancelFn: cancel,
	}
	go c.poll()
	return c
}

// poll 持续从各分区拉取新消息推入 msgCh
func (c *MemoryConsumer) poll() {
	defer close(c.msgCh)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.closeCtx.Done():
			return
		case <-ticker.C:
			c.pullAll()
		case <-c.anyNotify():
			c.pullAll()
		}
	}
}

// anyNotify 返回第一个分区的通知 channel（简化实现，足够触发 poll）
func (c *MemoryConsumer) anyNotify() <-chan struct{} {
	if len(c.t.partitions) == 0 {
		return nil
	}
	return c.t.partitions[0].notifyCh
}

func (c *MemoryConsumer) pullAll() {
	for i, pt := range c.t.partitions {
		msgs := pt.get(c.offsets[i])
		for _, msg := range msgs {
			select {
			case c.msgCh <- msg:
				c.offsets[i]++
			case <-c.closeCtx.Done():
				return
			}
		}
	}
}

func (c *MemoryConsumer) Consume(ctx context.Context) (*mq.Message, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case msg, ok := <-c.msgCh:
		if !ok {
			return nil, mq.ErrConsumerClosed
		}
		return msg, nil
	}
}

func (c *MemoryConsumer) ConsumeChan(ctx context.Context) (<-chan *mq.Message, error) {
	if c.closeCtx.Err() != nil {
		return nil, mq.ErrConsumerClosed
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return c.msgCh, nil
}

func (c *MemoryConsumer) Close() error {
	c.closeOnce.Do(func() {
		c.cancelFn()
	})
	return c.closeErr
}

// PublishedCount 返回指定 topic 某分区的消息数量（测试辅助）
func PublishedCount(m mq.MQ, topic string, partition int) int {
	mm, ok := m.(*MemoryMQ)
	if !ok {
		return 0
	}
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	t, ok := mm.topics[topic]
	if !ok || partition >= len(t.partitions) {
		return 0
	}
	t.partitions[partition].mu.RLock()
	defer t.partitions[partition].mu.RUnlock()
	return len(t.partitions[partition].msgs)
}

// 确保编译期接口检查
var _ mq.MQ = (*MemoryMQ)(nil)
var _ mq.Producer = (*MemoryProducer)(nil)
var _ mq.Consumer = (*MemoryConsumer)(nil)
