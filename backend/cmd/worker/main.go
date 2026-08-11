package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/chun/kada-backend/internal/infra"
	"github.com/chun/kada-backend/internal/mq"
	"github.com/chun/kada-backend/internal/service"
)

// processClickMessage 反序列化并落库一条点击消息
func processClickMessage(msg []byte, writer service.ClickWriter) error {
	var e mq.ClickEvent
	if err := json.Unmarshal(msg, &e); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return writer.WriteClick(ctx, e.LinkID, e.IP, e.UserAgent, e.Platform, e.Referer)
}

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	brokers := os.Getenv("KAFKA_BROKERS")
	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		topic = "clicks"
	}
	var brokerList []string
	for _, b := range strings.Split(brokers, ",") {
		if b = strings.TrimSpace(b); b != "" {
			brokerList = append(brokerList, b)
		}
	}
	if len(brokerList) == 0 {
		log.Fatal("KAFKA_BROKERS is required for worker")
	}

	db, err := infra.NewDB(databaseURL)
	if err != nil {
		log.Fatalf("database connect failed: %v", err)
	}
	defer infra.CloseDB(db)

	store := service.NewClickStore(db)
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokerList,
		Topic:    topic,
		GroupID:  "click-worker",
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("🧵 click-worker consuming topic %q from %s", topic, brokers)
	for {
		m, err := reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("shutdown signal received")
				return
			}
			log.Printf("read message failed: %v", err)
			continue
		}
		if err := processClickMessage(m.Value, store); err != nil {
			log.Printf("process message failed: %v", err)
			continue
		}
	}
}
