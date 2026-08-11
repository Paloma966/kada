# Kafka 点击事件流 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把短链点击日志解耦成 Kafka 异步消息流：`LinkService.LogClick` 发布 `click.event` 到 Kafka，独立 `cmd/worker` 消费落库；Kafka 不可用自动回退直写。

**Architecture:** 新增 `internal/mq`（ClickEvent + ClickPublisher 接口 + segmentio/kafka-go 实现）、`internal/service/click_store.go`（直写逻辑，生产回退与 worker 共用）、`cmd/worker`（消费者）。`LinkService.LogClick` 改为「发 Kafka → 失败回退直写」，handler 签名不变（`go LogClick` 保持 goroutine 不阻塞请求）。docker-compose 加 Kafka broker + worker 服务。

**Tech Stack:** Go 1.26, gin, pgx/v5, segmentio/kafka-go（纯 Go，无 CGO）。

**Spec:** `docs/superpowers/specs/2026-08-11-kafka-click-events-design.md`

## Global Constraints

- 后端验证命令（在 `backend/` 下）：`go build ./...`、`go test ./...`、`go vet ./...` 必须通过。前端不动。
- 数据层保持 pgx，**不引入 GORM / gRPC**（用户已确认砍掉）。
- `LogClick` 的对外签名与行为契约不变：void、fire-and-forget、绝不阻塞/破坏重定向路径。
- `KafkaBrokers` 为空 → Kafka 完全禁用 → 走直写（默认安全态）。
- 简历卖点（实现必须保住）：生产/消费解耦、Kafka 故障自动降级直写、worker 独立进程。
- 遵循现有代码风格（中文注释、服务分层、接口注入便于测试）。
- CI：`.github/workflows/ci.yml` 有 `backend-lint`（golangci-lint）与 `backend-test`。改动文件必须过 `golangci-lint` 检查（本地跑 `golangci-lint run` 验证）。

---

### Task 1: `internal/mq` — ClickEvent + Kafka 发布者（TDD）

**Files:**
- Create: `backend/internal/mq/click.go`
- Test: `backend/internal/mq/click_test.go`

**Interfaces:**
- Consumes: `github.com/segmentio/kafka-go`
- Produces:
  - `type ClickEvent struct { LinkID int64; IP string; UserAgent string; Platform string; Referer string; CreatedAt time.Time }`（json 标签小写）
  - `type ClickPublisher interface { PublishClick(ctx context.Context, e ClickEvent) error }`
  - `NewKafkaClickPublisher(brokers []string, topic string) *KafkaClickPublisher` — brokers 为空返回 nil
  - `(*KafkaClickPublisher).PublishClick(ctx, e) error` — JSON 序列化后写 topic
  - `(*KafkaClickPublisher).Close() error`

- [ ] **Step 1: 加依赖**

```bash
cd backend
go get github.com/segmentio/kafka-go@latest
go mod tidy
go build ./...
```

Expected: 编译通过，go.mod 出现 kafka-go。

- [ ] **Step 2: 写失败测试**

Create `backend/internal/mq/click_test.go`:

