package mq

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// ClickEvent is a short-link click event (message payload shared by producer and consumer)
type ClickEvent struct {
	EventID   string    `json:"event_id"` // idempotency key used by the worker for deduplication
	LinkID    int64     `json:"link_id"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Platform  string    `json:"platform"`
	Referer   string    `json:"referer"`
	CreatedAt time.Time `json:"created_at"`
}

// ClickPublisher publishes click events (producer-side interface, eases test mocks and degradation)
type ClickPublisher interface {
	PublishClick(ctx context.Context, e ClickEvent) error
}

// messageWriter narrows kafka.Writer's write interface so tests can inject a fake
type messageWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

// KafkaClickPublisher is the segmentio/kafka-go based implementation
type KafkaClickPublisher struct {
	writer messageWriter
}

// NewKafkaClickPublisher builds a publisher; returns nil when brokers is empty (Kafka disabled)
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

// PublishClick serializes a ClickEvent and writes it to the topic
func (p *KafkaClickPublisher) PublishClick(ctx context.Context, e ClickEvent) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{Value: b})
}

// Close closes the underlying writer
func (p *KafkaClickPublisher) Close() error {
	if w, ok := p.writer.(*kafka.Writer); ok {
		return w.Close()
	}
	return nil
}
