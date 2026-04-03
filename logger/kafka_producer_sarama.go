//go:build kafka

package logger

import (
	"context"
	"errors"

	"github.com/IBM/sarama"
)

// KafkaProducer 基于 Sarama 的 Kafka 生产者。
//
// 用法 1：传入已创建好的 sarama.SyncProducer / sarama.AsyncProducer：
//
//	p := logger.NewKafkaProducer(syncProducer, "logs", false)
//
// 用法 2：用 brokers 创建（推荐）：
//
//	p, _ := logger.NewKafkaProducerWithBrokers([]string{"localhost:9092"}, nil, "logs", false)
type KafkaProducer struct {
	sync    sarama.SyncProducer
	async   sarama.AsyncProducer
	isAsync bool
	topic   string
}

// NewKafkaProducer 创建 Kafka 生产者（Sarama 实现）。
//
// producer 支持：
// - sarama.SyncProducer（async=false）
// - sarama.AsyncProducer（async=true）
func NewKafkaProducer(producer any, topic string, async bool) *KafkaProducer {
	kp := &KafkaProducer{
		isAsync: async,
		topic:   topic,
	}

	if async {
		if ap, ok := producer.(sarama.AsyncProducer); ok {
			kp.async = ap
		}
	} else {
		if sp, ok := producer.(sarama.SyncProducer); ok {
			kp.sync = sp
		}
	}
	return kp
}

// NewKafkaProducerWithBrokers 使用 brokers 创建 Sarama producer。
//
// - config 为空时会使用默认配置（同步模式会开启 Return.Successes）
// - async=true 会创建 AsyncProducer；async=false 创建 SyncProducer
func NewKafkaProducerWithBrokers(brokers []string, config *sarama.Config, topic string, async bool) (*KafkaProducer, error) {
	if len(brokers) == 0 {
		return nil, errors.New("kafka brokers cannot be empty")
	}
	if config == nil {
		config = sarama.NewConfig()
	}

	if async {
		ap, err := sarama.NewAsyncProducer(brokers, config)
		if err != nil {
			return nil, err
		}
		return &KafkaProducer{async: ap, isAsync: true, topic: topic}, nil
	}

	// SyncProducer 需要 Return.Successes=true
	config.Producer.Return.Successes = true
	sp, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}
	return &KafkaProducer{sync: sp, isAsync: false, topic: topic}, nil
}

// Publish 实现 MQProducer 接口。
func (p *KafkaProducer) Publish(ctx context.Context, topic string, message []byte) error {
	if topic == "" {
		topic = p.topic
	}
	if topic == "" {
		return errors.New("kafka topic cannot be empty")
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(message),
	}

	if p.isAsync {
		if p.async == nil {
			return errors.New("kafka async producer is nil")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case p.async.Input() <- msg:
			return nil
		}
	}

	if p.sync == nil {
		return errors.New("kafka sync producer is nil")
	}
	_, _, err := p.sync.SendMessage(msg)
	return err
}

// Close 实现 MQProducer 接口。
func (p *KafkaProducer) Close() error {
	if p.isAsync {
		if p.async == nil {
			return nil
		}
		return p.async.Close()
	}
	if p.sync == nil {
		return nil
	}
	return p.sync.Close()
}