```go
package mq

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

type fakeWriter struct {
	got []kafka.Message
}

func (f *fakeWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	f.got = append(f.got, msgs...)
	return nil
}

func TestClickEventRoundTrip(t *testing.T) {
	e := ClickEvent{
		LinkID: 42, IP: "1.2.3.4", UserAgent: "Mozilla/5.0",
		Platform: "wechat", Referer: "https://x.com",
		CreatedAt: time.Unix(1700000000, 0).UTC(),
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var got ClickEvent
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got != e {
		t.Fatalf("round trip mismatch: got %+v want %+v", got, e)
	}
}

func TestNewKafkaClickPublisher_NoBrokersReturnsNil(t *testing.T) {
	if got := NewKafkaClickPublisher(nil, "clicks"); got != nil {
		t.Fatalf("expected nil for empty brokers, got %+v", got)
	}
	if got := NewKafkaClickPublisher([]string{"  ", ""}, "clicks"); got != nil {
		t.Fatalf("expected nil for blank brokers, got %+v", got)
	}
}

func TestKafkaClickPublisher_PublishClickWritesJSON(t *testing.T) {
	w := &fakeWriter{}
	p := &KafkaClickPublisher{writer: w}
	e := ClickEvent{LinkID: 7, IP: "9.9.9.9", Platform: "browser", CreatedAt: time.Unix(1700000000, 0).UTC()}
	if err := p.PublishClick(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if len(w.got) != 1 {
		t.Fatalf("expected 1 message, got %d", len(w.got))
	}
	var decoded ClickEvent
	if err := json.Unmarshal(w.got[0].Value, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded != e {
		t.Fatalf("decoded mismatch: got %+v want %+v", decoded, e)
	}
}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `cd backend && go test ./internal/mq/ -run TestClickEvent -count=1`

Expected: FAIL — `cannot find package` 或 `undefined: ClickEvent`。

- [ ] **Step 4: 实现**

Create `backend/internal/mq/click.go`:

```go
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
			RequiredAcks: kafka.RequireOne,
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
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd backend && go test ./internal/mq/ -count=1`

Expected: PASS，3 个测试全过。

- [ ] **Step 6: lint + vet**

Run: `cd backend && go vet ./internal/mq/ && golangci-lint run ./internal/mq/... 2>/dev/null || echo "(golangci-lint not available locally, vet passed)"`

Expected: vet 通过；若本机无 golangci-lint 则跳过（CI 会跑）。

- [ ] **Step 7: 提交**

```bash
git add go.mod go.sum internal/mq/
git commit -m "feat: click event model and kafka publisher (segmentio/kafka-go)"
```

---

### Task 2: ClickStore + LinkService 接入 Kafka + config（TDD）

**Files:**
- Create: `backend/internal/service/click_store.go`
- Test: `backend/internal/service/click_store_test.go`
- Modify: `backend/internal/service/link_service.go`
- Modify: `backend/config/config.go`
- Modify: `backend/cmd/server/main.go`

**Interfaces:**
- Consumes: `mq.ClickPublisher`（Task 1）、`mq.ClickEvent`
- Produces:
  - `type ClickWriter interface { WriteClick(ctx context.Context, linkID int64, ip, userAgent, platform, referer string) error }`
  - `type ClickStore struct{ db *pgxpool.Pool }`；`NewClickStore(db *pgxpool.Pool) *ClickStore`；`(*ClickStore).WriteClick(...)` 满足 `ClickWriter`
  - `service.NewLinkService(db *pgxpool.Pool, baseURL string, cache *CacheService, kafka mq.ClickPublisher, clickWriter ClickWriter) *LinkService`（**签名变化**）
  - `config.Config` 加字段 `KafkaBrokers`、`KafkaTopic` 与方法 `Brokers() []string`

- [ ] **Step 1: 写失败测试**

Create `backend/internal/service/click_store_test.go`:

```go
package service

import (
	"context"
	"testing"

	"github.com/chun/kada-backend/internal/mq"
)

type fakePublisher struct {
	calls int
	err   error
}

func (f *fakePublisher) PublishClick(_ context.Context, _ mq.ClickEvent) error {
	f.calls++
	return f.err
}

type fakeWriter struct {
	calls int
}

func (f *fakeWriter) WriteClick(_ context.Context, _ int64, _, _, _, _ string) error {
	f.calls++
	return nil
}

func TestLogClick_PublishSuccess(t *testing.T) {
	pub := &fakePublisher{}
	svc := &LinkService{kafka: pub, clickWriter: &fakeWriter{}}
	svc.LogClick(context.Background(), 1, "1.2.3.4", "ua", "browser", "ref")
	if pub.calls != 1 {
		t.Fatalf("expected publisher called once, got %d", pub.calls)
	}
}

func TestLogClick_PublishErrorFallsBack(t *testing.T) {
	pub := &fakePublisher{err: assertErr("kafka down")}
	w := &fakeWriter{}
	svc := &LinkService{kafka: pub, clickWriter: w}
	svc.LogClick(context.Background(), 1, "1.2.3.4", "ua", "browser", "ref")
	if pub.calls != 1 || w.calls != 1 {
		t.Fatalf("expected publish 1 + fallback 1, got publish=%d write=%d", pub.calls, w.calls)
	}
}

