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
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/chun/kada-backend/config"
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
	return writer.WriteClick(ctx, e.LinkID, e.IP, e.UserAgent, e.Platform, e.Referer, e.CreatedAt)
}

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	brokers := os.Getenv("KAFKA_BROKERS")
	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		topic = "clicks"
	}
	brokerList := config.SplitBrokers(brokers)
	if len(brokerList) == 0 {
		log.Fatal("KAFKA_BROKERS is required for worker")
	}

	// 加入消费组前先确保 topic 存在，避免 #585 空分配卡死。
	// 冷启动时 healthcheck 可能早于 controller 就绪，因此带退避重试；
	// 若最终仍失败则退出（否则继续会拿到空分配永久卡死）。
	var topicErr error
	for attempt := 1; attempt <= 10; attempt++ {
		topicCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		topicErr = ensureTopic(topicCtx, brokerList, topic)
		cancel()
		if topicErr == nil {
			break
		}
		log.Printf("⚠️ ensure kafka topic %q failed (attempt %d/10): %v", topic, attempt, topicErr)
		time.Sleep(2 * time.Second)
	}
	if topicErr != nil {
		log.Fatalf("cannot ensure kafka topic %q exists: %v", topic, topicErr)
	}

	db, err := infra.NewDB(databaseURL)
	if err != nil {
		log.Fatalf("database connect failed: %v", err)
	}
	defer infra.CloseDB(db)

	store := service.NewClickStore(db)
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokerList,
		Topic:   topic,
		GroupID: "click-worker",
		// MinBytes 保持 1：单条点击事件约 200B，设大值会攒批导致最多 MaxWait(10s) 的消费延迟
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("🧵 click-worker consuming topic %q from %s", topic, brokers)
	for {
		// 用 FetchMessage 而非 ReadMessage：ReadMessage 会自动提交 offset，
		// 处理失败也会被提交导致点击永久丢失。这里仅在落库成功后才提交。
		m, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("shutdown signal received")
				return
			}
			log.Printf("fetch message failed: %v", err)
			continue
		}
		if err := processClickMessage(m.Value, store); err != nil {
			// 处理失败不提交 offset，消息会重新投递，避免点击丢失
			log.Printf("process message failed: %v", err)
			continue
		}
		if err := reader.CommitMessages(ctx, m); err != nil {
			log.Printf("commit message failed: %v", err)
		}
	}
}
