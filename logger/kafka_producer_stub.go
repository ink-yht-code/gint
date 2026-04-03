//go:build !kafka

package logger

import (
	"context"
	"errors"
)

// KafkaProducer Kafka 生产者（默认 stub）。
//
// 如果要启用真正可用的 Kafka 后端（Sarama），请使用 `-tags kafka` 构建。
// 例如：`go test -tags kafka ./...` 或 `go build -tags kafka ./...`
type KafkaProducer struct{}

// NewKafkaProducer 创建 Kafka 生产者（stub）。
//
// 说明：默认构建不包含 Kafka 依赖，因此这里会返回一个“不可用”的 producer；
// 调用 Publish/Close 会返回明确错误。
func NewKafkaProducer(_ any, _ string, _ bool) *KafkaProducer {
	return &KafkaProducer{}
}

func (p *KafkaProducer) Publish(ctx context.Context, topic string, message []byte) error {
	_ = p
	_ = ctx
	_ = topic
	_ = message
	return errors.New("kafka producer is not enabled: build with -tags kafka to use Sarama implementation")
}

func (p *KafkaProducer) Close() error {
	_ = p
	return errors.New("kafka producer is not enabled: build with -tags kafka to use Sarama implementation")
}
