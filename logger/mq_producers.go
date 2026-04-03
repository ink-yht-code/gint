// Copyright 2025 ink-yht-code
//
// Proprietary License
//
// IMPORTANT: This software is NOT open source.
// You may NOT use, copy, modify, merge, publish, distribute, sublicense,
// or sell copies of this file, in whole or in part, without prior written
// permission from the copyright holder.
//
// This software is provided "AS IS", without warranty of any kind.

package logger

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

// --- Redis Stream 实现 ---

// RedisStreamProducer Redis Stream 生产者
type RedisStreamProducer struct {
	client *redis.Client
	maxLen int64 // 流最大长度（近似）
}

// RedisStreamProducerConfig Redis Stream 生产者配置
type RedisStreamProducerConfig struct {
	Client *redis.Client
	// MaxLen 流的最大长度（0 表示无限制）
	// 使用近似裁剪策略，提高性能
	MaxLen int64
}

// NewRedisStreamProducer 创建 Redis Stream 生产者
func NewRedisStreamProducer(cfg RedisStreamProducerConfig) *RedisStreamProducer {
	if cfg.MaxLen <= 0 {
		cfg.MaxLen = 100000 // 默认保留最近 10 万条
	}
	return &RedisStreamProducer{
		client: cfg.Client,
		maxLen: cfg.MaxLen,
	}
}

// Publish 实现 MQProducer 接口
func (p *RedisStreamProducer) Publish(ctx context.Context, topic string, message []byte) error {
	return p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: topic,
		MaxLen: p.maxLen,
		Approx: true, // 近似裁剪，提高性能
		Values: map[string]any{"data": message},
	}).Err()
}

// Close 实现 MQProducer 接口
func (p *RedisStreamProducer) Close() error {
	return p.client.Close()
}

// --- Redis List 实现（LPUSH 方式，更简单）---

// RedisListProducer Redis List 生产者（使用 LPUSH）
type RedisListProducer struct {
	client *redis.Client
}

// NewRedisListProducer 创建 Redis List 生产者
func NewRedisListProducer(client *redis.Client) *RedisListProducer {
	return &RedisListProducer{client: client}
}

// Publish 实现 MQProducer 接口
func (p *RedisListProducer) Publish(ctx context.Context, topic string, message []byte) error {
	return p.client.LPush(ctx, topic, message).Err()
}

// Close 实现 MQProducer 接口
func (p *RedisListProducer) Close() error {
	return p.client.Close()
}

// --- RabbitMQ 实现 ---

// RabbitMQProducer RabbitMQ 生产者（接口定义，避免强依赖）
// 实际使用时需要引入 github.com/rabbitmq/amqp091-go
type RabbitMQProducer struct {
	channel   any // *amqp091.Channel
	exchange  string
	mandatory bool
	immediate bool
}

// RabbitMQProducerConfig RabbitMQ 生产者配置
type RabbitMQProducerConfig struct {
	Channel   any    // *amqp091.Channel
	Exchange  string // 交换机名称（空字符串使用默认交换机）
	Mandatory bool   // 如果为 true，当路由不到队列时返回错误
	Immediate bool   // 如果为 true，当没有消费者时返回错误
}

// NewRabbitMQProducer 创建 RabbitMQ 生产者
func NewRabbitMQProducer(cfg RabbitMQProducerConfig) *RabbitMQProducer {
	return &RabbitMQProducer{
		channel:   cfg.Channel,
		exchange:  cfg.Exchange,
		mandatory: cfg.Mandatory,
		immediate: cfg.Immediate,
	}
}

// Publish 实现 MQProducer 接口
func (p *RabbitMQProducer) Publish(ctx context.Context, topic string, message []byte) error {
	// 用户需要根据实际使用的库实现
	// 示例（使用 amqp091-go）：
	/*
		ch := p.channel.(*amqp091.Channel)
		return ch.PublishWithContext(
			ctx,
			p.exchange,  // exchange
			topic,       // routing key (queue name when using default exchange)
			p.mandatory,
			p.immediate,
			amqp091.Publishing{
				ContentType: "application/json",
				Body:        message,
			},
		)
	*/
	return nil
}

// Close 实现 MQProducer 接口
func (p *RabbitMQProducer) Close() error {
	/*
		return p.channel.(*amqp091.Channel).Close()
	*/
	return nil
}

// --- RocketMQ 实现 ---

