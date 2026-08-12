package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/chun/kada-backend/internal/infra"
	"github.com/chun/kada-backend/internal/mq"
	"github.com/chun/kada-backend/internal/service"
)

// ensureTopic 幂等创建 topic（numPartitions=1 / replicationFactor=1）。
// 必须在创建 reader 之前调用：若 reader 在 topic 自动创建期间加入消费组，
// kafka-go 会拿到空分配并永久卡死（segmentio/kafka-go#585）。
func ensureTopic(ctx context.Context, brokerList []string, topic string) error {
	conn, err := kafka.DialContext(ctx, "tcp", brokerList[0])
	if err != nil {
		return err
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return err
	}
	ctrlAddr := net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port))
	ctrlConn, err := kafka.DialContext(ctx, "tcp", ctrlAddr)
	if err != nil {
		return err
	}
	defer ctrlConn.Close()

	return ctrlConn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
}

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

	// 加入消费组前先确保 topic 存在，避免 #585 空分配卡死
	topicCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := ensureTopic(topicCtx, brokerList, topic); err != nil {
		log.Printf("⚠️ ensure kafka topic %q failed: %v（worker 将继续尝试消费）", topic, err)
	}
	cancel()

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