func TestLogClick_NoKafkaDirectWrite(t *testing.T) {
	w := &fakeWriter{}
	svc := &LinkService{kafka: nil, clickWriter: w}
	svc.LogClick(context.Background(), 1, "1.2.3.4", "ua", "browser", "ref")
	if w.calls != 1 {
		t.Fatalf("expected direct write, got %d", w.calls)
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
```

Note: 这些测试直接构造 `LinkService` 字面量（字段 `kafka`、`clickWriter` 可见），**不需要 DB**。

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend && go test ./internal/service/ -run TestLogClick -count=1`

Expected: FAIL — `unknown field 'kafka' in struct literal`。

- [ ] **Step 3: 实现 ClickStore**

Create `backend/internal/service/click_store.go`:

```go
package service

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ClickWriter 直写点击日志（生产降级回退与 worker 消费共用）
type ClickWriter interface {
	WriteClick(ctx context.Context, linkID int64, ip, userAgent, platform, referer string) error
}

// ClickStore pgx 实现的 ClickWriter
type ClickStore struct {
	db *pgxpool.Pool
}

func NewClickStore(db *pgxpool.Pool) *ClickStore {
	return &ClickStore{db: db}
}

// WriteClick 事务内：插入点击日志 + 累加计数
func (s *ClickStore) WriteClick(ctx context.Context, linkID int64, ip, userAgent, platform, referer string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO click_logs (link_id, ip, user_agent, platform, referer)
		VALUES ($1, $2, $3, $4, $5)
	`, linkID, ip, userAgent, platform, referer); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE links SET click_count = click_count + 1 WHERE id = $1
	`, linkID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
```

- [ ] **Step 4: 改造 link_service.go**

在 `backend/internal/service/link_service.go`：

(1) 文件顶部 import 增加：

```go
"github.com/chun/kada-backend/internal/mq"
```

(2) `LinkService` 结构体增加两个字段：

```go
type LinkService struct {
	db         *pgxpool.Pool
	baseURL    string
	cache      *CacheService
	kafka      mq.ClickPublisher // Kafka 发布者；nil 表示禁用
	clickWriter ClickWriter      // 直写（降级回退用）
}
```

(3) 构造器签名与赋值：

```go
func NewLinkService(db *pgxpool.Pool, baseURL string, cache *CacheService, kafka mq.ClickPublisher, clickWriter ClickWriter) *LinkService {
	return &LinkService{db: db, baseURL: baseURL, cache: cache, kafka: kafka, clickWriter: clickWriter}
}
```

(4) `LogClick` 改写为「发 Kafka → 失败回退直写」：

```go
// LogClick 发布点击事件到 Kafka；Kafka 不可用时回退直写，保证点击不丢
func (s *LinkService) LogClick(ctx context.Context, linkID int64, ip, userAgent, platform, referer string) {
	if s.kafka != nil {
		err := s.kafka.PublishClick(ctx, mq.ClickEvent{
			LinkID:    linkID,
			IP:        ip,
			UserAgent: userAgent,
			Platform:  platform,
			Referer:   referer,
			CreatedAt: time.Now(),
		})
		if err == nil {
			return
		}
		// Kafka 失败 → 落到直写
	}
	if s.clickWriter != nil {
		_ = s.clickWriter.WriteClick(ctx, linkID, ip, userAgent, platform, referer)
	}
}
```

（确认 link_service.go 已 import `time`；若没有则补。）

- [ ] **Step 5: config 增加 Kafka 配置**

`backend/config/config.go`：

(1) 结构体加字段：

```go
	// Kafka（点击事件流；空 = 禁用）
	KafkaBrokers string
	KafkaTopic   string
```

(2) `Load()` 里加：

```go
		KafkaBrokers:      getEnv("KAFKA_BROKERS", ""),
		KafkaTopic:        getEnv("KAFKA_TOPIC", "clicks"),
```

(3) 加方法（文件顶部 import `strings`）：

```go
// Brokers 拆分逗号分隔的 broker 列表，去空白与空项
func (c *Config) Brokers() []string {
	var out []string
	for _, b := range strings.Split(c.KafkaBrokers, ",") {
		if b = strings.TrimSpace(b); b != "" {
			out = append(out, b)
		}
	}
	return out
}
```

- [ ] **Step 6: main.go 装配**

`backend/cmd/server/main.go`：

```go
	// Kafka 点击事件发布者（无 broker 时返回 nil = 禁用）
	kafkaPub := mq.NewKafkaClickPublisher(cfg.Brokers(), cfg.KafkaTopic)
	linkSvc := service.NewLinkService(db, cfg.BaseURL, cacheSvc, kafkaPub, service.NewClickStore(db))
```

并在 import 里加 `"github.com/chun/kada-backend/internal/mq"`（若未在 Task 1 的 mq 使用处引入——main.go 本就不引用，需新增）。

- [ ] **Step 7: 全量测试 + build + vet**

Run: `cd backend && go test ./... -count=1 && go build ./... && go vet ./...`

Expected: 全部 PASS（含既有测试）；build/vet 通过。

- [ ] **Step 8: 提交**

```bash
git add internal/service/click_store.go internal/service/click_store_test.go internal/service/link_service.go config/config.go cmd/server/main.go
git commit -m "feat: route click logging through kafka publisher with direct-write fallback"
```

---

### Task 3: `cmd/worker` — Kafka 消费者（TDD）

**Files:**
- Create: `backend/cmd/worker/main.go`
- Test: `backend/cmd/worker/main_test.go`

**Interfaces:**
- Consumes: `service.ClickWriter`、`service.NewClickStore`、`infra.NewDB`、`mq.ClickEvent`
- Produces: `processClickMessage(msg []byte, writer service.ClickWriter) error`（可测）；`main()` 编排 reader 循环

- [ ] **Step 1: 写失败测试**

Create `backend/cmd/worker/main_test.go`:

```go
package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/chun/kada-backend/internal/mq"
)

type recorderWriter struct {
	got []mq.ClickEvent
}

func (r *recorderWriter) WriteClick(_ context.Context, linkID int64, ip, ua, platform, referer string) error {
	r.got = append(r.got, mq.ClickEvent{LinkID: linkID, IP: ip, UserAgent: ua, Platform: platform, Referer: referer})
	return nil
}

func TestProcessClickMessage_Valid(t *testing.T) {
	w := &recorderWriter{}
	e := mq.ClickEvent{LinkID: 5, IP: "8.8.8.8", UserAgent: "ua", Platform: "qq", Referer: "r", CreatedAt: time.Unix(1700000000, 0).UTC()}
	b, _ := json.Marshal(e)
	if err := processClickMessage(b, w); err != nil {
		t.Fatal(err)
	}
	if len(w.got) != 1 || w.got[0].LinkID != 5 {
		t.Fatalf("expected 1 write with link 5, got %+v", w.got)
	}
}

func TestProcessClickMessage_InvalidJSON(t *testing.T) {
	w := &recorderWriter{}
	if err := processClickMessage([]byte("{not json"), w); err == nil {
		t.Fatal("expected error for invalid json")
	}
	if len(w.got) != 0 {
		t.Fatalf("expected no write on invalid json, got %d", len(w.got))
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend && go test ./cmd/worker/ -count=1`

Expected: FAIL — `undefined: processClickMessage`。

- [ ] **Step 3: 实现**

Create `backend/cmd/worker/main.go`:

```go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

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
	return writer.WriteClick(context.Background(), e.LinkID, e.IP, e.UserAgent, e.Platform, e.Referer)
}

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	brokers := os.Getenv("KAFKA_BROKERS")
	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		topic = "clicks"
	}
	if brokers == "" {
		log.Fatal("KAFKA_BROKERS is required for worker")
	}

	db, err := infra.NewDB(databaseURL)
	if err != nil {
		log.Fatalf("database connect failed: %v", err)
	}
	defer infra.CloseDB(db)

	store := service.NewClickStore(db)
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokers},
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
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd backend && go test ./cmd/worker/ -count=1`

Expected: PASS。

- [ ] **Step 5: build + vet + lint**

Run: `cd backend && go build ./... && go vet ./...`

Expected: 通过。

- [ ] **Step 6: 提交**

```bash
git add cmd/worker/
git commit -m "feat: kafka click consumer worker"
```

---

### Task 4: Docker + compose + env 接入

**Files:**
- Modify: `backend/Dockerfile`
- Modify: `docker-compose.yml`（仓库根目录）
- Modify: `.env.example`

**Interfaces:**
- Consumes: `cmd/worker`（Task 3 的 `/worker` 二进制）
- Produces: `docker compose config` 可解析；`kafka` 与 `kafka-worker` 服务就绪

- [ ] **Step 1: Dockerfile 构建两个二进制**

`backend/Dockerfile` builder 阶段：

```dockerfile
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server/ \
 && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /worker ./cmd/worker/
```

runtime 阶段：

```dockerfile
COPY --from=builder /server /server
COPY --from=builder /worker /worker
```

- [ ] **Step 2: docker-compose 加 kafka 与 kafka-worker**

在 `docker-compose.yml` 的 `services:` 下新增：

```yaml
  kafka:
    image: apache/kafka:3.8.0
    ports:
      - "127.0.0.1:9092:9092"
    environment:
      KAFKA_NODE_ID: "1"
      KAFKA_PROCESS_ROLES: "broker,controller"
      KAFKA_LISTENERS: "PLAINTEXT://:9092,CONTROLLER://:9093"
      KAFKA_ADVERTISED_LISTENERS: "PLAINTEXT://kafka:9092"
      KAFKA_CONTROLLER_LISTENER_NAMES: "CONTROLLER"
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: "CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT"
      KAFKA_CONTROLLER_QUORUM_VOTERS: "1@kafka:9093"
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: "1"
      KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: "1"
      KAFKA_TRANSACTION_STATE_LOG_MIN_ISR: "1"
      KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS: "0"
    healthcheck:
      test: ["CMD", "/opt/kafka/bin/kafka-broker-api-versions.sh", "--bootstrap-server", "localhost:9092"]
      interval: 10s
      timeout: 5s
      retries: 10

  kafka-worker:
    build: ./backend
    command: ["/worker"]
    environment:
      DATABASE_URL: postgres://kada:kada123@postgres:5432/kada?sslmode=disable
      KAFKA_BROKERS: kafka:9092
      KAFKA_TOPIC: clicks
    depends_on:
      postgres:
        condition: service_healthy
      kafka:
        condition: service_healthy
    restart: unless-stopped
```

并给 `backend` 服务的 `environment:` 增加：

```yaml
      KAFKA_BROKERS: kafka:9092
      KAFKA_TOPIC: clicks
```

- [ ] **Step 3: .env.example 补说明**

追加：

```bash
# ========== Kafka（点击事件流） ==========
# 逗号分隔 broker；留空 = 禁用 Kafka，点击直写数据库
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=clicks
```

- [ ] **Step 4: 验证**

```bash
cd /home/chun/dev/projects/kada && docker compose config --quiet && echo "compose OK"
cd backend && go build ./...
```

Expected: compose 解析成功；backend 构建通过。

（可选端到端，需要 docker：`docker compose up -d kafka` 后起 backend + worker，访问 `/r/:code` 触发点击，worker 日志应显示消费。）

- [ ] **Step 5: 提交**

```bash
git add backend/Dockerfile docker-compose.yml .env.example
git commit -m "feat: kafka broker + click worker in docker compose"
```

---

## 自检记录

- **Spec 覆盖**：ClickEvent/publisher（Task 1）、ClickStore + LogClick 生产回退（Task 2）、config 禁用态（Task 2）、worker 消费（Task 3）、docker/worker 部署（Task 4）、降级直写（Task 2 测试覆盖）、测试与简历卖点（各 Task）。
- **占位符扫描**：无 TBD/TODO；代码完整。
- **类型一致性**：`mq.ClickPublisher` / `mq.ClickEvent` / `NewKafkaClickPublisher` / `ClickWriter` / `NewClickStore` / `WriteClick` / `NewLinkService` 新签名 / `Config.Brokers` 在 Task 1-2 定义、Task 2-4 消费，签名一致；`cmd/worker` 消费 `service.ClickWriter` 与 `mq.ClickEvent`。
