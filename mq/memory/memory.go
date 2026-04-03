// Copyright 2025 ink-yht-code
//
// Proprietary License

package memory

import (
	"context"
	"errors"
	"sync"

	"github.com/ink-yht-code/gint/mq"
)

// MemoryProducer 内存生产者
type MemoryProducer struct {
	broker *MemoryBroker
	closed bool
	mu     sync.RWMutex
}

// MemoryConsumer 内存消费者
type MemoryConsumer struct {
	broker   *MemoryBroker
	topic    string
	handler  mq.Handler
	closed   bool
	mu       sync.RWMutex
	messages chan *mq.Message
	wg       sync.WaitGroup
}

// MemoryBroker 内存消息代理（用于测试）
type MemoryBroker struct {
	topics    map[string]chan *mq.Message
	mu        sync.RWMutex
	consumers map[string][]*MemoryConsumer
}

// NewBroker 创建内存消息代理
func NewBroker() *MemoryBroker {
	return &MemoryBroker{
		topics:    make(map[string]chan *mq.Message),
		consumers: make(map[string][]*MemoryConsumer),
	}
}

// NewProducer 创建内存生产者
func NewProducer(broker *MemoryBroker) mq.Producer {
	return &MemoryProducer{
		broker: broker,
	}
}

// NewConsumer 创建内存消费者
func NewConsumer(broker *MemoryBroker, topic string) mq.Consumer {
	broker.mu.Lock()
	defer broker.mu.Unlock()

	if _, ok := broker.topics[topic]; !ok {
		broker.topics[topic] = make(chan *mq.Message, 1000)
	}

	c := &MemoryConsumer{
		broker:   broker,
		topic:    topic,
		messages: broker.topics[topic],
	}

	broker.consumers[topic] = append(broker.consumers[topic], c)
	return c
}

// Send 发送消息
func (p *MemoryProducer) Send(ctx context.Context, topic string, msg *mq.Message) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return mq.ErrProducerClosed
	}

	p.broker.mu.RLock()
	ch, ok := p.broker.topics[topic]
	p.broker.mu.RUnlock()

	if !ok {
		p.broker.mu.Lock()
		if ch, ok = p.broker.topics[topic]; !ok {
			ch = make(chan *mq.Message, 1000)
			p.broker.topics[topic] = ch
		}
		p.broker.mu.Unlock()
	}

	select {
	case ch <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SendAsync 异步发送消息
func (p *MemoryProducer) SendAsync(ctx context.Context, topic string, msg *mq.Message, callback func(error)) {
	go func() {
		err := p.Send(ctx, topic, msg)
		if callback != nil {
			callback(err)
		}
	}()
}

// Close 关闭生产者
func (p *MemoryProducer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	return nil
}

// Subscribe 订阅主题
func (c *MemoryConsumer) Subscribe(topics ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, topic := range topics {
		c.broker.mu.Lock()
		if _, ok := c.broker.topics[topic]; !ok {
			c.broker.topics[topic] = make(chan *mq.Message, 1000)
		}
		c.broker.mu.Unlock()
	}

	return nil
}

// Unsubscribe 取消订阅
func (c *MemoryConsumer) Unsubscribe() error {
	return nil
}

// Consume 消费消息
func (c *MemoryConsumer) Consume(ctx context.Context, handler mq.Handler) error {
	c.mu.Lock()
	c.handler = handler
	c.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-c.messages:
			if !ok {
				return errors.New("message channel closed")
			}
			if err := handler(ctx, msg); err != nil {
				// 处理失败，重新入队
				go func() {
					c.messages <- msg
				}()
			}
		}
	}
}

// Close 关闭消费者
func (c *MemoryConsumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

// PublishedCount 已发布消息数量（用于测试）
func (b *MemoryBroker) PublishedCount(topic string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if ch, ok := b.topics[topic]; ok {
		return len(ch)
	}
	return 0
}
