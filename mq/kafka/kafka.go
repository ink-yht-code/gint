// Copyright 2025 ink-yht-code
//
// Proprietary License

package kafka

import (
	"context"
	"errors"
	"sync"

	"github.com/IBM/sarama"

	"github.com/ink-yht-code/gint/mq"
)

// KafkaProducer Kafka 生产者
type KafkaProducer struct {
	producer sarama.SyncProducer
	config   *mq.ProducerConfig
	closed   bool
	mu       sync.RWMutex
}

// KafkaProducerAsync 异步生产者
type KafkaProducerAsync struct {
	producer sarama.AsyncProducer
	config   *mq.ProducerConfig
	closed   bool
	mu       sync.RWMutex
	wg       sync.WaitGroup
}

// KafkaConsumer Kafka 消费者
type KafkaConsumer struct {
	consumer sarama.ConsumerGroup
	config   *mq.ConsumerConfig
	handler  mq.Handler
	closed   bool
	mu       sync.RWMutex
	ready    chan struct{}
}

// NewProducer 创建 Kafka 生产者
func NewProducer(config *mq.ProducerConfig) (mq.Producer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Producer.Retry.Max = config.RetryCount
	saramaConfig.Producer.Return.Successes = true

	if config.Timeout > 0 {
		saramaConfig.Net.ReadTimeout = 0
	}

	if config.Async {
		producer, err := sarama.NewAsyncProducer(config.Brokers, saramaConfig)
		if err != nil {
			return nil, err
		}
		return &KafkaProducerAsync{
			producer: producer,
			config:   config,
		}, nil
	}

	producer, err := sarama.NewSyncProducer(config.Brokers, saramaConfig)
	if err != nil {
		return nil, err
	}

	return &KafkaProducer{
		producer: producer,
		config:   config,
	}, nil
}

// Send 发送消息
func (p *KafkaProducer) Send(ctx context.Context, topic string, msg *mq.Message) error {
	if topic == "" {
		topic = p.config.Topic
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return mq.ErrProducerClosed
	}

	producerMsg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.ByteEncoder(msg.Key),
		Value: sarama.ByteEncoder(msg.Value),
	}

	for k, v := range msg.Headers {
		producerMsg.Headers = append(producerMsg.Headers, sarama.RecordHeader{
			Key:   []byte(k),
			Value: []byte(v),
		})
	}

	_, _, err := p.producer.SendMessage(producerMsg)
	return err
}

// SendAsync 异步发送消息
func (p *KafkaProducer) SendAsync(ctx context.Context, topic string, msg *mq.Message, callback func(error)) {
	go func() {
		err := p.Send(ctx, topic, msg)
		if callback != nil {
			callback(err)
		}
	}()
}

// Close 关闭生产者
func (p *KafkaProducer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}
	p.closed = true
	return p.producer.Close()
}

// Send 发送消息（异步生产者）
func (p *KafkaProducerAsync) Send(ctx context.Context, topic string, msg *mq.Message) error {
	if topic == "" {
		topic = p.config.Topic
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return mq.ErrProducerClosed
	}

	producerMsg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.ByteEncoder(msg.Key),
		Value: sarama.ByteEncoder(msg.Value),
	}

	for k, v := range msg.Headers {
		producerMsg.Headers = append(producerMsg.Headers, sarama.RecordHeader{
			Key:   []byte(k),
			Value: []byte(v),
		})
	}

	p.producer.Input() <- producerMsg
	return nil
}

// SendAsync 异步发送消息
func (p *KafkaProducerAsync) SendAsync(ctx context.Context, topic string, msg *mq.Message, callback func(error)) {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		err := p.Send(ctx, topic, msg)
		if callback != nil {
			callback(err)
		}
	}()
}

// Close 关闭生产者
func (p *KafkaProducerAsync) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}
	p.closed = true
	p.wg.Wait()
	return p.producer.Close()
}

// NewConsumer 创建 Kafka 消费者
func NewConsumer(config *mq.ConsumerConfig) (mq.Consumer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Consumer.Return.Errors = true

	if !config.AutoCommit {
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	}

	group, err := sarama.NewConsumerGroup(config.Brokers, config.GroupID, saramaConfig)
	if err != nil {
		return nil, err
	}

	return &KafkaConsumer{
		consumer: group,
		config:   config,
		ready:    make(chan struct{}),
	}, nil
}

// Subscribe 订阅主题
func (c *KafkaConsumer) Subscribe(topics ...string) error {
	return nil // Kafka 使用 ConsumerGroup 时在 Consume 中订阅
}

// Unsubscribe 取消订阅
func (c *KafkaConsumer) Unsubscribe() error {
	return nil
}

// Consume 消费消息
func (c *KafkaConsumer) Consume(ctx context.Context, handler mq.Handler) error {
	c.handler = handler

	topics := []string{c.config.Topic}
	if len(topics) == 0 {
		return errors.New("no topics to consume")
	}

	wg := &sync.WaitGroup{}

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()
		default:
			err := c.consumer.Consume(ctx, topics, &consumerGroupHandler{
				handler: handler,
				ready:   c.ready,
			})
			if err != nil {
				return err
			}
		}
	}
}

// Close 关闭消费者
func (c *KafkaConsumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true
	return c.consumer.Close()
}

// consumerGroupHandler Sarama ConsumerGroup Handler 实现
type consumerGroupHandler struct {
	handler mq.Handler
	ready   chan struct{}
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	close(h.ready)
	return nil
}

func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		message := &mq.Message{
			Topic:     msg.Topic,
			Key:       string(msg.Key),
			Value:     msg.Value,
			Timestamp: msg.Timestamp.UnixMilli(),
			Partition: msg.Partition,
			Offset:    msg.Offset,
			Headers:   make(map[string]string),
		}

		for _, header := range msg.Headers {
			message.Headers[string(header.Key)] = string(header.Value)
		}

		if err := h.handler(session.Context(), message); err != nil {
			return err
		}

		session.MarkMessage(msg, "")
	}
	return nil
}
