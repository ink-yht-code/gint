// Copyright 2025 ink-yht-code
//
// Proprietary License

// Package rabbitmq 提供基于 amqp091-go 的 RabbitMQ MQ 实现。
//
// RabbitMQ 没有原生的 topic/partition 概念，这里的映射关系：
//   - topic    → routing key（或 queue name）
//   - groupID  → queue name（同一 queue 的多个消费者共享消费，即竞争消费）
//   - partition → 不支持，SendWithPartition 忽略 partition 参数
package rabbitmq

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/ink-yht-code/gint/mq"
)

const msgChanSize = 1000

// RabbitMQConfig RabbitMQ 连接配置
type RabbitMQConfig struct {
	URL          string // amqp://user:pass@host:5672/vhost
	Exchange     string // 交换机名称，空则使用默认交换机
	ExchangeType string // direct / topic / fanout，默认 direct
	Durable      bool   // 队列/交换机是否持久化
}

// RabbitMQ 实现 mq.MQ 接口
type RabbitMQ struct {
	cfg       RabbitMQConfig
	mu        sync.Mutex
	closed    bool
	closeOnce sync.Once
	closeErr  error
	producers []*RabbitMQProducer
	consumers []*RabbitMQConsumer
}

// NewMQ 创建 RabbitMQ MQ
//
//	m := rabbitmq.NewMQ(rabbitmq.RabbitMQConfig{
//	    URL:          "amqp://guest:guest@localhost:5672/",
//	    Exchange:     "events",
//	    ExchangeType: "topic",
//	    Durable:      true,
//	})
//	p, _ := m.Producer("order.created")
//	c, _ := m.Consumer("order.created", "payment-queue")
func NewMQ(cfg RabbitMQConfig) (mq.MQ, error) {
	if cfg.ExchangeType == "" {
		cfg.ExchangeType = "direct"
	}
	return &RabbitMQ{cfg: cfg}, nil
}

// CreateTopic 在 RabbitMQ 中声明队列（topic 映射为 queue name）
func (r *RabbitMQ) CreateTopic(ctx context.Context, topic string, _ int) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return mq.ErrMQClosed
	}
	conn, err := amqp.Dial(r.cfg.URL)
	if err != nil {
		return err
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	_, err = ch.QueueDeclare(topic, r.cfg.Durable, false, false, false, nil)
	return err
}

// DeleteTopics 删除队列
func (r *RabbitMQ) DeleteTopics(ctx context.Context, topics ...string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return mq.ErrMQClosed
	}
	conn, err := amqp.Dial(r.cfg.URL)
	if err != nil {
		return err
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	for _, topic := range topics {
		if _, err := ch.QueueDelete(topic, false, false, false); err != nil {
			return err
		}
	}
	return nil
}

func (r *RabbitMQ) Producer(topic string) (mq.Producer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, mq.ErrMQClosed
	}
	conn, err := amqp.Dial(r.cfg.URL)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err := r.declareExchange(ch); err != nil {
		conn.Close()
		return nil, err
	}
	p := &RabbitMQProducer{
		conn:     conn,
		channel:  ch,
		cfg:      r.cfg,
		topic:    topic,
	}
	r.producers = append(r.producers, p)
	return p, nil
}

func (r *RabbitMQ) Consumer(topic string, groupID string) (mq.Consumer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, mq.ErrMQClosed
	}
	conn, err := amqp.Dial(r.cfg.URL)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err := r.declareExchange(ch); err != nil {
		conn.Close()
		return nil, err
	}
	// groupID 作为 queue name，同一 queue 多消费者竞争消费
	queueName := groupID
	if queueName == "" {
		queueName = topic
	}
	if _, err := ch.QueueDeclare(queueName, r.cfg.Durable, false, false, false, nil); err != nil {
		conn.Close()
		return nil, err
	}
	if r.cfg.Exchange != "" {
		if err := ch.QueueBind(queueName, topic, r.cfg.Exchange, false, nil); err != nil {
			conn.Close()
			return nil, err
		}
	}
	delivery, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		conn.Close()
		return nil, err
	}
	closeCtx, cancel := context.WithCancel(context.Background())
	c := &RabbitMQConsumer{
		conn:     conn,
		channel:  ch,
		topic:    topic,
		msgCh:    make(chan *mq.Message, msgChanSize),
		delivery: delivery,
		closeCtx: closeCtx,
		cancelFn: cancel,
	}
	r.consumers = append(r.consumers, c)
	go c.run()
	return c, nil
}