// RocketMQProducer RocketMQ 生产者（接口定义，避免强依赖）
// 实际使用时需要引入 github.com/apache/rocketmq-client-go/v2
type RocketMQProducer struct {
	producer any // rocketmq.Producer
	topic    string
	tag      string
}

// RocketMQProducerConfig RocketMQ 生产者配置
type RocketMQProducerConfig struct {
	Producer any    // rocketmq.Producer
	Topic    string // 主题
	Tag      string // 标签（可选）
}

// NewRocketMQProducer 创建 RocketMQ 生产者
func NewRocketMQProducer(cfg RocketMQProducerConfig) *RocketMQProducer {
	return &RocketMQProducer{
		producer: cfg.Producer,
		topic:    cfg.Topic,
		tag:      cfg.Tag,
	}
}

// Publish 实现 MQProducer 接口
func (p *RocketMQProducer) Publish(ctx context.Context, topic string, message []byte) error {
	// 用户需要根据实际使用的库实现
	// 示例（使用 rocketmq-client-go）：
	/*
		producer := p.producer.(rocketmq.Producer)
		msg := &primitive.Message{
			Topic: p.topic,
			Body:  message,
		}
		if p.tag != "" {
			msg.WithTag(p.tag)
		}
		_, err := producer.SendSync(ctx, msg)
		return err
	*/
	return nil
}

// Close 实现 MQProducer 接口
func (p *RocketMQProducer) Close() error {
	/*
		return p.producer.(rocketmq.Producer).Shutdown()
	*/
	return nil
}

// --- NATS 实现 ---

// NATSProducer NATS 生产者（接口定义，避免强依赖）
// 实际使用时需要引入 github.com/nats-io/nats.go
type NATSProducer struct {
	conn any // *nats.Conn
}

// NewNATSProducer 创建 NATS 生产者
func NewNATSProducer(conn any) *NATSProducer {
	return &NATSProducer{conn: conn}
}

// Publish 实现 MQProducer 接口
func (p *NATSProducer) Publish(ctx context.Context, topic string, message []byte) error {
	// 示例（使用 nats.go）：
	/*
		return p.conn.(*nats.Conn).Publish(topic, message)
	*/
	return nil
}

// Close 实现 MQProducer 接口
func (p *NATSProducer) Close() error {
	/*
		p.conn.(*nats.Conn).Close()
		return nil
	*/
	return nil
}

// --- 使用示例 ---

/*
# Redis Stream（推荐，简单高效）

import "github.com/redis/go-redis/v9"

rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
producer := logger.NewRedisStreamProducer(logger.RedisStreamProducerConfig{
    Client: rdb,
    MaxLen: 100000,
})

# Kafka（高吞吐量）

import "github.com/IBM/sarama"

config := sarama.NewConfig()
config.Producer.Return.Successes = true
syncProducer, _ := sarama.NewSyncProducer([]string{"localhost:9092"}, config)
producer := logger.NewKafkaProducer(syncProducer, "logs", false)

# RabbitMQ（灵活路由）

import "github.com/rabbitmq/amqp091-go"

conn, _ := amqp091.Dial("amqp://guest:guest@localhost:5672/")
ch, _ := conn.Channel()
producer := logger.NewRabbitMQProducer(logger.RabbitMQProducerConfig{
    Channel:  ch,
    Exchange: "logs",
})

# RocketMQ（阿里云生态）

import "github.com/apache/rocketmq-client-go/v2"

p, _ := rocketmq.NewProducer(...)
producer := logger.NewRocketMQProducer(logger.RocketMQProducerConfig{
    Producer: p,
    Topic:    "logs",
})

# NATS（轻量级）

import "github.com/nats-io/nats.go"

nc, _ := nats.Connect("nats://localhost:4222")
producer := logger.NewNATSProducer(nc)
*/

// --- 批量发送优化（JSON 数组格式）---

// BatchLogMessage 批量日志消息
type BatchLogMessage struct {
	Service string       `json:"service"`
	Host    string       `json:"host"`
	Logs    []LogMessage `json:"logs"`
}

// MarshalBatch 将批量日志序列化为 JSON
func MarshalBatch(logs []LogMessage) ([]byte, error) {
	if len(logs) == 0 {
		return nil, nil
	}
	batch := BatchLogMessage{
		Service: logs[0].Service,
		Host:    logs[0].Host,
		Logs:    logs,
	}
	return json.Marshal(batch)
}
