package mq

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// ClickEvent 短链点击事件（生产/消费共用的消息载荷）
type ClickEvent struct {
	LinkID    int64     `json:"link_id"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Platform  string    `json:"platform"`
	Referer   string    `json:"referer"`
	CreatedAt time.Time `json:"created_at"`
}

// ClickPublisher 发布点击事件（生产端接口，便于测试 mock 与降级）
type ClickPublisher interface {
	PublishClick(ctx context.Context, e ClickEvent) error
}

// messageWriter 收窄 kafka.Writer 的写接口，便于单测注入 fake
type messageWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

// KafkaClickPublisher 基于 segmentio/kafka-go 的实现
type KafkaClickPublisher struct {
	writer messageWriter
}

// NewKafkaClickPublisher 构造发布者；brokers 为空时返回 nil（表示 Kafka 禁用）
func NewKafkaClickPublisher(brokers []string, topic string) *KafkaClickPublisher {
	var valid []string
	for _, b := range brokers {
		if b = strings.TrimSpace(b); b != "" {
			valid = append(valid, b)
		}
	}
	if len(valid) == 0 {
		return nil
	}
	return &KafkaClickPublisher{
		writer: kafka.NewWriter(kafka.WriterConfig{
			Brokers:      valid,
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: int(kafka.RequireOne),
		}),
	}
}

// PublishClick 序列化 ClickEvent 并写入 topic
func (p *KafkaClickPublisher) PublishClick(ctx context.Context, e ClickEvent) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{Value: b})
}

// Close 关闭底层 writer
func (p *KafkaClickPublisher) Close() error {
	if w, ok := p.writer.(*kafka.Writer); ok {
		return w.Close()
	}
	return nil
}
