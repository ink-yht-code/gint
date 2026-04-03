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
	"sync"
	"time"
)

// LogMessage 日志消息结构（发送到 MQ 的消息格式）
type LogMessage struct {
	Level     string         `json:"level"`     // 日志级别
	Module    string         `json:"module"`    // 模块名（服务名）
	Message   string         `json:"message"`   // 日志消息
	Fields    map[string]any `json:"fields"`    // 日志字段
	TraceID   string         `json:"trace_id"`  // 链路追踪 ID
	UserID    string         `json:"user_id"`   // 用户 ID
	Timestamp time.Time      `json:"timestamp"` // 时间戳
	Service   string         `json:"service"`   // 服务名
	Host      string         `json:"host"`      // 主机名
}

// MQProducer MQ 生产者接口
// 用户需要根据实际使用的 MQ 实现此接口
type MQProducer interface {
	// Publish 发布消息到指定主题
	Publish(ctx context.Context, topic string, message []byte) error
	// Close 关闭生产者
	Close() error
}

// MQLogWriter MQ 日志写入器
// 实现 DBLogWriter 接口，将日志发送到 MQ
type MQLogWriter struct {
	producer      MQProducer
	topic         string
	service       string
	host          string
	buffer        chan LogMessage
	bufferSize    int
	batchSize     int
	flushInterval time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// MQLogWriterConfig MQ 日志写入器配置
type MQLogWriterConfig struct {
	// Producer MQ 生产者实例
	Producer MQProducer
	// Topic 日志主题/队列名
	Topic string
	// Service 服务名
	Service string
	// Host 主机名（可选，默认自动获取）
	Host string
	// BufferSize 异步缓冲区大小
	BufferSize int
	// BatchSize 批量发送大小
	BatchSize int
	// FlushInterval 刷新间隔
	FlushInterval time.Duration
}

// NewMQLogWriter 创建 MQ 日志写入器
func NewMQLogWriter(cfg MQLogWriterConfig) *MQLogWriter {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 1000
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 100 * time.Millisecond
	}

	ctx, cancel := context.WithCancel(context.Background())

	w := &MQLogWriter{
		producer:      cfg.Producer,
		topic:         cfg.Topic,
		service:       cfg.Service,
		host:          cfg.Host,
		buffer:        make(chan LogMessage, cfg.BufferSize),
		bufferSize:    cfg.BufferSize,
		batchSize:     cfg.BatchSize,
		flushInterval: cfg.FlushInterval,
		ctx:           ctx,
		cancel:        cancel,
	}

	// 启动异步发送协程
	w.wg.Add(1)
	go w.run()

	return w
}

// WriteLog 实现 DBLogWriter 接口
func (w *MQLogWriter) WriteLog(level, module, message string, fields map[string]any, traceID, userID string) {
	logMsg := LogMessage{
		Level:     level,
		Module:    module,
		Message:   message,
		Fields:    fields,
		TraceID:   traceID,
		UserID:    userID,
		Timestamp: time.Now(),
		Service:   w.service,
		Host:      w.host,
	}

	// 非阻塞写入缓冲区
	select {
	case w.buffer <- logMsg:
	default:
		// 缓冲区满，丢弃日志（避免阻塞业务）
		// 可以选择记录到本地文件作为降级
	}
}

// run 异步发送协程
func (w *MQLogWriter) run() {
	defer w.wg.Done()

	batch := make([]LogMessage, 0, w.batchSize)
	timer := time.NewTimer(w.flushInterval)
	defer timer.Stop()

	for {
		select {
		case <-w.ctx.Done():
			// 关闭时刷新剩余日志
			if len(batch) > 0 {
				w.sendBatch(batch)
			}
			return

		case msg := <-w.buffer:
			batch = append(batch, msg)
			if len(batch) >= w.batchSize {
				w.sendBatch(batch)
				batch = batch[:0]
				timer.Reset(w.flushInterval)
			}

		case <-timer.C:
			if len(batch) > 0 {
				w.sendBatch(batch)
				batch = batch[:0]
			}
			timer.Reset(w.flushInterval)
		}
	}
}

// sendBatch 批量发送日志
func (w *MQLogWriter) sendBatch(batch []LogMessage) {
	for _, msg := range batch {
		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		// 使用后台上下文，避免请求取消导致日志丢失
		_ = w.producer.Publish(context.Background(), w.topic, data)
	}
}

// Close 关闭写入器
func (w *MQLogWriter) Close() error {
	w.cancel()
	w.wg.Wait()
	return w.producer.Close()
}

/*
// 在 user 服务中初始化：

import (
    "github.com/redis/go-redis/v9"
    "github.com/ink-yht-code/gint/logger"
)

func initLogger() {
    // 1. 创建 Redis 客户端
    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    // 2. 创建 MQ 日志写入器
    mqWriter := logger.NewMQLogWriter(logger.MQLogWriterConfig{
        Producer:      logger.NewRedisStreamProducer(rdb),
        Topic:         "logs",
        Service:       "user-service",
        BufferSize:    1000,
        BatchSize:     100,
        FlushInterval: 100 * time.Millisecond,
    })

    // 3. 初始化 logger
    cfg := logger.Config{
        Level:    "info",
        Format:   "json",
        Output:   "stdout",
        EnableDB: true,
        DBLevel:  "info",
    }
    logger.Init(cfg)

    // 4. 设置 MQ 日志写入器
    logger.SetDBLogWriter(mqWriter)
}

// 在日志服务中消费：

func consumeLogs(rdb *redis.Client) {
    for {
        streams, err := rdb.XRead(ctx, &redis.XReadArgs{
            Streams: []string{"logs", "$"},
            Count:   100,
            Block:   5 * time.Second,
        }).Result()
        // 处理日志消息...
    }
}
*/