func (r *RabbitMQ) Close() error {
	r.closeOnce.Do(func() {
		r.mu.Lock()
		r.closed = true
		producers := r.producers
		consumers := r.consumers
		r.mu.Unlock()
		for _, p := range producers {
			if err := p.Close(); err != nil && r.closeErr == nil {
				r.closeErr = err
			}
		}
		for _, c := range consumers {
			if err := c.Close(); err != nil && r.closeErr == nil {
				r.closeErr = err
			}
		}
	})
	return r.closeErr
}

func (r *RabbitMQ) declareExchange(ch *amqp.Channel) error {
	if r.cfg.Exchange == "" {
		return nil
	}
	return ch.ExchangeDeclare(r.cfg.Exchange, r.cfg.ExchangeType, r.cfg.Durable, false, false, false, nil)
}

// --- RabbitMQProducer ---

// RabbitMQProducer RabbitMQ 生产者
type RabbitMQProducer struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	cfg       RabbitMQConfig
	topic     string
	mu        sync.RWMutex
	closed    bool
	closeOnce sync.Once
	closeErr  error
}

func (p *RabbitMQProducer) Send(ctx context.Context, msg *mq.Message) (*mq.ProducerResult, error) {
	return p.publish(ctx, msg)
}

// SendWithPartition RabbitMQ 不支持分区，等同于 Send
func (p *RabbitMQProducer) SendWithPartition(ctx context.Context, msg *mq.Message, _ int32) (*mq.ProducerResult, error) {
	return p.publish(ctx, msg)
}

func (p *RabbitMQProducer) publish(ctx context.Context, msg *mq.Message) (*mq.ProducerResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		return nil, mq.ErrProducerClosed
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	topic := msg.Topic
	if topic == "" {
		topic = p.topic
	}
	headers := amqp.Table{}
	for k, v := range msg.Headers {
		headers[k] = v
	}
	publishing := amqp.Publishing{
		ContentType:  "application/octet-stream",
		Body:         msg.Value,
		Headers:      headers,
		MessageId:    string(msg.Key),
		Timestamp:    time.Now(),
		DeliveryMode: amqp.Persistent,
	}
	exchange := p.cfg.Exchange
	err := p.channel.PublishWithContext(ctx, exchange, topic, false, false, publishing)
	if err != nil {
		return nil, err
	}
	return &mq.ProducerResult{}, nil
}

func (p *RabbitMQProducer) Close() error {
	p.closeOnce.Do(func() {
		p.mu.Lock()
		p.closed = true
		p.mu.Unlock()
		if p.channel != nil {
			_ = p.channel.Close()
		}
		if p.conn != nil {
			p.closeErr = p.conn.Close()
		}
	})
	return p.closeErr
}

// --- RabbitMQConsumer ---

// RabbitMQConsumer RabbitMQ 消费者
type RabbitMQConsumer struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	topic     string
	msgCh     chan *mq.Message
	delivery  <-chan amqp.Delivery
	closeCtx  context.Context
	cancelFn  context.CancelFunc
	closeOnce sync.Once
	closeErr  error
}

func (c *RabbitMQConsumer) run() {
	defer close(c.msgCh)
	for {
		select {
		case <-c.closeCtx.Done():
			return
		case d, ok := <-c.delivery:
			if !ok {
				return
			}
			msg := &mq.Message{
				Topic:     c.topic,
				Key:       []byte(d.MessageId),
				Value:     d.Body,
				Timestamp: d.Timestamp.UnixMilli(),
				Headers:   make(map[string]string, len(d.Headers)),
			}
			for k, v := range d.Headers {
				switch val := v.(type) {
				case string:
					msg.Headers[k] = val
				case []byte:
					msg.Headers[k] = string(val)
				default:
					if b, err := json.Marshal(v); err == nil {
						msg.Headers[k] = string(b)
					}
				}
			}
			select {
			case c.msgCh <- msg:
				_ = d.Ack(false)
			case <-c.closeCtx.Done():
				_ = d.Nack(false, true)
				return
			}
		}
	}
}

func (c *RabbitMQConsumer) Consume(ctx context.Context) (*mq.Message, error) {
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

func (c *RabbitMQConsumer) ConsumeChan(ctx context.Context) (<-chan *mq.Message, error) {
	if c.closeCtx.Err() != nil {
		return nil, mq.ErrConsumerClosed
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return c.msgCh, nil
}

func (c *RabbitMQConsumer) Close() error {
	c.closeOnce.Do(func() {
		c.cancelFn()
		if c.channel != nil {
			_ = c.channel.Close()
		}
		if c.conn != nil {
			c.closeErr = c.conn.Close()
		}
	})
	return c.closeErr
}

// 编译期接口检查
var _ mq.MQ = (*RabbitMQ)(nil)
var _ mq.Producer = (*RabbitMQProducer)(nil)
var _ mq.Consumer = (*RabbitMQConsumer)(nil)
