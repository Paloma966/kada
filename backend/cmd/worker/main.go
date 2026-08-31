package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/segmentio/kafka-go"

	"github.com/chun/kada-backend/config"
	"github.com/chun/kada-backend/internal/infra"
	"github.com/chun/kada-backend/internal/mq"
	"github.com/chun/kada-backend/internal/service"
)

// maxDeliveryAttempts 单条消息最大处理尝试次数。
// 超过则提交 offset 丢弃该消息：否则毒消息会让单分区单消费组永久卡死。
const maxDeliveryAttempts = 3

// isPermanentError 判断不可重试的永久性错误（毒消息）。
func isPermanentError(err error) bool {
	// 非法 JSON：重试永远失败
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return true
	}
	// 外键违反（23503）：如链接已删除，重试永远失败
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return true
	}
	return false
}

// attemptTracker 在内存中跟踪每条消息的处理尝试次数（重启后重置，可接受）
type attemptTracker struct {
	mu   sync.Mutex
	seen map[string]int
}

func newAttemptTracker() *attemptTracker {
	return &attemptTracker{seen: make(map[string]int)}
}

func (t *attemptTracker) record(topic string, partition int, offset int64) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	key := fmt.Sprintf("%s-%d-%d", topic, partition, offset)
	t.seen[key]++
	return t.seen[key]
}

func (t *attemptTracker) reset(topic string, partition int, offset int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.seen, fmt.Sprintf("%s-%d-%d", topic, partition, offset))
}

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
	return writer.WriteClick(ctx, e.EventID, e.LinkID, e.IP, e.UserAgent, e.Platform, e.Referer, e.CreatedAt)
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
		// #nosec G706 -- topic 来自环境配置、topicErr 为内部连接错误，非用户输入
		log.Printf("⚠️ ensure kafka topic %q failed (attempt %d/10): %v", topic, attempt, topicErr)
		time.Sleep(2 * time.Second)
	}
	if topicErr != nil {
		// #nosec G706 -- topic 来自环境配置、topicErr 为内部连接错误，非用户输入
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

	// #nosec G706 -- topic/brokers 来自环境配置，非用户输入
	log.Printf("🧵 click-worker consuming topic %q from %s", topic, brokers)
	tracker := newAttemptTracker()
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

		procErr := processClickMessage(m.Value, store)
		if procErr == nil {
			tracker.reset(m.Topic, m.Partition, m.Offset)
			if err := reader.CommitMessages(ctx, m); err != nil {
				log.Printf("commit message failed: %v", err)
			}
			continue
		}

		attempts := tracker.record(m.Topic, m.Partition, m.Offset)
		if isPermanentError(procErr) || attempts >= maxDeliveryAttempts {
			// 毒消息（非法 JSON / 外键违反）或重试超限：提交 offset 跳过，
			// 防止单分区消费组被同一条消息永久卡死。
			log.Printf("dropping click message after %d attempts (permanent error): %v", attempts, procErr)
			if err := reader.CommitMessages(ctx, m); err != nil {
				log.Printf("commit message failed: %v", err)
			}
			tracker.reset(m.Topic, m.Partition, m.Offset)
			continue
		}

		log.Printf("process message failed (attempt %d/%d): %v", attempts, maxDeliveryAttempts, procErr)
		time.Sleep(time.Second) // 短暂退避，避免空转
	}
}
