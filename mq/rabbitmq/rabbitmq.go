// Copyright 2025 ink-yht-code
//
// Proprietary License

package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/ink-yht-code/gint/mq"
)

// RabbitMQProducer RabbitMQ 生产者
type RabbitMQProducer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	config  *mq.ProducerConfig
	closed  bool
	mu      sync.RWMutex
}

// RabbitMQConsumer RabbitMQ 消费者
type RabbitMQConsumer struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	config   *mq.ConsumerConfig
	closed   bool
	mu       sync.RWMutex
	delivery <-chan amqp.Delivery
}

// NewProducer 创建 RabbitMQ 生产者
func NewProducer(config *mq.ProducerConfig) (mq.Producer, error) {
	conn, err := amqp.Dial(config.Brokers[0])
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	exchangeType := config.ExchangeType
	if exchangeType == "" {
		exchangeType = "direct"
	}

	// 声明交换机
	if config.Exchange != "" {
		if err := channel.ExchangeDeclare(
			config.Exchange,
			exchangeType,
			true,  // durable
			false, // auto-delete
			false, // internal
			false, // no-wait
			nil,   // args
		); err != nil {
			conn.Close()
			return nil, err
		}
	}

	return &RabbitMQProducer{
		conn:    conn,
		channel: channel,
		config:  config,
	}, nil
}

// Send 发送消息
func (p *RabbitMQProducer) Send(ctx context.Context, topic string, msg *mq.Message) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return mq.ErrProducerClosed
	}

	exchange := p.config.Exchange
	routingKey := topic
	if exchange == "" {
		// 直接发送到队列
		exchange = ""
		routingKey = topic
	}

	headers := amqp.Table{}
	for k, v := range msg.Headers {
		headers[k] = v
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(p.config.Timeout)*time.Second)
	defer cancel()

	return p.channel.PublishWithContext(ctx,
		exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         msg.Value,
			Headers:      headers,
			MessageId:    msg.Key,
			Timestamp:    time.Now(),
			DeliveryMode: amqp.Persistent,
		},
	)
}

// SendAsync 异步发送消息
func (p *RabbitMQProducer) SendAsync(ctx context.Context, topic string, msg *mq.Message, callback func(error)) {
	go func() {
		err := p.Send(ctx, topic, msg)
		if callback != nil {
			callback(err)
		}
	}()
}

// Close 关闭生产者
func (p *RabbitMQProducer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}
	p.closed = true

	var errs []error
	if p.channel != nil {
		if err := p.channel.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// NewConsumer 创建 RabbitMQ 消费者
func NewConsumer(config *mq.ConsumerConfig) (mq.Consumer, error) {
	conn, err := amqp.Dial(config.Brokers[0])
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	exchangeType := config.ExchangeType
	if exchangeType == "" {
		exchangeType = "direct"
	}

	// 声明交换机
	if config.Exchange != "" {
		if err := channel.ExchangeDeclare(
			config.Exchange,
			exchangeType,
			true,  // durable
			false, // auto-delete
			false, // internal
			false, // no-wait
			nil,   // args
		); err != nil {
			conn.Close()
			return nil, err
		}
	}

	// 声明队列
	queueName := config.Queue
	if queueName == "" {
		queueName = config.Topic
	}

	queue, err := channel.QueueDeclare(
		queueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		conn.Close()
		return nil, err
	}

	// 绑定队列到交换机
	if config.Exchange != "" {
		if err := channel.QueueBind(
			queue.Name,
			config.Topic, // routing key
			config.Exchange,
			false,
			nil,
		); err != nil {
			conn.Close()
			return nil, err
		}
	}

	// 设置 QoS
	concurrency := config.Concurrency
	if concurrency <= 0 {
		concurrency = 1
	}
	if err := channel.Qos(concurrency, 0, false); err != nil {
		conn.Close()
		return nil, err
	}

	// 开始消费
	delivery, err := channel.Consume(
		queue.Name,
		"",    // consumer tag
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &RabbitMQConsumer{
		conn:     conn,
		channel:  channel,
		config:   config,
		delivery: delivery,
	}, nil
}

// Subscribe 订阅主题（RabbitMQ 在创建时已绑定）
func (c *RabbitMQConsumer) Subscribe(topics ...string) error {
	return nil
}

// Unsubscribe 取消订阅
func (c *RabbitMQConsumer) Unsubscribe() error {
	return nil
}

// Consume 消费消息
func (c *RabbitMQConsumer) Consume(ctx context.Context, handler mq.Handler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case delivery, ok := <-c.delivery:
			if !ok {
				return errors.New("delivery channel closed")
			}

			msg := &mq.Message{
				Topic:     c.config.Topic,
				Key:       delivery.MessageId,
				Value:     delivery.Body,
				Timestamp: delivery.Timestamp.UnixMilli(),
				Headers:   make(map[string]string),
			}

			for k, v := range delivery.Headers {
				if str, ok := v.(string); ok {
					msg.Headers[k] = str
				} else if bytes, ok := v.([]byte); ok {
					msg.Headers[k] = string(bytes)
				} else if b, err := json.Marshal(v); err == nil {
					msg.Headers[k] = string(b)
				}
			}

			if err := handler(ctx, msg); err != nil {
				// 处理失败，nack 并重新入队
				_ = delivery.Nack(false, true)
				continue
			}

			// 处理成功，ack
			_ = delivery.Ack(false)
		}
	}
}

// Close 关闭消费者
func (c *RabbitMQConsumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	var errs []error
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
