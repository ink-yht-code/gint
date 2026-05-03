// Copyright 2025 ink-yht-code
//
// Proprietary License

package mq

import (
	"context"
	"errors"
	"io"
)

var (
	ErrMQClosed        = errors.New("mq is closed")
	ErrProducerClosed  = errors.New("producer is closed")
	ErrConsumerClosed  = errors.New("consumer is closed")
	ErrTopicEmpty      = errors.New("topic is empty")
	ErrInvalidTopic    = errors.New("invalid topic name")
	ErrInvalidPartition = errors.New("invalid partition")
	ErrMessageEmpty    = errors.New("message is empty")
)

// Message 消息
type Message struct {
	Topic     string            // 主题
	Key       []byte            // 消息键（用于分区路由）
	Value     []byte            // 消息值
	Headers   map[string]string // 消息头
	Timestamp int64             // 时间戳（UnixMilli）
	Partition int32             // 分区（消费时填充）
	Offset    int64             // 偏移量（消费时填充）
}

// ProducerResult 发送结果
type ProducerResult struct {
	Partition int32
	Offset    int64
}

// Producer 生产者接口，可被多个 goroutine 并发使用
type Producer interface {
	// Send 发送消息，不指定分区
	Send(ctx context.Context, msg *Message) (*ProducerResult, error)
	// SendWithPartition 指定分区发送消息
	SendWithPartition(ctx context.Context, msg *Message, partition int32) (*ProducerResult, error)
	// Close 关闭生产者，多次调用返回相同 error
	Close() error
}

// Consumer 消费者接口，可被多个 goroutine 并发使用
type Consumer interface {
	// Consume 阻塞获取单条消息
	Consume(ctx context.Context) (*Message, error)
	// ConsumeChan 返回消息 channel，适合 select 场景
	ConsumeChan(ctx context.Context) (<-chan *Message, error)
	// Close 关闭消费者，多次调用返回相同 error
	Close() error
}

// MQ 消息队列顶层接口，负责 topic 管理和创建生产者/消费者
type MQ interface {
	// CreateTopic 创建 topic，partitions 指定分区数（从 0 开始编号）
	CreateTopic(ctx context.Context, topic string, partitions int) error
	// DeleteTopics 删除一个或多个 topic
	DeleteTopics(ctx context.Context, topics ...string) error
	// Producer 为指定 topic 创建生产者
	Producer(topic string) (Producer, error)
	// Consumer 为指定 topic 创建消费者，groupID 指定消费组
	Consumer(topic string, groupID string) (Consumer, error)
	// Close 关闭 MQ，释放所有 Producer 和 Consumer 资源
	io.Closer
}
