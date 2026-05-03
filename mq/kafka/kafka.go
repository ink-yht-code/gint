// Copyright 2025 ink-yht-code
//
// Proprietary License

// Package kafka 提供基于 Sarama 的 Kafka MQ 实现。
package kafka

import (
	"context"
	"sync"
	"time"

	"github.com/IBM/sarama"

	"github.com/ink-yht-code/gint/mq"
)

const (
	defaultRetryMax     = 3
	defaultRetryBackoff = 100 * time.Millisecond
	msgChanSize         = 1000
)

// KafkaMQ 实现 mq.MQ 接口
type KafkaMQ struct {
	brokers   []string
	config    *sarama.Config
	admin     sarama.ClusterAdmin
	mu        sync.Mutex
	closed    bool
	closeOnce sync.Once
	closeErr  error
	producers []*KafkaProducer
	consumers []*KafkaConsumer
}

// NewMQ 创建 Kafka MQ
//
//	m := kafka.NewMQ([]string{"localhost:9092"}, nil)
//	_ = m.CreateTopic(ctx, "orders", 3)
//	p, _ := m.Producer("orders")
//	c, _ := m.Consumer("orders", "payment-svc")
func NewMQ(brokers []string, cfg *sarama.Config) (mq.MQ, error) {
	if cfg == nil {
		cfg = defaultConfig()
	}
	admin, err := sarama.NewClusterAdmin(brokers, cfg)
	if err != nil {
		return nil, err
	}
	return &KafkaMQ{
		brokers: brokers,
		config:  cfg,
		admin:   admin,
	}, nil
}

func defaultConfig() *sarama.Config {
	cfg := sarama.NewConfig()
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Retry.Max = defaultRetryMax
	cfg.Producer.Retry.Backoff = defaultRetryBackoff
	cfg.Producer.Return.Successes = true
	cfg.Consumer.Return.Errors = true
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	return cfg
}

func (k *KafkaMQ) CreateTopic(ctx context.Context, topic string, partitions int) error {
	if topic == "" {
		return mq.ErrInvalidTopic
	}
	if partitions <= 0 {
		return mq.ErrInvalidPartition
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.closed {
		return mq.ErrMQClosed
	}
	detail := &sarama.TopicDetail{
		NumPartitions:     int32(partitions),
		ReplicationFactor: 1,
	}
	err := k.admin.CreateTopic(topic, detail, false)
	// topic 已存在不视为错误
	if err == sarama.ErrTopicAlreadyExists {
		return nil
	}
	return err
}

func (k *KafkaMQ) DeleteTopics(ctx context.Context, topics ...string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.closed {
		return mq.ErrMQClosed
	}
	for _, topic := range topics {
		if err := k.admin.DeleteTopic(topic); err != nil && err != sarama.ErrUnknownTopicOrPartition {
			return err
		}
	}
	return nil
}

func (k *KafkaMQ) Producer(topic string) (mq.Producer, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.closed {
		return nil, mq.ErrMQClosed
	}
	sp, err := sarama.NewSyncProducer(k.brokers, k.config)
	if err != nil {
		return nil, err
	}
	p := &KafkaProducer{
		topic:    topic,
		producer: sp,
	}
	k.producers = append(k.producers, p)
	return p, nil
}

func (k *KafkaMQ) Consumer(topic string, groupID string) (mq.Consumer, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.closed {
		return nil, mq.ErrMQClosed
	}
	group, err := sarama.NewConsumerGroup(k.brokers, groupID, k.config)
	if err != nil {
		return nil, err
	}
	closeCtx, cancel := context.WithCancel(context.Background())
	c := &KafkaConsumer{
		topic:    topic,
		groupID:  groupID,
		group:    group,
		msgCh:    make(chan *mq.Message, msgChanSize),
		closeCtx: closeCtx,
		cancelFn: cancel,
	}
	k.consumers = append(k.consumers, c)
	go c.run()
	return c, nil
}

func (k *KafkaMQ) Close() error {
	k.closeOnce.Do(func() {
		k.mu.Lock()
		k.closed = true
		producers := k.producers
		consumers := k.consumers
		k.mu.Unlock()

		for _, p := range producers {
			if err := p.Close(); err != nil && k.closeErr == nil {
				k.closeErr = err
			}
		}
		for _, c := range consumers {
			if err := c.Close(); err != nil && k.closeErr == nil {
				k.closeErr = err
			}
		}
		if err := k.admin.Close(); err != nil && k.closeErr == nil {
			k.closeErr = err
		}
	})
	return k.closeErr
}

// --- KafkaProducer ---

// KafkaProducer Kafka 同步生产者
type KafkaProducer struct {
	topic     string
	producer  sarama.SyncProducer
	mu        sync.RWMutex
	closed    bool
	closeOnce sync.Once
	closeErr  error
}

func (p *KafkaProducer) Send(ctx context.Context, msg *mq.Message) (*mq.ProducerResult, error) {
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
	km := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.ByteEncoder(msg.Key),
		Value: sarama.ByteEncoder(msg.Value),
	}
	for k, v := range msg.Headers {
		km.Headers = append(km.Headers, sarama.RecordHeader{Key: []byte(k), Value: []byte(v)})
	}
	partition, offset, err := p.producer.SendMessage(km)
	if err != nil {
		return nil, err
	}
	return &mq.ProducerResult{Partition: partition, Offset: offset}, nil
}

