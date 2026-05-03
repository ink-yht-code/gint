// Copyright 2025 ink-yht-code
//
// Proprietary License

package logger

import (
	"context"

	"github.com/ink-yht-code/gint/mq"
)

// MQProducerAdapter 将 mq.Producer 适配为 logger.MQProducer。
//
// 用于将 gint/mq 的生产者直接接入日志 MQ 写入器，无需重复创建连接。
//
//	p, _ := kafka.NewMQ(brokers, nil).Producer("logs")
//	writer := logger.NewMQLogWriter(logger.MQLogWriterConfig{
//	    Producer: logger.NewMQProducerAdapter(p),
//	    Topic:    "logs",
//	    Service:  "order-service",
//	})
//	logger.SetDBLogWriter(writer)
type MQProducerAdapter struct {
	producer mq.Producer
	topic    string
}

// NewMQProducerAdapter 创建适配器，将 mq.Producer 包装为 logger.MQProducer
func NewMQProducerAdapter(producer mq.Producer, topic string) MQProducer {
	return &MQProducerAdapter{producer: producer, topic: topic}
}

// Publish 实现 logger.MQProducer 接口
func (a *MQProducerAdapter) Publish(ctx context.Context, _ string, message []byte) error {
	_, err := a.producer.Send(ctx, &mq.Message{
		Topic: a.topic,
		Value: message,
	})
	return err
}

// Close 实现 logger.MQProducer 接口
func (a *MQProducerAdapter) Close() error {
	return a.producer.Close()
}
