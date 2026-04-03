// Copyright 2025 ink-yht-code
//
// Proprietary License

package mq

import (
	"context"
	"errors"
)

var (
	ErrProducerClosed  = errors.New("producer is closed")
	ErrConsumerClosed  = errors.New("consumer is closed")
	ErrTopicEmpty      = errors.New("topic is empty")
	ErrMessageEmpty    = errors.New("message is empty")
	ErrHandlerNotSet   = errors.New("handler not set")
	ErrUnsupportedType = errors.New("unsupported message type")
)

// Message 消息
type Message struct {
	Topic     string            // 主题
	Key       string            // 消息键（用于分区）
	Value     []byte            // 消息值
	Headers   map[string]string // 消息头
	Timestamp int64             // 时间戳
	Partition int32             // 分区（仅 Kafka）
	Offset    int64             // 偏移量（仅 Kafka）
}

// Producer 生产者接口
type Producer interface {
	// Send 发送消息
	Send(ctx context.Context, topic string, msg *Message) error
	// SendAsync 异步发送消息
	SendAsync(ctx context.Context, topic string, msg *Message, callback func(error))
	// Close 关闭生产者
	Close() error
}

// Consumer 消费者接口
type Consumer interface {
	// Subscribe 订阅主题
	Subscribe(topics ...string) error
	// Unsubscribe 取消订阅
	Unsubscribe() error
	// Consume 消费消息（阻塞）
	Consume(ctx context.Context, handler Handler) error
	// Close 关闭消费者
	Close() error
}

// Handler 消息处理函数
type Handler func(ctx context.Context, msg *Message) error

// ProducerConfig 生产者配置
type ProducerConfig struct {
	Brokers      []string // Kafka brokers 或 RabbitMQ 地址
	Topic        string   // 默认主题
	Exchange     string   // RabbitMQ 交换机
	ExchangeType string   // RabbitMQ 交换机类型
	RetryCount   int      // 重试次数
	Timeout      int      // 超时时间（秒）
	Async        bool     // 是否异步发送
}

// ConsumerConfig 消费者配置
type ConsumerConfig struct {
	Brokers      []string // Kafka brokers 或 RabbitMQ 地址
	Topic        string   // 主题
	Queue        string   // RabbitMQ 队列名
	Exchange     string   // RabbitMQ 交换机
	ExchangeType string   // RabbitMQ 交换机类型
	GroupID      string   // Kafka 消费组 ID
	AutoCommit   bool     // 是否自动提交
	Concurrency  int      // 并发数
}

// MQ 消息队列抽象层
type MQ interface {
	Producer(config *ProducerConfig) (Producer, error)
	Consumer(config *ConsumerConfig) (Consumer, error)
}