func (p *KafkaProducer) SendWithPartition(ctx context.Context, msg *mq.Message, partition int32) (*mq.ProducerResult, error) {
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
	km := &sarama.ProducerMessage{
		Topic:     topic,
		Partition: partition,
		Key:       sarama.ByteEncoder(msg.Key),
		Value:     sarama.ByteEncoder(msg.Value),
	}
	for k, v := range msg.Headers {
		km.Headers = append(km.Headers, sarama.RecordHeader{Key: []byte(k), Value: []byte(v)})
	}
	part, offset, err := p.producer.SendMessage(km)
	if err != nil {
		return nil, err
	}
	return &mq.ProducerResult{Partition: part, Offset: offset}, nil
}

func (p *KafkaProducer) Close() error {
	p.closeOnce.Do(func() {
		p.mu.Lock()
		p.closed = true
		p.mu.Unlock()
		p.closeErr = p.producer.Close()
	})
	return p.closeErr
}

// --- KafkaConsumer ---

// KafkaConsumer Kafka 消费者，基于 ConsumerGroup
type KafkaConsumer struct {
	topic     string
	groupID   string
	group     sarama.ConsumerGroup
	msgCh     chan *mq.Message
	closeCtx  context.Context
	cancelFn  context.CancelFunc
	closeOnce sync.Once
	closeErr  error
}

// run 持续从 Kafka 拉取消息，rebalance 后自动重连
func (c *KafkaConsumer) run() {
	defer close(c.msgCh)
	handler := &cgHandler{msgCh: c.msgCh, closeCtx: c.closeCtx}
	for {
		if err := c.group.Consume(c.closeCtx, []string{c.topic}, handler); err != nil {
			return
		}
		if c.closeCtx.Err() != nil {
			return
		}
		// rebalance 后重置 ready，继续消费
		handler.ready = false
	}
}

func (c *KafkaConsumer) Consume(ctx context.Context) (*mq.Message, error) {
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

func (c *KafkaConsumer) ConsumeChan(ctx context.Context) (<-chan *mq.Message, error) {
	if c.closeCtx.Err() != nil {
		return nil, mq.ErrConsumerClosed
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return c.msgCh, nil
}

func (c *KafkaConsumer) Close() error {
	c.closeOnce.Do(func() {
		c.cancelFn()
		c.closeErr = c.group.Close()
	})
	return c.closeErr
}

// --- cgHandler ---

type cgHandler struct {
	msgCh    chan *mq.Message
	closeCtx context.Context
	ready    bool
}

func (h *cgHandler) Setup(_ sarama.ConsumerGroupSession) error {
	h.ready = true
	return nil
}

func (h *cgHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *cgHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case km, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			msg := &mq.Message{
				Topic:     km.Topic,
				Key:       km.Key,
				Value:     km.Value,
				Timestamp: km.Timestamp.UnixMilli(),
				Partition: km.Partition,
				Offset:    km.Offset,
				Headers:   make(map[string]string, len(km.Headers)),
			}
			for _, h := range km.Headers {
				msg.Headers[string(h.Key)] = string(h.Value)
			}
			select {
			case h.msgCh <- msg:
				session.MarkMessage(km, "")
			case <-h.closeCtx.Done():
				return nil
			}
		case <-session.Context().Done():
			return nil
		}
	}
}

// 编译期接口检查
var _ mq.MQ = (*KafkaMQ)(nil)
var _ mq.Producer = (*KafkaProducer)(nil)
var _ mq.Consumer = (*KafkaConsumer)(nil)
